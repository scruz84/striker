# Proposal

## Why

When a manager opens the transfer offer dialog ("Make Offer") they have to guess a fee: the
only reference is the asking price of a listed player, and nothing tells them what similar
players have actually cost recently. The data already exists (`player_transfers` records
every completed move with its fee), but it is only shown per player or per league, never as
"what does a player like this one go for". A comparison of recent comparable sales makes bids
better informed without exposing anything private.

## What Changes

- The offer dialog (`TransferOfferDialog`) gains a "Recent comparable transfers" section: up
  to 8 recent completed transfers of players with the same best position, a similar average
  skill and a similar age, each with the fee paid, date, selling and buying club, and a
  summary line (median fee and range).
- Comparables are matched and displayed using only information that is public for any
  player: best position, age and average skill. Individual skill ratings, traits, stamina,
  form, wage and contract of the comparable players are never returned or shown.
- Every new `player_transfers` row stores a snapshot of the moved player's best position, age
  and average skill at the moment of the transfer, so a comparable is judged by what the
  player was when sold, not what he has become since (and his current private ratings never
  leak through the matching).
- A new authenticated endpoint `GET /v1/teams/player/:id/transfer-comparables` returns the
  comparables for a player.

## Capabilities

### New Capabilities

- `transfer-comparables`: which past transfers count as comparable to a given player, what a
  comparable entry exposes (and what it must never expose), and how the offer dialog presents
  them.

### Modified Capabilities

None. The transfer flows themselves (`team-finances` "Transfers") keep their behavior; the
snapshot columns are an implementation detail of recording a transfer.

## Impact

- Backend: new migration `backend/resources/sql/init88_player_transfer_snapshot.sql`;
  `common/playertransfers.go` (`RecordPlayerTransfer` writes the snapshot, new
  `GetTransferComparables`); new handler in `handlers/playerHandlers.go`; route in
  `service/fmservice.go`.
- Frontend: `api/types.ts`, `api/hooks.ts`, `components/TransferOfferDialog.tsx` (plus a new
  `components/TransferComparables.tsx`), `i18n/locales/{en,es}.json`.
- Database: every new or changed query must be served by an index on its `WHERE`/`JOIN`/`ORDER BY`
  columns, added in the same migration when none exists, and checked with an `EXPLAIN` test
  (design §6-7). The comparables query gets a partial index on `player_transfers`; the other
  lookups use existing primary keys.
- No money movement, no engine job, no notification, no CPU behavior change.

## Out of scope

- Using comparables in CPU valuation or bidding (`common.ScoreTarget`, `aitransfers.go`).
- Showing comparables on the "Buy" (asking price) confirmation, the listing form, or the
  player page outside the offer dialog.
- Back-filling snapshots for transfers recorded before this change: their past skills are
  unknown, so those rows are never comparables. The list fills up as new transfers happen.
- Loans, loan fees, free-agent premiums and releases as comparables.

## Open questions for the owner

1. The section is shown in the offer dialog for every target, listed or not, because it only
   uses public figures. Should it be restricted to transfer-listed players only?
2. Each row shows the comparable's average skill at the time of sale (the same single figure
   already public on any player page). If even that is considered too revealing, rows can
   show only position and age, keeping average skill for matching only.
