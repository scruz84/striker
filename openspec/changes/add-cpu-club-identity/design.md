# Design

## Context

`common.RandomAiLevel(tier)` draws a level for a new club; the season rollover applies
promotion/relegation moves in `engine/season.go` under the country's rollover lock (one replica).
Every CPU market decision reads `common.AiTransferProfileFor(team.AiLevel)`.

## Decisions

### Level after a division change
`common.RandomAiLevelFrom(tier, rng)` (the weighted draw with an explicit rng; `RandomAiLevel`
calls it with a fresh one) and pure `StepAiLevelToward(current, drawn)`. In the rollover, after a
move is applied, `redrawAiLevelAfterMove` reads the team (`GetTeamById`), skips owned or level-0
clubs, reads the new tier and writes `SetTeamAiLevel`. The rollover already runs once per country
under its lock, so no replica can apply it twice.

### Style
`common.AiClubStyleFor(teamId)`: `teamId mod 5` → 0,1 balanced; 2 youth; 3 frugal; 4 ambitious
(ids are sequential, so the shares hold across a league). The style is stored in the new nullable
`teams.ai_style` (`init102_team_ai_style.sql`; codes 0-3, NULL reads as balanced) when a club is
created (`handlers.createTeam` → `AssignAiStyle`) and by `BackfillAiStyles` at service startup;
`GetTeamById` reads it into `Team.AiStyle` (never serialised). Storing it, rather than deriving it
on every read, keeps tests deterministic (a club inserted by a test is balanced whatever its id)
and lets a future admin tool change it. `ApplyClubStyle(profile, style)` is pure and level-aware;
`AiTransferProfileForTeam(team)` combines both. The engine reads
the profile through one helper, `aiProfile(team)`, which replaces every
`common.AiTransferProfileFor(team.AiLevel)` in the CPU market code.

Adjustments (Manager or Director only): youth → `SignsProjects`, `LoansOutYouth`,
`EstimatesPotential`; frugal → `BudgetReserve + 0.15`, `KeyPlayerPriceMultiplier − 0.5`;
ambitious → `BudgetReserve − 0.10` (min 0.05), `MinScore − 1` (min 1),
`KeyPlayerPriceMultiplier + 0.5`.

### Prospects by potential
`promoteProspectByPotential` runs after `promoteFromAcademyForWeakestPosition` returns false,
for `profile.SignsProjects` teams (every Director and youth Managers). Pure pick
`pickProspectByPotential(cantera, caps, squad, profile, rng)`: for each prospect,
`ProjectPlayer` then `PositionValue` at `BestCorePosition`, compared with `weakestStarterValue`
of the senior squad at that position; the best margin wins. Caps via
`common.GetPlayerSkillCaps`; the rng is seeded from the team id and the week so a re-run picks
the same one. Promotion goes through `CanPromoteFromAcademy` / `PromoteFromAcademy`.

## Risks / Trade-offs

- A released club keeps its style through its reset: the style belongs to the club, not to its
  former owner.
- Levels drift towards each tier's mix over seasons, which is the point; the gauges show it.

## Migration

`resources/sql/init102_team_ai_style.sql`: `ALTER TABLE teams ADD COLUMN IF NOT EXISTS ai_style
smallint` (re-runnable). No TRANSACTION_* reason or notification is added.
