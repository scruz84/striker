# Proposal

## Why

CPU teams pick lineups with one blunt rule: the best XI from the whole squad, refreshed after
each match. They never schedule a substitution, so a CPU side only changes players when someone
gets injured. In friendlies they field the 4-4-2 best XI with no substitutes, so their reserves
never play. The XI is chosen without looking at who is injured or suspended, and the engine
quietly patches it at kickoff. None of this depends on the AI level, apart from the formation
rank.

Human teams have related gaps around injuries. The engine already brings on the best player for
an unavailable starter, both before kickoff and mid-match, but:

- the match's stored lineup is written *before* that replacement, so the Lineups tab shows the
  injured player as a starter and never says who took his place;
- a scheduled substitution can bring on a player who is injured, suspended, or was already
  substituted off;
- the injury replacement can pick a player who was already substituted off;
- scheduled substitutions ignore the 3-substitution cap. After an early injury replacement, all
  three planned changes still fire, so a team can make 4;
- the lineup editor flags an unavailable starter but doesn't say who will play instead.

## What Changes

- **The CPU picks its lineup at kickoff.** For every league match, cup tie and friendly, an
  AI-managed team decides its XI, formation and scheduled substitutions just before the match
  is simulated. The engine then validates the choice with the same rules a human's lineup goes
  through. The post-match default-lineup refresh stays.
- **Unavailable players by AI level.** A Manager or Director builds its XI and chooses its
  formation from available players only. A Caretaker keeps picking from the whole squad and
  relies on the kickoff backfill.
- **Planned substitutions by AI level** (competitive matches), from a fatigue-gain model that
  uses the engine's own fatigue formula:
  - Caretaker: at most 1 substitution, at minute 70. It replaces its most tired starter
    (Stamina < 50) when the best bench player for that slot is worth at least 80% of him.
  - Manager: at most 2, at minute 70, the pairs with the highest expected gain (≥ 5%). It always
    keeps one substitution free for injuries.
  - Director: at most 3. It picks the best minute per pair from 55–85 (gain ≥ 2%), and its third
    change is never before minute 75.
- **Friendly rotation by AI level.** Every CPU team gives its reserves minutes in friendlies:
  - Caretaker: starts its regulars and makes 3 changes at minute 60 for its best bench players.
  - Manager: rests its 5 strongest outfield regulars, starts the best XI from the rest, and
    makes 3 changes at minute 60 among the reserves.
  - Director: starts a reserve XI (a backup goalkeeper too if it has one) and keeps key players
    out unless it needs them to field 11. Its 3 changes at minute 60 favour players aged 21 or
    under.
- **Human injury handling (and the engine for everyone):**
  - The stored match lineup is the XI that actually kicked off, with the backfilled
    replacement in place of the unavailable starter.
  - When several positions need backfilling, they are filled in a fixed slot order, so
    "the best player for that position" doesn't depend on map iteration order.
  - A scheduled substitution whose incoming player is unavailable (injured, suspended, sent off,
    injured earlier this match, or already substituted off) brings on the best eligible bench
    player for that position instead. If nobody is eligible, it is skipped.
  - The injury replacement never brings back a player who was already substituted off.
  - A scheduled substitution is skipped once the team has used its 3 substitutions.
  - The lineup editors (default, friendly default, per-match, cup tie, friendly) show, for each unavailable
    starter, the player who will replace him at kickoff.

### Out of scope

- Substitutions that react to the score or to cards during the match. Planned substitutions are
  still fixed before kickoff.
- Changing the fatigue, Form or growth formulas, or the 3-substitution limit.
- Rotation in competitive matches. Fatigue doesn't carry over between matches, so there is
  nothing to rotate for.
- Rewriting a human's saved lineup. The human's choice stays as it is: the injured player
  returns to the XI automatically once he recovers.
- National teams (AI level 0).

### Owner decisions (please confirm)

1. A Caretaker gets at most 1 planned substitution rather than none, so that every level
   "considers it if needed" while still doing the least.
2. If a scheduled substitution's incoming player is unavailable, the change still happens with
   the best eligible bench player for that position, instead of being dropped.
3. In friendlies, rested regulars lose 1 Form point (benched), and reserves gain Form and growth
   chances from playing. We accept this trade-off.
4. The CPU's kickoff lineup is computed in memory and stored only as the match snapshot.
   The CPU's default lineup is still refreshed after each match as it is today.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `cpu-team-management`: lineup selection moves to kickoff, depends on AI level (unavailable
  players, planned substitutions) and adds friendly rotation.
- `match-simulation`: the stored lineup is the actual kickoff XI, backfill order is deterministic,
  scheduled substitutions respect the shared cap and replace an ineligible incoming player, and
  nobody re-enters after being substituted off.

## Impact

- Backend: `common/lineup.go` (new CPU planner), new `common/lineupvalidation.go` (validators
  moved out of `handlers/lineupHandlers.go`), `engine/simulation.go`, `engine/cups.go`,
  `engine/friendlies.go`, `engine/substitution_simulation.go`, `engine/injury_simulation.go`,
  `engine/aiteams.go`; the five lineup GET handlers gain a `projectedReplacements` field.
- Frontend: `features/lineup` (pitch chip / badge shows "Replaced by …"), en/es locales.
- No schema migration, no new TRANSACTION_* reason, no notification.
