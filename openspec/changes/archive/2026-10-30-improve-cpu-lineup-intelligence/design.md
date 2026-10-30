# Design

## Context

See proposal.md (Why) for the motivation. The current code, as it stands:

- CPU lineup: `engine.refreshAiTeamLineup` runs after each match (from `RunAITeamMatchReaction`,
  called by `UpdateTeamFinancesAfterMatch`). It calls `common.SelectBestFormationAndLineup` on the
  whole squad and writes the default lineup. It never writes substitutions.
  `formationRankForAiLevel` uses the global `rand`.
- CPU friendlies: `common.GetLineupsForFriendly` falls back to `BestLineupForTeam`, which is
  always 4-4-2 with no substitutions. `engine/friendlies.go` reads substitutions only from the
  per-match friendly override, which a CPU team never has.
- Kickoff: `simulateMatch`, `simulateCupTie` and `simulateFriendly` resolve the lineups and call
  `persistResolvedLineupSnapshot` (or the friendly or cup equivalent). Only then do they call
  `runMatchSimulation`, which does `excludeUnavailablePlayers` → `backfillEmptyPositions`. So the
  snapshot is taken before the refill. `backfillEmptyPositions` loops over a Go map, so the order
  it fills positions in is random.
- `applyScheduledSubstitutions` only checks that the outgoing player is on the pitch and the
  incoming one is not. It doesn't check the incoming player's availability, whether he was
  already substituted off, or the 3-substitution cap. `pickInjuryReplacement` doesn't exclude
  players who were substituted off.
- `handlers.validateLineup` / `validateSubstitutions` hold the human lineup rules.
  `validateLineup` already has an `enforceAvailability` flag.
- Fatigue: `engine.fatigueMultiplier(stamina, minutesPlayedSoFar)`, where minutes are counted
  per player from the moment he came on.
- Lineups are only readable by the owning team, so a CPU team's lineup is never visible to
  anyone before its match page shows the snapshot.

## Goals / Non-Goals

**Goals:**
- A pure, deterministic CPU lineup planner that is unit-testable without a database.
- One shared "best eligible bench player for position X" rule. The pre-kickoff refill, the
  injury replacement, the scheduled-substitution fallback and the editor's projection all use it.
- No schema change and no new engine job.

**Non-Goals:**
- Decisions during the match (reacting to the score). Substitutions stay pre-scheduled.
- Changing how a human builds or saves a lineup.

## Decisions

1. **The CPU plans at kickoff, in memory, and the snapshot records it.** Each simulate function
   resolves lineups as today. Then, for each side that `isAiManagedTeam` and has no per-match
   override, it replaces that side's lineup and substitutions with
   `common.PlanAiMatchLineup(...)`. *Alternative:* write the CPU's default lineup before kickoff
   through `SetDefaultLineup`. Rejected: `SetDefaultLineup` silently does nothing when its
   advisory lock is busy, and it adds writes on the hot path. *Alternative:* plan in the
   post-match refresh. Rejected: it runs before the next match's injuries and suspensions are
   known and before newer ones are written. The post-match `refreshAiTeamLineup` stays, so the
   CPU's stored default lineup still looks sensible.

2. **Planner API (new `common/ailineup.go`).**
   `PlanAiMatchLineup(squad []*Player, leagueUnavailable, matchUnavailable map[int64]bool,
   aiLevel int, isFriendly bool, rng *rand.Rand) AiMatchPlan` returns
   `{Formation, Positions []LineupPosition, Substitutions []LineupSubstitution}`.
   - Regulars = `SelectBestFormationAndLineup` on the squad minus `leagueUnavailable` (minus
     nothing for a Caretaker), using rank from `formationRankForAiLevel(aiLevel, n, rng)`. The
     function gains an `rng` parameter, and the post-match caller passes a time-seeded one.
   - For a competitive match, the XI is the regulars. Substitutions come from
     `planCompetitiveSubstitutions(level, ...)` using the gain formula in the spec.
   - For a friendly, the XI and substitutions come from `planFriendlyRotation(level, ...)`. The
     candidate pool is the squad minus `matchUnavailable` (friendly unavailability: injuries only).
   - The engine seeds `rng` with `matchId*31 + teamId`, so a re-simulation after stuck-match
     recovery picks the same lineup.

