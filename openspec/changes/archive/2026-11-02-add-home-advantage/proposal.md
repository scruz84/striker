# Proposal

## Why

The simulator treats both sides identically: nothing in `engine/simulation.go` or the
`match-simulation` spec gives the home side any edge, so playing at home and away are the same
and a full stadium is only money. In real football home sides win about 45% of matches between
even teams (away about 28%). Giving the home side a modest edge that grows with the crowd that
actually turns up makes the fixture list matter, and makes socios, ticket price and stadium size
sporting decisions as well as financial ones.

## What Changes

- The home side's zone ratings (attack, midfield and defense, centre and flanks) are multiplied
  every minute by `1 + 0.08 + 0.14 × fill`, where `fill` is that match's attendance divided by the
  home stadium's capacity (0 when the capacity is 0). An empty stadium still gives +8%, a half-full
  one +15% and a full one +22%. (First proposed as `1 + 0.04 + 0.06 × fill`; the calibration in
  task 2.2 showed that was too weak to reach the targets, see tasks.md.)
- To know the crowd before kickoff, the weather and attendance of league matches, cup ties and
  friendlies are drawn **before** the simulation instead of after it. The same numbers are still
  used for the gate revenue and the match report.
- World Cup fixtures (group friendlies and knockout ties) are played as neutral: no home
  advantage.
- The zone duel and minute ratings of the match viewer show the ratings including the home
  advantage, since they come from the stored instants.
- **BREAKING (spec)**: the fanbase "Crowd influence" requirement no longer says fanbase cannot
  affect the simulation: through the attendance it now does, a little.

Out of scope:
- Travel distance, pitch type, or any per-club "fortress" bonus.
- Showing the computed home advantage as its own number in the UI.
- Neutral venues for cup finals (no neutral venue exists in the game today).

Open questions for the owner:
- Should a national team's friendly at home get the advantage (the default here: yes, like any
  home match; only World Cup fixtures are neutral)?
- Are +8% / +14% the right magnitudes? Task 2.2 calibrated them against the targets in the spec
  (the first guess, +4% / +6%, fell short); the owner may prefer a smaller effect, which means
  lowering the targets too.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `match-simulation`: new "Home advantage" requirement; "Post-match processing" draws the crowd
  before the simulation.
- `socios`: "Weather drawn once" — weather and attendance are drawn before kickoff.
- `fanbase`: "Crowd influence" — fanbase reaches the simulation through attendance.

## Impact

- Backend: `engine/simulation.go`, `engine/cups.go`, `engine/friendlies.go` (draw weather and
  crowd before `runMatchSimulation`, credit the gate after the save commits), `engine/stadium.go`
  (split `stadiumAttendanceAndRevenue` into "draw" and "credit"), `engine/initial_simulation.go`
  (apply the multiplier after the zone blend), `common.MatchSimulation` (new `HomeAdvantage`).
- No migration, API or frontend change; no new strings.
- Results shift towards home wins in every competition; checked with a calibration test and the
  economy simulation (prize money and sponsorship bonuses depend on results).
