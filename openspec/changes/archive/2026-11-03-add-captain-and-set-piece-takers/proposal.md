# Proposal

## Why

Penalties and direct free kicks always go to the on-pitch player with the best Shot (or best
`0.6 × Shot + 0.4 × Pass`) (`bestOnPitchOutfielder` in `engine/scorer_simulation.go`), and a team
has no captain at all. Choosing who takes the set pieces and who wears the armband is one of the
most familiar decisions in football management: it is cheap to build, easy to understand, and it
lets a manager reward a specialist even when he is not the best finisher.

## What Changes

- Every lineup (default, friendly default, per league match, per cup tie, per friendly) can name a
  **captain**, a **penalty taker** and a **free-kick taker**; each is optional and defaults to
  "automatic".
- A chosen penalty or free-kick taker takes the set piece whenever he is on the pitch; otherwise the
  current automatic rule applies.
- The **captain** steadies his side: while he is on the pitch, a card that would go to one of his
  team-mates (not to himself) is cancelled with probability 10%.
- CPU teams name the oldest outfield starter as captain (ties: lowest id) and leave the takers
  automatic.
- The lineup editor lets a manager pick the three roles from the XI or the bench; the viewer's
  lineups pitch shows the captain's armband (C) from kickoff on.

Out of scope:
- Corner takers or any other set-piece roles (corners are not simulated).
- Morale, leadership attributes or any captain effect besides the card one.
- A "vice-captain" fallback.

Open questions for the owner:
- Is a 10% card cancellation the right size for the captain's effect, or should the captain be
  purely cosmetic?
- Should a goalkeeper be allowed as penalty taker? The default here is no (outfield only, like the
  automatic rule).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-simulation`: "Penalty resolution" and "Direct free-kick resolution" honour a chosen
  taker; new "Lineup roles" and "Captain's calming effect" requirements.
- `cpu-team-management`: new "CPU captain" requirement.
- `match-viewer`: new "Captain's armband" requirement.

## Impact

- Backend: new migration (role columns), `common.Lineup` role fields in every lineup getter/setter
  and the kickoff snapshot, validation, `engine/scorer_simulation.go` (taker choice),
  `engine/discipline_simulation.go` (captain), `common/ailineup.go`.
- API: optional `captainId`, `penaltyTakerId`, `freeKickTakerId` in lineup GET/PUT payloads.
- Frontend: role pickers in `MatchLineupForm` / `SelectedPlayerPanel`, armband in the viewer's
  `Lineups` and `MatchPitchChip`; en/es strings.
