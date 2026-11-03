# Tasks

## 1. Squad view (frontend)

- [x] 1.1 Add the `contract` sort to `SquadList` (own squad only, starts descending, flat list, seasons left on each row) and the A-Z name tie-break for every sort; verify "Longest contracts first", "Flip the order" and "Rival squad" with vitest tests.
- [x] 1.2 Add the "Last season" chip with its tooltip and en/es strings; verify "Last season" and "Loaned-in player" with vitest tests.

## 2. Checks

- [x] 2.1 Run `npx tsc --noEmit && npx oxlint && npx vitest run` and `openspec validate add-squad-contract-sort --strict`; verify all pass.
