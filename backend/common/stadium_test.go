package common

import (
	"math/rand"
	"testing"
)

func TestRandomStadiumCapacity_WithinTierRange(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	cases := []struct {
		tier     int
		min, max int
	}{
		{1, StadiumCapacityTier1Min, StadiumCapacityTier1Max},
		{2, StadiumCapacityTier2Min, StadiumCapacityTier2Max},
		{3, StadiumCapacityTier3Min, StadiumCapacityTier3Max},
		{99, StadiumCapacityTier3Min, StadiumCapacityTier3Max}, // unknown tier falls back to tier 3
	}
	for _, tc := range cases {
		for i := 0; i < 100; i++ {
			capacity := RandomStadiumCapacity(tc.tier, r)
			if capacity < tc.min || capacity > tc.max {
				t.Fatalf("tier %d: capacity %d out of range [%d, %d]", tc.tier, capacity, tc.min, tc.max)
			}
		}
	}
}

func TestRandomStadiumCapacity_HigherTiersAreBigger(t *testing.T) {
	midpoint := func(min, max int) int { return (min + max) / 2 }
	tier1 := midpoint(StadiumCapacityTier1Min, StadiumCapacityTier1Max)
	tier2 := midpoint(StadiumCapacityTier2Min, StadiumCapacityTier2Max)
	tier3 := midpoint(StadiumCapacityTier3Min, StadiumCapacityTier3Max)

	if tier1 <= tier2 || tier2 <= tier3 {
		t.Fatalf("expected each tier's typical capacity to be bigger than the tier below it: tier1=%d tier2=%d tier3=%d", tier1, tier2, tier3)
	}
}

func TestSetTicketPrice_RejectsOutOfRange(t *testing.T) {
	// SetTicketPrice validates before touching the database, so this doesn't need a live one.
	if err := SetTicketPrice(nil, 1, MinTicketPrice-1); err != ErrInvalidTicketPrice {
		t.Fatalf("expected ErrInvalidTicketPrice for a too-low price, got %v", err)
	}
	if err := SetTicketPrice(nil, 1, MaxTicketPrice+1); err != ErrInvalidTicketPrice {
		t.Fatalf("expected ErrInvalidTicketPrice for a too-high price, got %v", err)
	}
}
