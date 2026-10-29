# Proposal

## Why

The "Recent comparable transfers" section (capability `transfer-comparables`) only appears in the
offer dialog for another club's player. A manager about to list one of their own players has no
reference for the asking price and has to guess, or open some other club's similar player and
start a fake offer to see the figures. The data and the endpoint already work for any player,
so showing the same section when listing costs little.

## What Changes

- Show the existing "Recent comparable transfers" section in both dialogs that put one of the
  manager's own players on the transfer market. It sits below the asking-price field:
  - `ListPlayerForTransferDialog`, opened by "List for transfer" on the player profile and by
    "List for sale" in the own-squad player drawer;
  - the transfer market's "List a player" dialog (`ListPlayerDialog` in `TransferMarketPage.tsx`),
    once a player is picked from the dropdown. It follows the selection and is hidden while no
    player is picked.
- The comparables are fetched only while the dialog is open and a player is chosen. They are
  cached per player (same query key as the offer dialog), so switching players in the dropdown
  does not refetch one already seen within a minute.
- If loading fails, the listing dialog says the player can still be listed, not that an offer
  can still be sent. The other texts (title, summary, empty, broadened, row meta) stay shared
  with the offer dialog.
- No backend change. `GET /v1/teams/player/:id/transfer-comparables` already accepts any player
  id, own players included, and returns only public figures.

### Out of scope

- Showing comparables for a player who is already listed (to reprice him). There is no edit-price
  flow today: the owner has to unlist and list again, and the listing dialog will then show them.
- Showing comparables when answering a received offer for one of the manager's players.
- Loan listings (`OfferLoanDialog`). Loans are never comparables, and a loan fee is a different
  kind of price.
- A suggested asking price computed from the comparables. The section only shows figures.
- Any change to which transfers count as comparables, the bands, the 60-day window or the cap
  of 8.

### Open questions for the owner

1. Should the received-offer flow (accept/reject an offer for your player) also show the
   comparables? It is the other moment when a manager judges a price for their own player.
2. Should the profile get a standalone "Comparable transfers" button for own players, so they
   can check a price without starting a listing (useful for a player already listed)? This
   proposal assumes the listing dialogs are enough.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `transfer-comparables`: adds a requirement for showing the comparables section in the dialogs
  that list one of the manager's own players for transfer. The offer-dialog requirement and the
  endpoint are unchanged.

## Impact

- Frontend only: `frontend/src/components/ListPlayerForTransferDialog.tsx`,
  `frontend/src/features/transfers/TransferMarketPage.tsx` (`ListPlayerDialog` moves to its own
  file `ListPlayerDialog.tsx` so it can be tested),
  `frontend/src/components/TransferComparables.tsx` (a context-specific error text), new tests,
  `frontend/src/i18n/locales/{en,es}.json` (one new key).
- One extra GET per listing dialog opened. It is served by the existing partial index
  `player_transfers_comparables_idx`, so there is no new SQL, migration or index.
- Spec: the `transfer-comparables` purpose and Code map name the listing dialogs.
