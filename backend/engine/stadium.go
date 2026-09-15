package engine

import (
	"fmt"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

// A home crowd sits at roughly half of stadium capacity on average, swinging up to
// OccupancyJitter (as a fraction of capacity) either way on any given matchday - there's no
// attempt yet to model anything that would actually drive that swing (recent form, the
// opponent, weather...), just enough randomness that it isn't a flat number every match.
const BaseOccupancyRate = 0.5
const OccupancyJitter = 0.2
const MinOccupancyRate = 0.1
const MaxOccupancyRate = 1.0

// CalculateAttendance derives one home match's crowd size from a team's stadium capacity.
func CalculateAttendance(capacity int, r *rand.Rand) int {
	if capacity <= 0 {
		return 0
	}
	rate := BaseOccupancyRate + (r.Float64()*2-1)*OccupancyJitter
	if rate < MinOccupancyRate {
		rate = MinOccupancyRate
	}
	if rate > MaxOccupancyRate {
		rate = MaxOccupancyRate
	}
	return int(float64(capacity) * rate)
}

// TicketRevenue is deliberately simple: attendance times price, nothing more (no concessions,
// no season tickets, no away-fan allocation).
func TicketRevenue(attendance int, ticketPrice int) int64 {
	return int64(attendance) * int64(ticketPrice)
}

// creditTicketRevenue pays the home team its matchday gate receipts - see
// UpdateTeamFinancesAfterMatch, which is the only caller (both league matches and cup ties route
// through it, so both generate ticket revenue for the home side). Returns the attendance figure
// it computed (0 if the stadium lookup failed or the stadium is empty) so a league match's report
// (see common.UpdateMatchFlavor) can show the exact same crowd size rather than drawing its own,
// independent random attendance that could disagree with this transaction's own description.
func creditTicketRevenue(dbCtx *common.DatabaseContext, homeTeamId int64) int {
	stadium, err := common.GetStadium(dbCtx, homeTeamId)
	if err != nil {
		log.Errorf("error getting stadium for team %d: %s", homeTeamId, err)
		return 0
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	attendance := CalculateAttendance(stadium.Capacity, r)
	revenue := TicketRevenue(attendance, stadium.TicketPrice)
	if revenue == 0 {
		return attendance
	}

	description := fmt.Sprintf("Matchday ticket revenue (%d attendance)", attendance)
	if err := common.AdjustTeamBudget(dbCtx, homeTeamId, revenue, common.TRANSACTION_TICKET_REVENUE, description); err != nil {
		log.Errorf("error crediting ticket revenue for team %d: %s", homeTeamId, err)
	}
	return attendance
}
