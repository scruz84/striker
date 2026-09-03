package engine

import (
	"encoding/json"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

func UpdatePlayedMatch(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	log.Infof("Updating played match %d home team %d from division %d", match.Id, match.Home, match.DivisionId)
	_, err := dbCtx.Db.Exec(common.UpdateMatchStmtSql, match.HomeGoals, match.AwayGoals, true, false, match.Id)
	if err != nil {
		log.Errorf("error updating match: %s", err)
		return
	}

	//Match events
	insertEvent, _ := dbCtx.Db.Prepare(common.InsertMatchEventStmtSql)
	defer insertEvent.Close()
	for _, e := range *match.Events {
		teamOwnerPlayersJson, _ := json.Marshal(e.TeamOwnerPlayers)
		teamOwnerPlayerNamesJson, _ := json.Marshal(e.TeamOwnerPlayersNames)
		otherTeamOwnerPlayersJson, _ := json.Marshal(e.OtherTeamPlayers)
		otherTeamOwnerPlayerNamesJson, _ := json.Marshal(e.OtherTeamPlayersNames)
		_, err := insertEvent.Exec(match.Id, e.EventType, e.TeamOwner, e.OtherTeam, e.Minute, teamOwnerPlayersJson,
			teamOwnerPlayerNamesJson, otherTeamOwnerPlayersJson, otherTeamOwnerPlayerNamesJson, e.HomeGoals, e.AwayGoals)
		if err != nil {
			log.Errorf("error inserting event: %s", err)
		}
	}

	//Match instants
	insertInstant, _ := dbCtx.Db.Prepare(common.InsertMatchInstantStmtSql)
	defer insertInstant.Close()
	for _, i := range *match.MatchInstants {
		homeQuaJson, _ := json.Marshal(i.HomeQua)
		awayQuaJson, _ := json.Marshal(i.AwayQua)
		_, err := insertInstant.Exec(i.MatchId, i.Minute, i.HomeGoals, i.AwayGoals, homeQuaJson, awayQuaJson, i.MidfieldDiff,
			i.HomeAttackRightDiff, i.HomeAttackLeftDiff, i.HomeAttackCenterDiff,
			i.AwayAttackRightDiff, i.AwayAttackLeftDiff, i.AwayAttackCenterDiff)
		if err != nil {
			log.Errorf("error inserting match instant: %s", err)
		}
	}
}

// UpdatePlayerAvailability counts this match against every already-unavailable player on
// both teams (clearing anyone whose suspension/injury has now run out), then records any new
// suspensions/injuries picked up during this match - in that order, so a player booked in
// this match doesn't have their brand-new unavailability immediately decremented by the same
// match that caused it.
func UpdatePlayerAvailability(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	if err := common.DecrementUnavailabilityForTeam(dbCtx, match.Home); err != nil {
		log.Errorf("error decrementing unavailability for team %d: %s", match.Home, err)
	}
	if err := common.DecrementUnavailabilityForTeam(dbCtx, match.Away); err != nil {
		log.Errorf("error decrementing unavailability for team %d: %s", match.Away, err)
	}
	for _, u := range match.PendingUnavailabilities {
		if err := common.AddPlayerUnavailability(dbCtx, u.PlayerId, u.Reason, u.Matches); err != nil {
			log.Errorf("error recording unavailability for player %d: %s", u.PlayerId, err)
		}
	}
}

func UpdateClassification(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	log.Infof("Updating classification for match %d home team %d from division %d", match.Id, match.Home, match.DivisionId)
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
	_, err := dbCtx.Db.Exec(common.UpdateTeamClassificationStmtSql, 1, points, win, loss, draws, match.HomeGoals, match.AwayGoals, match.DivisionId, match.Home)
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
	_, err2 := dbCtx.Db.Exec(common.UpdateTeamClassificationStmtSql, 1, points, win, loss, draws, match.AwayGoals,
		match.HomeGoals, match.DivisionId, match.Away)
	if err2 != nil {
		log.Errorf("Error updating classification: %s", err2)
	}
}
