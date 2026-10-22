# Proposal

## Why

A simulation of the shipped economy (`backend/engine/economysim_test.go`) showed total money
inflating ~23x in 18 seasons and ~80x in 36: attendance scaled with whatever stadium was built,
the "optimal" ticket price was 7x the default, and tier income grew ~5x per division while wages
stayed flat. See `design.md` for the full audit.

## What Changes

- Attendance is driven by club members (socios), price, form, opponent and weather, capped by
  capacity, with no minimum occupancy; weather is drawn once and shown in the match report.
- New per-team `socios` count, drifting at each season rollover towards a Fanbase- and
  position-based target.
- `optimalTicketPrice` scans prices against the new demand model.
- Director-level CPU teams only expand a stadium that keeps selling out.
- A division wage factor (×4.0 / ×2.1 / ×1.1 for tiers 1 / 2 / 3) applied when wages are paid.

## Capabilities

### New Capabilities

- `socios`: club members, attendance model and optimal ticket price.

### Modified Capabilities

- `team-finances`: wages scaled by division; gate driven by socios.
- `cpu-team-management`: stadium expansion requires recent sell-outs.
- `fanbase`: attendance influence moved from a direct multiplier to socios and the rival factor.

## Impact

Shipped in `5f2f17a` (scruz84/striker#264), with migration `resources/sql/init80_socios.sql`.
This change was archived when the project adopted OpenSpec; the resulting requirements were
written straight into the main specs rather than through deltas.
