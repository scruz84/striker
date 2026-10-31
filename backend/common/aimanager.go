package common

import "math/rand"

// AI difficulty tiers for an unowned team - see RunAITeamManagement (added in a later phase).
// Each tier is a strict superset of the capabilities below it: Caretaker only reacts (keeps the
// lineup fresh, accepts clearly-good incoming offers); Manager adds roster maintenance (contract
// renewals/releases, free-agent signings, ticket pricing); Director adds active market
// participation (transfer shopping, stadium upgrades, trainer hiring).
const (
	AiLevelCaretaker = 1
	AiLevelManager   = 2
	AiLevelDirector  = 3
)

// AiTransferProfile concentrates every knob that differs between AI levels when a CPU team buys,
// sells or answers an offer - the functions that implement transfer behavior (ScoreTarget,
// EvaluateIncomingOffer, and the engine's market pass) take one of these instead of scattering
// `if level == ...` checks. Each level includes the capabilities of the ones below it (a Director
// also protects key players, a Manager also accepts offers that clear its valuation), so the
// profiles are built bottom-up by AiTransferProfileFor.
type AiTransferProfile struct {
	Level int

	// ValuationNoise is the relative error (+/-) with which this team values a player - a
	// Caretaker misjudges what a player is worth by up to 30%, a Director by 5%. It is what lets
	// a Caretaker sell a future star cheaply, which is an opportunity for human managers.
	ValuationNoise float64
	// EstimatesPotential is whether the team tries to read a young player's hidden ceiling at
	// all (Caretaker: no). PotentialNoise is how wrong that read is, as a fraction of the
	// player's real headroom - the CPU never reads the exact ceiling, so it is neither omniscient
	// nor betrayed by its own signings.
	EstimatesPotential bool
	PotentialNoise     float64

	// ProtectsKeyPlayers makes the team ask KeyPlayerPriceMultiplier times a starter's value
	// before selling one it can't replace. InternationalPremium is the extra a Director asks for
	// a national-team player (0 = none).
	ProtectsKeyPlayers       bool
	KeyPlayerPriceMultiplier float64
	InternationalPremium     float64

	// Where it shops: free agents, the transfer list, and (Director) every player in the world.
	SearchesFreeAgents   bool
	SearchesTransferList bool
	SearchesUnlisted     bool
	// SignsProjects lets a young player's projected growth justify a signing on its own.
	SignsProjects bool
	// MinScore is how high ScoreTarget must come out before the team acts: a Manager only moves
	// for a clear starter upgrade or a hole, a Director for smaller gains.
	MinScore float64

	// ListsSurplus: put squad surplus on the transfer list. SellsToBuy: list a starter (or a
	// surplus player) to fund a target the budget can't cover yet.
	ListsSurplus bool
	SellsToBuy   bool

	// BudgetReserve is the fraction of the current budget that must still be there after a
	// signing. ConsidersWageBill additionally keeps WageRunwayRounds of the new payroll covered.
	BudgetReserve     float64
	ConsidersWageBill bool

	// MovesPerPass caps signings per management pass (0 = the team never shops);
	// MaxNewOffersPerPass caps how many of those may be new offers for unlisted players.
	MovesPerPass        int
	MaxNewOffersPerPass int
}

// AiTransferProfileFor returns the transfer profile of an AI level; an unknown level gets the
// Caretaker profile, the most passive one.
func AiTransferProfileFor(level int) AiTransferProfile {
	profile := AiTransferProfile{
		Level:                    AiLevelCaretaker,
		ValuationNoise:           0.30,
		KeyPlayerPriceMultiplier: 1,
	}
	if level >= AiLevelManager {
		profile.Level = AiLevelManager
		profile.ValuationNoise = 0.15
		profile.EstimatesPotential = true
		profile.PotentialNoise = 0.6
		profile.ProtectsKeyPlayers = true
		profile.KeyPlayerPriceMultiplier = 2
		profile.SearchesFreeAgents = true
		profile.SearchesTransferList = true
		profile.MinScore = 5
		profile.ListsSurplus = true
		profile.BudgetReserve = 0.5
		profile.MovesPerPass = 1
	}
	if level >= AiLevelDirector {
		profile.Level = AiLevelDirector
		profile.ValuationNoise = 0.05
		profile.PotentialNoise = 0.15
		profile.KeyPlayerPriceMultiplier = 2.5
		profile.InternationalPremium = 0.25
		profile.SearchesUnlisted = true
		profile.SignsProjects = true
		profile.MinScore = 3
		profile.SellsToBuy = true
		profile.BudgetReserve = 0.15
		profile.ConsidersWageBill = true
		profile.MovesPerPass = 3
		profile.MaxNewOffersPerPass = 2
	}
	return profile
}

