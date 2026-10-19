# Game mechanics

How Striker's core simulation actually works under the hood: when matches get simulated and how
a result is calculated, when and how players' skills change, how a team's money moves, and how an
unowned (CPU-controlled) team resolves the transfer market. Written for whoever is reading the
code later - each section names the exact files/functions to open for the real implementation,
since constants and formulas here will drift out of date faster than this prose does.

See also `ARCHITECTURE.md` for the overall codebase layout and `ECONOMY_ROADMAP.md` for planned
(not yet built) economy features.

## Match simulator

### When it runs

Every league match, cup tie, and friendly is simulated by the same per-minute cron job pattern
(`backend/engine/scheduler.go`, using `go-quartz`), each polling its own table for anything whose
kickoff has arrived:

- League matches: `StartSimulateMatchesScheduler` → `SimulateMatches` (`engine/engine.go`), every
  minute.
- Cup ties: `StartCupScheduler` → `SimulateCupTies` (`engine/cups.go`), every minute.
- Friendlies: `StartFriendlyScheduler` → `SimulateFriendlies` (`engine/friendlies.go`), every
  minute.
- Season rollover check: `StartAdvanceSeasonScheduler` → `AdvanceSeasonIfComplete`
  (`engine/season.go`), every 5 minutes (cheaper to check, so it doesn't need per-minute
  granularity).
- Player birthdays: `StartBirthdayScheduler` → `ProcessBirthdays` (`engine/birthdays.go`), once a
  day (01:00 UTC) - see "Player skill progression" below for how this drives aging independently
  of season rollover.
- CPU team daily activity: `StartAiActivityScheduler` → `RunAiTeamDailyActivity`
  (`engine/aiteams.go`), every 5 minutes - see "How CPU (unowned) teams resolve the transfer
  market" below for why this exists alongside the per-match trigger.

A match becomes eligible the moment its `match_date` is **up to 15 minutes in the future**, not
only once kickoff has strictly passed - see `common.GetSimulableMatchesStmtSql` /
`GetSimulableFriendliesStmtSql` (`WHERE match_date > now() AND match_date < now() + interval '15
minutes'`). The whole 90-minute result is computed instantly at that point, then paced back out
over a live WebSocket feed (`engine/livematch.go`'s `BroadcastMatchLive`) at one simulated minute
per real minute, so a viewer who opens the match page at any point in that window sees something
that looks like it's happening live rather than an already-finished score. The result notification
(`NotifyMatchResult`/`notifyCupTieResult`/`notifyFriendlyResult`) is deliberately deferred until
that broadcast actually finishes (`BroadcastMatchLive`'s `onComplete` callback) - it used to fire
the instant the match was simulated, which meant a notification could arrive spelling out the
final score before the match's own kickoff time had even passed.

Up to `maxConcurrentSimulations` (8, `engine/engine.go`) matches are simulated concurrently via a
buffered channel + `sync.WaitGroup`, each in its own goroutine with a `recover()` so one panic
can't take down the rest of that tick's batch.

**Lineup lock:** since a league match/friendly's lineup is already being read into simulation from
T-15 onward, `SetMatchLineupHandler`/`SetCupTieLineupHandler`/`SetFriendlyLineupHandler`
(`backend/handlers`) all reject a lineup change once `common.IsLineupLocked` reports true - within
`common.LineupLockMinutes` (20, `backend/common/lineuplock.go`) of kickoff, or any time after it -
returning `common.ErrLineupLocked`. This gives a 5-minute margin over the actual T-15 read for
league matches/friendlies; for cup ties (which simulate at kickoff itself, with no early read) it's
simply a flat 20-minute cutoff. The frontend mirrors this with its own `useIsLineupLocked` hook
(`frontend/src/api/useLiveMatch.ts`), swapping each per-match lineup editor for a
`LineupLockedNotice` once locked, so a save is never attempted only to be rejected. The team-wide
default lineup (`SetLineupHandler`, no specific match) is unaffected - it isn't tied to any one
kickoff.

A league's kickoff day/time is admin-configured per league (`common.MatchSchedule` -
`WeekdaysMask`/`Hour`/`Minute`, see `common/leagueschedule.go` and `common/matchschedule.go`) and
fixtures are generated a season at a time (`GenerateMatches`, `engine/engine.go`). A cup tie or
friendly instead kicks off a short fixed delay after being scheduled (accepted challenge, or
paired into a bracket round) rather than on the league's weekly cadence.

Changing a schedule (a league's own override, or the app-wide default) doesn't just affect
fixtures generated afterward - `AdminSetLeagueScheduleHandler`/`AdminClearLeagueScheduleHandler`/
`AdminSetMatchScheduleHandler` (`backend/handlers`) also immediately re-date every currently
unplayed match onto the new schedule (`engine.RescheduleUnplayedMatches`/
`RescheduleUnplayedMatchesForLeagues`), preserving each match's week-number ordering the same way
`ComputeFixtureDates` dates a freshly generated season. A league with its own override is left
alone by an app-wide default change, same as for new fixtures.

### How a result is calculated

`runMatchSimulation` (`engine/simulation.go`) builds each team's *qualification* (`TeamQualification`
- `Attack`/`Defense`/`Midfield` plus left/right-flank variants) from whoever's actually in the
lineup that minute, then runs a fixed pipeline once per simulated minute, 1 to 90
(`SimulationPipelineImpl.Simulate`):

1. **SubstitutionSimulation** - applies any of the manager's scheduled substitutions (see
   "Player skill progression" below for where those minutes end up being credited) due this
   minute.
2. **StaminaSimulation** (`engine/stamina_simulation.go`) - rebuilds `EffectiveHomePlayers`/
   `EffectiveAwayPlayers`, a fatigue-adjusted copy of each team's players that every later stage
   reads instead of the raw `HomePlayers`/`AwayPlayers` (`InitialSimulation`'s qualifications and
   `ScorerSimulation`'s individual goalkeeper/finisher rolls alike). Each player's seven core
   skills are scaled down by `fatigueMultiplier`, which combines how many minutes they've played
   so far this match (from `HomeMinutesByPosition`/`AwayMinutesByPosition`, see "Player skill
   progression" below) with their own `Stamina` (0-99): a `Stamina`-99 player never fatigues at
   all, a `Stamina`-0 player who has played every minute loses up to `FatigueMaxPenalty` (30%) off
   every skill. A player who hasn't played yet this match (0 minutes) is always unaffected,
   whatever their `Stamina`.
3. **InitialSimulation** (`engine/initial_simulation.go`) - recomputes both teams'
   `TeamQualification` from the current lineup (a red card or injury this match already
   thinned a position group, so this has to be recalculated every minute, not once at kickoff).
   A core stat like `Attack` is just the sum of the relevant players' raw skill values (e.g.
   every striker's `Shot`); a flank stat blends the winger/full-back's own skill 50/50 with their
   `Lateral` rating. `blendLine` then gives each zone of a line (defense, midfield, attack) a
   share of its neighbors' *per-player average* strength (`ZoneSpillover`, 15% each way - a flank
   picks up some of the center's strength and vice versa, the same way a back four or midfield
   three actually covers for each other) - the average, not the center's raw sum, since a flank is
   always exactly one player (see `common/formation.go`) while the center fields one to three, and
   comparing raw totals would swing a flank's own rating just from this team's own formation width
   rather than anyone's actual skill; the center's blended result is then scaled back up by its
   player count, so a wider line's greater total presence (the reason the game sums it) is
   otherwise unaffected. `TeamQualification`'s `Defense`/`Midfield`/`Attack` and their
   `Left`/`Right` variants all carry a bit of their neighbors' strength before ScorerSimulation
   ever reads them - not just the display-only `MidfieldDiff`/per-side attack-vs-defense diffs
   (now plain center-vs-center diffs on top of that already-blended strength, used for match
   reporting/flavor).
4. **ScorerSimulation** (`engine/scorer_simulation.go`) - each minute has a `14%` base chance of
   producing an attacking "chance" (`ProbChance`), boosted by up to another 10 percentage points
   the more one side's `Midfield` dominates the other's (`chanceProbability`). Whichever team gets
   the chance is itself weighted by that same midfield split (`getTeamChance`). The chance is then
   randomly center/left/right (50/25/25); a left/right chance's attack value also picks up a
   crossed midfield edge (`flankMidfieldSupport`, weighted `FlankMidfieldSupportWeight`) - the
   attacking side's own flank midfield against the defending side's *mirrored* flank midfield, not
   the same-named one, since the two teams attack in opposite directions and so a team's own left
   flank is physically the same strip of pitch as the opponent's right flank (see
   `ZoneComparisonRows.tsx`'s doc comment on the frontend for the same reasoning applied to
   attack-vs-defense). A player is picked to take the chance (70% a striker, 20% midfielder, 10%
   defender - `getPlayerForChance`), and it succeeds (a goal) with a probability derived from a
   logistic curve over the attacking side's build-up plus the individual finisher against the
   defending side's back line plus their goalkeeper (`chanceSuccessProbability`) - bounded to a
   realistic conversion-rate band instead of scaling linearly with the stat gap.
