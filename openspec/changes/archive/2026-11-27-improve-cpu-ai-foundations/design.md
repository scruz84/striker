# Design

## Context

CPU behaviour lives in `common/ailineup.go` (kickoff plan: XI, formation, substitutions, orders,
captain), `common/mentality.go` (`PickAiMentality`), `engine/aikickoff.go` (applies the plan at
kickoff, picks the mentality from both XIs' strengths) and `engine/aiteams.go` (management pass).
National-team lineups are chosen by `common.selectNationalSquadLineup`, which today returns
`SelectBestLineup` (always FormationBalanced) over the whole pool; the national team's match
roster is only the saved XI (`engine.resolveMatchTeamPlayers` → `GetPlayersByIds`), and its
unavailability query (`GetUnavailablePlayerIdsForFriendly(nationalTeamId)`) matches no player
because called-up players keep their club's `team_id` - so an injured international can play.

## Decisions

### Gauges from one grouped query
`common.GetAiLevelStats` runs one statement:

```sql
SELECT t.ai_level, (t.owner_id IS NOT NULL) owned, count(*), COALESCE(SUM(t.budget),0),
       count(*) FILTER (WHERE t.budget < -MaxDebt), COALESCE(SUM(c.points),0), COALESCE(SUM(c.matches),0)
FROM teams t LEFT JOIN classification c ON c.team_id = t.team_id
WHERE t.is_national_team = false AND t.reset_pending_since IS NULL
GROUP BY 1, 2
```

and folds the rows into four buckets (owned → `human`; otherwise by `ai_level`, level 0 unowned
clubs are skipped). `classification` holds one row per team for its current season, so no season
join is needed. The gauges are registered in `RegisterMarketGauges` (engine only, read with
max()) through one `RegisterCallback`, so it is one round trip per tick. The admin overview calls
the same function (`AdminOverview.AiLevels`).

### Calibration harness
`engine/ai_level_calibration_test.go` reuses the in-memory pipeline of
`home_advantage_test.go` (`SubstitutionSimulation`, `StaminaSimulation`, `InitialSimulation`,
`ScorerSimulation`, `DisciplineSimulation`; no injuries, they need the database). The squad is a
deliberately uneven one (strong strikers, a weak flank, tired veterans) so formation choice,
substitutions and mentality matter. Each side gets `PlanAiMatchLineup` at its level, then
`PickAiMentality` against the other's XI, as `planAiKickoff` does. Home and away alternate so the
venue cancels out (neutral home advantage 1).

### Captain
`AiCaptain(squad, positions, substitutions)` skips any `PlayerOutId`; when that leaves nobody it
falls back to the old rule. `PlanAiMatchLineup` passes the plan's own substitutions (friendly
rotation included).

### Sponsor deal
`chooseSponsorDealIfManager` in `engine/aiteams.go`, called from `manageRosterIfManager`: reads
`IsSponsorDealLocked`, the team's `classification` row (`common.GetTeamSeasonRecord`), and
computes the expected payout with the engine's own `sponsorshipAmount(1000, deal, outcome)` so the
formula can never drift from the payout. The switch goes through `common.SetSponsorDeal`, which
locks it for the season. Pure helper `bestSponsorDeal(current, wins, draws, matches)`.

### Manager trainer
`upgradeTrainerIfBetterAvailable` gains a level-aware twin `hireTrainerIfManager` (pure pick:
`pickManagerTrainer(available)` = first with skill < `aiManagerTrainerMaxSkill` = 80, the elite
threshold of `trainerBonus`). Runs from `manageRosterIfManager` for Managers only (Directors keep
their rule in `manageMarketIfDirector`).

### Cup order
`common.AiCupDrawOrder(aiLevel, ownStrength, opponentStrength)` returns the order or nil.
`planAiKickoff` already computes both strengths for the mentality; it appends the order for
`kickoffCup` when the side has fewer than `MaxConditionalOrders`, then validates with
`ValidateOrders` as today.

### National call-up
`selectNationalSquadLineup` reads the pool, drops players with an injury
(`common.GetInjuredPlayerIds(playerIds)`: `player_unavailability` with reason injury and
`matches_remaining > 0`), then uses `SelectBestFormationAndLineup(fit, AiLevelDirector, nil)`;
the saved lineup gets that formation, `AiCaptain` and `AiConditionalOrders(AiLevelDirector)`.
No bench, so no substitutions (out of scope). Caps are still counted from the saved XI.

## Risks / Trade-offs

- Managers taking non-elite trainers shrinks the market humans see; the market is topped up to
  one candidate per team every season (`TopUpTrainerMarket`), and elite ones stay untouched.
- The sponsor choice uses the current season's record only, so a promoted club is not judged on
  last season's division.

## Migration

None: no schema change, no new TRANSACTION_* reason, no new notification key.
