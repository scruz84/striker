# Design

## Context

The management pass (`RunAITeamManagement`) and the kickoff plan (`planAiKickoff` →
`common.PlanAiMatchLineup`) know nothing about the table. Standings come from
`common.GetRankedClassification(leagueId)` (best to worst), promotion/relegation slot counts from
`common.GetPromotionRelegationConfig`, a team's league and tier from `GetTeamById` /
`GetTeamTier`, and the country's tiers from `GetLeaguesWithTierForCountryStmtSql`.

## Decisions

### Pure objective
`common.SeasonObjectiveFor(s SeasonSituation) string` takes `Points []int` (the league, best to
worst), `Index` (the club's place), `Played`, `Remaining`, `PromotionSlots` (the configured slots
when a tier above exists, 1 in a top division), `RelegationSlots` (0 without a tier below).
"Best-placed side in the zone" is `Points[len-RelegationSlots]`; "last promotion place" is
`Points[PromotionSlots-1]`. The "caught" check uses the club's own remaining matches for the side
in the zone (fixtures are a double round robin, so remaining counts differ by at most one).
Constants: `SeasonObjectiveStartShare` (1/3), `SeasonObjectiveMargin` (3).

### Reading the situation
`common.GetTeamSeasonSituation(teamId)`: the team's league and tier, its ranked classification,
the country's highest tier number, the config, and `Remaining` from
`CountRemainingLeagueMatchesStmtSql` (unplayed `matches` rows of the team in its league this
season). One function, used by the pass and the kickoff; an error means `open`.

### Market urgency
`common.AdjustProfileForObjective(profile, objective)` returns a copy; `runAiTransferPass(dbCtx,
team, objective)` applies it right after `AiTransferProfileFor`. Counter-offers keep the plain
profile (they are answered on the offer sweep, away from the pass).

### Youth minutes
`common.AiPlanOptions{YouthStarts int}` and `PlanAiMatchLineupWithOptions`; `PlanAiMatchLineup`
keeps its signature and passes zero options. In the competitive branch, after the regulars are
chosen, `giveYouthStarts(squad, regulars, unavailable, n)` swaps in up to n bench players aged
`aiYouthMaxAge` (21) or under worth `aiYouthMinValueRatio` (0.8) of the starter, walking the
outfield starters from the lowest value at their slot; substitutions and captain are planned
afterwards. `planAiKickoff` sets `YouthStarts = aiDirectorYouthStarts` (2) for a Director's
league match when `GetTeamSeasonSituation` gives `nothing_at_stake`.

## Risks / Trade-offs

- Two extra queries per CPU league kickoff and per management pass; both are indexed lookups on
  one league.
- A lower bar in a relegation fight can make a club overspend; the crisis rules of
  add-cpu-financial-discipline still stop it below 0.

## Migration

None.
