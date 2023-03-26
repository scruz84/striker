package engine

import (
	"encoding/json"
	"footmanager/fmcommon"
	log "github.com/sirupsen/logrus"
)

func UpdatePlayedMatch(dbCtx *fmcommon.DatabaseContext, match *MatchSimulation) {
	updateMatchStmt := dbCtx.PreparedStmts[fmcommon.UpdateMatchStmt]
	_, err := updateMatchStmt.Exec(match.HomeGoals, match.AwayGoals, true, match.Id)
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

func UpdateClassification(dbCtx *fmcommon.DatabaseContext, match *MatchSimulation) {
	log.Infof("Updating classification for match %d home team %d from division %d", match.Id, match.Home, match.DivisionId)
	stmt := dbCtx.PreparedStmts[fmcommon.UpdateTeamClassificationStmt]
	points := 0
	win := 0
	loss := 1
	draws := 0
	//home
	if match.HomeGoals > match.AwayGoals {
		points = 3
		draws = 0
		win = 1
		loss = 0
	} else if match.HomeGoals < match.AwayGoals {
		points = 0
		draws = 0
		loss = 1
		win = 0
	} else {
		points = 1
		draws = 1
		loss = 0
		win = 0
	}
	_, err := stmt.Exec(1, points, win, loss, draws, match.HomeGoals, match.AwayGoals, match.DivisionId, match.Home)
	if err != nil {
		log.Errorf("Error updating classification: %s", err)
	}
	points = 0
	win = 0
	loss = 1
	draws = 0
	//away
	if match.AwayGoals > match.HomeGoals {
		points = 3
		draws = 0
		win = 1
		loss = 0
	} else if match.AwayGoals < match.HomeGoals {
		points = 0
		draws = 0
		loss = 1
		win = 0
	} else {
		points = 1
		draws = 1
		loss = 0
		win = 0
	}
	_, err2 := stmt.Exec(1, points, win, loss, draws, match.AwayGoals, match.HomeGoals, match.DivisionId, match.Away)
	if err2 != nil {
		log.Errorf("Error updating classification: %s", err2)
	}
}
