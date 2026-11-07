# Proposal

## Why

The regional (international) club cups start on a fixed 60-day clock: a daily 03:00 UTC job
claims a new "international cycle" every 60 days and starts one cup per region from each
country's most recent tier-1 champion. That clock was a stand-in for "about one domestic season"
when countries stopped rolling over together, but it is not derived from any real schedule:

- A 10-team league plays 18 matchdays. With daily matches a season lasts about 19 days, so each
  country plays about three seasons per edition. Only the latest champion counts, so two of every
  three champions never play the cup, and a region spends ~57 of every 60 days with no cup.
- The very first claim fires on the first 03:00 tick (the clock starts NULL), usually before any
  country has finished a season. It creates nothing and still pushes the next attempt 60 days out.
- Leagues can have their own schedule (`league_schedule`), so no single fixed period fits every
  region: a weekly league needs ~18 weeks per season.

## What Changes

- **Each region starts its own edition when its countries have new champions**, instead of on the
  60-day clock. A new check runs every 5 minutes. A region starts an edition when no edition of
  its cup is running, at least 2 of its countries have a champion that has not played the cup yet,
  and either every country in the region with a tier-1 league has one, or some country has
  finished two seasons since it last entered (so a slow or brand-new country never holds the
  others back by more than one season of the fastest country).
- **Entrants are recorded.** A new table `international_cup_entries` stores which domestic season's
  champion each country sent to each edition, so "has not played the cup yet" is exact. A
  migration backfills it from the editions that already exist.
- **Edition numbers per region.** A regional cup's `season` becomes that region's edition number
  (1, 2, 3...), shown as "European Cup - Edition 3", instead of the shared global counter.
- **The World Cup keeps its clock.** The daily 03:00 UTC job and its 60-day counter stay, but only
  to start the World Cup every 4th cycle (every 240 days, as today). The estimate of the next World
  Cup is unchanged.
- The empty-state, entry-criteria and help texts (en/es) describe the new rule.

Out of scope:
- Recording an international cup win as a domestic `cup_result` in `season_history`. Today
  `advanceCup` writes "Champion" into the winner's `season_history` row for the cup's `season`
  number, which for an international cup is not a domestic season. It is a separate bug; this
  change does not touch it.
- Seeding byes by the entrants' current-season points (unchanged).
- Changing World Cup frequency.

Decided by the owner:
- Start editions from new champions rather than a fixed period (option chosen in chat).

Open questions for the owner:
- The fallback is "a country finished two seasons since it last entered" rather than the
  "30 days" floated in chat: it adapts to each region's real pace and needs no timestamp. A fixed
  maximum wait can be added later if a region of only weekly leagues waits too long.

## Capabilities

### New Capabilities
- `international-cup`: when a regional club cup edition starts, who enters, how editions are
  numbered and how they are started safely on several replicas.

### Modified Capabilities
- `world-cup`: the 60-day counter only drives the World Cup now.
- `engine-replicas`: the regional cup start runs under its own lock.
- `match-scheduling`: the new 5-minute regional cup check.

## Impact

- Backend: migration `init103_international_cup_entries.sql`; `engine/cups.go`
  (`StartRegionalCupsIfDue`, `regionalCupEntrants`, `StartWorldCupIfDue`), `engine/tasks.go`
  (`RegionalCupJob`), `engine/scheduler.go`, `engine/fmengine.go`; `common/cups.go`
  (`GetRegionalCupCountryStates`, `CreateRegionalCup`), `common/enginelocks.go`
  (`LockNamespaceRegionalCupStart`); `GetMostRecentTier1Champions` is removed.
- Frontend: `features/cup/InternationalCupPage.tsx` (edition title), en/es strings.
- Specs: new `international-cup`; `world-cup`, `engine-replicas`, `match-scheduling` deltas.
