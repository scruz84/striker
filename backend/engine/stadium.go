package engine

import (
	"fmt"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

// A home crowd sits at roughly half of stadium capacity on average when charging
// common.DefaultTicketPrice (see demandRate below), swinging up to OccupancyJitter (as a
// fraction of capacity) either way on any given matchday - there's still no attempt to model
// anything else that would plausibly drive that swing (recent form, the opponent, weather...),
// just enough randomness that it isn't a flat number every match.
const BaseOccupancyRate = 0.5
const OccupancyJitter = 0.2
const MinOccupancyRate = 0.1
const MaxOccupancyRate = 1.0

// PriceSensitivity is how much occupancy rate a team gives up by charging common.MaxTicketPrice
// instead of common.DefaultTicketPrice - the whole point being that maximizing price is no longer
// a free lunch (see engine.maximizeTicketPrice in aiteams.go, which used to be a strictly
// dominant move precisely because attendance never responded to price at all). 0.35 takes
// occupancy from the 0.5 base down near MinOccupancyRate at the maximum price - a real,
// noticeable penalty, comparable in size to OccupancyJitter's own swing, not a rounding error.
const PriceSensitivity = 0.35

// demandRate returns the occupancy rate a given ticket price implies, before jitter: exactly
// BaseOccupancyRate at common.DefaultTicketPrice, sliding down toward MinOccupancyRate as price
// rises to common.MaxTicketPrice, and up (a smaller move - MinTicketPrice sits much closer to
// DefaultTicketPrice than MaxTicketPrice does) as price drops toward common.MinTicketPrice.
// Linear in between - there's no claim this is the real shape of ticket demand, just that it has
// to slope the right direction for pricing to be an actual tradeoff instead of a one-way lever.
func demandRate(ticketPrice int) float64 {
	priceRange := float64(common.MaxTicketPrice - common.DefaultTicketPrice)
	return BaseOccupancyRate - PriceSensitivity*(float64(ticketPrice-common.DefaultTicketPrice)/priceRange)
}

// optimalTicketPrice returns whichever integer price in [common.MinTicketPrice,
// common.MaxTicketPrice] maximizes expected per-seat revenue (price * demandRate(price)) - the
// price a rational owner charges now that demandRate means a higher price can cost attendance
// (see engine.maximizeTicketPrice, the Manager-level AI policy that uses this). Ignores jitter
// and capacity: jitter averages to zero and capacity is just a constant multiplier on revenue, so
// neither changes which price comes out on top. A plain integer scan over at most 100 candidates
// rather than solving the maximization analytically, so this stays correct automatically if
// demandRate's shape ever changes instead of silently going stale like a hardcoded price would.
func optimalTicketPrice() int {
	best := common.MinTicketPrice
	bestRevenue := -1.0
	for price := common.MinTicketPrice; price <= common.MaxTicketPrice; price++ {
		rate := demandRate(price)
		if rate < MinOccupancyRate {
			rate = MinOccupancyRate
		}
		if rate > MaxOccupancyRate {
			rate = MaxOccupancyRate
		}
		if revenue := float64(price) * rate; revenue > bestRevenue {
			bestRevenue, best = revenue, price
		}
	}
	return best
}

// CalculateAttendance derives one home match's crowd size from a team's stadium capacity and its
// current ticket price (see demandRate) - a higher price now genuinely costs some attendance,
// rather than attendance being independent of price.
func CalculateAttendance(capacity int, ticketPrice int, r *rand.Rand) int {
	if capacity <= 0 {
		return 0
	}
	rate := demandRate(ticketPrice) + (r.Float64()*2-1)*OccupancyJitter
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
	attendance := CalculateAttendance(stadium.Capacity, stadium.TicketPrice, r)
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
