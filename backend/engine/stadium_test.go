package engine

import (
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

// avgAttendance averages CalculateAttendance over many draws with a fixed seed, so a comparison
// between two scenarios isn't drowned by the +/- AttendanceNoise swing.
func avgAttendance(capacity, price, socios, formPoints, formMatches, rivalFanbase, cupRound int, weather string) float64 {
	const trials = 4000
	r := rand.New(rand.NewSource(7))
	total := 0
	for i := 0; i < trials; i++ {
		total += CalculateAttendance(capacity, price, socios, formPoints, formMatches, rivalFanbase, cupRound, weather, r)
	}
	return float64(total) / trials
}

func TestCalculateAttendance_NeverExceedsCapacityOrGoesNegative(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for _, capacity := range []int{50, 1000, 30000} {
		for _, socios := range []int{1, 500, 100000} {
			for i := 0; i < 500; i++ {
				got := CalculateAttendance(capacity, common.DefaultTicketPrice, socios, 15, 5, 99, 3, "sunny", r)
				if got < 0 || got > capacity {
					t.Fatalf("attendance %d out of [0, %d] (socios %d)", got, capacity, socios)
				}
			}
		}
	}
}

func TestCalculateAttendance_ZeroCapacity(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	if got := CalculateAttendance(0, common.DefaultTicketPrice, common.DefaultSocios, 0, 0, -1, 0, "", r); got != 0 {
		t.Fatalf("expected 0 attendance for 0 capacity, got %d", got)
	}
}

// A default club (500 socios, default price, no history, neutral opponent/weather) keeps drawing
// the ~500 a default ground drew before socios existed.
func TestCalculateAttendance_DefaultClubDrawsItsSocios(t *testing.T) {
	got := avgAttendance(common.DefaultStadiumCapacity, common.DefaultTicketPrice, common.DefaultSocios, 0, 0, common.DefaultFanbase, 0, "cloudy")
	// rivalFactor(50) is ~1.0, so the crowd should sit within a few percent of the member count.
	if got < 470 || got > 530 {
		t.Fatalf("average attendance %.0f too far from the 500 socios", got)
	}
}

// The point of the whole model: a bigger ground does not draw a bigger crowd by itself.
func TestCalculateAttendance_CapacityDoesNotCreateDemand(t *testing.T) {
	small := avgAttendance(1000, common.DefaultTicketPrice, common.DefaultSocios, 0, 0, common.DefaultFanbase, 0, "cloudy")
	huge := avgAttendance(50000, common.DefaultTicketPrice, common.DefaultSocios, 0, 0, common.DefaultFanbase, 0, "cloudy")
	if huge > small*1.05 {
		t.Fatalf("a 50,000-seat ground drew %.0f vs %.0f in a 1,000-seat one - capacity must not create demand", huge, small)
	}
}

func TestCalculateAttendance_MoreSociosDrawMore(t *testing.T) {
	few := avgAttendance(30000, common.DefaultTicketPrice, 400, 0, 0, common.DefaultFanbase, 0, "cloudy")
	many := avgAttendance(30000, common.DefaultTicketPrice, 1600, 0, 0, common.DefaultFanbase, 0, "cloudy")
	if many <= few*3 {
		t.Fatalf("4x the socios should draw about 4x the crowd, got %.0f vs %.0f", many, few)
	}
}

func TestCalculateAttendance_HigherPriceReducesAttendance(t *testing.T) {
	cheap := avgAttendance(30000, 5, 1000, 0, 0, common.DefaultFanbase, 0, "cloudy")
	normal := avgAttendance(30000, common.DefaultTicketPrice, 1000, 0, 0, common.DefaultFanbase, 0, "cloudy")
	dear := avgAttendance(30000, 30, 1000, 0, 0, common.DefaultFanbase, 0, "cloudy")
	if !(cheap > normal && normal > dear) {
		t.Fatalf("attendance should fall as price rises, got %.0f / %.0f / %.0f", cheap, normal, dear)
	}
}

// No occupancy floor: at and above SociosPriceZeroAt nobody turns up, however empty the ground.
func TestCalculateAttendance_NobodyPaysAbsurdPrices(t *testing.T) {
	for _, price := range []int{SociosPriceZeroAt, 70, common.MaxTicketPrice} {
		if got := avgAttendance(30000, price, 5000, 15, 5, 99, 0, "sunny"); got != 0 {
			t.Fatalf("price %d drew %.0f, want 0", price, got)
		}
	}
}

func TestCalculateAttendance_FormRivalWeatherAndCupRound(t *testing.T) {
	base := func(formPoints, formMatches, rival, cup int, weather string) float64 {
		return avgAttendance(30000, common.DefaultTicketPrice, 1000, formPoints, formMatches, rival, cup, weather)
	}
	if base(15, 5, 50, 0, "cloudy") <= base(0, 5, 50, 0, "cloudy") {
		t.Error("a winning streak should draw more than a losing one")
	}
	if base(0, 5, 99, 0, "cloudy") <= base(0, 5, 0, 0, "cloudy") {
		t.Error("a big visitor should draw more than an unknown one")
	}
	if base(0, 5, 50, 0, "sunny") <= base(0, 5, 50, 0, "rain") {
		t.Error("sun should draw more than rain")
	}
	if base(0, 5, 50, 3, "cloudy") <= base(0, 5, 50, 0, "cloudy") {
		t.Error("a later cup round should draw more than a league match")
	}
}

func TestFormFactor(t *testing.T) {
	if got := formFactor(0, 0); got != 1.0 {
		t.Errorf("no history should be neutral, got %v", got)
	}
	if got := formFactor(0, 5); got != FormFactorMin {
		t.Errorf("five defeats should be %v, got %v", FormFactorMin, got)
	}
	if got := formFactor(15, 5); got < FormFactorMin+FormFactorSpan-1e-9 {
		t.Errorf("five wins should be %v, got %v", FormFactorMin+FormFactorSpan, got)
	}
}

func TestOptimalTicketPrice_WithinBounds(t *testing.T) {
	for _, socios := range []int{50, 500, 5000} {
		price := optimalTicketPrice(1000, socios, 0, 0)
		if price < common.MinTicketPrice || price > common.MaxTicketPrice {
			t.Fatalf("optimal price %d out of bounds [%d, %d]", price, common.MinTicketPrice, common.MaxTicketPrice)
		}
	}
}

// With a ground the members can't fill, the best price is the revenue-maximising one on the demand
// curve, and no longer the old runaway 69 - it must stay well under SociosPriceZeroAt.
func TestOptimalTicketPrice_UnfilledGroundStaysModerate(t *testing.T) {
	price := optimalTicketPrice(30000, 500, 0, 0)
	if price >= SociosPriceZeroAt || price > 25 {
		t.Fatalf("optimal price %d is too high for a ground its members cannot fill", price)
	}
}

// When there are more members than seats, the best price is the one that just fills the ground -
// charging more than that means empty seats, charging less leaves money on the table.
func TestOptimalTicketPrice_OversubscribedGroundFillsUp(t *testing.T) {
	capacity, socios := 1000, 3000
	price := optimalTicketPrice(capacity, socios, 0, 0)
	if price <= common.DefaultTicketPrice {
		t.Fatalf("with 3x demand the price (%d) should rise above the default", price)
	}
	revenue := expectedGateRevenue(capacity, socios, 0, 0, price)
	for p := common.MinTicketPrice; p <= common.MaxTicketPrice; p++ {
		if expectedGateRevenue(capacity, socios, 0, 0, p) > revenue+1e-9 {
			t.Fatalf("price %d beats the reported optimum %d", p, price)
		}
	}
}

func TestShouldUpgradeStadium(t *testing.T) {
	tests := []struct {
		soldOut, total int
		want           bool
	}{
		{0, 0, false},
		{4, 4, false}, // not enough history on record
		{1, 10, false},
		{3, 10, false},
		{4, 10, true},
		{10, 10, true},
		{2, 5, true},
		{1, 5, false},
	}
	for _, tt := range tests {
		if got := shouldUpgradeStadium(tt.soldOut, tt.total); got != tt.want {
			t.Errorf("shouldUpgradeStadium(%d, %d) = %v, want %v", tt.soldOut, tt.total, got, tt.want)
		}
	}
}

func TestTicketRevenue(t *testing.T) {
	if got := TicketRevenue(1000, 10); got != 10000 {
		t.Fatalf("expected 10000, got %d", got)
	}
	if got := TicketRevenue(0, 10); got != 0 {
		t.Fatalf("expected 0 revenue for 0 attendance, got %d", got)
	}
}

func TestStadiumMaintenanceCost_ZeroCapacity(t *testing.T) {
	if got := StadiumMaintenanceCost(0); got != 0 {
		t.Fatalf("expected 0 maintenance cost for 0 capacity, got %d", got)
	}
}

// TestStadiumMaintenanceCost_ScalesWithCapacity guards the whole point of this feature: a bigger
// stadium must cost strictly more to maintain than a smaller one, not just more to build.
func TestStadiumMaintenanceCost_ScalesWithCapacity(t *testing.T) {
	small := StadiumMaintenanceCost(1000)
	big := StadiumMaintenanceCost(5000)
	if big <= small {
		t.Fatalf("expected maintenance cost to increase with capacity, got %d for 1000 and %d for 5000", small, big)
	}
	if big != small*5 {
		t.Fatalf("expected maintenance cost to scale linearly with capacity, got %d for 1000 and %d for 5000", small, big)
	}
}

func TestSplitRevenueEvenly(t *testing.T) {
	tests := []struct {
		revenue  int64
		wantHome int64
		wantAway int64
	}{
		{0, 0, 0},
		{10000, 5000, 5000},
		{10001, 5001, 5000}, // odd unit stays with home
	}
	for _, tt := range tests {
		home, away := splitRevenueEvenly(tt.revenue)
		if home != tt.wantHome || away != tt.wantAway {
			t.Errorf("splitRevenueEvenly(%d) = (%d, %d), want (%d, %d)", tt.revenue, home, away, tt.wantHome, tt.wantAway)
		}
		if home+away != tt.revenue {
			t.Errorf("splitRevenueEvenly(%d): shares %d+%d don't sum back to the original revenue", tt.revenue, home, away)
		}
	}
}
