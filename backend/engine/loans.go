package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// ReturnExpiredLoans sends every loaned player whose return_season has arrived back to its
// original owner - called once per season rollover, right after the season counter advances
// (see AdvanceSeasonIfComplete), so return_season <= the new season means "due now."
func ReturnExpiredLoans(dbCtx *common.DatabaseContext, newSeason int) {
	loans, err := common.GetExpiredLoans(dbCtx, newSeason)
	if err != nil {
		log.Errorf("error listing expired loans: %s", err)
		return
	}

	for _, loan := range loans {
		if err := common.ReturnLoan(dbCtx, loan); err != nil {
			log.Errorf("error returning loan %d (player %d): %s", loan.LoanId, loan.PlayerId, err)
			continue
		}
		notifyLoanReturn(dbCtx, loan)
	}
}

func notifyLoanReturn(dbCtx *common.DatabaseContext, loan common.PlayerLoan) {
	player, err := common.GetPlayerById(dbCtx, loan.PlayerId)
	if err != nil {
		log.Errorf("error getting player %d for loan return notification: %s", loan.PlayerId, err)
		return
	}
	fromTeam, err := common.GetTeamById(loan.FromTeamId, dbCtx)
	if err != nil {
		log.Errorf("error getting team %d for loan return notification: %s", loan.FromTeamId, err)
		return
	}

	playerName := player.FirstName + " " + player.LastName
	if err := common.NotifyTeamOwner(dbCtx, loan.FromTeamId, common.NOTIFICATION_TRANSFER, playerName+" has returned from loan"); err != nil {
		log.Errorf("error notifying loan return to team %d: %s", loan.FromTeamId, err)
	}
	if err := common.NotifyTeamOwner(dbCtx, loan.ToTeamId, common.NOTIFICATION_TRANSFER, playerName+" has returned to "+fromTeam.TeamName); err != nil {
		log.Errorf("error notifying loan return from team %d: %s", loan.ToTeamId, err)
	}
}
