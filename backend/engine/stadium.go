package engine

import (
	"fmt"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"time"
)

// A home crowd is driven by the club's socios (members - see common.GetSocios), not by how many
// seats the stadium happens to have: demand = socios x price x form x rival x weather x noise,
// capped at capacity. Building a bigger ground therefore only helps a club whose socios actually
// outgrow the one it has - a stadium of 50,000 seats still draws what its members, price, form and
// opponent bring through the gate, which is what keeps stadium expansion from being a free money
// machine.
//
// There is deliberately no minimum occupancy: nobody pays 100 for a ticket just because the seat
// exists. The price that earns the most is therefore either the one that just fills the ground
// (when demand exceeds capacity) or a moderate one (when it doesn't) - see optimalTicketPrice.

// SociosPriceZeroAt is the ticket price at which a club's socios stop turning up entirely:
// the price factor is 1.0 at common.DefaultTicketPrice, falls linearly to 0 here, and rises above
// 1.0 (up to MaxPriceFactor) for a price below the default.
const SociosPriceZeroAt = 40
const MaxPriceFactor = 1.3

// FormFactorMin/FormFactorSpan: after five straight league defeats the crowd is 0.8x its normal
// size, after five wins 1.2x (see formFactor).
const FormFactorMin = 0.8
const FormFactorSpan = 0.4

// RivalFactorMin/RivalFactorSpan: an opponent nobody follows (Fanbase 0) pulls 0.85x of the usual
// crowd, the biggest club (Fanbase 99) 1.15x. A cup tie adds CupRoundAttendanceBoost per round on
// top (a later round is a bigger occasion).
const RivalFactorMin = 0.85
const RivalFactorSpan = 0.30
const CupRoundAttendanceBoost = 0.1

// AttendanceNoise is the +/- fraction of random swing around the modelled crowd.
const AttendanceNoise = 0.1

// weatherAttendanceFactors maps common.RandomWeather's codes to how much each moves turnout.
var weatherAttendanceFactors = map[string]float64{"sunny": 1.05, "cloudy": 1.0, "windy": 0.92, "rain": 0.82}

// meanWeatherAttendanceFactor is the expected weatherAttendanceFactor over the four equally
// likely conditions - what optimalTicketPrice assumes, since it can't know tomorrow's weather.
const meanWeatherAttendanceFactor = (1.05 + 1.0 + 0.92 + 0.82) / 4

// weatherFactor returns the turnout multiplier for a weather code; an unknown or empty code (a
// friendly, a stored match from before weather existed) counts as neutral.
func weatherFactor(weather string) float64 {
	if f, ok := weatherAttendanceFactors[weather]; ok {
		return f
	}
	return 1.0
}

// priceFactor is 1.0 at common.DefaultTicketPrice, 0 at SociosPriceZeroAt and above, capped at
// MaxPriceFactor for very cheap tickets.
func priceFactor(ticketPrice int) float64 {
	f := 1 - float64(ticketPrice-common.DefaultTicketPrice)/float64(SociosPriceZeroAt-common.DefaultTicketPrice)
	return math.Min(MaxPriceFactor, math.Max(0, f))
}

// formFactor turns a team's league points over its last formMatches matches into a crowd
// multiplier: FormFactorMin with no points, FormFactorMin+FormFactorSpan with every point won, and
// the neutral 1.0 when there is no recent league history at all.
func formFactor(formPoints int, formMatches int) float64 {
	if formMatches <= 0 {
		return 1.0
	}
	return FormFactorMin + FormFactorSpan*float64(formPoints)/float64(3*formMatches)
}

// rivalFactor is how much the visiting side pulls in a crowd, by its Fanbase (0-99); a cup tie
// (cupRound > 0) adds CupRoundAttendanceBoost per round. A nil-equivalent rival (fanbase < 0, e.g.
// unknown) is treated as the default fanbase.
func rivalFactor(rivalFanbase int, cupRound int) float64 {
	if rivalFanbase < 0 {
		rivalFanbase = common.DefaultFanbase
	}
	return (RivalFactorMin + RivalFactorSpan*float64(rivalFanbase)/99.0) * (1 + CupRoundAttendanceBoost*float64(cupRound))
}

// attendanceDemand is the modelled crowd before noise and before the capacity cap.
func attendanceDemand(socios int, ticketPrice int, formPoints int, formMatches int, rivalFanbase int, cupRound int, weather string) float64 {
	return float64(socios) * priceFactor(ticketPrice) * formFactor(formPoints, formMatches) *
		rivalFactor(rivalFanbase, cupRound) * weatherFactor(weather)
}

// CalculateAttendance derives one home match's crowd size: the modelled demand (see
// attendanceDemand) with +/- AttendanceNoise of random swing, never more than the stadium's
// capacity and never negative. cupRound is 0 for a league match or friendly.
func CalculateAttendance(capacity int, ticketPrice int, socios int, formPoints int, formMatches int, rivalFanbase int, cupRound int, weather string, r *rand.Rand) int {
	if capacity <= 0 {
		return 0
	}
	noise := 1 + (r.Float64()*2-1)*AttendanceNoise
	demand := attendanceDemand(socios, ticketPrice, formPoints, formMatches, rivalFanbase, cupRound, weather) * noise
	return int(math.Max(0, math.Min(float64(capacity), demand)))
}

