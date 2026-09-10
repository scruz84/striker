package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// RunAITeamManagement drives every automatic action an unowned team's AI level entitles it to -
// called once per match for each side involved (see UpdateTeamFinancesAfterMatch) instead of a
// separate scheduler job, so AI upkeep happens exactly as often as game time actually advances.
// See common.AiLevelCaretaker/Manager/Director for what each tier adds on top of the one below
// it; a no-op for any team with a real owner.
func RunAITeamManagement(dbCtx *common.DatabaseContext, teamId int64) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching team %d for AI management: %s", teamId, err)
		return
	}
	if team.OwnerId > 0 || team.AiLevel < common.AiLevelCaretaker {
		return
	}

	refreshAiTeamLineup(dbCtx, teamId)
	// Level 2 (contracts/releases/free agents/ticket pricing) and Level 3 (transfer shopping/
	// stadium/trainer) actions are added in later phases, gated on team.AiLevel.
}

// refreshAiTeamLineup re-picks and persists an unowned team's best-XI default lineup from its
// current squad - the Caretaker-level baseline every AI-managed team gets, every AI level and up.
// Runs every match rather than only at season rollover (see autoManageLineupIfUnowned in
// progression.go, which still covers the retirement/academy-intake refresh at season end) so a
// mid-season squad change - a Level 2/3 signing or release, once those phases land - shows up in
// the default lineup right away instead of waiting for the next season.
func refreshAiTeamLineup(dbCtx *common.DatabaseContext, teamId int64) {
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching squad for AI-managed team %d lineup refresh: %s", teamId, err)
		return
	}
	if err := common.SetDefaultLineup(dbCtx, teamId, common.FlattenLineup(common.SelectBestLineup(*players))); err != nil {
		log.Errorf("error refreshing lineup for AI-managed team %d: %s", teamId, err)
	}
}
