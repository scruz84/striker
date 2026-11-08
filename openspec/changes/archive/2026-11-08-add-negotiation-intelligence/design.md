# Design

## Context

`ExpectedWageWithIntelligence` / `MaxAcceptableRenewalYearsWithIntelligence` scale the trend by
`IntelligenceReactionFactor`; `PlayerSkillTrend` reads `GetLatestPlayerEvolution`. The floor lives
in `wageFloorShare` (`common/contracts.go`), the move premium in `PreContractMovePremium`
(`common/precontracts.go`), the clause discount in `ReleaseClauseDiscount`
(`common/releaseclauses.go`). All negotiation goes through `PersonalTermsExpectation` /
`PlayerTermsExpectation`, `MakeContractRenewalOfferWithClause`, `SignFreeAgentWithClause`,
`MakeTransferOfferWithClause`, `MakePreContractOfferWithClause` and `PayReleaseClause`.

## Decisions

### D1. Tier tables
`intelligenceTable` maps each tier to `{lastSeasonFloor, movePremium, clauseDiscount}`: Naive
0.75/1.05/0.05, Casual 0.80/1.075/0.075, Aware 0.85/1.10/0.10, Shrewd 0.90/1.15/0.125, Mastermind
0.95/1.20/0.15; an unknown tier (0) reads as Aware, so a not-yet-backfilled row behaves as before.
`LastSeasonFloorShare(tier)`, `PreContractPremium(tier)`, `ClauseDiscountFor(tier, clause, base)`.

### D2. Signatures
`PersonalTermsExpectation(p, intelligence, trend, kind)` already has the tier and uses
`LastSeasonFloorShare`. `PreContractMinimumWage(expected, tier)` and `PreContractKeepWage(best,
tier)` take it; `PreContractCounterBar` reads the player's tier. `ReleaseClauseDiscount(clause,
base, tier)` and `DiscountedWage(p, expected, clause, tier)` take it; callers already read it
through `PlayerTermsExpectation`, which now also returns it (`PlayerTerms{Wage, Years,
Intelligence}`).

### D3. Trend
`PlayerSkillTrend` averages `SkillTrendFromEvolution` over the player's last
`SkillTrendRuns` (5) runs (`GetRecentPlayerEvolutionsStmtSql`, `ORDER BY` the log id desc
`LIMIT 5`), rounded half away from zero; no runs is 0, as before.

### D4. Insight
`resources/sql/init106_negotiation_insights.sql`: `negotiation_insights (team_id bigint REFERENCES
teams ON DELETE CASCADE, player_id bigint REFERENCES players ON DELETE CASCADE, created_at
timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (team_id, player_id))`. `RecordNegotiationInsight
(ex, teamId, playerId)` (`INSERT … ON CONFLICT DO NOTHING`) runs in each negotiation function once
the player has judged the terms, accepted or not, before returning; it is best-effort (logged on
error). The player details handler adds `negotiationStyle` (1-5) for the viewer's club when it has
an insight (`HasNegotiationInsight`), never otherwise. The club reset clears its insights.

## Risks / Trade-offs

- Shrewd and Mastermind players (25% of new players) get more expensive to keep and to poach; Naive
  ones cheaper. Aware keeps today's numbers.

## Migration

`init106_negotiation_insights.sql`. Existing players keep their tier.

## Code map (for the new `player-negotiation` spec when the change is archived)

- Tiers: `backend/common/playerintelligence.go` (`intelligenceTable`, `LastSeasonFloorShare`,
  `PreContractPremium`, `ClauseDiscountFor`, `IntelligenceReactionFactor`); used by
  `wageFloorShare` (`common/contracts.go`), `PreContractMinimumWage` / `PreContractKeepWage` /
  `PreContractCounterBar` (`common/precontracts.go`, tier read by `negotiationTier`) and
  `ReleaseClauseDiscount` / `DiscountedWage` (`common/releaseclauses.go`); the CPU reads it with
  `playerTier` (`engine/aiteams.go`).
- Trend: `common.PlayerSkillTrend`, `AverageSkillTrend`, `SkillTrendRuns`,
  `GetRecentPlayerEvolutionsStmtSql` (`common/playerprogression.go`).
- Insight: `resources/sql/init106_negotiation_insights.sql`, `RecordNegotiationInsight` /
  `NegotiationStyleFor` (`common/playerintelligence.go`), recorded by `noteNegotiation`
  (`handlers/releaseClauseHandlers.go`) from the renewal, free-agent, offer, pre-contract and clause
  handlers; `PlayerDetails.NegotiationStyle` set by `PlayerDetailsHandler`; cleared by
  `ResetReleasedTeam`. Frontend: "Negotiator" row in `components/PlayerProfileRail.tsx`, strings
  `playerPage.stats.negotiator`, `playerPage.negotiationStyles.*`; tier ranges in
  `utils/contractTerms.ts` and the `noPayCutRange` hint.
- Tests: `common/personalterms_test.go`, `common/precontracts_test.go`,
  `common/releaseclauses_test.go`, `engine/contract_terms_integration_test.go`
  (`TestContractTerms_TrendAveragesTheLatestRuns`), `handlers/negotiationInsight_test.go`,
  `components/PlayerProfileRail.test.tsx`, `utils/contractTerms.test.ts`.
