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

// FanbaseAttendanceMinFactor/MaxFactor bound how much Fanbase can scale a team's base occupancy
// rate (see fanbaseAttendanceFactor) before price/jitter apply on top - 0.5x at Fanbase 0 up to
// 1.5x at Fanbase 99, a real, noticeable swing (comparable in size to OccupancyJitter's own)
// without letting either extreme dominate price itself. common.DefaultFanbase (50) lands almost
// exactly at 1.0x, so an unestablished team's attendance behaves the same as before Fanbase
// existed.
const FanbaseAttendanceMinFactor = 0.5
const FanbaseAttendanceMaxFactor = 1.5

// fanbaseAttendanceFactor maps a 0-99 Fanbase value onto [FanbaseAttendanceMinFactor,
// FanbaseAttendanceMaxFactor] - see CalculateAttendance, which multiplies demandRate's own
// price-based rate by this before applying jitter/clamping.
func fanbaseAttendanceFactor(fanbase int) float64 {
	return FanbaseAttendanceMinFactor + float64(fanbase)/99.0*(FanbaseAttendanceMaxFactor-FanbaseAttendanceMinFactor)
}

// CalculateAttendance derives one home match's crowd size from a team's stadium capacity, its
// current ticket price (see demandRate - a higher price costs some attendance), and its Fanbase
// (see fanbaseAttendanceFactor - a bigger supporter base fills more of the same stadium at the
// same price).
func CalculateAttendance(capacity int, ticketPrice int, fanbase int, r *rand.Rand) int {
	if capacity <= 0 {
		return 0
	}
	rate := demandRate(ticketPrice)*fanbaseAttendanceFactor(fanbase) + (r.Float64()*2-1)*OccupancyJitter
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

// StadiumMaintenanceCost returns one round's upkeep bill for a stadium of the given capacity -
// linear in capacity (see common.StadiumMaintenanceCostPerSeat/StadiumMaintenanceCostDivisor), so
// a bigger stadium costs more to run every round regardless of whether it's actually full, not
// just more to build in the first place (see common.UpgradeStadium).
func StadiumMaintenanceCost(capacity int) int64 {
	return int64(capacity) * common.StadiumMaintenanceCostPerSeat / common.StadiumMaintenanceCostDivisor
}

// deductStadiumMaintenance charges teamId its stadium's upkeep for the round - see
// StadiumMaintenanceCost. Called from PayMatchdayWages (wages.go) rather than getting its own
// scheduler hook, for the same cadence reason wages are: piggybacking on the same per-round call
// site that already fires exactly once per league match keeps this proportional to match count
// instead of real calendar time.
func deductStadiumMaintenance(dbCtx *common.DatabaseContext, teamId int64) {
	stadium, err := common.GetStadium(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting stadium for team %d: %s", teamId, err)
		return
	}
	cost := StadiumMaintenanceCost(stadium.Capacity)
	if cost <= 0 {
		return
	}
	if err := common.AdjustTeamBudget(dbCtx, teamId, -cost, common.TRANSACTION_STADIUM_MAINTENANCE, "stadium maintenance"); err != nil {
		log.Errorf("error deducting stadium maintenance for team %d: %s", teamId, err)
	}
}

// stadiumAttendanceAndRevenue draws one home match's crowd size and resulting gate revenue from
// homeTeamId's own stadium (capacity and ticket price) - shared by creditTicketRevenue (which
// pays the home side the full amount) and creditFriendlyTicketRevenue (which splits it in two).
// Returns a zero attendance/revenue pair, rather than erroring, when the stadium lookup fails -
// callers treat that the same as a genuinely empty house.
func stadiumAttendanceAndRevenue(dbCtx *common.DatabaseContext, homeTeamId int64) (int, int64) {
	stadium, err := common.GetStadium(dbCtx, homeTeamId)
	if err != nil {
		log.Errorf("error getting stadium for team %d: %s", homeTeamId, err)
		return 0, 0
	}
	fanbase, err := common.GetFanbase(dbCtx, homeTeamId)
	if err != nil {
		log.Errorf("error getting fanbase for team %d, using the neutral default: %s", homeTeamId, err)
		fanbase = common.DefaultFanbase
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	attendance := CalculateAttendance(stadium.Capacity, stadium.TicketPrice, fanbase, r)
	return attendance, TicketRevenue(attendance, stadium.TicketPrice)
}

// creditTicketRevenue pays the home team its matchday gate receipts - see
// UpdateTeamFinancesAfterMatch, which is the only caller (both league matches and cup ties route
// through it, so both generate ticket revenue for the home side only). Returns the attendance
// figure it computed (0 if the stadium lookup failed or the stadium is empty) so a league match's
// report (see common.UpdateMatchFlavor) can show the exact same crowd size rather than drawing
// its own, independent random attendance that could disagree with this transaction's own
// description.
func creditTicketRevenue(dbCtx *common.DatabaseContext, homeTeamId int64) int {
	attendance, revenue := stadiumAttendanceAndRevenue(dbCtx, homeTeamId)
	if revenue == 0 {
		return attendance
	}

	description := fmt.Sprintf("Matchday ticket revenue (%d attendance)", attendance)
	if err := common.AdjustTeamBudget(dbCtx, homeTeamId, revenue, common.TRANSACTION_TICKET_REVENUE, description); err != nil {
		log.Errorf("error crediting ticket revenue for team %d: %s", homeTeamId, err)
	}
	return attendance
}

// creditFriendlyTicketRevenue pays both sides of a friendly half of the home team's own matchday
// gate receipts each - see engine.simulateFriendly, the only caller. Unlike a league match or cup
// tie (see creditTicketRevenue above), where the away side gets nothing, a friendly's away team
// travels purely as a favor to the home side rather than for any competitive stake, so the gate it
// helped draw is split evenly instead of going entirely to the home side. Any odd unit from
// splitting an odd total stays with the home side, whose stadium actually generated it. Returns
// the attendance figure it computed (0 if the stadium lookup failed or the stadium is empty),
// same convention as creditTicketRevenue, even though no friendly report currently displays it.
func creditFriendlyTicketRevenue(dbCtx *common.DatabaseContext, homeTeamId int64, awayTeamId int64) int {
	attendance, revenue := stadiumAttendanceAndRevenue(dbCtx, homeTeamId)
	if revenue == 0 {
		return attendance
	}

	homeShare, awayShare := splitRevenueEvenly(revenue)

	description := fmt.Sprintf("Friendly ticket revenue (%d attendance, half share)", attendance)
	if err := common.AdjustTeamBudget(dbCtx, homeTeamId, homeShare, common.TRANSACTION_TICKET_REVENUE, description); err != nil {
		log.Errorf("error crediting friendly ticket revenue for home team %d: %s", homeTeamId, err)
	}
	if err := common.AdjustTeamBudget(dbCtx, awayTeamId, awayShare, common.TRANSACTION_TICKET_REVENUE, description); err != nil {
		log.Errorf("error crediting friendly ticket revenue for away team %d: %s", awayTeamId, err)
	}
	return attendance
}

// splitRevenueEvenly divides revenue into two integer shares that sum back to revenue exactly -
// any odd unit (an integer division always rounds toward zero, never the other way) goes to the
// first (home) share, so a stray cent never just vanishes from the two sides' combined total.
func splitRevenueEvenly(revenue int64) (homeShare int64, awayShare int64) {
	awayShare = revenue / 2
	homeShare = revenue - awayShare
	return
}