5. **DisciplineSimulation** (`engine/discipline_simulation.go`) - a flat 3% chance per minute of a
   card incident (`ProbCardIncident`), 6% of those a straight red, otherwise a yellow (a second
   yellow for the same player converts to a red). A sent-off player is removed from their team's
   active lineup for the rest of the match (nobody replaces them - there's no substitution model
   for this) and banned 1-2 matches (`SecondYellowSuspensionMatches`/`StraightRedSuspensionMatches`).
6. **InjurySimulation** (`engine/injury_simulation.go`) - independently, a flat 0.5% chance per
   minute (`ProbInjuryIncident`) of an injury, removing the player from the rest of the match and
   sidelining them 1-5 more matches. Unlike a card, this doesn't necessarily leave the team a man
   down: if they haven't already used all `MaxScheduledSubstitutions` (3, shared with the
   manager's own scheduled swaps above) this match, the best available bench player is brought on
   in the injured player's place (`bringOnInjuryReplacement`), logged as an ordinary substitution
   event. With no substitutions left, the injured player simply isn't replaced.

Goalkeepers are exempt from cards/injuries (there's no substitute to bring on for one). Both
teams' finances are updated once the full 90 minutes resolve (ticket revenue, sponsorship - see
"Economy" below) and any unowned team involved gets its automatic AI upkeep run (see the CPU
transfer-market section).

### Suspended/injured players never actually take the field

Before any of the above runs, `excludeUnavailablePlayers` (`engine/simulation.go`) strips every
player still serving a suspension or injury (`common.GetUnavailablePlayerIds`/
`GetUnavailablePlayerIdsForFriendly`) out of the saved starting lineup, even if the manager never
updated it - so a banned or injured player simply can't appear, whatever the lineup says.
`backfillEmptyPositions` then tries to restore each thinned position back to its original
headcount from the rest of the squad (the next-best eligible player by `common.PositionValue`,
mirroring a real substitute coming off the bench); if the squad doesn't have enough fit players
left to fill every slot, the position is simply left short rather than fielding someone twice or
someone unavailable - the match is played with fewer players on that side instead.

### Forfeit when a side runs out of players

`applyForfeitIfUnderstaffed` (`engine/simulation.go`) checks, before kickoff and again after every
simulated minute, whether either side has fallen to `MinPlayersToContinueMatch` (7, goalkeeper
included) or fewer - whether from a squad crisis the pre-kickoff backfill above couldn't cover, or
from unreplaced red cards/injuries during the match. The moment it does, the match ends right
there: the actual scoreline up to that point is discarded in favor of a flat `ForfeitGoals`-0 (3-0)
win for the side still standing, a `common.MATCH_ABANDONED` event records it, and no further
minutes are simulated - the same way a real match gets abandoned and awarded rather than played out
short-handed.

## Player skill progression

Skills drift match-by-match, not once a season: right after every league match, cup tie, or
friendly is simulated, each player who featured gets an independent shot at growing (or
declining), scaled by how many minutes of that specific match they played. A friendly counts here
too, even though it carries none of a competitive match's financial stakes (see "Economy" below) -
the players who feature in it still develop from playing. `updateSkillsAfterPlayedMatch`
(`engine/simulation.go`, called from `simulateMatch`, `engine/cups.go`'s `simulateCupTie`, and
`engine/friendlies.go`'s `simulateFriendly`, right after each one's other post-processing) is the
trigger; `updatePlayerSkillsAfterMatch` (`engine/progression.go`) does the actual drifting.

For each player who took the field this match, on the side they played for:

1. `primaryPosition` picks whichever single tactical position they logged the most minutes at
   *in this match* (from the match simulation's own in-memory per-player-per-position minutes -
   nothing is read back from the database), and only the skills that position actually uses
   drift (mirrors `common.PositionValue`'s own position→skill weighting - e.g. a striker's
   shot/pass/dribbling can move, their defense never does).
2. Each relevant skill independently has a chance to drift at all, scaling with minutes played
   *in this match* up to `MatchSkillGrowthMinutesReference` (900 - the same 10-full-matches budget
   the old season-end system used, just now spent match by match: someone who plays every minute
   of a game still only has roughly a 1-in-10 chance of a given skill moving that game, rather
   than near-guaranteed growth every single match).
3. When a skill does drift (`driftSkill`), the delta is an age trend (+1 under 24, 0 from 24-29,
   -1 from 30 on) plus a hired trainer's bonus (`trainerBonus` - +1 for a decent coach, +2 for an
   elite one, nothing below a minimum threshold, and never applied to a player already in decline
   - a coach accelerates growth but can't reverse aging) plus a small ±1 random wobble, clamped to
   `[1, max]`. Age and trainer skill are both read fresh for that match, not cached from season
   start.
4. `max` is that specific player's own hidden ceiling for that skill, not a universal 99 - every
   player gets one individually rolled per skill at creation time (`common.rollSkillCap`, applied
   by `doCreatePlayers` for new squads/academy/cantera players and `InjectFreeAgents` for injected
   free agents), stored in `players.*_max` columns and fetched in batch per match
   (`common.GetPlayerSkillCaps`). Most players (70%) get a modest 5-15 point gap above their
   starting value, 25% get a real 15-30 point gap, and a rare 5% get a wonderkid-sized 30-45 point
   gap - always capped at 99. Once a skill reaches its own ceiling, further positive drift simply
   has nowhere left to go, mirroring how few real players ever develop into world-class talents.
   This ceiling is deliberately never exposed anywhere in the `Player` struct or any API response
   - a manager sees a player's current skills, never their hidden potential.
5. If nothing actually drifted this match (quite possible - see the small per-skill chance
   above), there's no database write and no log row; if something did, it's persisted immediately
   and logged to the same `player_progression_log` table the old season-end system used - see
   `/players/evolutions`, unchanged, just now fed by many small match-triggered updates across a
   season instead of one big batch at rollover. When at least one of a team's players evolved this
   match, its owner also gets a `NOTIFICATION_PROGRESSION` notification (how many players
   developed, linking to `/players/evolutions?logIds=...` - the exact `player_progression_log`
   row ids this match produced) so the change is visible without having to think to check a
   player's page. The `logIds` link scopes that page to precisely the player(s) who evolved after
   this one match - opened without `logIds` (direct navigation), it instead falls back to every
   current squad player's own all-time latest row, which can span many unrelated matches.

**Stamina and Form** are two further per-player attributes (0-99, alongside the seven core
skills), both updated in the same `updateTeamPlayersAfterMatch` pass but on their own rules -
persisted separately (`common.UpdatePlayerStaminaAndForm`) rather than through
`UpdatePlayerProgression`, and never part of a `NOTIFICATION_PROGRESSION` notification or the
`player_progression_log` audit trail (they're expected to move after nearly every match, so a
notification for them would lose all signal value):

