# retired-players Specification

## Purpose
What is kept of a player once he retires: a snapshot of who he was, and his awards, transfers,
club history, match stats and best XI selections under his id, shown on a read-only profile.

## Requirements

### Requirement: Retirement snapshot
When a player retires (by age, or a free agent moving into coaching), the game SHALL store, in the same transaction that removes him, his name, birth country, best position, age, last club (his club, or a free agent's last club) with its name, and that club's country's current season. A second retirement of the same id MUST change nothing.

#### Scenario: Rostered veteran
- **WHEN** a 38-year-old striker of a club whose country is in season 7 retires
- **THEN** his snapshot reads age 38, best position striker, that club and season 7

#### Scenario: Free agent into coaching
- **WHEN** a free agent whose last club was C retires into coaching in season 1
- **THEN** his snapshot names C and season 1, and a trainer with his name joins the market as before

### Requirement: Record kept after retirement
A retired player's season awards, transfers and club history SHALL be kept under his id. A world reset MUST clear them together with the retirement snapshots.

#### Scenario: Golden Boot winner retires
- **WHEN** a player who won a Golden Boot retires
- **THEN** his Golden Boot count, his transfer and his clubs are still returned for his id

### Requirement: Retired player profile
The player endpoint SHALL answer a retired player's id with his snapshot (a `retired` object), career totals, awards, season lines, clubs and transfers, and 404 for an id that never existed. The player page SHALL show it read-only: a "Retired" chip, "Retired in season N, aged A", the last club linked, and no skills, contract or market actions.

#### Scenario: Open a retired player
- **WHEN** a user opens `/player/5` for a player who retired in season 9 aged 37
- **THEN** the page reads "Retired in season 9, aged 37" with his totals and no offer button

### Requirement: Market panels list players in the game
The biggest fee on the transfer market panel, a league's recent transfers and the transfer comparables SHALL only count transfers of players still in the game, as they did when a retired player's transfers were deleted with him.

#### Scenario: Record fee holder retires
- **WHEN** the player of the biggest fee ever paid retires
- **THEN** the panel shows the next biggest fee of a player still in the game

## Code map

- `backend/resources/sql/init122_retired_players.sql`
- `backend/common/retiredplayers.go` (`recordRetiredPlayerTx`, `GetRetiredPlayer`, `RetiredPlayerDetails`)
- `backend/common/database.go` `RetirePlayer`, `backend/common/trainers.go` `RetireFreeAgentAsTrainer`
- `backend/handlers/playerHandlers.go` `PlayerDetailsHandler`, `retiredPlayerDetails`
- `backend/handlers/initializer.go` `teamPlayerDependentTables`
- `backend/common/transferstats.go`, `leaguetransfers.go`, `playertransfers.go` (the `EXISTS` filters)
- `frontend/src/features/player/RetiredPlayerView.tsx`, `PlayerDetailsPage.tsx`, `api/hooks.ts` `usePlayerProfile`
