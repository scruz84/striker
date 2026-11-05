package engine

import (
	"math/rand"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// kickoffKind says which lineup tables a match's per-match override lives in, and so which
// override-exists query and which unavailability rules apply.
type kickoffKind int

const (
	kickoffLeague kickoffKind = iota
	kickoffCup
	kickoffFriendly
)

func (k kickoffKind) overrideExistsSql() string {
	switch k {
	case kickoffCup:
		return common.CupTieLineupOverrideExistsStmtSql
	case kickoffFriendly:
		return common.FriendlyLineupOverrideExistsStmtSql
	default:
		return common.MatchLineupOverrideExistsStmtSql
	}
}

// aiKickoffSeed is the seed of the rng a CPU side's kickoff plan draws from, so re-simulating the
// same match (stuck-match recovery) picks the same formation.
func aiKickoffSeed(matchId int64, teamId int64) int64 { return matchId*31 + teamId }

// planAiKickoff lets an AI-managed team decide its XI, formation and scheduled substitutions just
// before its match is simulated (see common.PlanAiMatchLineup). A side that is human-managed, has
// a per-match lineup saved (its owner's, or the snapshot of an earlier attempt at this match), or
// whose plan fails the same validation a human's lineup goes through, keeps lineup and
// substitutions exactly as passed in - a failed plan is logged. The plan lives in memory and is
// stored only as the match's kickoff snapshot (see runMatchSimulation).
func planAiKickoff(dbCtx *common.DatabaseContext, kind kickoffKind, matchId int64, teamId int64,
	lineup *common.Lineup, substitutions []common.LineupSubstitution) (*common.Lineup, []common.LineupSubstitution) {
	return planAiKickoffWithFormation(dbCtx, kind, matchId, teamId, lineup, substitutions, nil)
}

// planAiKickoffWithFormation is planAiKickoff with the formation fixed when formation is set (a
// Director's pick against its opponent, see directorFormationAgainst).
func planAiKickoffWithFormation(dbCtx *common.DatabaseContext, kind kickoffKind, matchId int64, teamId int64,
	lineup *common.Lineup, substitutions []common.LineupSubstitution, formation *int) (*common.Lineup, []common.LineupSubstitution) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching team %d for its kickoff lineup in match %d: %s", teamId, matchId, err)
		return lineup, substitutions
	}
	if !isAiManagedTeam(team) {
		return lineup, substitutions
	}
	var hasOverride bool
	if err := dbCtx.Db.QueryRow(kind.overrideExistsSql(), matchId, teamId).Scan(&hasOverride); err != nil {
		log.Errorf("error checking team %d's lineup override for match %d: %s", teamId, matchId, err)
		return lineup, substitutions
	}
	if hasOverride {
		return lineup, substitutions
	}

	squad, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching squad of team %d for its kickoff lineup in match %d: %s", teamId, matchId, err)
		return lineup, substitutions
	}
	leagueUnavailable, err := common.GetUnavailablePlayerIds(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching unavailable players of team %d for match %d: %s", teamId, matchId, err)
		return lineup, substitutions
	}
	matchUnavailable := leagueUnavailable
	if kind == kickoffFriendly {
		if matchUnavailable, err = common.GetUnavailablePlayerIdsForFriendly(dbCtx, teamId); err != nil {
			log.Errorf("error fetching friendly-unavailable players of team %d for match %d: %s", teamId, matchId, err)
			return lineup, substitutions
		}
	}

	rng := rand.New(rand.NewSource(aiKickoffSeed(matchId, teamId)))
	opts := aiPlanOptions(dbCtx, kind, team)
	opts.Formation = formation
	plan := common.PlanAiMatchLineupWithOptions(*squad, leagueUnavailable, matchUnavailable, team.AiLevel, kind == kickoffFriendly, rng, opts)

	// The same checks a human's saved lineup goes through. A Caretaker's XI may name unavailable
	// players (the kickoff backfill replaces them); a Manager's or Director's may not.
	enforceAvailability := team.AiLevel >= common.AiLevelManager
	if err := common.ValidateLineup(plan.Formation, plan.Positions, squad, matchUnavailable, enforceAvailability); err != nil {
		log.Errorf("kickoff lineup of team %d for match %d is invalid, keeping the stored lineup: %s", teamId, matchId, err)
		return lineup, substitutions
	}
	if err := common.ValidateSubstitutions(plan.Positions, plan.Substitutions, squad, matchUnavailable); err != nil {
		log.Errorf("kickoff substitutions of team %d for match %d are invalid, keeping the stored lineup: %s", teamId, matchId, err)
		return lineup, substitutions
	}
	if err := common.ValidateOrders(plan.Positions, plan.Orders, squad, matchUnavailable); err != nil {
		log.Errorf("kickoff orders of team %d for match %d are invalid, keeping the stored lineup: %s", teamId, matchId, err)
		return lineup, substitutions
	}
	if err := common.ValidateRoles(plan.Positions, plan.Roles, squad); err != nil {
		log.Errorf("kickoff roles of team %d for match %d are invalid, keeping the stored lineup: %s", teamId, matchId, err)
		return lineup, substitutions
	}

	planned := &common.Lineup{MatchId: matchId, TeamId: teamId, Formation: plan.Formation, Positions: plan.Positions, Substitutions: plan.Substitutions, Orders: plan.Orders, LineupRoles: plan.Roles}
	if lineup != nil {
		planned.MatchId = lineup.MatchId
	}
	return planned, plan.Substitutions
}

