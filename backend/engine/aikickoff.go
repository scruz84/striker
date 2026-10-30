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
	plan := common.PlanAiMatchLineup(*squad, leagueUnavailable, matchUnavailable, team.AiLevel, kind == kickoffFriendly, rng)

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

	planned := &common.Lineup{MatchId: matchId, TeamId: teamId, Formation: plan.Formation, Positions: plan.Positions, Substitutions: plan.Substitutions}
	if lineup != nil {
		planned.MatchId = lineup.MatchId
	}
	return planned, plan.Substitutions
}

// planAiKickoffs applies planAiKickoff to both sides of a match, replacing their entries in
// lineups and returning each side's substitutions.
func planAiKickoffs(dbCtx *common.DatabaseContext, kind kickoffKind, matchId int64, homeTeamId int64, awayTeamId int64,
	lineups *map[int64]*common.Lineup, homeSubstitutions []common.LineupSubstitution, awaySubstitutions []common.LineupSubstitution,
) ([]common.LineupSubstitution, []common.LineupSubstitution) {
	(*lineups)[homeTeamId], homeSubstitutions = planAiKickoff(dbCtx, kind, matchId, homeTeamId, (*lineups)[homeTeamId], homeSubstitutions)
	(*lineups)[awayTeamId], awaySubstitutions = planAiKickoff(dbCtx, kind, matchId, awayTeamId, (*lineups)[awayTeamId], awaySubstitutions)
	return homeSubstitutions, awaySubstitutions
}