3. **Fatigue formula shared.** Move `fatigueMultiplier` and its 0.30 constant to
   `common.FatigueMultiplier` (engine calls it). The planner's average factor over minutes
   [a, b] is the multiplier at the midpoint, because it is linear in minutes. This keeps the
   planner and the engine from drifting apart.

4. **Validation moves to common.** `handlers.validateLineup` / `validateSubstitutions` become
   `common.ValidateLineup` / `common.ValidateSubstitutions`, and the handlers call them. The
   engine validates every CPU plan, with `enforceAvailability = aiLevel >= Manager`. If the plan
   fails, it logs and falls back to the stored lineup. This puts the "same rules as humans"
   constraint into code, not just into the spec.

5. **One eligibility rule in the engine.** Add `HomeSubstitutedOff` / `AwaySubstitutedOff`
   (`map[int64]bool`) to `common.MatchSimulation`. A new `engine.bestEligibleBenchPlayer(match,
   side, position)` excludes: on the pitch, `*UnavailablePlayers`, `SentOff`,
   `PendingUnavailabilities` and substituted off. `pickInjuryReplacement` and the
   scheduled-substitution fallback both call it. `applyScheduledSubstitutions` first checks that
   `*SubstitutionsUsed < MaxScheduledSubstitutions`. Then it checks that the outgoing player is on
   the pitch. Then it swaps the incoming player for `bestEligibleBenchPlayer` if he isn't
   eligible. Every fired change marks the outgoing player as substituted off.

6. **Refill moves to common, in slot order.** `common.BackfillEmptyPositions` iterates
   `formationSlotOrder`'s position order (de-duplicated) instead of a map. The engine's
   `excludeUnavailablePlayers` and the editor projection both use it.

7. **Snapshot after the refill.** `runMatchSimulation` takes a `snapshot func(teamId, *Lineup,
   []LineupSubstitution)` callback, the same pattern as `reportFlavorStore`. It is called right
   after `excludeUnavailablePlayers`, with the lineup rebuilt from `*PlayersByPosition`. The three
   simulate functions pass their existing persist function and stop calling it before
   simulation. National-team fixtures keep their own explicit snapshot, because their call-ups
   are never unavailable.

8. **Projected replacements in the API.** `Lineup` gains
   `ProjectedReplacements []ProjectedReplacement` (`playerOutId`, `playerInId`, `position`;
   `omitempty`). A helper `common.ProjectKickoffReplacements(lineup, squad, unavailable)` runs
   `BackfillEmptyPositions` on a copy and diffs it. The five owner GET handlers (default,
   friendly default, match, cup tie, friendly) fill it in, using friendly unavailability for the
   two friendly ones. Frontend: `PlayerChip` / `UnavailabilityBadge` render "Replaced by {name}"
   with the new locale key `lineup.replacedBy` (en "Replaced by {{name}}", es
   "Le sustituye {{name}}").

No migration file, no TRANSACTION_* reason and no notification key are added.

## Risks / Trade-offs

- [The ratings code may treat `HomeLineup` as the starters] → make it read the kickoff
  snapshot / `*PlayersByPosition` at minute 0, and add a test where an injured player in the
  saved lineup gets no rating.
- [CPU friendlies get weaker, and humans win more friendlies against them] → accepted.
  Friendlies carry no table points. Revenue doesn't depend on the result.
- [Rested regulars lose 1 Form per friendly] → accepted (owner decision 3). Form doesn't affect
  match strength.
- [Manager/Director planned substitutions change CPU results slightly] → they only fire when the
  modelled gain is positive. Check them with `TestEconomySimulation`, which isn't affected
  financially, and an engine test that compares mean goal difference over 500 seeded matches
  with and without planning.
- [A stuck-match re-run must pick the same lineup] → seeded `rng`, with a test.

## Migration Plan

Code only. Deploy with the usual rolling engine and API update. Matches simulated before the
deploy keep their pre-refill snapshots. Rollback is a revert, with no data to undo.
