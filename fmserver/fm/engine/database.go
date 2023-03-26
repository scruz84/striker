package engine

import (
	"encoding/json"
	"footmanager/fmcommon"
	log "github.com/sirupsen/logrus"
)

func UpdatePlayedMatch(dbCtx *fmcommon.DatabaseContext, match *MatchSimulation) {
	updateMatchStmt := dbCtx.PreparedStmts[fmcommon.UpdateMatchStmt]
	_, err := updateMatchStmt.Exec(match.HomeGoals, match.AwayGoals, match.Id)
	if err != nil {
		log.Errorf("error updating match: %s", err)
		return
	}

	insertEvent := dbCtx.PreparedStmts[fmcommon.InsertMatchEventStmt]
	for _, e := range *match.Events {
		teamOwnerPlayersJson, _ := json.Marshal(e.TeamOwnerPlayers)
		otherTeamOwnerPlayersJson, _ := json.Marshal(e.OtherTeamPlayers)
		_, err := insertEvent.Exec(match.Id, e.EventType, e.TeamOwner, e.OtherTeam, e.Minute, teamOwnerPlayersJson, otherTeamOwnerPlayersJson)
		if err != nil {
			log.Errorf("error inserting event: %s", err)
		}
	}
}