// aiDirectorYouthStarts is how many youngsters a Director with nothing at stake starts in a
// league match (see common.AiPlanOptions).
const aiDirectorYouthStarts = 2

// aiPlanOptions are the situation options of a CPU side's kickoff plan: a Director's league match
// with nothing left at stake gives aiDirectorYouthStarts youngsters a start.
func aiPlanOptions(dbCtx *common.DatabaseContext, kind kickoffKind, team *common.Team) common.AiPlanOptions {
	if kind != kickoffLeague || team.AiLevel < common.AiLevelDirector {
		return common.AiPlanOptions{}
	}
	if common.TeamSeasonObjective(dbCtx, team.Id) != common.SeasonObjectiveNothingAtStake {
		return common.AiPlanOptions{}
	}
	return common.AiPlanOptions{YouthStarts: aiDirectorYouthStarts}
}

// planAiKickoffs applies planAiKickoff to both sides of a match, replacing their entries in
// lineups and returning each side's substitutions.
func planAiKickoffs(dbCtx *common.DatabaseContext, kind kickoffKind, matchId int64, homeTeamId int64, awayTeamId int64,
	lineups *map[int64]*common.Lineup, homeSubstitutions []common.LineupSubstitution, awaySubstitutions []common.LineupSubstitution,
) ([]common.LineupSubstitution, []common.LineupSubstitution) {
	homeBefore, awayBefore := (*lineups)[homeTeamId], (*lineups)[awayTeamId]
	(*lineups)[homeTeamId], homeSubstitutions = planAiKickoff(dbCtx, kind, matchId, homeTeamId, homeBefore, homeSubstitutions)
	(*lineups)[awayTeamId], awaySubstitutions = planAiKickoff(dbCtx, kind, matchId, awayTeamId, awayBefore, awaySubstitutions)

	// A Director re-picks its formation against the other side's XI now that it is known (see
	// directorFormationAgainst) - each side against the other's first-pass XI, so the order of the
	// two re-picks doesn't matter.
	if kind != kickoffFriendly {
		homeFirst, awayFirst := (*lineups)[homeTeamId], (*lineups)[awayTeamId]
		if homeFirst != nil && homeFirst != homeBefore {
			if f, ok := directorFormationAgainst(dbCtx, homeTeamId, homeFirst, awayTeamId, awayFirst, true); ok {
				(*lineups)[homeTeamId], homeSubstitutions = planAiKickoffWithFormation(dbCtx, kind, matchId, homeTeamId, homeBefore, homeSubstitutions, &f)
			}
		}
		if awayFirst != nil && awayFirst != awayBefore {
			if f, ok := directorFormationAgainst(dbCtx, awayTeamId, awayFirst, homeTeamId, homeFirst, false); ok {
				(*lineups)[awayTeamId], awaySubstitutions = planAiKickoffWithFormation(dbCtx, kind, matchId, awayTeamId, awayBefore, awaySubstitutions, &f)
			}
		}
	}

	// A side whose plan was taken (a fresh lineup, not the one passed in) picks its mentality now
	// that both XIs are known - see common.PickAiMentality.
	homePlanned := (*lineups)[homeTeamId] != nil && (*lineups)[homeTeamId] != homeBefore
	awayPlanned := (*lineups)[awayTeamId] != nil && (*lineups)[awayTeamId] != awayBefore
	if homePlanned || awayPlanned {
		homeStrength, awayStrength := kickoffStrength(dbCtx, homeTeamId, (*lineups)[homeTeamId]), kickoffStrength(dbCtx, awayTeamId, (*lineups)[awayTeamId])
		if homePlanned {
			applyAiKickoffTactics(dbCtx, kind, homeTeamId, (*lineups)[homeTeamId], homeStrength, awayStrength, true)
		}
		if awayPlanned {
			applyAiKickoffTactics(dbCtx, kind, awayTeamId, (*lineups)[awayTeamId], awayStrength, homeStrength, false)
		}
	}
	return homeSubstitutions, awaySubstitutions
}

