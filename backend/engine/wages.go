package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// PayWeeklyWages charges every team its current player and head-coach wage bill - the game's
// one real-calendar-time recurring cost (see StartWeeklyWagesScheduler for the cadence),
// independent of whether that team actually played a match this week. Ticket revenue and
// sponsorship stay matchday-only (see UpdateTeamFinancesAfterMatch) since those are genuinely
// event-driven; a squad and coaching staff cost money to keep on the books whether or not the
// team took the pitch, the same way a real club's payroll doesn't pause between fixtures.
func PayWeeklyWages(dbCtx *common.DatabaseContext) {
	teamIds, err := common.GetAllTeamIds(dbCtx)
	if err != nil {
		log.Errorf("error fetching team ids for weekly wages: %s", err)
		return
	}
	for _, teamId := range teamIds {
		deductWages(dbCtx, teamId)
		deductTrainerWage(dbCtx, teamId)
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
	if err := common.AdjustTeamBudget(dbCtx, teamId, -wageBill, common.TRANSACTION_WAGES, "weekly wages"); err != nil {
		log.Errorf("error deducting wages for team %d: %s", teamId, err)
	}
}

// deductTrainerWage charges a team its head coach's weekly wage - a no-op for a team that
// hasn't hired one.
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
