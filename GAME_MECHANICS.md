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
- Weekly wages: `StartWeeklyWagesScheduler` → `PayWeeklyWages` (`engine/wages.go`), once a week
  (Monday 00:00 UTC).
- Player birthdays: `StartBirthdayScheduler` → `ProcessBirthdays` (`engine/birthdays.go`), once a
  day (01:00 UTC) - see "Player skill progression" below for how this drives aging independently
  of season rollover.

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
2. **InitialSimulation** (`engine/initial_simulation.go`) - recomputes both teams'
   `TeamQualification` from the current lineup (a red card or injury this match already
   thinned a position group, so this has to be recalculated every minute, not once at kickoff).
   A core stat like `Attack` is just the sum of the relevant players' raw skill values (e.g.
   every striker's `Shot`); a flank stat blends the winger/full-back's own skill 50/50 with their
   `Lateral` rating. These roll up into a `MidfieldDiff` and per-side attack-vs-defense diffs used
   for match reporting/flavor, not for the goal math itself.
3. **ScorerSimulation** (`engine/scorer_simulation.go`) - each minute has a `14%` base chance of
   producing an attacking "chance" (`ProbChance`), boosted by up to another 10 percentage points
   the more one side's `Midfield` dominates the other's (`chanceProbability`). Whichever team gets
   the chance is itself weighted by that same midfield split (`getTeamChance`). The chance is then
   randomly center/left/right (50/25/25), a player is picked to take it (70% a striker, 20%
   midfielder, 10% defender - `getPlayerForChance`), and it succeeds (a goal) with probability
   `(attack + player'sOwnValue) / (defense + goalkeeperSkill + attack + player'sOwnValue)` - i.e.
   pure relative strength between the attacking side's build-up plus the individual finisher
   against the defending side's back line plus their goalkeeper.
4. **DisciplineSimulation** (`engine/discipline_simulation.go`) - a flat 3% chance per minute of a
   card incident (`ProbCardIncident`), 15% of those a straight red, otherwise a yellow (a second
   yellow for the same player converts to a red). A sent-off player is removed from their team's
   active lineup for the rest of the match (nobody replaces them - there's no substitution model
   for this) and banned 1-2 matches (`SecondYellowSuspensionMatches`/`StraightRedSuspensionMatches`).
5. **InjurySimulation** (`engine/injury_simulation.go`) - independently, a flat 1.5% chance per
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

## Player skill progression

Skills drift match-by-match, not once a season: right after every league match or cup tie is
simulated, each player who featured gets an independent shot at growing (or declining), scaled by
how many minutes of that specific match they played. `updateSkillsAfterCompetitiveMatch`
(`engine/simulation.go`, called from `simulateMatch` and `engine/cups.go`'s `simulateCupTie`,
right after the match's other post-processing) is the trigger; `updatePlayerSkillsAfterMatch`
(`engine/progression.go`) does the actual drifting. Deliberately not called for friendlies - see
`engine/friendlies.go`'s `simulateFriendly` and `init26_friendlies.sql`'s own rationale for why
friendlies have no effect beyond the match itself.

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
   `[1, 99]`. Age and trainer skill are both read fresh for that match, not cached from season
   start.
4. If nothing actually drifted this match (quite possible - see the small per-skill chance
   above), there's no database write and no log row; if something did, it's persisted immediately
   and logged to the same `player_progression_log` table the old season-end system used - see
   `/players/evolutions`, unchanged, just now fed by many small match-triggered updates across a
   season instead of one big batch at rollover.

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
  capacity × an occupancy rate: a base rate (`demandRate`) that's exactly `BaseOccupancyRate`
  (0.5) at `DefaultTicketPrice` (10) and slides linearly down toward `MinOccupancyRate` (0.1) as
  price rises to `MaxTicketPrice` (100, scaled by `PriceSensitivity` = 0.35) - or up, more gently,
  toward `MaxOccupancyRate` (1.0) as price drops toward `MinTicketPrice` (1) - plus ±`OccupancyJitter`
  (0.2) random noise on top, clamped to `[MinOccupancyRate, MaxOccupancyRate]`. There's still no
  model for what else should plausibly drive attendance (form, rivalry, weather), but price itself
  is now a real tradeoff rather than a free lever: `optimalTicketPrice` scans every price in
  `[MinTicketPrice, MaxTicketPrice]` for whichever maximizes expected `price × demandRate(price)`
  (currently 69, well below the maximum - see the CPU-team section below, which charges exactly
  this rather than simply maxing out).
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

- **Wages** - deducted weekly, not per match (`PayWeeklyWages`, Monday 00:00 UTC via its own cron
  job), as the sum of every squad player's wage (`GetTeamWageBill`) plus the team's trainer's wage
  if it has one. A player's own wage is derived from their skills
  (`common.PlayerWageFromSkills`): the sum of all seven core skills × `WagePerSkillPoint` (5) - a
  higher-rated player costs more to keep, with no separate negotiation needed for the baseline
  figure (negotiated wages on signing/renewal are checked against this as a floor - see
  `common.MaxAcceptableRenewalYears`/offer-acceptance logic in `common/contracts.go`).
- **Transfers** - a fixed-price listing (`common/transfers.go`) or an accepted negotiated offer
  (`common/offers.go`) debits the buyer and credits the seller the agreed fee in one atomic
  transaction; a free-agent signing costs nothing but the player's wage.
- **Stadium expansion** (`UpgradeStadium`, `common/stadium.go`) - adds an owner-chosen number of
  seats, between `MinStadiumUpgradeSeats` (100) and `MaxStadiumUpgradeSeats` (5,000), costing
  current capacity × `StadiumUpgradeCostPerCurrentSeat` (10) × seats requested ÷
  `StadiumUpgradeSeats` (500) - so both a bigger stadium and a bigger expansion cost more in
  absolute terms. AI-managed teams always request the `StadiumUpgradeSeats` default.
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

`common/database.go`'s `UpdateTeamFinancesAfterMatch` is the one place both league matches and cup
ties route through after a result: it credits the home side's ticket revenue, credits sponsorship
to both sides, and triggers `RunAITeamManagement` for either side that's currently unowned (see
below). Friendlies deliberately skip all of this - no ticket revenue, no sponsorship, no AI
upkeep hook - since a friendly isn't a competitive result with financial stakes.

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
`UpdateTeamFinancesAfterMatch` above) - there's no separate scheduler job for it, so AI upkeep
happens exactly as often as that team actually plays.

