# Proposal

## Why

A study of the three CPU AI levels (Caretaker, Manager, Director) found that nothing measures
whether the levels actually play differently, and that several cheap decisions a human makes are
never taken by the CPU or are taken badly: no CPU club ever picks a sponsor deal, Caretakers and
Managers never hire a trainer (so their players miss the +1/+2 growth bonus), a CPU captain can be
one of the starters its own plan substitutes off, Directors play a knockout tie that is level late
on exactly like a league match although a draw goes to a coin-flip shootout, and national teams
always field a 4-4-2 that can include injured players. This change is the first of a series of CPU
improvements; it adds the measurements the later ones are tuned against and the low-risk fixes.

## What Changes

- **Measurement**: per-AI-level gauges (`striker.ai.teams`, `striker.ai.average_budget`,
  `striker.ai.insolvent_teams`, `striker.ai.points_per_match`, labelled `ai_level` = human,
  caretaker, manager, director), panels for them in the Grafana dashboard's Simulated world row, the same
  breakdown in the admin overview, and an opt-in calibration test (`AI_LEVEL_SIM=1`) that plays
  each level against a Caretaker with identical squads.
- **CPU captain**: the oldest outfield starter that the team's own scheduled substitutions do not
  take off.
- **CPU sponsor deal**: a Manager or Director, once it has played 6 league matches in the season,
  switches to the deal with the highest expected payout for its win and draw rates so far.
- **Manager trainer**: a Manager with no trainer hires the best available one below elite skill
  (80), leaving elite coaches to Directors and humans; it never fires one.
- **Director cup orders**: in a cup tie a Director adds "from 75, if drawing" - go Attacking when
  its XI is at least as strong as the opponent's, Defensive otherwise.
- **National teams**: the call-up skips injured players, picks the strongest of the six
  formations, names a captain and adds the Director's conditional mentality orders.

Out of scope:
- Substitutions for national teams (their match roster is only the called-up XI; a bench needs a
  roster change and a decision on caps).
- Director substitution orders (they compete with its 3 planned changes for the substitution cap).
- Any change to how CPU teams buy or sell players (later changes of this series).
- Showing a club's AI level to players.

Open questions for the owner:
- Should a Caretaker also hire a (non-elite) trainer? The default here is no, to keep the levels
  apart; the calibration test will show how far behind Caretakers fall.
- Should suspensions bar a player from his national team? The default here is no: only injuries do,
  as in any friendly.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cpu-team-management`: "CPU captain" changes; new "CPU sponsor deal", "Manager trainer",
  "Director cup orders" and "AI level calibration" requirements.
- `national-teams`: "Call-up shortly before kickoff" picks only fit players, the best formation, a
  captain and mentality orders.
- `observability`: new "CPU level gauges" and "CPU level panels" requirements.

## Impact

- Backend: `common/ailineup.go` (captain, cup order), `common/nationalteam.go` (call-up),
  `common/telemetry.go` + new `common/ailevelstats.go` (gauges, admin stats), `common/admin.go`
  (overview), `engine/aiteams.go` (sponsor, trainer), `engine/aikickoff.go` (cup order),
  new `engine/ai_level_calibration_test.go`.
- Frontend: admin overview tab shows the per-level table (en/es strings).
- Grafana: `backend/resources/grafana/striker-dashboard.json` gains the CPU level panels.
- No schema change.
