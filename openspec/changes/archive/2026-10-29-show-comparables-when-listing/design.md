# Design

## Context

See proposal.md for the why. Current state:

- `TransferComparables` (`frontend/src/components/TransferComparables.tsx`) is a presentational
  component. It takes `result`, `isLoading` and `isError` and renders the loading, error, empty,
  broadened and list states. Its texts live under `transfers.offerDialog.comparables.*`. The
  error text ("…You can still send your offer.") only fits the offer dialog.
- `useTransferComparables(playerId, enabled)` (`api/hooks.ts`) fetches
  `GET /v1/teams/player/:id/transfer-comparables` with query key
  `['transferComparables', id]` and a 60 s stale time. Today only `TransferOfferDialog` calls it,
  with `enabled = open`.
- Own players are listed through two dialogs:
  - `ListPlayerForTransferDialog` (`components/`) has a fixed `player`. It is mounted by
    `PlayerProfileRail` (always mounted on the own player's profile and opened by its "List for
    transfer" button) and by the own-squad drawer in `features/team/SquadList.tsx`.
  - `ListPlayerDialog` (local to `features/transfers/TransferMarketPage.tsx`) has a player
    `Select` (`selectedPlayerId`) and is mounted once on the market's transfers tab.
- The endpoint has no ownership check and reads only public snapshot fields, so it already
  serves own players. Nothing in the backend changes.

## Goals / Non-Goals

**Goals:**
- Reuse `TransferComparables` and `useTransferComparables` unchanged in behavior. One component
  must keep both dialogs and the offer dialog looking the same.
- No request until a listing dialog is open with a player chosen.

**Non-Goals:**
- Refactoring the two listing dialogs into one. They differ in how the player is chosen. Merging
  them is unrelated to this change.
- New backend code, SQL, migration, TRANSACTION_* reason or notification key. This change adds
  none of them.

## Decisions

1. **Error text by prop, not by duplicating the component.** Add an optional
   `errorText?: string` prop to `TransferComparables`. It defaults to
   `t('transfers.offerDialog.comparables.error')`, so `TransferOfferDialog` needs no change.
   The listing dialogs pass `t('transfers.listPlayer.comparablesError')`.
   *Alternatives:* a `context: 'offer' | 'listing'` enum (couples the component to its callers);
   one neutral error string for both (it would lose the "you can still …" reassurance the spec
   asks for in each place).

2. **Shared i18n keys stay where they are.** The title, summary, empty, broadened and meta texts
   keep their `transfers.offerDialog.comparables.*` keys. Moving them to a neutral namespace
   would touch every locale entry and the existing tests for no user-visible gain. The only new
   key is `transfers.listPlayer.comparablesError`:
   - en: "Could not load comparable transfers. You can still list the player."
   - es: "No se pudieron cargar los traspasos comparables. Aún puedes ponerlo en venta."

3. **`ListPlayerForTransferDialog`**: call
   `useTransferComparables(player.id, open)` and render
   `<TransferComparables … errorText={…} />` below the asking-price `NumberField`, before the
   listing error alert. `PlayerProfileRail` mounts this dialog permanently, so gating on `open`
   is what keeps profile visits from firing the request.

4. **Market `ListPlayerDialog`**: call
   `useTransferComparables(selectedPlayerId === '' ? undefined : selectedPlayerId, open)`. The hook
   already disables itself for an `undefined` id. Render the section only when `selected` is
   defined, below the asking-price field. Switching the pick changes the query key. A player seen
   within the last 60 s is served from cache, and the new pick shows the loading skeleton.
   *Alternative:* keep the section always visible with a "pick a player" placeholder. That adds a
   string and a state for no information, so it is rejected.
   To make it testable, move `ListPlayerDialog` unchanged out of `TransferMarketPage.tsx` into its
   own file, `features/transfers/ListPlayerDialog.tsx`, before adding the section. Exporting it
   from the page instead would trip oxlint's `only-export-components` rule.

5. **Same query key as the offer dialog.** Comparables depend only on the target player, not on
   who is asking. Sharing `['transferComparables', id]` is correct and gives free cache hits.

## Risks / Trade-offs

- [The section makes the listing dialogs taller, especially on a phone] → It already fits the offer
  dialog at 360 px with the same max of 8 rows. `ResponsiveDialog` scrolls its content and the
  list button sits in the fixed actions bar, so it stays reachable. Check this at 360 px.
- [A manager may read the median as a recommended price] → Out of scope by design: the section
  shows figures only, the same as in the offer dialog.
- [One more request per listing dialog] → It is served by `player_transfers_comparables_idx`
  (`TestTransferComparables_UsesIndex`). Opening a listing dialog is rare compared with page
  loads.

## Migration Plan

Frontend-only and additive. Deploy with the normal frontend build. Roll back by reverting the
commit. There are no data or schema changes.

## Code map additions (for the main spec)

- `frontend/src/components/ListPlayerForTransferDialog.tsx` (comparables below the asking price).
- `frontend/src/features/transfers/ListPlayerDialog.tsx` (moved out of `TransferMarketPage.tsx`;
  comparables for the picked player).
- `frontend/src/components/TransferComparables.tsx` (`errorText` prop).
- Tests: `frontend/src/components/ListPlayerForTransferDialog.test.tsx`,
  `frontend/src/features/transfers/ListPlayerDialog.test.tsx`,
  `frontend/src/components/TransferComparables.test.tsx` (`errorText`).