- **Caretaker** (tier 1, the baseline every level gets):
  1. **Keeps the default lineup fresh** (`refreshAiTeamLineup` re-picks the current best XI from
     the squad every match, so retirements, academy call-ups, and signings show up in the lineup
     immediately rather than waiting for season end).
  2. **Resolves every pending transfer offer on its own players** - this is the direct answer to
     "how does the CPU decide on a transfer offer": accept if the fee is at least
     `minAcceptableTransferFeeWageMultiplier` (20) × the player's own market-rate wage, otherwise
     reject. The player's personal terms are never in question here - `MakeTransferOffer` never
     even lets a human create an offer the player themself wouldn't accept - so this check is
     purely about whether the fee is good enough for the selling club. This runs at every AI
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
     the weakest player there who isn't loan-listed is put on the transfer list at 20x their
     market wage (the same bar used to accept an incoming offer) - at most one new listing per
     match, so listings build up gradually while a surplus lasts, and never one that would leave
     fewer than `minSquadSizeAfterSale` owned, unlisted players. Needs the transfer window open.
  3. **Sign a free agent for the weakest position** - only when `WeakestPositionNeed` flags a core
     position as genuinely thin, not merely below-average; picks whichever available free agent
     rates highest there.
  4. **Charge whichever ticket price maximizes expected revenue** (`optimalTicketPrice`, see
     above) - this used to simply be `MaxTicketPrice`, back when attendance never fell as price
     rose, but now that it does, always maxing out would just as often cost revenue as earn it.
- **Director** (tier 3) adds active market participation on top (`manageMarketIfDirector`):
  1. **Buy a transfer-listed upgrade** for its single neediest core position - but only a listing
     that's a genuine improvement over that position's current average quality
     (`PositionNeed.AvgValue`), unlike the free Manager-tier signing above which takes any free
     agent to fill a thin spot regardless of quality, since that one costs nothing.
  2. **Expand the stadium** once the budget comfortably covers the cost with room to spare -
     specifically `stadiumUpgradeBudgetCushionMultiplier` (3) × one upgrade's cost, so it keeps a
     working cash reserve rather than spending down to the wire.
  3. **Upgrade its trainer** to a clearly better one on the open market (at least
     `trainerUpgradeMinSkillMargin` (10) skill points above its current coach, or hire one at all
     if it has none) - trainers have no signing fee, so budget never gates this, only that skill
     margin (avoiding constant churn over noise-level differences).

Every one of these calls the same production code path a human manager would use
(`common.MakeContractRenewalOffer`, `common.ReleasePlayer`, `common.SignFreeAgentAtMarketRate`,
`common.AcceptOffer`/`RejectOffer`, `common.ExecuteTransfer`, `common.UpgradeStadium`,
`common.HireTrainer`) - an AI team gets no special-cased shortcuts or exemptions from the same
validation (budget, transfer window, contract state) a human's action would have to pass.
