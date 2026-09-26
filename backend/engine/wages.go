package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// PayMatchdayWages charges one team its current player and head-coach wage bill - called once
// for each side of every simulated league match (see simulateMatch), so a team's wage cadence
// exactly tracks its league's own match cadence (common.MatchSchedule.WeekdaysMask) instead of a
// fixed real-calendar-time cron.
//
// This used to be PayWeeklyWages, fired once a week (Monday 00:00 UTC) regardless of how many
// rounds a league actually played that week. That decoupling was the game's core inflation bug:
// ticket revenue and sponsorship (see UpdateTeamFinancesAfterMatch) are paid per match, so a
// league scheduled to play daily earned up to 7x the income of a weekly league in the same real
// week while paying the exact same, cadence-independent wage bill - the same fixed number of
// season-long rounds cost a fraction of the wages once the season was compressed into fewer real
// weeks. Charging the same per-round amount this function always charged, but on every round
// instead of every calendar week, keeps a season's total wage cost proportional to its total
// match count (which doesn't change with cadence), the same way ticket/sponsorship income already
// is - so no schedule choice can out-earn its own payroll for free.
//
// Not called from cup ties or friendlies: a league's WeekdaysMask is the one schedule this is
// meant to track, and a team already gets exactly one league match per round (a round-robin
// schedule has no byes), so this fires exactly once per round per team, matching the cadence the
// income side of the ledger already follows.
func PayMatchdayWages(dbCtx *common.DatabaseContext, teamId int64) {
	deductWages(dbCtx, teamId)
	deductTrainerWage(dbCtx, teamId)
}

// PayAllTeamsWagesNow charges every team its current wage bill on demand - used by
// handlers.AdminPayWagesNowHandler as a manual testing convenience, not by any recurring
// scheduler (wages are cadence-driven now - see PayMatchdayWages).
func PayAllTeamsWagesNow(dbCtx *common.DatabaseContext) {
	teamIds, err := common.GetAllTeamIds(dbCtx)
	if err != nil {
		log.Errorf("error fetching team ids for on-demand wage payment: %s", err)
		return
	}
	for _, teamId := range teamIds {
		PayMatchdayWages(dbCtx, teamId)
	}
}

func deductWages(dbCtx *common.DatabaseContext, teamId int64) {
	wageBill, err := common.GetTeamWageBill(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting wage bill for team %d: %s", teamId, err)
		return
	}
	if wageBill == 0 {
		return
	}
	if err := common.AdjustTeamBudget(dbCtx, teamId, -wageBill, common.TRANSACTION_WAGES, "matchday wages"); err != nil {
		log.Errorf("error deducting wages for team %d: %s", teamId, err)
	}
}

// deductTrainerWage charges a team its head coach's wage - a no-op for a team that hasn't hired
// one.
func deductTrainerWage(dbCtx *common.DatabaseContext, teamId int64) {
	trainer, err := common.GetTeamTrainer(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting trainer for team %d: %s", teamId, err)
		return
	}
	if trainer == nil {
		return
	}
	description := "trainer wages: " + trainer.FirstName + " " + trainer.LastName
	if err := common.AdjustTeamBudget(dbCtx, teamId, -int64(trainer.Wage), common.TRANSACTION_TRAINER_WAGES, description); err != nil {
		log.Errorf("error deducting trainer wage for team %d: %s", teamId, err)
	}
}
