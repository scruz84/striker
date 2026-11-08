package common

import "math/rand"

// Intelligence tiers for a player's own contract savvy - how sharply they personally react, when
// negotiating, to their recent skill trend (see PlayerSkillTrend). Unlike AiLevelCaretaker/Manager/
// Director (which gate a CPU *team's* management behavior and are meaningless once a human owns
// it), a player's intelligence always matters, whoever owns their team - it's a property of their
// own agent, not of whoever currently employs them. See IntelligenceReactionFactor for how each
// tier actually scales a negotiation.
const (
	IntelligenceNaive      = 1
	IntelligenceCasual     = 2
	IntelligenceAware      = 3
	IntelligenceShrewd     = 4
	IntelligenceMastermind = 5
)

// intelligenceReactionFactors scales how much of a trend-driven adjustment (see
// ExpectedWageWithIntelligence/MaxAcceptableRenewalYearsWithIntelligence) each tier actually acts
// on: a Naive player (0) ignores their own trend entirely, reproducing this feature's pre-existing
// behavior exactly; a Mastermind (1.0) reacts to the full calculated adjustment. The three tiers in
// between split that range evenly rather than skewing toward either end - there's no in-game
// signal (yet) that would justify a non-linear curve here.
var intelligenceReactionFactors = map[int]float64{
	IntelligenceNaive:      0,
	IntelligenceCasual:     0.25,
	IntelligenceAware:      0.5,
	IntelligenceShrewd:     0.75,
	IntelligenceMastermind: 1.0,
}

// IntelligenceReactionFactor returns how strongly a player of this tier reacts to their own skill
// trend. An unrecognized level (0, the pre-migration sentinel, or any other value outside 1-5)
// falls back to 0 - the same "ignore the trend" behavior as IntelligenceNaive - rather than
// panicking or guessing, so a not-yet-backfilled row never blocks a contract offer.
func IntelligenceReactionFactor(level int) float64 {
	if factor, ok := intelligenceReactionFactors[level]; ok {
		return factor
	}
	return 0
}

// intelligenceWeights biases which tier a newly created player gets, 1=Naive..5=Mastermind -
// lower tiers are deliberately far more common than the rare, fully market-savvy few, the same
// shape as defaultAiLevelWeights but with no team/division-tier dimension to split by (a player's
// own intelligence doesn't depend on which league they end up in).
var intelligenceWeights = [5]int{30, 25, 20, 15, 10}

// RandomIntelligence picks a weighted-random intelligence tier (1-5) for a newly created player -
// see doCreatePlayers and InjectFreeAgents, the only callers.
func RandomIntelligence() int {
	total := 0
	for _, w := range intelligenceWeights {
		total += w
	}
	roll := rand.Intn(total)
	cumulative := 0
	for i, w := range intelligenceWeights {
		cumulative += w
		if roll < cumulative {
			return i + 1
		}
	}
	return IntelligenceMastermind // unreachable: intelligenceWeights always sums to `total`
}

const GetPlayerIntelligenceStmtSql = "SELECT intelligence FROM players WHERE player_id = $1"

// GetPlayerIntelligence fetches one player's hidden intelligence tier - kept out of the Player
// struct and GetPlayerById's own SELECT (same treatment as GetPlayerSkillCaps) since nothing but a
// contract negotiation (MakeContractRenewalOffer, SignFreeAgent, SignFreeAgentAtMarketRate) ever
// needs it.
func GetPlayerIntelligence(dbCtx *DatabaseContext, playerId int64) (int, error) {
	var level int
	err := dbCtx.Db.QueryRow(GetPlayerIntelligenceStmtSql, playerId).Scan(&level)
	return level, err
}

// intelligenceTerms are the contract terms a player's negotiation tier shapes (openspec/specs/
// player-negotiation, team-finances "Contract renewal and release", pre-contracts "Player
// demands", release-clauses "Wage discount for a clause"): the share of his wage he keeps when he
// renews in his last season, how much above his usual wage he asks to leave on a pre-contract,
// and the wage he gives up for a release clause of at most twice his value (half that for one of
// at most four times).
type intelligenceTerms struct {
	lastSeasonFloor float64
	movePremium     float64
	clauseDiscount  float64
}

var intelligenceTable = map[int]intelligenceTerms{
	IntelligenceNaive:      {lastSeasonFloor: 0.75, movePremium: 1.05, clauseDiscount: 0.05},
	IntelligenceCasual:     {lastSeasonFloor: 0.80, movePremium: 1.075, clauseDiscount: 0.075},
	IntelligenceAware:      {lastSeasonFloor: 0.85, movePremium: 1.10, clauseDiscount: 0.10},
	IntelligenceShrewd:     {lastSeasonFloor: 0.90, movePremium: 1.15, clauseDiscount: 0.125},
	IntelligenceMastermind: {lastSeasonFloor: 0.95, movePremium: 1.20, clauseDiscount: 0.15},
}

// termsForIntelligence is tier's row; an unknown tier (0 before the backfill) reads as Aware, the
// middle tier, which keeps the flat values these rules had before tiers shaped them.
func termsForIntelligence(tier int) intelligenceTerms {
	if t, ok := intelligenceTable[tier]; ok {
		return t
	}
	return intelligenceTable[IntelligenceAware]
}

// LastSeasonFloorShare is the share of his wage a player of tier keeps when he renews in the last
// season of his contract.
func LastSeasonFloorShare(tier int) float64 { return termsForIntelligence(tier).lastSeasonFloor }

// PreContractPremium is how many times his expected wage a player of tier asks to leave his club on
// a pre-contract.
func PreContractPremium(tier int) float64 { return termsForIntelligence(tier).movePremium }

// ClauseDiscountFor is the share of his wage a player of tier gives up for a release clause of at
// most twice his value.
func ClauseDiscountFor(tier int) float64 { return termsForIntelligence(tier).clauseDiscount }

const recordNegotiationInsightStmtSql = "INSERT INTO negotiation_insights (team_id, player_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
const hasNegotiationInsightStmtSql = "SELECT EXISTS (SELECT 1 FROM negotiation_insights WHERE team_id = $1 AND player_id = $2)"
const clearTeamNegotiationInsightsStmtSql = "DELETE FROM negotiation_insights WHERE team_id = $1"

// RecordNegotiationInsight notes that teamId has negotiated with playerId, so it sees his
// negotiation tier from now on (openspec/specs/player-negotiation "Negotiation insight"). The
// handlers call it once the player has judged the terms, after the negotiation's own transaction
// is over (its player row lock would block the insert's foreign-key check).
func RecordNegotiationInsight(dbCtx *DatabaseContext, teamId int64, playerId int64) error {
	_, err := dbCtx.Db.Exec(recordNegotiationInsightStmtSql, teamId, playerId)
	return err
}

// NegotiationStyleFor is playerId's negotiation tier as teamId may see it: the tier once teamId has
// negotiated with him, nil before.
func NegotiationStyleFor(dbCtx *DatabaseContext, teamId int64, playerId int64) (*int, error) {
	var known bool
	if err := dbCtx.Db.QueryRow(hasNegotiationInsightStmtSql, teamId, playerId).Scan(&known); err != nil || !known {
		return nil, err
	}
	tier, err := GetPlayerIntelligence(dbCtx, playerId)
	if err != nil || tier == 0 {
		return nil, err
	}
	return &tier, nil
}
