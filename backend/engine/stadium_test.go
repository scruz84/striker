package engine

import (
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

func TestCalculateAttendance_WithinBounds(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	capacity := 10000
	for i := 0; i < 1000; i++ {
		attendance := CalculateAttendance(capacity, common.DefaultTicketPrice, common.DefaultFanbase, r)
		if attendance < 0 || attendance > capacity {
			t.Fatalf("attendance %d out of bounds for capacity %d", attendance, capacity)
		}
	}
}

func TestCalculateAttendance_ZeroCapacity(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	if got := CalculateAttendance(0, common.DefaultTicketPrice, common.DefaultFanbase, r); got != 0 {
		t.Fatalf("expected 0 attendance for 0 capacity, got %d", got)
	}
}

func TestCalculateAttendance_AveragesNearBaseRate(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	capacity := 10000
	total := 0
	const trials = 5000
	for i := 0; i < trials; i++ {
		// demandRate(DefaultTicketPrice) == BaseOccupancyRate exactly by construction, and
		// fanbaseAttendanceFactor(common.DefaultFanbase) is ~1.0 by the same kind of
		// construction - this is the one price/fanbase combination where attendance should still
		// average out to the historical baseline.
		total += CalculateAttendance(capacity, common.DefaultTicketPrice, common.DefaultFanbase, r)
	}
	average := float64(total) / float64(trials)
	expected := float64(capacity) * BaseOccupancyRate
	// Random jitter is symmetric, so over enough trials the average should land close to the
	// base rate - loose tolerance since this is just a sanity check, not a distribution test.
	if average < expected*0.9 || average > expected*1.1 {
		t.Fatalf("average attendance %.0f too far from expected ~%.0f", average, expected)
	}
}

// TestCalculateAttendance_HigherPriceReducesAttendance guards the whole point of demandRate:
// before it existed, attendance was independent of price, making "always charge the maximum" a
// strictly dominant (and unrealistic) strategy - see maximizeTicketPrice/optimalTicketPrice.
func TestCalculateAttendance_HigherPriceReducesAttendance(t *testing.T) {
	capacity := 10000
	const trials = 5000

	average := func(price int) float64 {
		r := rand.New(rand.NewSource(7))
		total := 0
		for i := 0; i < trials; i++ {
			total += CalculateAttendance(capacity, price, common.DefaultFanbase, r)
		}
		return float64(total) / float64(trials)
	}

	cheap := average(common.MinTicketPrice)
	expensive := average(common.MaxTicketPrice)
	if expensive >= cheap {
		t.Fatalf("expected average attendance at max price (%.0f) to be lower than at min price (%.0f)", expensive, cheap)
	}
}

// TestCalculateAttendance_BiggerFanbaseIncreasesAttendance guards the whole point of
// fanbaseAttendanceFactor: two teams with identical stadiums/prices should draw different crowds
// once their Fanbase differs.
func TestCalculateAttendance_BiggerFanbaseIncreasesAttendance(t *testing.T) {
	capacity := 10000
	const trials = 5000

	average := func(fanbase int) float64 {
		r := rand.New(rand.NewSource(7))
		total := 0
		for i := 0; i < trials; i++ {
			total += CalculateAttendance(capacity, common.DefaultTicketPrice, fanbase, r)
		}
		return float64(total) / float64(trials)
	}

	small := average(1)
	big := average(99)
	if big <= small {
		t.Fatalf("expected average attendance at fanbase 99 (%.0f) to be higher than at fanbase 1 (%.0f)", big, small)
	}
}

func TestOptimalTicketPrice_WithinBounds(t *testing.T) {
	price := optimalTicketPrice()
	if price < common.MinTicketPrice || price > common.MaxTicketPrice {
		t.Fatalf("optimal price %d out of bounds [%d, %d]", price, common.MinTicketPrice, common.MaxTicketPrice)
	}
}

// TestOptimalTicketPrice_BeatsChargingTheMaximum pins down the actual bug this whole feature
// fixes: maximizeTicketPrice used to always charge common.MaxTicketPrice because that was
// strictly optimal when attendance didn't respond to price - now that it does, that price should
// no longer win on expected revenue.
func TestOptimalTicketPrice_BeatsChargingTheMaximum(t *testing.T) {
	optimal := optimalTicketPrice()
	optimalRevenue := float64(optimal) * demandRate(optimal)
	maxRevenue := float64(common.MaxTicketPrice) * demandRate(common.MaxTicketPrice)
	if optimalRevenue < maxRevenue {
		t.Fatalf("optimal price %d (revenue %.2f) should be at least as good as charging the maximum (revenue %.2f)",
			optimal, optimalRevenue, maxRevenue)
	}
	if optimal == common.MaxTicketPrice {
		t.Fatalf("expected the optimal price to no longer be the maximum once attendance responds to price")
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
