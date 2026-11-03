# Tasks

## 1. Storage and API (backend)

- [x] 1.1 Add `resources/sql/init99_player_watchlist.sql`, re-runnable; verify by applying it twice on the test DB.
- [x] 1.2 Add `common/watchlist.go` (`WatchPlayer` with the 30 cap, `UnwatchPlayer`, `GetWatchlist`, `ClearWatchlist`); verify "Follow", "Limit" and "Own player" with integration tests, including two concurrent follows at 29 that end at exactly 30.
- [x] 1.3 Add the three `/v1/user/watchlist` endpoints; verify with handler tests that a user only sees and changes his own club's list.

## 2. Events and notifications (backend)

- [x] 2.1 Remove the player from the buyer's / borrower's watchlist inside `movePlayerToTeam` and `AcceptLoan`; verify "Signed by the follower" with an integration test.
- [x] 2.2 Add `NOTIFICATION_WATCHLIST`, the four keys with Spanish templates and `NotifyWatchers`, called after the commit of every market event and the season-end expiry; verify "Player listed", "Player released" and "Own club's sale" with integration tests.
- [x] 2.3 Clear the watchlist in the club reset and add it to the data export; verify the user-accounts "Watchlist" and "Watchlist in the export" scenarios with integration tests.
- [x] 2.4 Verify "Selling club" by asserting that no team, player or transfer payload contains watcher data (handler test over the public endpoints).

## 3. Frontend

- [x] 3.1 Add `useWatchlist`, `WatchToggle` and the toggles on the player page, other clubs' squads, the market rows and search; add en/es strings; verify with vitest tests that toggling calls the API and updates every star.
- [x] 3.2 Add the Watchlist tab to `TransferMarketPage` with status and actions per row; verify "Listed player" with a vitest test.

## 4. Specs and checks

- [x] 4.1 Add the Code map to the new `player-watchlist` spec and update the Code map of `user-accounts`; verify `openspec validate add-player-watchlist --strict` passes.
- [x] 4.2 Run backend `go build ./... && go vet ./... && go test ./...` and frontend `npx tsc --noEmit && npx oxlint && npx vitest run`; verify all pass.