// expectedGateRevenue is the revenue a ticket price is expected to earn at one league home match
// against an average opponent in average weather - what optimalTicketPrice maximises.
func expectedGateRevenue(capacity int, socios int, formPoints int, formMatches int, ticketPrice int) float64 {
	demand := attendanceDemand(socios, ticketPrice, formPoints, formMatches, common.DefaultFanbase, 0, "") * meanWeatherAttendanceFactor
	return math.Min(float64(capacity), demand) * float64(ticketPrice)
}

// optimalTicketPrice returns whichever integer price in [common.MinTicketPrice,
// common.MaxTicketPrice] maximises expected gate revenue (see expectedGateRevenue) for a stadium
// of this capacity and this many socios, given the team's recent form - the price a rational owner
// charges (see engine.maximizeTicketPrice, the Manager-level AI policy that uses this). A plain
// integer scan rather than solving analytically so it stays correct if the demand model changes.
// Ties go to the lower price.
func optimalTicketPrice(capacity int, socios int, formPoints int, formMatches int) int {
	best := common.MinTicketPrice
	bestRevenue := -1.0
	for price := common.MinTicketPrice; price <= common.MaxTicketPrice; price++ {
		if revenue := expectedGateRevenue(capacity, socios, formPoints, formMatches, price); revenue > bestRevenue {
			bestRevenue, best = revenue, price
		}
	}
	return best
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

// matchdayContext is everything besides the home side itself that shapes one home match's crowd:
// who is visiting, how big an occasion it is (cupRound, 0 for league/friendly), the weather
// already drawn for the match, and the match id whose own result must not count toward the home
// side's pre-kickoff form (0 for a cup tie or friendly, which has no row in matches).
type matchdayContext struct {
	awayTeamId     int64
	cupRound       int
	weather        string
	excludeMatchId int64
}

// stadiumAttendanceAndRevenue draws one home match's crowd size and resulting gate revenue from
// homeTeamId's own stadium (capacity and ticket price), socios, recent form and the visiting
// side's Fanbase - shared by creditTicketRevenue (which pays the home side the full amount) and
// creditFriendlyTicketRevenue (which splits it in two). Returns a zero attendance/revenue pair,
// rather than erroring, when the stadium lookup fails - callers treat that the same as a
// genuinely empty house. A failed socios/form/rival lookup falls back to the neutral value
// instead, so one bad read doesn't blank out a whole matchday's gate.
func stadiumAttendanceAndRevenue(dbCtx *common.DatabaseContext, homeTeamId int64, mc matchdayContext) (int, int64) {
	stadium, err := common.GetStadium(dbCtx, homeTeamId)
	if err != nil {
		log.Errorf("error getting stadium for team %d: %s", homeTeamId, err)
		return 0, 0
	}
	socios, err := common.GetSocios(dbCtx, homeTeamId)
	if err != nil {
		log.Errorf("error getting socios for team %d, using the default: %s", homeTeamId, err)
		socios = common.DefaultSocios
	}
	formPoints, formMatches, err := common.GetRecentLeagueForm(dbCtx, homeTeamId, mc.excludeMatchId)
	if err != nil {
		log.Errorf("error getting recent form for team %d, treating it as neutral: %s", homeTeamId, err)
		formPoints, formMatches = 0, 0
	}
	rivalFanbase := -1
	if mc.awayTeamId != 0 {
		if rivalFanbase, err = common.GetFanbase(dbCtx, mc.awayTeamId); err != nil {
			log.Errorf("error getting fanbase for visiting team %d, using the default: %s", mc.awayTeamId, err)
			rivalFanbase = -1
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	attendance := CalculateAttendance(stadium.Capacity, stadium.TicketPrice, socios, formPoints, formMatches, rivalFanbase, mc.cupRound, mc.weather, r)
	return attendance, TicketRevenue(attendance, stadium.TicketPrice)
}

// creditTicketRevenue pays the home team its matchday gate receipts - see
// UpdateTeamFinancesAfterMatch, which is the only caller (both league matches and cup ties route
// through it, so both generate ticket revenue for the home side only). Returns the attendance
// figure it computed (0 if the stadium lookup failed or the stadium is empty) so a league match's
// report (see common.UpdateMatchFlavor) can show the exact same crowd size rather than drawing
// its own, independent random attendance that could disagree with this transaction's own
// description.
func creditTicketRevenue(dbCtx *common.DatabaseContext, homeTeamId int64, mc matchdayContext) int {
	attendance, revenue := stadiumAttendanceAndRevenue(dbCtx, homeTeamId, mc)
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
	attendance, revenue := stadiumAttendanceAndRevenue(dbCtx, homeTeamId, matchdayContext{awayTeamId: awayTeamId, weather: common.RandomWeather()})
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
