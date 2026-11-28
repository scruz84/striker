# Design

## Snapshot instead of soft delete

Keeping the `players` row with a `retired` flag would make every query over players (squads,
markets, AI, national-team pools, search) need a filter, and a forgotten one would put a retiree
back on a pitch. A separate `retired_players` table keeps those queries untouched: a retired player
simply is not in `players`. The history tables keep his id; readers resolve the name with
`LEFT JOIN players ... LEFT JOIN retired_players`.

## Migration `init122_retired_players.sql`

- `CREATE TABLE IF NOT EXISTS retired_players (player_id PK, first_name, last_name, birth_country,
  avatar, best_position, age, last_team_id REFERENCES teams ON DELETE SET NULL, last_team_name,
  retired_season, retired_at)`.
- `ALTER TABLE ... DROP CONSTRAINT IF EXISTS` on `player_season_awards_player_id_fkey`,
  `player_transfers_player_id_fkey`, `player_club_history_player_id_fkey`.

Re-runnable. No TRANSACTION_* reason and no notification key.

## Writing the snapshot

`recordRetiredPlayerTx` inserts from the player's own row in the retiring transaction
(`ON CONFLICT DO NOTHING`): last club is `COALESCE(team_id, last_team_id)`, the season that club's
country's `current_season`, the best position `BestCorePosition` computed in Go.

## Readers that must not change

Without the foreign key, a retired player's transfer stays in `player_transfers`. The three readers
that list transfers of players in the game add `EXISTS (SELECT 1 FROM players ...)`:
`getTopTransferFeeStmtSql`, `getRecentLeagueTransfersStmtSql` and, inside its ordered subquery so
the limit is not wasted, `getTransferComparablesStmtSql` (the plan keeps the ordered index walk; the
plan test allows the primary-key probe between the limit and the index scan).

## World reset

`handlers.teamPlayerDependentTables` lists `player_season_awards`, `player_transfers`,
`player_club_history` and `retired_players`, since the cascade from `players` no longer clears them.

## Code map

- `backend/resources/sql/init122_retired_players.sql`
- `backend/common/retiredplayers.go` (`recordRetiredPlayerTx`, `GetRetiredPlayer`, `RetiredPlayerDetails`)
- `backend/common/database.go` `RetirePlayer`, `backend/common/trainers.go` `RetireFreeAgentAsTrainer`
- `backend/handlers/playerHandlers.go` `PlayerDetailsHandler`, `retiredPlayerDetails`
- `backend/handlers/initializer.go` `teamPlayerDependentTables`
- `backend/common/transferstats.go`, `leaguetransfers.go`, `playertransfers.go` (the `EXISTS` filters)
- `frontend/src/features/player/RetiredPlayerView.tsx`, `PlayerDetailsPage.tsx`, `api/hooks.ts` `usePlayerProfile`