// directorFormationAgainst is the formation a Director side (planned, its first-pass XI) should
// play against the opponent's XI (see chooseFormationAgainst), and whether it differs from the
// planned one; false for any other level, an unknown opponent XI or a read error. The XIs are
// rated on a neutral ground: the home advantage multiplies every zone alike, so it hardly changes
// which shape is best.
func directorFormationAgainst(dbCtx *common.DatabaseContext, teamId int64, planned *common.Lineup, opponentId int64, opponentLineup *common.Lineup, isHome bool) (int, bool) {
	if opponentLineup == nil || len(opponentLineup.Positions) == 0 {
		return 0, false
	}
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil || team.AiLevel < common.AiLevelDirector {
		return 0, false
	}
	squad, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error reading the squad of team %d to pick its formation: %s", teamId, err)
		return 0, false
	}
	unavailable, err := common.GetUnavailablePlayerIds(dbCtx, teamId)
	if err != nil {
		log.Errorf("error reading the unavailable players of team %d to pick its formation: %s", teamId, err)
		return 0, false
	}
	opponentSquad, err := common.GetPlayersForMatchTeam(dbCtx, opponentId, opponentLineup)
	if err != nil {
		log.Errorf("error reading the squad of team %d to pick its opponent's formation: %s", opponentId, err)
		return 0, false
	}
	available := make([]*common.Player, 0, len(*squad))
	for _, p := range *squad {
		if !unavailable[p.Id] {
			available = append(available, p)
		}
	}
	f := chooseFormationAgainst(available, ZoneDuelSide{Lineup: opponentLineup, Squad: *opponentSquad}, 1, isHome)
	return f, f != planned.Formation
}

// kickoffStrength is a side's XI strength as the CPU judges it (see common.LineupStrength), from
// its own squad; 0 when it can't be read.
func kickoffStrength(dbCtx *common.DatabaseContext, teamId int64, lineup *common.Lineup) int {
	if lineup == nil {
		return 0
	}
	squad, err := common.GetPlayersForMatchTeam(dbCtx, teamId, lineup)
	if err != nil {
		log.Errorf("error reading the squad of team %d to weigh its kickoff XI: %s", teamId, err)
		return 0
	}
	byId := make(map[int64]*common.Player, len(*squad))
	for _, p := range *squad {
		byId[p.Id] = p
	}
	return common.LineupStrength(lineup, byId)
}

// applyAiKickoffTactics sets a CPU side's kickoff mentality from its AI level and both XIs'
// strengths (see common.PickAiMentality) and, in a cup tie, appends a Director's drawing order
// (see common.AiCupDrawOrder) while the lineup still has room for one.
func applyAiKickoffTactics(dbCtx *common.DatabaseContext, kind kickoffKind, teamId int64, lineup *common.Lineup, own int, opponent int, isHome bool) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching team %d for its kickoff mentality: %s", teamId, err)
		lineup.Mentality = common.MENTALITY_BALANCED
		return
	}
	setAiKickoffTactics(team.AiLevel, kind, lineup, own, opponent, isHome)
}

// setAiKickoffTactics is applyAiKickoffTactics once the AI level is known - pure, for tests.
func setAiKickoffTactics(aiLevel int, kind kickoffKind, lineup *common.Lineup, own int, opponent int, isHome bool) {
	lineup.Mentality = common.PickAiMentality(aiLevel, own, opponent, isHome, kind == kickoffFriendly)
	if kind == kickoffCup && len(lineup.Orders) < common.MaxConditionalOrders {
		if order := common.AiCupDrawOrder(aiLevel, own, opponent); order != nil {
			lineup.Orders = append(append([]common.LineupOrder(nil), lineup.Orders...), *order)
		}
	}
}
