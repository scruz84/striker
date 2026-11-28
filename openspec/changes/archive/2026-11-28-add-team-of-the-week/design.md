# Design

## When a matchday is complete

A matchday is complete when it has matches, all played, and none still has a
`pending_post_match_effects` row - those are claimed when the replay ends (or by the recovery job
once its live window has passed). So `applyPostMatchEffects` of every league match calls
`maybeCompleteBestXi`, and the last of the matchday finds it complete. Nothing is visible before
the replays end, as the architecture rules require.

## Exactly once across replicas

Several replicas can finish the last matches of a matchday at once and both see it complete.
`SaveBestXi` claims the round with `INSERT INTO best_xi_rounds ... ON CONFLICT DO NOTHING RETURNING`
and writes the players in the same transaction; only the caller that inserted the round notifies.
The concurrent test runs four replicas' recovery pass and completion together.

## Migration `init124_best_xi.sql`

`best_xi_rounds (league_id, season, week_number PK)` and `best_xi_players (round, slot PK, line,
player_id, team_id, rating, goals, matches)`; `week_number` 0 is the team of the season (league
matchdays start at 1). The player's position comes from `player_match_stats.position` (add-player-stats).

## Selection

`SelectBestXi` (pure): per line (1 goalkeeper, 2 defence, 3 midfield, 4 attack - `BestXiLineOf`
maps the ten tactical positions), the highest rating, then minutes, goals and lower player id;
slots 1, 2-5, 6-9, 10-11. A short line leaves slots empty. The season candidates take the line they
played most (the lower on a tie) and their average rating, among players with
`RatingLeaderMinutesPerMatch` (36) minutes per match of the team that played the most.

## Notification keys

`notifications.messages.teamOfTheWeek` (`week`, `names`, `count`) linking to
`/division/<league>?xi=<week>`, and `notifications.messages.teamOfTheSeason` (`season`, `names`,
`count`) linking to `/division/<league>?season=<s>&xi=0`; Spanish templates in
`common/notificationtranslations.go`. No TRANSACTION_* reason.

## Code map

- `backend/resources/sql/init124_best_xi.sql`
- `backend/common/bestxi.go` (`SelectBestXi`, `IsLeagueWeekComplete`, candidates, `SaveBestXi`, `GetBestXi`, `GetPlayerBestXiCounts`)
- `backend/engine/bestxi.go` (`maybeCompleteBestXi`, `awardSeasonBestXi`, `notifyBestXi`); `engine/postmatch.go`; `engine/awards.go`
- `backend/handlers/statsHandlers.go` `LeagueBestXiHandler`
- `frontend/src/features/stats/BestXiCard.tsx`, `bestXiLayout.ts`
