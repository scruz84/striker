# Proposal

## Why

There is no way to keep an eye on a player: a manager who likes a striker on another team has to
remember him and keep checking the transfer list, the loan list and the free agents by hand, and
he finds out too late that the player was listed, released or sold. A watchlist with
notifications turns scouting into a routine ("follow, get told when he becomes available, act")
and reuses the notification feed and Web Push that already exist.

## What Changes

- A manager can **follow** up to 30 players (star toggle) from the player page, squad lists of
  other clubs, the transfer, loan and free-agent lists and search results.
- A **Watchlist** tab in the transfer market lists the followed players with their current status
  (club, transfer-listed with asking price, loan-listed with terms, free agent) and quick actions
  (make an offer, buy, sign, take on loan).
- A followed player triggers a `NOTIFICATION_WATCHLIST` to the follower when he is
  **transfer-listed**, **loan-listed**, **becomes a free agent** or **changes club**. The follower
  is never told about his own club's actions on the player.
- Following is private: no club learns who follows its players.
- The watchlist belongs to the club: it is cleared when the club is reset, removed when the
  account is deleted, and included in the data export.

Out of scope:
- Notifications on skill changes, injuries or price drops (listings cannot change price; a re-list
  is a new listing and notifies again).
- Shared watchlists between members of a group, or notes on players.

Open questions for the owner:
- Is 30 the right cap?
- Should following a player of a human club ever be visible to that club (e.g. "3 clubs are
  following him")? The default here is never.

## Capabilities

### New Capabilities
- `player-watchlist`: following players, the watchlist view, its notifications and its privacy.

### Modified Capabilities
- `user-accounts`: "Club reset pending business" clears the watchlist; "Data export" includes it.

## Impact

- Backend: new migration (`player_watchlist`), `common/watchlist.go`, hooks in
  `common/transfers.go` (`CreateTransferListing`, `ExecuteTransfer`), `common/offers.go`
  (`AcceptOffer`), `common/loans.go` (`CreateLoanListing`, `AcceptLoan`), `common/contracts.go`
  (`ReleasePlayer`, contract expiry), the club reset and account deletion/export code.
- API: `GET /v1/user/watchlist`, `PUT` / `DELETE /v1/user/watchlist/:id`. The follow toggles read
  the followed ids from the watchlist response, so no other payload changes.
- Frontend: `WatchToggle` component, Watchlist tab in `TransferMarketPage`; en/es strings; Spanish
  notification templates.
