package engine

import (
	"database/sql"
	"os"
	"testing"

	"github.com/scruz84/striker/common"
)

// TestRecommendedTicketPrice runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when
// unset): openspec fix-empty-gate-high-ticket-price shows owners the price a CPU Manager would
// charge, so it must follow the team's own capacity and socios.
func TestRecommendedTicketPrice(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer db.Close()
	dbCtx := &common.DatabaseContext{Db: db}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	const country = "zu3"
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`, country); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
	var leagueId int64
	if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, country).Scan(&leagueId); err != nil {
		t.Fatalf("seeding league: %s", err)
	}
	recommend := func(capacity, socios int) int {
		t.Helper()
		var teamId int64
		if err := db.QueryRow(`INSERT INTO teams (name, country, league_id, stadium_capacity) VALUES ('T', $1, $2, $3) RETURNING team_id`,
			country, leagueId, capacity).Scan(&teamId); err != nil {
			t.Fatalf("seeding team: %s", err)
		}
		if err := common.SetSocios(dbCtx, teamId, socios); err != nil {
			t.Fatalf("SetSocios: %s", err)
		}
		price, err := RecommendedTicketPrice(dbCtx, teamId)
		if err != nil {
			t.Fatalf("RecommendedTicketPrice: %s", err)
		}
		return price
	}

	if got := recommend(1000, 3000); got <= common.DefaultTicketPrice || got > common.MaxTicketPrice {
		t.Errorf("1,000 seats / 3,000 socios recommends %d, want above %d and within the range", got, common.DefaultTicketPrice)
	}
	if got := recommend(30000, 500); got > 25 || got < common.MinTicketPrice {
		t.Errorf("30,000 seats / 500 socios recommends %d, want at most 25", got)
	}
}