- **Stamina** is read every simulated minute, not just post-match - see `StaminaSimulation` above,
  which fatigues a player's *effective* in-match skills based on it. After the match, a player who
  featured gets an independent chance to drift (`driftStamina`), scaled by the same
  minutes-played-this-match chance the core skills use. The drift itself is purely an age trend
  (`ageTrendForStamina` - rises under 24, flat 24-29, declines from 30 on and steeper again from
  34, reflecting that physical conditioning fades earlier and faster than footballing skill) plus
  a small ±1 wobble - no trainer bonus, since a coach develops footballing skill, not raw fitness.
  A player who didn't feature in a match doesn't have their Stamina touched at all (no fatigue to
  recover from, nothing played to drift it either way).
- **Form** updates for *every* squad player each match, not just whoever played - it also feeds
  back into that very match's own player ratings (`common.ComputeMatchRatings`, see below), so a
  player's current Form shapes their rating before the match, and their rating that match then
  reshapes their Form afterward. A player who featured gets a baseline rise
  (`FormRisePerMatch`, +2) for simply playing, adjusted up or down by how far their own match
  rating landed from the average 6.0 (`driftFormPlayed`, `FormPerformanceWeight` - a poor enough
  performance can outweigh the baseline rise and net Form down despite having played). A squad
  player who sat out the match entirely instead gets a flat decay (`driftFormBenched`,
  `FormDecayPerMatchBenched`, -1). Both clamp to `[1, 99]`, same as every other attribute.
  `ComputeMatchRatings` itself reads a player's Form defensively: a `nil` player map or a missing
  player id (every call site that predates Form, and any hand-built `WeekMatch` in a test) falls
  back to the neutral midpoint (`common.DefaultForm`, 50 - no rating effect either way) rather than
  erroring or crashing.

New players start with Form at the neutral midpoint (`common.DefaultForm`) and Stamina randomized
independently of their footballing skills (`common.StaminaGenerationMin`/`Max`, 50-90) - unlike the
seven core skills, physical conditioning isn't meant to track a prospect's lower starting quality
(see `doCreatePlayers`/`InjectFreeAgents`).

Aging is on its own, separate, independent cadence: `ProcessBirthdays` (`engine/birthdays.go`),
run daily by its own cron job, bumps a player's age by one every `BirthdayCycleDays` (20) days
since their `birth_day` anchor. `birth_day` rolls forward by the same 20 days each time so it's
positioned for the next one, and the player details page shows "days until next birthday" derived
straight from it.

Season rollover (`AdvanceSeasonIfComplete`, `engine/season.go`, checked every tick by the 5-minute
cron job once `GetSeasonMatchCounts` reports the season's matches all played) still calls
`ProgressPlayers`/`ProgressFreeAgents` (`engine/progression.go`), but skills are no longer any
part of what those do - just:

1. **Retirement** - `ShouldRetire` checks the player's current age (already wherever
   `ProcessBirthdays` last left it): forced at 40 (`RetirementForcedAge`), and from 33
   (`RetirementEligibleAge`) the odds climb 15 percentage points per year past that. A retiree is
   dropped immediately - no contract step, nothing else below applies to them this season.
2. **Contract countdown** - `AdvancePlayerContract` ticks the remaining contract length down by
   one regardless of anything above; hitting zero releases the player to free agency.

A team left short of a full squad afterward (retirements, expired contracts) gets topped back up
from the academy (`CreateAcademyPlayers`, also trainer-boosted). `ProgressFreeAgents` checks
unemployed players the same way but they never drift skills at all, match-triggered or otherwise
(no club, no matches, no coach to develop them) - one forced past `FreeAgencyRetirementSeasons`
(2) consecutive unsigned seasons retires regardless of age, converting into a fresh trainer
candidate on the open market (`RetireFreeAgentAsTrainer`) rather than just vanishing.

## Season awards

At the same rollover point, right alongside that season's prize money and season-ticket payouts
(`applyPromotionRelegation`, `engine/awards.go`), each league hands out two awards, both delivered
as `NOTIFICATION_AWARD` notifications to the winner's team owner (distinguished by
`NotificationMessage.Key`, the same convention `NOTIFICATION_CUP` uses for a tie result vs. the
champion) and linking to the winning player's page:

- **Golden Boot** - the league's top scorer for the season (`common.GetTopScorers`, limit 1),
  skipped if nobody scored at all.
- **Player of the Season** - `common.GetPlayerOfTheSeason`: a simple goals/cards weighted score
  (+3 per goal, -1 per yellow card, -3 per red card) tallied straight from `match_events`, same
  "not a real performance model, just enough to make it feel earned" spirit as
  `ComputeMatchRatings`. Skipped if nobody's score comes out positive.

Each winner is also saved permanently in `player_season_awards` (`common.RecordPlayerSeasonAward`;
migration `init74` backfills seasons finished earlier from `match_events`), which feeds the
"Golden Boots" / "MVP" counters on the player's profile page (`common.GetPlayerSeasonAwards`).

Both read `match_events` for the season just finished, which classification's in-place reset
never touches, so they can run at any point in `applyPromotionRelegation` regardless of the
already-run `ProgressPlayers`/`ProgressFreeAgents` steps. Neither award is archived anywhere
beyond the notification itself - same reasoning as `SeasonRecap.tsx`'s doc comment on why there's
no separate "final standings" to show: classification is reset in place rather than kept per
season, so this, too, is a point-in-time recap rather than a permanent record.

## Economy

