package engine

import (
	"math/rand"
	"testing"
)

func TestCalculateAttendance_WithinBounds(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	capacity := 10000
	for i := 0; i < 1000; i++ {
		attendance := CalculateAttendance(capacity, r)
		if attendance < 0 || attendance > capacity {
			t.Fatalf("attendance %d out of bounds for capacity %d", attendance, capacity)
		}
	}
}

func TestCalculateAttendance_ZeroCapacity(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	if got := CalculateAttendance(0, r); got != 0 {
		t.Fatalf("expected 0 attendance for 0 capacity, got %d", got)
	}
}

func TestCalculateAttendance_AveragesNearBaseRate(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	capacity := 10000
	total := 0
	const trials = 5000
	for i := 0; i < trials; i++ {
		total += CalculateAttendance(capacity, r)
	}
	average := float64(total) / float64(trials)
	expected := float64(capacity) * BaseOccupancyRate
	// Random jitter is symmetric, so over enough trials the average should land close to the
	// base rate - loose tolerance since this is just a sanity check, not a distribution test.
	if average < expected*0.9 || average > expected*1.1 {
		t.Fatalf("average attendance %.0f too far from expected ~%.0f", average, expected)
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
