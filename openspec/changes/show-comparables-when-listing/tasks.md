# Tasks

## 1. Shared section

- [ ] 1.1 Add the optional `errorText?: string` prop to `frontend/src/components/TransferComparables.tsx`, defaulting to `t('transfers.offerDialog.comparables.error')` (design §1). Add a case to `TransferComparables.test.tsx` that renders the error state with a custom `errorText` and asserts it replaces the default. Verify with `npx vitest run src/components/TransferComparables.test.tsx`, and confirm `TransferOfferDialog.tsx` is unchanged.
- [ ] 1.2 Add `transfers.listPlayer.comparablesError` to `frontend/src/i18n/locales/en.json` ("Could not load comparable transfers. You can still list the player.") and `es.json` ("No se pudieron cargar los traspasos comparables. Aún puedes ponerlo en venta."). Verify both files parse and the key exists in both (`npx vitest run` covers any locale-parity test).

## 2. Listing from the player page and the squad drawer

- [ ] 2.1 In `frontend/src/components/ListPlayerForTransferDialog.tsx`, call `useTransferComparables(player.id, open)` and render `<TransferComparables … errorText={t('transfers.listPlayer.comparablesError')} />` below the asking-price field (design §3). Verify with `npx tsc --noEmit`.
- [ ] 2.2 Add `frontend/src/components/ListPlayerForTransferDialog.test.tsx` (mocking `../api/hooks` as `app/Header.test.tsx` does). Assert that the hook is called with `enabled = false` while closed and `true` while open, that the comparables section renders below the asking price, and that on a comparables error the custom note shows while the list button still calls `useListPlayer().mutate` with the typed price. Verify it passes.

## 3. Listing from the transfer market

- [ ] 3.1 Move `ListPlayerDialog` unchanged from `frontend/src/features/transfers/TransferMarketPage.tsx` into `frontend/src/features/transfers/ListPlayerDialog.tsx` and import it back (design §4). Verify with `npx tsc --noEmit && npx oxlint` (no new warnings) and confirm the market's "List a player" button still opens it in the running app.
- [ ] 3.2 In `ListPlayerDialog.tsx`, call `useTransferComparables(selectedPlayerId === '' ? undefined : selectedPlayerId, open)` and render the section with the listing error text only when a player is picked, below the asking-price field. Verify with `npx tsc --noEmit`.
- [ ] 3.3 Add `frontend/src/features/transfers/ListPlayerDialog.test.tsx`. Cover these cases: with no player picked there is no section and the hook gets an `undefined` id; picking a player shows his comparables; picking another player calls the hook with the new id; a comparables error still lets the player be listed. Verify it passes.

## 4. Spec and docs

- [ ] 4.1 Update `openspec/specs/transfer-comparables/spec.md` directly: widen its Purpose to "a manager making a transfer offer or listing one of their own players" (a Purpose is not carried by the delta). Verify by reading the file.
- [ ] 4.2 Run `openspec validate show-comparables-when-listing --strict` and verify it passes.

## 5. Final checks

- [ ] 5.1 In the running app, open the listing dialog from the player profile, the own-squad drawer and the market's "List a player" dialog, in English and Spanish and at 360 px width. Verify each dialog shows the section below the asking price, the list button stays reachable, and the browser console shows no missing-key warning.
- [ ] 5.2 Backend: `cd backend && go build ./... && go vet ./... && go test ./...`. Verify all pass (no backend change is expected, but this confirms it).
- [ ] 5.3 Frontend: `cd frontend && npx tsc --noEmit && npx oxlint && npx vitest run`. Verify all pass.

## Workflow follow-up

- Resolve the proposal's two open questions with the owner (received-offer flow, standalone profile button) before or during review.
- Archive the change (`/opsx:archive`) once merged. That merges the ADDED requirements into `openspec/specs/transfer-comparables/spec.md`. Then append the "Code map additions" from `design.md` to that spec's Code map.
