# Design

## Context

See proposal.md - Why. Current state that shapes the approach:

- Market events happen in a few production functions, all following the "lock, validate, write,
  commit, then notify best-effort" rule: `CreateTransferListing`, `ExecuteTransfer`
  (`common/transfers.go`), `AcceptOffer` (`common/offers.go`), `CreateLoanListing`, `AcceptLoan`,
  `ExerciseLoanBuyOption` (`common/loans.go`), `ReleasePlayer` and the season-end contract expiry
  (`common/contracts.go`, `AdvancePlayerContract`). Every permanent move goes through
  `movePlayerToTeam(tx, …)`.
- Notifications: `NotifyTeamOwner` / `NotifyTeamOwnerWithLink` with a `NotificationMessage{Key,
  Params}`; Spanish templates in `common/notificationtranslations.go`; Web Push picks them up.
- Club reset (`common/worldreset.go` / account lifecycle in `engine/accountlifecycle.go`) and data
  export (`handlers/accountHandlers.go` `ExportAccountHandler`) enumerate what a club owns.

## Goals / Non-Goals

**Goals:**
- One fan-out helper called after the commit of each market event, so a new event type is one line.
- The watchlist never blocks or slows a transfer: fan-out is best-effort and after the commit.

**Non-Goals:**
- Digest notifications or per-event notification preferences.

## Decisions

### D1. Storage
Migration `resources/sql/init99_player_watchlist.sql` (renumber if needed):
`player_watchlist (team_id bigint REFERENCES teams ON DELETE CASCADE, player_id bigint REFERENCES
players ON DELETE CASCADE, created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (team_id,
player_id))`, index on `player_id`. `MaxWatchlistSize = 30`. `common/watchlist.go`:
`WatchPlayer` (an `INSERT … SELECT … WHERE (SELECT count(*) …) < 30 ON CONFLICT DO NOTHING`, so
the cap holds under concurrent requests), `UnwatchPlayer`, `GetWatchlist(teamId)` (players with
club, active transfer listing, active loan listing) and `ClearWatchlist(tx, teamId)`.

### D2. Leaving the watchlist on signing
`movePlayerToTeam` also runs `DELETE FROM player_watchlist WHERE team_id = $1 AND player_id = $2`
in the same transaction, which covers transfers, accepted offers and buy options; `AcceptLoan`
does the same for the borrower.

### D3. Notification fan-out
`NotifyWatchers(dbCtx, playerId, event, actingTeamIds ...int64)` selects the followers of the
player except the acting clubs (seller and buyer, lender and borrower) and sends
`NOTIFICATION_WATCHLIST = 16` with keys `notifications.messages.watchlistListed`,
`watchlistLoanListed`, `watchlistFreeAgent`, `watchlistMoved` (params `player`, `playerId`,
`team`, `price` where relevant), each linking to the player. Called after the commit of every
function listed in Context; for the season-end expiry it is called per expired player after the
batch commits. Spanish templates in `common/notificationtranslations.go`.

### D4. API and privacy
`GET /v1/user/watchlist`, `PUT /v1/user/watchlist/:id`, `DELETE /v1/user/watchlist/:id` under
the authenticated user group, acting on the caller's club only. No other endpoint reads
`player_watchlist`. The data export adds a `watchlist` array (player id, name, followed since);
the club reset calls `ClearWatchlist` with its other pending business.

### D5. Frontend
`useWatchlist()` loads the list once (TanStack Query) and exposes `isWatched(id)`;
`WatchToggle` (star `IconButton`) is used on `PlayerDetailsPage`, other clubs' `SquadList`, the
transfer/loan/free-agent rows and `SearchPage`. A sixth tab "Watchlist" in `TransferMarketPage`
reuses the existing buy / offer / loan / sign dialogs. Keys `watchlist.*` in en/es.

## Risks / Trade-offs

- [A popular player listed by a big club fans out many notifications] → at most one per follower
  and the number of followers is bounded by human clubs × 30.
- [Season-end expiry notifies many followers at once] → it runs after the rollover commits,
  outside the rollover lock.

## Migration Plan

Additive migration; empty watchlists for everyone. Rollback: redeploy the previous service; the
table is ignored.

## Code map

To copy into `openspec/specs/player-watchlist/spec.md` as its `## Code map` when the change is
archived.

- Storage: `backend/resources/sql/init99_player_watchlist.sql` (`player_watchlist`, index on
  `player_id`).
- Core: `common/watchlist.go` (`MaxWatchlistSize`, `WatchPlayer` under the
  `LockNamespaceWatchlist` advisory lock, `UnwatchPlayer`, `GetWatchlist` with each entry's
  listing and the same redaction as a rival squad, `ClearWatchlist`, `exportWatchlist`,
  `NotifyWatchers` with the `WatchEvent*` events, `DeletePlayerFromWatchlistStmtSql`).
- Leaving the list on signing: `movePlayerToTeam` (`common/transfers.go`, covering transfers,
  accepted offers and counter-offers, loans) and `SignFreeAgent` (`common/contracts.go`).
- Notifications (`NOTIFICATION_WATCHLIST = 16`, keys `notifications.messages.watchlist*`, Spanish
  templates in `common/notificationtranslations.go`), sent after each commit by
  `CreateTransferListing`, `ExecuteTransfer` (`common/transfers.go`), `finishOfferTransfer`
  (`common/offers.go`), `CreateLoanListing`, `notifyLoanAccepted`, `ExerciseLoanBuyOption`
  (`common/loans.go`), `ReleasePlayer`, `SignFreeAgent` (`common/contracts.go`) and, for expired
  contracts, `engine.ProgressPlayers` once every team is progressed.
- API: `GET /v1/user/watchlist`, `PUT` / `DELETE /v1/user/watchlist/:id`
  (`handlers/watchlistHandlers.go`, routes in `service/fmservice.go`).
- Frontend: `src/components/WatchToggle.tsx` (on `PlayerProfileRail`, other clubs' `SquadList`
  sheet, the transfer / loan / free-agent rows of `TransferMarketPage` and `SearchPage`),
  `src/features/transfers/WatchlistTab.tsx` (the market's sixth tab), hooks `useWatchlist`,
  `useWatchPlayer`, `useUnwatchPlayer`; strings `watchlist.*`.
- Tests: `common/watchlist_integration_test.go`, `handlers/watchlist_test.go`,
  `src/components/WatchToggle.test.tsx`.
