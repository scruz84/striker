# player-negotiation Specification

## Purpose
How a player's hidden negotiation intelligence shapes what he asks, and how a club learns it.

## Requirements

### Requirement: Skill trend in negotiations
The skill trend a player reacts to when negotiating SHALL be the average change of his seven core skills over his last 5 progression runs, rounded half away from zero; a player with no run has a trend of 0.

#### Scenario: One good match
- **WHEN** a player's last 5 runs changed his skills by +10, 0, 0, 0 and 0
- **THEN** his trend is +2

#### Scenario: Fewer runs
- **WHEN** a player has 2 runs of +3 and +4
- **THEN** his trend is +4 (3.5 rounded)

### Requirement: Negotiation insight
A club SHALL see a player's negotiation tier (Naive, Casual, Aware, Shrewd, Mastermind) on his page once it has negotiated with him - a renewal, a free-agent signing, a transfer offer, a pre-contract offer or a clause payment that he judged, accepted or not - and MUST NOT see it before, nor see another club's insight.

#### Scenario: After a refused offer
- **WHEN** a manager's offer is refused on personal terms by a Shrewd player
- **THEN** his page shows "Negotiator: Shrewd" to that manager

#### Scenario: Another club
- **WHEN** a manager who never negotiated with him opens the same page
- **THEN** no negotiator row is shown

## Code map

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
