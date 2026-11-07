# Design

## Decisions

### When a region starts an edition

- **Per region, from data, every 5 minutes.** A new `RegionalCupJob` (cron `30 */5 * * * *`)
  calls `engine.StartRegionalCupsIfDue`. It groups the active countries by
  `common.RegionForCountry` and, for each region, reads one state row per country that has a
  tier-1 league (`common.GetRegionalCupCountryStates`):
  - `LastEnteredSeason`: the highest domestic season that country already sent to the cup
    (`international_cup_entries`), 0 if never.
  - `ChampionSeason`, `ChampionTeamId`: its most recent tier-1 champion in `season_history`
    (`final_position = 1`, newest season, lowest `league_id` among several tier-1 groups - the
    same pick the old `GetMostRecentTier1Champions` made).
  - `UnusedChampions`: how many of its tier-1 seasons with a champion are newer than
    `LastEnteredSeason`.
- **Pure decision.** `engine.regionalCupEntrants(states)` returns the entrants, or nil when no
  edition is due. A country is *fresh* when `ChampionSeason > LastEnteredSeason`. An edition is
  due when at least `MinRegionalCupCountries = 2` countries are fresh and either every country is
  fresh or one has `UnusedChampions >= RegionalCupMaxUnusedChampions = 2`. Only fresh countries
  enter, each with its latest champion.
- **Why this fallback.** Waiting for every country makes the slowest one (weekly schedule, just
  activated, or a stuck season) set the pace. Starting once any country has a second unused
  champion bounds the wait at one season of the fastest country and never skips more than one of
  its champions. It follows each region's real pace instead of a fixed number of days and needs no
  timestamp. A champion that does not enter stays fresh for the next edition.
- **One edition at a time.** A region with an unfinished cup (`cups.region = $1 AND completed =
  false`) is skipped. The `world` region (World Cup knockout) is never a country's region, so it is
  never checked.

### Writing an edition

- **One transaction.** `common.CreateRegionalCup(dbCtx, region, name, entries, ties, matchDate)`
  inserts the `cups` row with `season = COALESCE(max(season), 0) + 1` for that region (the
  edition number), one `international_cup_entries` row per entrant and the first round's
  `cup_ties`, then commits. Byes are stored as already won, as `CreateCupTie` does. No national team
  can be an entrant, so no lineup is auto-selected. A failure leaves nothing behind, so a
  half-written cup can never block its region.
- **Pairings** come from the existing `BuildFirstRoundPairings` with
  `classificationPointsForSeeding`, and kick off `cupRoundGap` (24 h) later, as before.

### Schema: `init103_international_cup_entries.sql`

```sql
CREATE TABLE IF NOT EXISTS international_cup_entries (
    cup_id  bigint NOT NULL REFERENCES cups(cup_id) ON DELETE CASCADE,
    country varchar(4) NOT NULL,
    season  int NOT NULL,      -- the country's domestic season whose champion entered
    team_id bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    PRIMARY KEY (cup_id, country)
);
CREATE INDEX IF NOT EXISTS international_cup_entries_country_idx ON international_cup_entries (country, season);
```

Backfill (re-runnable, `ON CONFLICT DO NOTHING`): for every existing regional cup (`region IS NOT
NULL AND region <> 'world'`), each round-1 entrant gets a row with its country and the newest
season it won a tier-1 league. That marks the champions that already played as used, so the
first new edition does not repeat them. A team that won again after its edition is marked used
for that later title too; at worst that country waits one more season.

### World Cup

- `StartInternationalCupsIfDue` becomes `StartWorldCupIfDue`: it still claims the 60-day cycle
  with `ClaimInternationalCupSeason` and calls `StartWorldCupForNewSeason(counter)`, but no longer
  starts regional cups. The column and constant names (`international_cup_season`,
  `InternationalCupCycle`) stay, so the next-World-Cup estimate and existing data are unchanged.
  The engine constant becomes `WorldCupCycleCadence`.
- `StartInternationalCupsForNewSeason`, `startInternationalCupForRegion` and
  `common.GetMostRecentTier1Champions` are removed (no other caller).

### Replicas

- `StartRegionalCupsIfDue` runs under `TryWithSessionAdvisoryLock(NamespacedLockKey(
  LockNamespaceRegionalCupStart = 6, 0))`. A replica that loses the lock skips the tick. Inside it
  each region's states and its "cup running" check are re-read, so two replicas ticking together
  start each edition once. The test `TestStartRegionalCupsIfDue_ConcurrentReplicasStartOneEdition`
  covers it.

### Frontend

- `InternationalCupPage` titles the cup with `cup.titleWithEdition` ("{{name}} - Edition
  {{edition}}" / "{{name}} - Edición {{edition}}").
- `cup.entryCriteria`, `cup.noInternationalCup` and the help answer `help.sections.cup.items[1].a`
  describe the new rule.

No new notification or `TRANSACTION_*` reason. Economy: the prizes are unchanged
(`CupPrizePerRound` x round per tie won, `CupChampionBonus` = 100,000 to the winner, so 130,000
for winning a 3-round, 8-country edition), but they are now paid up to once per domestic season
instead of about once every three daily-schedule seasons - the pace the 60-day clock was meant to
approximate. Only domestic champions take part. `TestEconomySimulation` does not model the
international cup, so its output is unchanged.
