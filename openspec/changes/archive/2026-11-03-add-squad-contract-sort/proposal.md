# Proposal

## Why

Deciding whether to renew a player starts with knowing how long each contract still runs, but the
squad view only shows it inside each player's sheet. A manager has to open every player to find
the contracts that are about to run out, and a player in the last season of his deal is easy to
forget until he leaves for free at season end.

## What Changes

- The squad's "Sort by" menu gets a **Contract** option, on the manager's own squad only. Picking
  it sorts from the most seasons left to the fewest; the existing arrow flips the order.
- While sorted by contract, the squad is one flat list (no position groups) and every row shows
  the seasons left ("3 seasons left").
- On the own squad, every player in the last season of his contract carries a **Last season**
  chip, whatever the sort. The threshold is the dashboard's `CONTRACT_EXPIRY_WARNING_YEARS` (1),
  so the chip and the dashboard's "contracts expiring" item agree. A loaned-in player never gets
  it: renewing is his parent club's call.
- Ties in any sort fall back to the player's name, always A-Z.

Out of scope:
- Remembering the chosen sort between visits.
- Contract information on a rival squad: those terms stay redacted (`Player.RedactFinancials`).
- Any backend change: the own squad already carries `contractYearsRemaining`.

Open questions for the owner: none.

## Capabilities

### New Capabilities
- `squad-list`: the squad view's sorting and contract warnings.

### Modified Capabilities
- None.

## Impact

- Frontend: `frontend/src/features/team/SquadList.tsx`, `SquadList.test.tsx`; en/es strings
  (`team.contractExpiringChip`, `team.contractExpiringTooltip`).