// aiLevelWeightsByTier biases which AI level a newly created CPU team gets by the division tier
// it starts in, so top-flight leagues field sharper CPU competition than the lower ones - see
// RandomAiLevel. Values are {weight(Caretaker), weight(Manager), weight(Director)}; a tier not
// listed here falls back to defaultAiLevelWeights.
var aiLevelWeightsByTier = map[int][3]int{
	1: {10, 30, 60},
	2: {30, 50, 20},
}
var defaultAiLevelWeights = [3]int{60, 30, 10}

// RandomAiLevel picks a weighted-random AI level (1-3) for a newly created team in the given
// division tier.
func RandomAiLevel(tier int) int {
	weights, ok := aiLevelWeightsByTier[tier]
	if !ok {
		weights = defaultAiLevelWeights
	}
	total := weights[0] + weights[1] + weights[2]
	roll := rand.Intn(total)
	if roll < weights[0] {
		return AiLevelCaretaker
	}
	if roll < weights[0]+weights[1] {
		return AiLevelManager
	}
	return AiLevelDirector
}

const SetTeamAiLevelStmtSql = "UPDATE teams SET ai_level = $1 WHERE team_id = $2"

// BackfillAiLevelsStmtSql finds every unowned team that hasn't been assigned an AI level yet,
// along with the division tier it plays in (so RandomAiLevel can weight correctly) - covers a
// world that existed before this feature shipped, same idea as TopUpTrainerMarket backfilling
// the trainer market.
const BackfillAiLevelsStmtSql = "SELECT t.team_id, l.tier FROM teams t JOIN leagues l ON t.league_id = l.league_id " +
	"WHERE t.owner_id IS NULL AND t.ai_level = 0 AND t.reset_pending_since IS NULL"

// SetTeamAiLevel overrides a team's AI difficulty level directly - used by the admin endpoint
// (handlers.AdminSetTeamAiLevelHandler) to hand-tune a specific CPU team, and by
// BackfillAiLevels/team creation to assign the initial one.
func SetTeamAiLevel(dbCtx *DatabaseContext, teamId int64, level int) error {
	_, err := dbCtx.Db.Exec(SetTeamAiLevelStmtSql, level, teamId)
	return err
}

// BackfillAiLevels assigns a weighted-random AI level to every unowned team that doesn't have
// one yet. Called once at service startup - see service.StartSvc.
func BackfillAiLevels(dbCtx *DatabaseContext) error {
	rows, err := dbCtx.Db.Query(BackfillAiLevelsStmtSql)
	if err != nil {
		return err
	}
	type pendingTeam struct {
		teamId int64
		tier   int
	}
	pending := make([]pendingTeam, 0)
	for rows.Next() {
		var p pendingTeam
		if err := rows.Scan(&p.teamId, &p.tier); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, p)
	}
	rows.Close()

	for _, p := range pending {
		if err := SetTeamAiLevel(dbCtx, p.teamId, RandomAiLevel(p.tier)); err != nil {
			return err
		}
	}
	return nil
}

const ClearOwnedTeamAiLevelsStmtSql = "UPDATE teams SET ai_level = 0 WHERE owner_id IS NOT NULL AND ai_level != 0"

// ClearOwnedTeamAiLevels zeroes ai_level on every currently-owned team that still carries a
// leftover value from before AssignTeam's own UPDATE was fixed to reset it at claim time - a
// stale nonzero value there was otherwise indistinguishable from a real CPU team's difficulty
// tier, showing e.g. "AI Director" on a manager's own squad page even though Team.AiLevel's doc
// comment already promises 0 for an owned team. Called once at service startup, same as
// BackfillAiLevels, so an already-affected database heals itself without a manual fix.
func ClearOwnedTeamAiLevels(dbCtx *DatabaseContext) error {
	_, err := dbCtx.Db.Exec(ClearOwnedTeamAiLevelsStmtSql)
	return err
}
