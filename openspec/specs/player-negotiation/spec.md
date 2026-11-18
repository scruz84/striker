# player-negotiation Specification

## Purpose
How a player's hidden negotiation intelligence shapes what he asks, and that it stays hidden.

## Requirements

### Requirement: Skill trend in negotiations
The skill trend a player reacts to when negotiating SHALL be the average change of his seven core skills over his last 5 progression runs, rounded half away from zero; a player with no run has a trend of 0.

#### Scenario: One good match
- **WHEN** a player's last 5 runs changed his skills by +10, 0, 0, 0 and 0
- **THEN** his trend is +2

#### Scenario: Fewer runs
- **WHEN** a player has 2 runs of +3 and +4
- **THEN** his trend is +4 (3.5 rounded)

### Requirement: Private negotiation tier
A player's negotiation tier (Naive, Casual, Aware, Shrewd, Mastermind) SHALL stay private: no manager MUST see it, on the player's page or in any API response, whether or not his club has negotiated with the player.

#### Scenario: After a refused offer
- **WHEN** a manager's offer is refused on personal terms by a Shrewd player
- **THEN** his page shows no negotiator row and the player details carry no negotiation tier

#### Scenario: Never negotiated
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
- Privacy: no response carries the tier (`PlayerDetails` has no tier field, guarded by
  `TestPlayerDetailsCarryNoNegotiationTier` in `common/playerdetails_test.go`); the old
  `negotiation_insights` table is dropped by `resources/sql/history/init109_drop_negotiation_insights.sql`.
  Frontend: only the tier ranges, never a player's own tier, in `utils/contractTerms.ts` and the
  `noPayCutRange` hint.
- Tests: `common/personalterms_test.go`, `common/precontracts_test.go`,
  `common/releaseclauses_test.go`, `engine/contract_terms_integration_test.go`
  (`TestContractTerms_TrendAveragesTheLatestRuns`), `common/playerdetails_test.go`,
  `utils/contractTerms.test.ts`.
