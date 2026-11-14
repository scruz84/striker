# Design

## Context

`engine.startRegionalCupIfDue` reads `common.GetRegionalCupCountryStates`, picks the fresh
countries with `regionalCupEntrants` (unchanged), and writes the edition with
`common.CreateRegionalCup` in one transaction. `international_cup_entries` has one row per
(cup, country), primary key `(cup_id, country)`, and `max(season)` per country decides freshness.
The first round comes from the generic `BuildFirstRoundPairings`, which shuffles and gives byes by
points and puts every bye first, so the bye holders meet each other in round 2.

## Decisions

### 1. Reading the runner-up
`GetRegionalCupCountryStatesStmtSql`'s `latest` lateral also returns the champion's `league_id`. A
second lateral reads `season_history` with `final_position = 2` for that league and season.
`RegionalCupCountryState.RunnerUpTeamId` is 0 when there is none. Freshness is unchanged: it is
still about the champion's season.

### 2. Entrants
`const RegionalCupEntrantsPerCountry = 2` (engine). A new `common.RegionalCupEntrant
{Country, Season, TeamId, FinalPosition}`. The function `regionalCupTeams(states)` expands each
fresh state into its champion and, when present, its runner-up. `CreateRegionalCup` takes
`[]RegionalCupEntrant` and inserts one row per team, `final_position` included, in the same
transaction as before. `InternationalCupEntry` gains `FinalPosition`, and
`GetInternationalCupEntries` orders by country, then position.

### 3. Draw: `engine/regionalcupdraw.go`
`regionalCupFirstRound(entrants []common.RegionalCupEntrant, points map[int64]int, r *rand.Rand)
[][2]int64`, pure:
1. Shuffle, then stable-sort by (champion first, points desc). The first `2P − N` (when N ≠ P) are
   the byes.
2. The rest are split into champions and runners-up, each shuffled. Each champion takes the first
   runner-up of another country. Leftover champions pair with each other, leftover runners-up
   with each other, avoiding one country when possible.
3. A repair pass swaps opponents between two ties while any tie has two teams of one country and a
   swap removes it without adding one.
4. Output order: bye, tie, bye, tie... then the remaining ties. A bye is `[team, ByeTeamId]`.
   `BuildNextRoundPairings` pairs slot 2i with 2i+1, so each bye holder meets a round-1 winner.

In a real tie the home side is the champion, or the side picked first in step 2.

### 4. Migration `resources/sql/init112_international_cup_runners_up.sql`
- `ALTER TABLE international_cup_entries ADD COLUMN IF NOT EXISTS final_position smallint NOT NULL
  DEFAULT 1`: existing rows are champions.
- In a `DO` block, only while the primary key still contains `country`: drop
  `international_cup_entries_pkey` and add `PRIMARY KEY (cup_id, team_id)`.
- Every migration re-runs at startup, and `init103`'s backfill used `ON CONFLICT (cup_id, country)`,
  which needs that old key. It is changed to skip editions that already have entries
  (`NOT EXISTS`) and to use a bare `ON CONFLICT DO NOTHING`. The result on an upgraded database
  is the same.

### 5. Replicas
Nothing new: the same global lock (`LockNamespaceRegionalCupStart`), and the state is re-read
inside it. The concurrent test now expects 4 entries and 2 ties.

No `TRANSACTION_*` reason or notification key is added.
