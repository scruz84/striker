package engine

import (
	"database/sql"
	"os"
	"testing"

	"github.com/scruz84/striker/common"
)

// TestInternationalCup_EntrantsAreMostRecentTier1Champions runs against a real Postgres
// (STRIKER_TEST_DB_DSN; skipped when unset). Regional/international cups no longer read a
// country's live classification table - countries now advance their own season independently
// (see init75_season_per_country.sql), so there's no single moment when every country is
// "between seasons" at once to read standings from. common.GetMostRecentTier1Champions reads
// season_history instead (the persisted record of who actually won), and this pins that it picks
// the right team even when a country's CURRENT classification table - already reset and reused
// for a newer season, see common.ResetClassificationForSeason - shows something else entirely at
// rank 1.
func TestInternationalCup_EntrantsAreMostRecentTier1Champions(t *testing.T) {
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

	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}

	// Two made-up countries: unknown codes fall back to the "europe" region (see
	// common.RegionForCountry), so they share one international cup.
	seed := func(code string) (champion, stale int64) {
		exec(`INSERT INTO countries (name) VALUES ($1) ON CONFLICT DO NOTHING`, code)
		tier1 := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, code)
		champion = scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, code+" champion", code, tier1)
		stale = scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, code+" stale-leader", code, tier1)

		// season_history: this country's season 1 ended with `champion` on top - the persisted
		// fact GetMostRecentTier1Champions must read.
		exec(`INSERT INTO season_history (team_id, season, league_id, final_position) VALUES ($1, 1, $2, 1)`, champion, tier1)

		// The live classification table has since moved on to a newer season and shows `stale`
		// at rank 1 instead - exactly what reading live standings (the old regionChampions) would
		// wrongly pick, and exactly what this test must NOT pick.
		exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 2, 30)`, tier1, stale)
		exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 2, 10)`, tier1, champion)
		return champion, stale
	}
	champion1, stale1 := seed("zq")
	champion2, stale2 := seed("zr")

	champions, err := common.GetMostRecentTier1Champions(dbCtx, CupTier)
	if err != nil {
		t.Fatalf("GetMostRecentTier1Champions: %s", err)
	}
	if champions["zq"] != champion1 || champions["zr"] != champion2 {
		t.Fatalf("GetMostRecentTier1Champions: got %v, want zq=%d zr=%d", champions, champion1, champion2)
	}

	byRegion := make(map[string][]int64)
	for country, teamId := range champions {
		byRegion[common.RegionForCountry(country)] = append(byRegion[common.RegionForCountry(country)], teamId)
	}

	const season = 2
	StartInternationalCupsForNewSeason(dbCtx, season, byRegion)

	cupId := scan(`SELECT cup_id FROM cups WHERE region = 'europe' AND season = $1`, season)
	ties, err := common.GetCupTies(dbCtx, cupId)
	if err != nil {
		t.Fatalf("GetCupTies: %s", err)
	}
	entrants := make(map[int64]bool)
	for _, tie := range ties {
		if tie.HomeTeamId != nil {
			entrants[*tie.HomeTeamId] = true
		}
		if tie.AwayTeamId != nil {
			entrants[*tie.AwayTeamId] = true
		}
	}
	for _, c := range []struct{ champion, stale int64 }{{champion1, stale1}, {champion2, stale2}} {
		if !entrants[c.champion] {
			t.Errorf("tier-1 champion %d is missing from the cup entrants %v", c.champion, entrants)
		}
		if entrants[c.stale] {
			t.Errorf("stale live-standings team %d must not be a cup entrant (entrants %v)", c.stale, entrants)
		}
	}
}
