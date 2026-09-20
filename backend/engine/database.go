package engine

import (
	"encoding/json"
	"fmt"
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

// UpdateTeamFinancesAfterMatch credits both teams their matchday income - gate receipts for the
// home side, sponsorship for both - since that's genuinely event-driven, tied to actually
// playing. Wages are not: see PayWeeklyWages, which runs on a real-calendar-time schedule
// independent of who played, the same way a real club's payroll doesn't pause between fixtures.
// This is also the once-per-match hook point for any unowned team's AI upkeep (see
// RunAITeamManagement) - matches are the natural cadence for that, and reusing this call site
// avoids a separate scheduler job for it.
//
// Returns the home side's computed attendance (see creditTicketRevenue) - simulateMatch reuses
// this exact figure for the match's report flavor rather than drawing a second, independent one.
func UpdateTeamFinancesAfterMatch(dbCtx *common.DatabaseContext, match *common.MatchSimulation) int {
	attendance := creditTicketRevenue(dbCtx, match.Home)
	creditSponsorship(dbCtx, match.Home, match.HomeGoals, match.AwayGoals)
	creditSponsorship(dbCtx, match.Away, match.AwayGoals, match.HomeGoals)
	RunAITeamManagement(dbCtx, match.Home)
	RunAITeamManagement(dbCtx, match.Away)
	return attendance
}

// NotifyMatchResult tells both teams' owners how their league match went.
func NotifyMatchResult(dbCtx *common.DatabaseContext, match *common.MatchSimulation) {
	homeTeam, errHome := common.GetTeamById(match.Home, dbCtx)
	awayTeam, errAway := common.GetTeamById(match.Away, dbCtx)
	if errHome != nil || errAway != nil {
		log.Errorf("error getting team names for match result notification: %v / %v", errHome, errAway)
		return
	}
	msg := common.NotificationMessage{
		Text: fmt.Sprintf("%s %d - %d %s", homeTeam.TeamName, match.HomeGoals, match.AwayGoals, awayTeam.TeamName),
		Key:  "notifications.messages.matchResult",
		Params: map[string]any{
			"home": homeTeam.TeamName, "homeGoals": match.HomeGoals,
			"away": awayTeam.TeamName, "awayGoals": match.AwayGoals,
		},
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Home, common.NOTIFICATION_MATCH_RESULT, msg); err != nil {
		log.Errorf("error notifying home team owner of match result: %s", err)
	}
	if err := common.NotifyTeamOwner(dbCtx, match.Away, common.NOTIFICATION_MATCH_RESULT, msg); err != nil {
		log.Errorf("error notifying away team owner of match result: %s", err)
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
