package common

import "testing"

// TestCapTicketPriceMigration covers init87 of openspec fix-empty-gate-high-ticket-price: a team
// left on a price above MaxTicketPrice is reset to the default, one already in range keeps its
// price, and applying the migrations again changes nothing.
func TestCapTicketPriceMigration(t *testing.T) {
	db, dbCtx := openSimulationTestDb(t)
	f := newSimulationFixture(t)

	setPrice := func(teamId int64, price int) {
		t.Helper()
		if _, err := db.Exec(`UPDATE teams SET ticket_price = $1 WHERE team_id = $2`, price, teamId); err != nil {
			t.Fatalf("seeding ticket price: %s", err)
		}
	}
	priceOf := func(teamId int64) int {
		t.Helper()
		var price int
		if err := db.QueryRow(`SELECT ticket_price FROM teams WHERE team_id = $1`, teamId).Scan(&price); err != nil {
			t.Fatalf("reading ticket price: %s", err)
		}
		return price
	}
	setPrice(f.home, 69)
	setPrice(f.away, 25)

	for run := 1; run <= 2; run++ {
		if err := EnsureSchema(dbCtx); err != nil {
			t.Fatalf("EnsureSchema run %d: %s", run, err)
		}
		if got := priceOf(f.home); got != DefaultTicketPrice {
			t.Errorf("run %d: team at 69 has price %d, want %d", run, got, DefaultTicketPrice)
		}
		if got := priceOf(f.away); got != 25 {
			t.Errorf("run %d: team at 25 has price %d, want 25", run, got)
		}
	}
}
