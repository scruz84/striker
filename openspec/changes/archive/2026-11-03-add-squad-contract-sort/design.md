# Design

## Decisions

- **Frontend only.** `GetFullTeamDetailsByUser` already returns the real `contractYearsRemaining`
  for the caller's own squad; a rival's reads 0 (`Player.RedactFinancials`), which is why the
  Contract sort is offered on the own squad only rather than shown and useless.
- **Starts descending.** Changing the sort field otherwise keeps the arrow's direction; the
  Contract sort resets it to descending so it opens as asked (longest contracts first), and the
  arrow gives the "about to run out first" view.
- **Flat list.** The position groups would split the renewal overview into four short lists; the
  contract sort shows the whole squad at once.
- **One threshold.** The chip reuses `CONTRACT_EXPIRY_WARNING_YEARS` from the dashboard's Pending
  panel, so the squad and the dashboard agree on who is expiring. Unlike the dashboard count, the
  chip skips loaned-in players: their renewal is the parent club's.
- **Tie-break outside the direction.** The name tie-break is applied after the direction factor,
  so equal contracts stay A-Z in both directions.

## Code map

- `frontend/src/features/team/SquadList.tsx`: `compareSquadPlayers`, `contractExpiring`, the
  `sorted` memo (tie-break), `changeSortField`, `SquadPlayerCard` (`showContract`, chip).
- Threshold: `CONTRACT_EXPIRY_WARNING_YEARS` in `frontend/src/features/dashboard/pendingItems.ts`.
- Strings: `team.contract`, `team.contractSeasons`, `team.contractExpiringChip`,
  `team.contractExpiringTooltip` in `frontend/src/i18n/locales/{en,es}.json`.
- Tests: `frontend/src/features/team/SquadList.test.tsx` ("SquadList contract sort").

After archiving, copy this Code map into `openspec/specs/squad-list/spec.md` as its `## Code map`.
