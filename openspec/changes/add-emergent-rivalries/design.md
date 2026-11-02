# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- Meetings are already queryable (`common.GetHeadToHead`, `common/headtohead.go`) but nothing is
  stored per pair. Team names are random (`common.RandomTeamName`) and clubs can be renamed
  (`common/teamrename.go`), so nothing in a name is a reliable link between clubs.
- A league match is simulated and saved (`engine/simulation.go` `simulateMatch`,
  `common.SaveLeagueMatchResultTx`) up to 15 minutes before kickoff. Money and the fanbase drift
  (`UpdateTeamFinancesAfterMatch` → `applyFanbaseDrift`) run right after the save; availability,
  skill growth and notifications wait for the replay through the pending post-match record
  (`engine/postmatch.go`: `completePostMatchEffects` / `ApplyDuePostMatchEffects`, each record
  claimed by exactly one replica). Cup ties follow the same path (`engine/cups.go`).
- Season rollover runs per country under its advisory lock (`engine/season.go`
  `advanceCountrySeasonIfComplete`, `recordSeasonHistory` has the final standings).
- The crowd is drawn in `engine/stadium.go` (`stadiumAttendanceAndRevenue`, `rivalFactor`), cards in
  `engine/discipline_simulation.go` (`ProbCardIncident = 0.03`), fanbase drift in
  `engine/fanbase.go` (`driftFanbaseForResult`, 15%).
- `common.IsKeyPlayer(squad, playerId)` (`common/transferscoring.go`) already defines "a player
  whose sale leaves the squad lame".

## Goals / Non-Goals

**Goals:**
- One row per pair, updated by single atomic statements, so concurrent replicas and concurrent
  matches of the same pair can never lose or double an update.
- The level a match is played with is fixed at kickoff and stored with the match, so a played
  match always shows the level it had, and a replay can never be spoiled by a newer level.

**Non-Goals:**
- Recomputing rivalries from the full match history (no backfill: every pair starts at 0 when this
  ships; see Migration Plan).
- Any admin tooling to edit rivalries.

## Decisions

### D1. Storage: `team_rivalries`, one row per unordered pair
Migration `resources/sql/init94_team_rivalries.sql` (renumber to the next free `initN` if another
change lands first):
`team_rivalries (team_low_id bigint, team_high_id bigint, country varchar, score real NOT NULL
DEFAULT 0, updated_at timestamptz, PRIMARY KEY (team_low_id, team_high_id), CHECK (team_low_id <
team_high_id))`, index on `team_high_id` and on `country`. Both team ids reference `teams` with
`ON DELETE CASCADE`. The same migration adds `rivalry_level smallint NOT NULL DEFAULT 0` to
`matches` and `cup_ties`.
`common/rivalries.go` holds `AddRivalryPointsStmtSql` (an `INSERT … ON CONFLICT DO UPDATE SET
score = LEAST(40, team_rivalries.score + EXCLUDED.score)` inside a CTE that reads the old score
`FOR UPDATE`, returning old and new), `GetRivalryScoreStmtSql`, `GetTopRivalriesStmtSql` and
`DecayCountryRivalriesStmtSql`, plus `RivalryLevel(score) int` (`RIVALRY_NONE = 0`,
`RIVALRY_RIVAL = 1`, `RIVALRY_CLASICO = 2`).
*Alternative:* a row per (team, other team). Rejected: two rows per pair can drift apart and every
update would need two writes.

### D2. Level at kickoff, stored with the match
`simulateMatch` and `simulateCupTie` read the pair's level before `runMatchSimulation` (clubs of
the same country only) and put it on `MatchSimulation.RivalryLevel`. `DisciplineSimulation` reads it
(3% / 3.6% / 4.2%), `matchdayContext` gets a `rivalryLevel` for `rivalFactor` (×1 / ×1.10 / ×1.20)
and `applyFanbaseDrift` takes it (15% / 30% / 45%). `SaveLeagueMatchResultTx` /
`SaveCupTieResultTx` write it to `rivalry_level`, and the post-match payload carries it with the
red-card count already present in the events.
Friendlies pass level 0 everywhere.

### D3. Match events applied with the post-match effects
`applyPostMatchEffects` calls `applyRivalryForMatch(kind, payload)` for `league` and `cup` kinds:
points per the "Rivalry events" scenarios, ×1.5 when both teams have a `user_id` at that moment,
then `AddRivalryPoints`. Because the record is claimed once, the points are added once; because it
runs after the replay, a level rise (old level < new level) can notify at once without spoiling.
A cup tie knows it eliminated someone (every tie is single-leg knockout) and whether it is the
final from its round and the bracket size (`engine/bracket.go`).

### D4. Season events and decay inside the rollover
In `advanceCountrySeasonIfComplete`, right after `recordSeasonHistory`, for each division add +4
(×1.5 if both human) to the champion–runner-up pair, then run `DecayCountryRivalriesStmtSql`
(`score = CASE WHEN score * 0.6 < 0.5 THEN 0 ELSE score * 0.6 END WHERE country = $1`). It runs
under the existing country rollover lock, so a second replica never decays twice.

### D5. Key-player sales
`ExecuteTransfer` and `AcceptOffer` already lock the player and both budgets. Before the move they
evaluate `IsKeyPlayer(sellerSquad, playerId)` from the squad they read inside the lock; after the
commit, best-effort, they add +3 (×1.5 if both human) when buyer and seller share a country.
Loans (`AcceptLoan`, `ExerciseLoanBuyOption`) are untouched.

### D6. Notification and API
New `NOTIFICATION_RIVALRY = 14` in `common/fmconstants.go`, keys
`notifications.messages.rivalryRival` and `notifications.messages.rivalryClasico` (params `team`,
`teamId`), Spanish templates in `common/notificationtranslations.go`, sent with a link to the other
club. `GET /v1/teams/team/:id/rivalries` returns up to 3 `{teamId, teamName, score, level}` with a
level. `TeamMatchSummary` gains `rivalry` (stored `rivalry_level` for a played match, current level
for an upcoming one); the match viewer payload gains the same field.

## Risks / Trade-offs

- [Every pair in a division drifts into "Rival" through ordinary close games] → per-event points
  are small (≈1.9 a season for a typical pair, steady state ≈4.75 with ×0.6 decay, below 8); a
  calibration test (tasks 2.4) simulates seasons and asserts the share of Rival and Clásico pairs.
- [Positive feedback: a Clásico has more cards, cards add points] → capped at +2 per match and by
  the 40 ceiling; decay still pulls it down.
- [Bigger gates in rival matches shift the economy] → only a few pairs qualify; checked with the
  economy simulation.

## Migration Plan

Additive migration; every pair starts at 0, so the first rivalries appear during the first season
after release. Rollback: the feature reads level 0 when the table is empty, so dropping the reads
restores today's behavior; the table and columns can stay.