Every team starts with a budget of **400,000** (`teams.budget` default, `resources/sql/init.sql`)
and every change to it is a signed ledger entry (`common.AdjustTeamBudget`, reason codes in
`common/fmconstants.go`'s `TRANSACTION_*` constants) - there's no other bookkeeping, the running
`budget` column and the transaction history are the whole financial model.

**Income:**

- **Ticket revenue** (`engine/stadium.go`) - paid to the home side only, once per match:
  attendance × ticket price, nothing more (no concessions, no away allocation). Attendance is
  driven by the club's **socios** (members, `teams.socios` - see "Socios" below), not by how many
  seats the ground has: `min(capacity, socios × price × form × rival × weather × noise)`
  (`CalculateAttendance`), where
  - *price* is 1.0 at `DefaultTicketPrice` (10), falls linearly to 0 at `SociosPriceZeroAt` (40)
    and is capped at `MaxPriceFactor` (1.3) for very cheap tickets;
  - *form* (racha) is 0.8 after five straight league defeats up to 1.2 after five wins, from the
    league points of the home side's last `RecentFormMatches` (5) league matches (the match being
    played doesn't count; neutral 1.0 with no history);
  - *rival* is 0.85 for an unknown visitor up to 1.15 for the biggest (by the visitor's Fanbase),
    plus `CupRoundAttendanceBoost` (10%) per cup round;
  - *weather* is sunny 1.05, cloudy 1.00, windy 0.92, rain 0.82 - drawn once in `simulateMatch`,
    used here and stored as the same value the match report shows;
  - *noise* is ±`AttendanceNoise` (10%).

  There is deliberately **no minimum occupancy**: nobody pays 100 for a ticket just because the
  seat exists. `optimalTicketPrice(capacity, socios, form)` scans every price for whichever
  maximizes expected `price × min(capacity, demand)` - the price that just fills the ground when
  demand exceeds capacity, a moderate one (~20) when it doesn't (the CPU charges exactly this - see
  the CPU-team section). Capacity starts at `DefaultStadiumCapacity` (1000) and, being only a cap,
  no longer creates demand by itself: building seats helps only a club whose socios outgrow its
  ground.
- **Sponsorship** (`engine/sponsorship.go`) - a per-match payment to *both* sides regardless of
  result, scaled by the division tier they currently play in (8,000/4,000/1,500 base for tiers
  1/2/3, `sponsorshipForTier`) and further shaped by each team's own chosen `common.SponsorDeal`
  (`sponsorshipAmount`, changeable any time via `FinancesPage.tsx`, same as ticket price):
  `SponsorDealSteady` always pays 1.3× the base regardless of result; `SponsorDealBalanced` (the
  default) pays the base amount plus a bonus for a win or draw; `SponsorDealPerformance` pays only
  0.6× the base guaranteed but the biggest bonus on top - worse than Steady for a team that mostly
  loses, better for one that mostly wins or draws.
- **Prize money** (`engine/prizemoney.go`) - at season rollover, each tier's total pool
  (500,000/200,000/50,000 for tiers 1/2/3) is split across that tier's teams by final league
  position, roughly linearly from ~2× the per-team average share for 1st down to ~0 for last
  (`prizeMoneyForPosition`). Cup ties pay `CupPrizePerRound` (5,000) × the round number to the
  winner of each tie, plus a one-off `CupChampionBonus` (100,000) to the eventual champion.
- **Season tickets and merchandising** (`engine/seasonincome.go`) - two more once-a-season lump
  sums, paid at the same rollover point as prize money (`awardSeasonTicketsAndMerchandising`,
  called alongside `awardLeaguePrizeMoney`). Season tickets are stadium capacity ×
  `SeasonTicketRevenuePerSeat` (5) - independent of the per-match attendance-driven ticket revenue
  above, so a big stadium earns more here even if rarely full on a given matchday. Merchandising
  is a much flatter, smaller pool than prize money (100,000/40,000/10,000 for tiers 1/2/3) split
  1.5× the per-team average for 1st down to 0.5× for last (`merchandisingForPosition`) - a
  struggling team's shirts still sell, unlike prize money's 2×-to-0 spread.

**Expenses:**

- **Wages** - deducted once per league match played (`PayMatchdayWages`, called from
  `simulateMatch` for both sides - a round-robin schedule gives every team exactly one league
  match per round, so this fires exactly once per round), as the sum of every squad player's wage
  (`GetTeamWageBill`) plus the team's trainer's wage if it has one, all multiplied by the team's
  **division wage factor** (`common.DivisionWageFactor`: ×4.0 / ×2.1 / ×1.1 for tiers 1 / 2 / 3).
  The factor is applied only when wages are *paid* - in `GetTeamWageBill`,
  `GetTeamLoanWageContribution` and `deductTrainerWage` - and nowhere else: a player's stored wage
  stays the base figure that offers, renewals, the market-value formulas and the CPU's valuation
  work with (the CPU's affordability check scales it through `TargetContext.WageFactor`), so no
  migration is needed and a promotion or relegation raises or lowers income and payroll at once.
  Income scales roughly 5x from third to first division while skills (and so base wages) are the
  same everywhere; without the factor first-division clubs printed money. Together with the socios
  attendance model this brings a passive club's income/expense ratio to ~1.0 / ~1.1-1.2 / ~1.3 for
  tiers 3 / 2 / 1 and stabilises the world's total money at ~2x its starting value (run
  `ECON_SIM=1 go test ./engine -run TestEconomySimulation -v` to check - `engine/economysim_test.go`
  replays 40 seasons with the real engine functions). The finance page shows the factor. A player's own wage is derived
  from their skills (`common.PlayerWageFromSkills`): the sum of all seven core skills ×
  `WagePerSkillPointNumerator` (5) ÷ `WagePerSkillPointDenominator` (7) - a higher-rated player
  costs more to keep, with no separate negotiation needed for the baseline figure (negotiated
  wages on signing/renewal are checked against this as a floor - see
  `common.MaxAcceptableRenewalYears`/offer-acceptance logic in `common/contracts.go`). This used
  to fire on a fixed real-calendar cron (Monday 00:00 UTC) regardless of how many rounds a league
  actually played that week - since ticket revenue and sponsorship are paid per match, a league
  scheduled to play daily could earn up to 7x a weekly league's income in the same real week while
  paying the same, cadence-independent wage bill (a season's fixed number of rounds costing a
  fraction of the wages once compressed into fewer real weeks). Tying the charge to the same
  per-round cadence the income side already follows removes that gap, so a season's total wage
  cost is proportional to its total match count the same way ticket/sponsorship income already is
  - but the old flat per-skill-point rate (previously just `WagePerSkillPoint` = 5) also needed to
  come down by the same ÷7 once the charge itself moved from once a week to once a round under the
  default daily schedule, or the per-round bill would simply be the old weekly bill charged up to
  7x as often. That second half of the fix landed later than the cadence change (see
  `resources/sql/init65_rescale_wages_to_match_cadence.sql`, which also recalculates every
  pre-existing player's/trainer's stored wage - freshly created ones get the corrected rate
  automatically); until then, a daily-schedule team's payroll really was up to 7x too high
  relative to its income, which is what drained the in-game economy.
- **Transfers** - a fixed-price listing (`common/transfers.go`) or an accepted negotiated offer
  (`common/offers.go`) debits the buyer and credits the seller the agreed fee in one atomic
  transaction; a free-agent signing costs nothing but the player's wage. `GetOffersReceived`
  already returns every pending offer from every interested buyer (at most one per team, enforced
  by a unique constraint on `transfer_offers`); the Offers tab's `ReceivedOffersByPlayer`
  (`TransferMarketPage.tsx`) groups a player's competing offers together, sorted by amount, so a
  seller comparing bids sees them side by side instead of interleaved by date with every other
  player's offers (Economy Phase 4, "auction/bidding").
- **Loan money** (`common/loans.go`, Economy Phase 4) - `AcceptLoan` used to move no money at
  all; a lender can now attach three independent, all-optional terms to a loan listing
  (`CreateLoanListing`), each zero/off by default so an existing loan-listing flow is unaffected:
  a `loanFee` the borrower pays the lender once, upfront, when the loan is accepted (like a mini
  transfer fee); a `wageContributionPercent` the lender keeps covering for the loan's duration -
  the borrower's own wage bill (`GetTeamWageBill`) already excludes that slice, and the lender is
  charged it separately each matchday (`GetTeamLoanWageContribution`, `engine.deductWages`,
  `TRANSACTION_LOAN_WAGE_CONTRIBUTION`) so the two sides always add up to exactly the player's full
  wage; and a `buyOptionFee` the borrower can pay at any time during the loan to make the move
  permanent (`common.ExerciseLoanBuyOption`) - the same money movement as a normal transfer, minus
  moving the player, since they already play for the borrower. A player's own page shows an
  "Exercise buy option" button when the viewer's team is the current borrower and the loan carries
  one (`PlayerStatus.OnLoanBuyOptionFee`).
- **Stadium expansion** (`UpgradeStadium`, `common/stadium.go`) - adds an owner-chosen number of
  seats, between `MinStadiumUpgradeSeats` (100) and `MaxStadiumUpgradeSeats` (5,000), costing
  current capacity × `StadiumUpgradeCostPerCurrentSeat` (10) × seats requested ÷
  `StadiumUpgradeSeats` (500) - so both a bigger stadium and a bigger expansion cost more in
  absolute terms. AI-managed teams always request the `StadiumUpgradeSeats` default.
- **Stadium maintenance** (`StadiumMaintenanceCost`, `engine/stadium.go`) - deducted once per
  round alongside wages (`deductStadiumMaintenance`, called from `PayMatchdayWages`), linear in
  the stadium's current capacity: capacity × `StadiumMaintenanceCostPerSeat` (1) ÷
  `StadiumMaintenanceCostDivisor` (10) - 100/round at `DefaultStadiumCapacity` (1,000). Gives
  capacity an ongoing cost, not just a one-time purchase price - before this, `UpgradeStadium`
  above was the only cost tied to size, so a bigger stadium was pure upside once paid for.
- **Contract renewal/release** - a renewal offered at or above the player's own market wage is
  accepted for free; releasing a player early before their contract expires costs a severance (see
  `common.EarlyReleaseWageMultiplier` in `common/contracts.go`).

**Insolvency:** a team's budget can go negative with no immediate effect, but once it crosses
`-MaxDebt` (-200,000, `common/fmconstants.go`) `common.IsTeamSolvent` starts reporting false, and
every spend-side action - `ExecuteTransfer` (buying a listed player), `MakeTransferOffer`,
`AcceptLoan` (borrowing one), and `SignFreeAgent` (which `SignFreeAgentAtMarketRate` also routes
through, so this covers Director-tier AI buying too) - returns `ErrTeamInsolvent` instead of going
through. Selling, releasing, or cancelling a listing are never gated by this: those are exactly
how a team is meant to dig back out, so blocking them would make a debt spiral impossible to
escape rather than just painful. `GetFinancesHandler` exposes this as `insolvent` (and the
threshold itself as `maxDebt`) on the finances response so the frontend can show a standing
warning rather than the player only discovering it from a failed action.

**Financial planning tools:** `FinancesPage.tsx` also renders a cash-flow forecast card and two
charts, all computed client-side from the same `transactions` list the finances table already
uses - no separate backend endpoint. `cashFlowForecast.ts`'s `computeCashFlowForecast` reads the
current weekly wage bill directly off the most recent `TRANSACTION_WAGES`/`TRANSACTION_TRAINER_WAGES`
transactions (wages are a fixed weekly debit, not something to average) and averages
`TRANSACTION_TICKET_REVENUE`/`TRANSACTION_SPONSORSHIP` transactions per week they actually span
(deliberately excluding once-a-season lumps and transfers, which would otherwise wildly skew a
"weekly" figure) - if the resulting net is negative and the team isn't already insolvent, it
projects how many weeks until budget hits `-maxDebt`. `BudgetOverTimeChart` reconstructs a running
balance backward from the current budget across the visible transaction window (a hand-rolled SVG
line chart, the same convention `SeasonPositionChart` already established) and `IncomeBreakdownChart`
sums every positive transaction by reason into horizontal bars.

`engine/database.go`'s `UpdateTeamFinancesAfterMatch` is the one place both league matches and cup
ties route through after a result: it credits the home side's ticket revenue, credits sponsorship
to both sides, drifts each side's Fanbase (see below), and triggers `RunAITeamMatchReaction` for
either side that's currently unowned (see below). Friendlies deliberately skip all of this - no
sponsorship (tied to goals scored in a result that actually matters, which a friendly's never is),
no Fanbase drift, no *per-match* AI upkeep hook - since a friendly isn't a competitive result
(skill growth is one exception - see "Player skill progression" above). Ticket revenue is the
other: `engine/friendlies.go`'s `simulateFriendly` calls `creditFriendlyTicketRevenue` instead,
which draws attendance/revenue from the home side's own stadium exactly like `creditTicketRevenue`
does, but splits the gate evenly between both sides instead of paying it entirely to the home side
- the away side showed up purely as a favor for a non-competitive match, so it isn't left
empty-handed. A CPU team's market upkeep isn't actually gated on the per-match hook firing, though
- see the daily scheduler described in the next section.

### Fanbase

Every team also has a Fanbase index (`teams.fanbase`, 0-99 - the same scale as every other
rateable attribute in this game: skills, Stamina, Form), representing how big/loyal its supporter
base currently is. Unlike those, it's a team-level stat, not a player one, and it never appears in
`player_progression_log` or any `NOTIFICATION_PROGRESSION` notification - it has its own type
instead (`NOTIFICATION_FANBASE`, see below). New teams, and every team that existed before this
feature shipped, start at `common.DefaultFanbase` (50, the same neutral-midpoint convention
`common.DefaultForm` uses).

Fanbase drifts slowly, not every match - `engine/fanbase.go`'s `applyFanbaseDrift` runs from the
same `UpdateTeamFinancesAfterMatch` hook sponsorship uses (league matches and cup ties only, never
friendlies), and rolls two independent, deliberately rare per-match chances:

- **Result drift** (`driftFanbaseForResult`, `FanbaseResultDriftProbability` = 15% chance per
  match) - a win moves it `+FanbaseWinDelta` (1), a loss `FanbaseLossDelta` (-1), a draw never
  moves it even when the roll succeeds. Rare on purpose: a club's Fanbase is a slow-moving
  identity, not a weekly mood the way Form is (Form moves every single match a player features
  in - see "Player skill progression" above).
- **Tier pull** (`driftFanbaseTierPull`, `FanbaseTierPullProbability` = 5% chance per match) -
  independent of this match's result, nudges Fanbase one point toward wherever its current
  division tier's own target sits (`FanbaseTierTarget`: 80/50/30 for tiers 1/2/3, falling back to
  tier 3's target for anything deeper). This is what gives promotion/relegation a slow,
  multi-season effect on crowd size on top of the immediate prize-money/sponsorship-tier jump they
  already cause (see "Season awards" above).

Both rolls are independent pure functions (directly unit tested without a database, same
`driftStamina`/`driftFormPlayed`-style split as player attribute drift above), clamped to `[1, 99]`
the same way every other 0-99 attribute is. If a match's drift actually changes the value, the
owner is told via `NOTIFICATION_FANBASE` - but only when it crosses into a new *ten-point band*
(e.g. 47 -> 51, or 53 -> 49, see `notifyFanbaseMilestone`), not on every single one-point move
(most of which don't even happen, given how rare both rolls are) - frequent enough to notice a
real trend, rare enough not to spam.

What it actually affects:

- **Matchday attendance** - no longer a direct multiplier: Fanbase now moves attendance through
  **socios** (below), whose season-end target is `2.2 × Fanbase^1.5` (`sociosTarget`), and through
  the *rival* factor, where the visiting side's Fanbase pulls a bigger or smaller crowd.
- **Sponsorship** (`engine/sponsorship.go`'s `creditSponsorship`) and **season tickets/
  merchandising** (`engine/seasonincome.go`'s `awardSeasonTicketsAndMerchandising`) - each
  multiplies its own base figure by `engine/fanbase.go`'s `fanbaseCommercialFactor`, a gentler
  `[FanbaseCommercialMinFactor, FanbaseCommercialMaxFactor]` (0.8x-1.2x) swing than attendance
  gets, since a sponsor or season-ticket buyer already cares most about which tier a team plays in
  or how big its stadium is - a bigger Fanbase just makes an otherwise-identical club a somewhat
  more attractive commercial partner on top of that. `SponsorDealPreviews` (what `FinancesPage.tsx`
  shows before a manager commits to a deal) takes the team's current Fanbase as a parameter now,
  so the previewed numbers match what will actually be paid.

Deliberately **not** affected (at least not yet): match simulation itself. The simulator has no
"home advantage"/crowd-noise concept at all today, and giving Fanbase one would be a separate,
bigger design decision - see `ECONOMY_ROADMAP.md`.

Public via `common.TeamDetails.Fanbase` (any team's page - same visibility tier as `StadiumSummary`)
and `FinancesResponse.Fanbase` (the owner's own finances page, shown alongside Stadium/Budget in a
small `FanbaseCard`, same pattern `StadiumCard`/`SponsorDealCard` use).

### Socios

Every team also has a count of club members, `teams.socios` (`common.DefaultSocios` = 500 for a new
team; existing teams were backfilled to `round(2.2 × fanbase^1.5)` by
`resources/sql/init80_socios.sql`). They are the paying core of the home crowd - see "Ticket
revenue" above - so demand is created by members, not by seats.

At every season rollover (`updateSociosAtSeasonEnd`, called from
`awardSeasonTicketsAndMerchandising` with each league's final standings) socios close
`SociosDriftRate` (30%) of the gap to the team's target,
`sociosTarget = 2.2 × Fanbase^1.5 × position factor`, where the position factor runs from +10%
(champion) to -10% (last). The curve is superlinear on purpose: a big first-division club
(Fanbase towards 80) ends up with ~1,400-1,700 socios and outgrows a 1,000-seat ground; a small
third-division one (Fanbase towards 30) settles at ~360-500 and never needs to expand. The owner
is told (`NOTIFICATION_FANBASE`, `sociosGrew`/`sociosShrank`) when the count moves by 5% or more,
and the finance page shows it next to Fanbase.

## Notifications

In-game events (a match result, a transfer offer, a player's progression, etc.) are recorded as
rows via `common.CreateNotification` and shown in the bell menu (`NotificationBell.tsx`), which
also polls unread count. Separately, each user can opt into having them pushed to them as they
happen rather than only seen on next visit - `common.NotificationPreferences`
(`emailEnabled`/`pushEnabled`, both default false) gates a call to `common.DefaultNotifier.Send`,
a package-level `Notifier` swapped at startup so callers don't need to know which channel(s) are
actually wired up.

Every call site builds a `common.NotificationMessage{Text, Key, Params}` rather than a bare
string: `Text` is English, used for external delivery (below) and as the frontend's fallback;
`Key`/`Params` are an i18n key (e.g. `notifications.messages.transferBought`) and its
interpolation values, stored alongside `Text` in the `notifications.message_key`/`message_params`
columns (both nullable - empty for a row written before this existed). The bell and the dashboard
`SeasonRecap` banner both render `t(messageKey, messageParams)` when present, falling back to the
plain `message` text otherwise - this is what lets the same match-result/transfer/cup notification
show in whichever language the viewer has chosen instead of always English. Adding a new
notification type means adding both the `Key`/`Params` at the call site and the matching
`notifications.messages.*` entry in `frontend/src/i18n/locales/{en,es}.json`.

External delivery (email/push) also localizes: `LanguageSwitcher.tsx` calls `PUT
/v1/user/language` whenever a logged-in user changes language, storing it in `users.language`
(`common.SetUserLanguage`/`GetUserLanguage`). `deliverExternalNotifications` looks this up and, for
`"es"` with a recognized `Key`, renders through `common.spanishNotificationTemplates`
(`common/notificationtranslations.go`) - a hand-maintained Go mirror of `es.json`'s
`notifications.messages.*` block, needed because the backend has no runtime access to the
frontend's JSON locale files. Any other language, an unset preference, or an unrecognized key falls
back to the English `Text`, same as before this existed. Adding a new notification type's
translation here is a manual step alongside the frontend JSON entry - nothing enforces the two stay
in sync.

- `LogNotifier` is the fallback: every channel just gets logged, never actually delivered - what
  runs when nothing else is configured.
- `WebPushNotifier` (`common/webpushnotifier.go`) is the real implementation for the `push`
  channel (`email` still falls through to log-only - no email transport exists yet). It sends via
  the Web Push protocol (VAPID) to every one of a user's subscribed devices/browsers
  (`push_subscriptions` table, `common/pushsubscriptions.go`; a user can have several - phone,
  laptop, etc.), dropping a subscription outright once its push service reports it permanently
  gone (404/410). `common.ConfigureWebPushNotifier`, called once at startup by both
  `service.StartSvc` and `engine.StartEngine` (either process can trigger a notification), swaps
  `DefaultNotifier` to this only when `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` are set in the
  environment - otherwise push notifications are still tracked in-app, just never actually pushed.
- On the frontend, `src/pushNotifications.ts` handles subscribing/unsubscribing this browser
  (requesting `Notification` permission, then `PushManager.subscribe` with the VAPID public key -
  exposed to the browser the same "baked placeholder + `set-env.sh` rewrite" way
  `GOOGLE_OAUTH_CLIENT_ID` is, via `config.ts`/`public/config.js`), and `public/sw.js`'s `push`/
  `notificationclick` handlers display the notification and focus/open the app when clicked.
  Toggling the push switch in `NotificationBell` only flips `pushEnabled` on once the browser
  subscription actually succeeds - a denied permission leaves it off rather than claiming push is
  live when nothing was registered.

## How CPU (unowned) teams resolve the transfer market

An unowned team (nobody has claimed it) isn't just idle - it runs one of three AI tiers,
weighted-randomly assigned at creation and biased by the division tier it starts in (top-flight
leagues skew toward sharper CPU competition - `RandomAiLevel`, `common/aimanager.go`). Each tier is
a strict superset of the one below it. All of this is driven from `RunAITeamManagement`
(`engine/aiteams.go`), called once per match for each unowned side involved (via
`UpdateTeamFinancesAfterMatch` above) - and, independently of any match at all, once a day on
average for every AI-managed team via `RunAiTeamDailyActivity`/`StartAiActivityScheduler`
(`isAiManagedTeam` is the one shared gate all three of `RunAITeamManagement`,
`RespondToAllPendingOffers`, and `RunAiTeamDailyActivity` check: unowned and at least Caretaker
level). Calling `RunAITeamManagement` twice over is harmless - every action inside is itself a
no-op once there's nothing left to do - so before match activity even starts (a freshly created
world, or simply a quiet week for a given team) a CPU team's market upkeep no longer depends on it
having played anything.

`RunAiTeamDailyActivity`'s own scheduler ticks every 5 minutes (288 times a day);
each AI-managed team independently rolls a 1-in-288 chance per tick for its market-management
pass and, separately, the same 1-in-288 chance to challenge another AI-managed team to a friendly
(`ChallengeToFriendly`, `common/friendlies.go`) - both average out to about once a day, at
whichever random tick happens to hit first, rather than every team acting in lockstep at a fixed
hour. A friendly opponent is always another AI-managed team, never a human's: a human still only
ever gets challenged by another human, so this can't spam a manager's inbox with unsolicited CPU
challenges. Since both sides are unowned, `ChallengeToFriendly` auto-accepts the challenge on the
spot (there's nobody who'd otherwise act on it) and schedules the resulting friendly the same short
delay ahead as any other accepted challenge (`friendlyScheduleDelay`) - it's picked up by the
regular per-minute `StartFriendlyScheduler` from there, same as a human-arranged one, subject to
the same weekly cap (`max_friendlies_per_week`) on both sides. Beyond the market-trigger and
matchday-activity purpose, a friendly a CPU team plays this way still counts for skill growth (see
"Player skill progression" above) exactly like a competitive match would.

- **Caretaker** (tier 1, the baseline every level gets):
  1. **Keeps the default lineup - and formation - fresh** (`refreshAiTeamLineup` re-picks the
     current best XI from the squad every match, so retirements, academy call-ups, and signings
     show up in the lineup immediately rather than waiting for season end). The formation itself
     is picked fresh too, not left at `FormationBalanced`'s 4-4-2 forever:
     `common.SelectBestFormationAndLineup` builds every one of the six formations' own best-XI for
     this exact squad, ranks them by total strength, and picks the rank this team's own AI level
     settles for (`common.formationRankForAiLevel`) - only Director reliably finds the genuinely
     strongest shape (rank 0); Manager settles for the second-best (rank 1); Caretaker settles for
     the third or fourth best (a coin flip each time), reflecting that it's the least sharp of the
     three at reading its own squad.
  2. **Resolves every pending transfer offer on its own players** - this is the direct answer to
     "how does the CPU decide on a transfer offer" (`respondToReceivedOffers`, deciding with
     `common.EvaluateIncomingOffer`): accept if the fee reaches the team's *perceived value* of the
     player, otherwise reject. The player's personal terms are never in question here -
     `MakeTransferOffer` never even lets a human create an offer the player themself wouldn't
     accept - so this is purely about whether the fee is good enough for the selling club. The
     perceived value starts from `common.MarketValue` (market wage x 20, scaled down with age) and
     is then seen through the level's valuation noise (`AiTransferProfile.ValuationNoise`: 30% for a
     Caretaker, 15% Manager, 5% Director) - which is why a Caretaker will sometimes let a bargain
     go, and a Caretaker never tries to read a teenager's hidden potential at all, so it can sell a
     future star for the price of any other teenager. Every level also refuses a sale that would
     leave the squad under `MinSquadSizeAfterSale` or the player's position under
     `MinPositionDepth`; a Manager and a Director additionally protect a key starter (one who
     starts and whose loss would leave the position short or cost the XI 6+ points) by asking 2x
     (Manager) or 2.5x (Director) his value, and a Director asks 25% more for a national-team
     player. The highest offers are judged first, each offer's noise is seeded from its own id (so
     it gets the same answer on every sweep while it waits out the negotiation window), and an
     acceptable offer still inside that window is simply left pending. This runs at every AI
     level (not gated behind Manager) since it's baseline business any unowned team should
     handle; it's also swept independently once or twice a day by a dedicated scheduler
     (`RespondToAllPendingOffers`/`StartOfferResponseScheduler`), so an offer on a team that
     hasn't played a match in a while still gets answered within a day or two rather than sitting
     pending indefinitely (issue #61).
- **Manager** (tier 2) adds roster/business upkeep on top, in this order every match
  (`manageRosterIfManager`):
  1. **Renew expiring contracts** - any squad player down to 1 season or fewer remaining gets
     offered a renewal at exactly their own market wage and their own age-appropriate maximum term
     - guaranteed to be accepted, since it's exactly what the player would ask for.
  2. **Transfer-list surplus players** (`listSurplusPlayer`) - only players the team actually
     owns count: anyone on loan *to* it is ignored, and players already transfer-listed don't
     count towards depth. First, any existing listing that's no longer genuine surplus is
     withdrawn - the listed player's position would be thin (below `MinPositionDepth`) without
     them, or the unlisted squad has shrunk below `minSquadSizeAfterSale` (`SquadSize - 2`).
     Then, if a core position's unlisted depth is more than one player above `MinPositionDepth`,
     the weakest player there who isn't loan-listed is put on the transfer list at his market value
     (`common.ListingAskingPrice`, the same yardstick the team starts from when judging an
     incoming offer) - at most one new listing per match, so listings build up gradually while a
     surplus lasts, and never one that would leave fewer than `minSquadSizeAfterSale` owned,
     unlisted players. Anyone signed in the last two weeks (`common.AiMinHoldPeriod`) is never
     listed - otherwise a signing that happens to be the weakest at a deep position bounced between
     clubs at the same fee - and neither is a youngster on a team that signs projects (below).
     Needs the transfer window open.
  3. **Promote a cantera prospect for the weakest position** (`promoteFromAcademyForWeakestPosition`)
     - only when that position is genuinely thin (below `MinPositionDepth`), weighing its own
     academy exactly the choice `CanPromoteFromAcademy`/`PromoteFromAcademy` exist to offer a human
     manager: whichever cantera prospect rates highest at the thin position is promoted, subject to
     the same once-a-real-calendar-week cap a human promotion uses.
  4. **Charge whichever ticket price maximizes expected revenue** (`optimalTicketPrice`, see
     above) for its own stadium, socios and recent form - not simply `MaxTicketPrice`, since demand
     falls to nothing at `SociosPriceZeroAt` (40).
- **Director** (tier 3) adds the rest of the active market on top (`manageMarketIfDirector`, plus a
  wider transfer pass - below):
  1. **Expand the stadium** once the budget comfortably covers the cost with room to spare -
     specifically `stadiumUpgradeBudgetCushionMultiplier` (3) x one upgrade's cost, so it keeps a
     working cash reserve rather than spending down to the wire - *and* demand really outgrew the
     ground: at least 40% of its latest 10 home league matches sold out (`shouldUpgradeStadium`,
     needing at least 5 on record). A freshly expanded ground resets this naturally, since the old,
     smaller crowds no longer reach the new capacity.
  2. **Upgrade its trainer** to a clearly better one on the open market (at least
     `trainerUpgradeMinSkillMargin` (10) skill points above its current coach, or hire one at all
     if it has none) - trainers have no signing fee, so budget never gates this, only that skill
     margin (avoiding constant churn over noise-level differences).

### How a CPU team shops (the transfer pass)

Every Manager and Director runs one shopping pass per management tick (`runAiTransferPass`,
`engine/aitransfers.go`), right after the roster upkeep above. There is **no signing quota**: the
team tries to improve its squad, and a team that finds nothing worth having signs nobody. How it goes
about it is spelled out by one `common.AiTransferProfile` per level (`AiTransferProfileFor`,
`common/aimanager.go`) - every level includes the capabilities of the ones below it:

| | Caretaker | Manager | Director |
|---|---|---|---|
| Answers offers | accepts at its valuation | + protects key starters (2x) | + asks 2.5x for them and +25% for internationals |
| Valuation noise | 30% | 15% | 5% |
| Reads hidden potential | no | yes, very noisily (+-60% of headroom) | yes, closely (+-15%) |
| Shops | - | free agents and the transfer list | + every player in the world, humans' included, and the national-team radar |
| Signs for | - | holes and clear starter upgrades (score >= 5) | + smaller upgrades (score >= 3) and young prospects |
| Selling | doesn't list | lists surplus | + lists a player to fund a target |
| Budget kept in reserve | - | 50% | 15%, and 20 rounds of the new payroll |
| Signings per pass | 0 | 1 | up to 3 (at most 2 of them new offers) |

Every candidate is scored by the pure function `common.ScoreTarget`: how many points the player adds
to the team's best XI as the team sees him (through its valuation noise) plus a bonus for filling a
thin position; for a Director, his projected growth over three seasons (for under-24s, read from a
*noisy estimate* of the hidden skill ceiling - the CPU never reads the exact ceiling, so it is
neither omniscient nor betrayed by its own signings - weighted more for a team with settled
starters, and capped at `MaxProjects` places and a quarter of the budget per prospect); and whether
the price is a bargain or an overpay against its own valuation. A candidate only counts if the
budget (and payroll, for a Director) can take it and the score clears the profile's `MinScore`.

The market it looks at depends on the profile: free agents (best 60, with the usual premium to
the previous club), the transfer list (bought at the asking price), and - Director only - the best
players at each core position anywhere in the world plus a random scouting sample of youngsters.
For a player nobody listed, it **makes an offer** (`MakeTransferOffer`) at its valuation plus a
margin that grows with how much it wants him, never beyond 1.5x its valuation. Limits, all constants
in `common/offerrejections.go`:

- at most **3 CPU offers open at once on one human manager's player**, always from different teams
  (the human is notified of each); and only one at a time on another CPU team's player, so CPU teams
  don't all pile onto the same target;
- at most 4 offers open per CPU team and 2 new ones per pass, and an offer left unanswered for 3 days
  is withdrawn;
- after a rejection (or a withdrawal) the team leaves that player alone for 3 days
  (`AiOfferCooldown`, remembered in `player_offer_rejections`), and then only returns with a bid at
  least 10% higher - and only if its valuation allows it;
- a team never signs beyond `MaxAiSquadSize` (`SquadSize + 4`) players that aren't on the transfer
  list. A Director that finds a target it can't pay for - or has no room for - **lists the player the
  target would replace** (only when that leaves the position at its minimum depth and the squad above
  its minimum size, and when it has no other listing out) and buys him in a later pass.

**National teams as a radar.** A player in his country's national-team pool (the
`NationalSquadPoolSize` best by birth country - `GetPlayersByBirthCountry`) is an *international*,
and his caps (`GetPlayerCapsBatch`) make him more so. `common.GetNationalRadar` keeps a snapshot of
every active country's pool, rebuilt every 15 minutes. It matters to the CPU in four ways:

- a **Director's radar**: the five best players of every national team, wherever they play and
  whether or not anyone listed them, are candidates too (narrowed to those who beat its weakest
  starter at their position, best 20), so it finds talent in other leagues;
- an international (Director only) earns a **score bonus** of 2 plus 0.1 per cap (up to 20 caps);
- a Director **asks more for one**: 25% above its valuation as a pool member, up to 50% with 20
  caps - both when answering an offer and in a listing's asking price (`ListingAskingPrice`);
- **no CPU team lists an international as surplus**, nor sells one to fund a signing.

Each action is counted in the `striker.ai.market_actions` metric (by action) and logged as
`ai-market team=... action=...`, so the CPU's daily volume of signings and offers can be read off.

Every one of these calls the same production code path a human manager would use
(`common.MakeContractRenewalOffer`, `common.ReleasePlayer`, `common.SignFreeAgentAtMarketRate`,
`common.AcceptOffer`/`RejectOffer`, `common.ExecuteTransfer`, `common.UpgradeStadium`,
`common.HireTrainer`) - an AI team gets no special-cased shortcuts or exemptions from the same
validation (budget, transfer window, contract state) a human's action would have to pass.

## National team friendlies

Every active country has a national team - a real, unowned `teams` row (flagged
`is_national_team`, `backend/common/nationalteam.go`) rather than any club, standing in for a
virtual squad of that country's best eligible players by birth country (`GetPlayersByBirthCountry`
- currently on a club's senior squad, not an academy prospect or free agent, ranked by overall
skill and capped at `NationalSquadPoolSize` (40)). It's deliberately financially/competitively
inert: zero budget, zero stadium capacity (so the one money flow a friendly ever triggers - gate
receipts off the home side's stadium - is always zero), no real league, and an `ai_level` of 0, the
same combination that already keeps every other unowned-team system (new-user assignment, AI-level
backfill, AI market/friendly automation) from ever touching it - see that file's own comments for
exactly which existing queries this piggybacks on for free versus the one explicit exclusion it
does need (`common.Search`'s team listing, which has no league to filter it out through).

Once a real calendar week (`engine.StartNationalTeamFriendlyScheduler`, Sunday 03:00 UTC - the cron
cadence itself is the "once a week," there's no per-team cap to enforce the way club friendlies
need one, since there's no human on either side to challenge/accept), every active country is
paired off for a friendly (`common.ScheduleNationalTeamFriendlies`): same-region pairings are
preferred (`pairNationalTeamCountries`, mirroring how the international club cup already groups by
region), with any country left over within its own region pooled across regions so at most one
country site-wide sits out in a given week. Each fixture is created directly in `friendly_matches`
(no challenge/accept dance) and each side's starting XI is picked and persisted immediately
(`SetNationalTeamLineup`) as that match's `friendly_lineup_positions` override - there's no club
roster to fall back to the way `BestLineupForTeam` provides for a real team. From there it's an
ordinary friendly: picked up and simulated by the same per-minute `StartFriendlyScheduler`, and
(being a friendly) still counts for skill growth for whoever featured, same as any other friendly.

**International caps**: a player's profile shows how many national-team appearances they were
selected to start (`PlayerCareerStats.Caps`, `GetPlayerCareerStats`) - counted from
`friendly_lineup_positions` rows on a national team's side of a played friendly, plus (since the
World Cup below added a second way to be capped) `cup_tie_lineup_positions` rows on a national
team's side of a played knockout tie. An appearance alone (not a goal/card action) is what a cap
actually is, unlike the event-based `match_events` queries the rest of that function's stats use.
Being selected is the whole simplification here (the same "best XI by raw skill" selection
`SetNationalTeamLineup`/`SetNationalTeamCupTieLineup` always makes, not any notion of current
form/fitness/actual call-up history) - it doesn't check whether the player was actually fit to
play that week, the same tolerance the engine already affords any saved default lineup that
happens to include an unavailable player.

**Call-up notifications**: whenever `SetNationalTeamLineup`/`SetNationalTeamCupTieLineup` saves a
lineup, every selected player's actual club owner (not the national team, which has no owner) is
notified (`NOTIFICATION_CALL_UP`) with a link straight to the fixture - the friendly or cup tie
itself, not the player's own page - so a click leads directly to the match they were called up
for.

**Frontend**: `/national-teams` (`NationalTeamsPage.tsx`) is where any of this is actually watched
- a manager's own team is never one side of a national-team fixture, so none of it could otherwise
surface through a team-scoped page the way club friendlies do through `/friendlies`. It has two
tabs: a global upcoming/recent national-team friendlies feed (`GET
/v1/national-teams/friendlies/{upcoming,recent}`, the any-country counterpart to
`GetUpcomingFriendlies`/`GetRecentFriendlies`), and the World Cup section below. A player's own
profile additionally lists their own recent call-ups (`GetRecentNationalTeamCallUps`), each
linking straight to that friendly or cup tie.

## World Cup

Every `WorldCupFrequencySeasons` (4) completed seasons, if no World Cup is currently running and
at least `MinWorldCupCountries` (8) countries are active, a new one starts
(`engine.StartWorldCupForNewSeason`, called from `AdvanceSeasonIfComplete` alongside
`StartInternationalCupsForNewSeason`) - a `world_cups` row (`init72_world_cup.sql`) plus a group
stage: active countries are split into balanced groups of ~4 (`common.AssignWorldCupGroups`, every
group's size within one of every other's), each scheduled a full round-robin (the classic "circle
method", `common.BuildRoundRobinRounds`) one matchday a week apart. Every group fixture is a plain
`friendly_matches` row (so `/friendlies/:id` renders it with no changes needed, and it's picked up
by the ordinary per-minute friendly scheduler) with both sides' lineups set immediately via the
existing `SetNationalTeamLineup` - a World Cup group match is, mechanically, just a national-team
friendly that happens to also be linked to a `world_cup_group_matches` row.

Once a linked friendly is simulated, `UpdateWorldCupStandingsAfterMatch` (hooked into
`simulateFriendly`, a no-op for every other friendly) updates both sides' running
`world_cup_group_teams` record. Every tick (`CupJob`, alongside `AdvanceCups`),
`engine.AdvanceWorldCups` checks every active World Cup: once every group match is played
(`IsWorldCupGroupStageComplete`), it seeds the knockout stage from each group's top two
(`RankGroupStandings`: points, then goal difference, then goals for, then team name as a last
deterministic tiebreak) as an *ordinary* `Cup` (`CreateInternationalCup`, region `"world"`) -
`startWorldCupKnockout` shuffles the qualifiers (no attempt at a real tournament's seeded bracket,
since this game has no strength ranking to seed by) and hands them to the exact same
`BuildFirstRoundPairings`/`createRoundTies`/`AdvanceCups`/`SimulateCupTies` machinery every other
cup already uses. The one addition needed there: `createRoundTies` now checks, for a real tie's
each side, whether that team is a national team (`GetNationalTeamCountry`) and if so auto-selects
its lineup (`SetNationalTeamCupTieLineup`) for every round the knockout stage generates - a
national team has no manager to do that itself, the same reason `SetNationalTeamLineup` exists for
the group stage/weekly friendlies. A club cup's ties are entirely unaffected (the check is simply
false for every one of them). Once that `Cup` itself completes, `AdvanceWorldCups` copies its
champion back onto the `world_cups` row.

While a World Cup is running, the ordinary weekly national-team friendly cadence
(`ScheduleNationalTeamFriendlies`) pauses entirely (`HasActiveWorldCup`) - a simplification over a
real international calendar, which only pauses the countries actually still in the tournament, but
avoids double-booking a country's national team in the same week without having to track which
teams have already been eliminated.

Frontend: `GET /v1/national-teams/world-cup/current` returns the latest World Cup plus its groups
(standings and fixtures); the knockout bracket, once it exists, is fetched separately through the
existing `GET /v1/cups/:id` and rendered with `CupPage`'s own exports (`groupByRound`/`roundName`/
`TieRow`) - a tie is a tie regardless of which competition it belongs to, same reuse
`InternationalCupPage` already makes.
