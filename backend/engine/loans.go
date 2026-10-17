package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// ReturnExpiredLoans sends every player on loan TO a team in country whose return_season has
// arrived back to its original owner - called once per that country's own season rollover,
// right after its season counter advances (see advanceCountrySeasonIfComplete), so
// return_season <= the new season means "due now." Scoped to country because a loan's
// return_season is denominated in the borrower's own season count (see
// common.GetExpiredLoansStmtSql).
func ReturnExpiredLoans(dbCtx *common.DatabaseContext, country string, newSeason int) {
	loans, err := common.GetExpiredLoans(dbCtx, country, newSeason)
	if err != nil {
		log.Errorf("error listing expired loans for %s: %s", country, err)
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
	homeMsg := common.NotificationMessage{
		Text:   playerName + " has returned from loan",
		Key:    "notifications.messages.loanReturnedHome",
		Params: map[string]any{"name": playerName},
	}
	if err := common.NotifyTeamOwner(dbCtx, loan.FromTeamId, common.NOTIFICATION_TRANSFER, homeMsg); err != nil {
		log.Errorf("error notifying loan return to team %d: %s", loan.FromTeamId, err)
	}
	awayMsg := common.NotificationMessage{
		Text:   playerName + " has returned to " + fromTeam.TeamName,
		Key:    "notifications.messages.loanReturnedAway",
		Params: map[string]any{"name": playerName, "team": fromTeam.TeamName},
	}
	if err := common.NotifyTeamOwner(dbCtx, loan.ToTeamId, common.NOTIFICATION_TRANSFER, awayMsg); err != nil {
		log.Errorf("error notifying loan return from team %d: %s", loan.ToTeamId, err)
	}
}
