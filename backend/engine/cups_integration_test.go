package engine

import (
	"database/sql"
	"os"
	"testing"

	"github.com/scruz84/striker/common"
)

// TestInternationalCup_EntrantsAreTierOneChampionsNotPromotedTeams runs against a real Postgres
// (STRIKER_TEST_DB_DSN, a database not reused across runs; skipped when unset). Promotion
// repoints the promoted teams' classification rows at the tier-1 league, so the champions have to
// be read before it and handed to StartInternationalCupsForNewSeason - reading afterwards finds
// the tier-2 champions at rank 1.
func TestInternationalCup_EntrantsAreTierOneChampionsNotPromotedTeams(t *testing.T) {
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

	type country struct {
		tier1, tier2, champion, runnerUp, promoted int64
	}
	// Two made-up countries: unknown codes fall back to the "europe" region (see
	// common.RegionForCountry), so they share one international cup.
	seed := func(code string) country {
		exec(`INSERT INTO countries (name) VALUES ($1) ON CONFLICT DO NOTHING`, code)
		var c country
		c.tier1 = scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, code)
		c.tier2 = scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L2', 2) RETURNING league_id`, code)
		newTeam := func(name string, leagueId int64, points int) int64 {
			teamId := scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, name, code, leagueId)
			exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 1, $3)`, leagueId, teamId, points)
			return teamId
		}
		c.champion = newTeam(code+" champion", c.tier1, 30)
		c.runnerUp = newTeam(code+" runner-up", c.tier1, 10)
		c.promoted = newTeam(code+" promoted", c.tier2, 40)
		return c
	}
	countries := []country{seed("zq"), seed("zr")}

	// What AdvanceSeasonIfComplete does, in order.
	champions, err := regionChampions(dbCtx)
	if err != nil {
		t.Fatalf("regionChampions: %s", err)
	}
	for _, c := range countries {
		if err := common.UpdateTeamLeague(dbCtx, c.promoted, c.tier1); err != nil {
			t.Fatalf("promoting: %s", err)
		}
		if err := common.UpdateClassificationLeague(dbCtx, c.promoted, c.tier1); err != nil {
			t.Fatalf("repointing classification: %s", err)
		}
	}

	// Premise of the bug: reading the standings now yields the promoted teams.
	late, err := regionChampions(dbCtx)
	if err != nil {
		t.Fatalf("regionChampions after promotion: %s", err)
	}
	for _, c := range countries {
		if !containsTeam(late["europe"], c.promoted) {
			t.Fatalf("test premise: after promotion %d should rank first in its new league, got %v", c.promoted, late["europe"])
		}
	}

	const season = 2
	StartInternationalCupsForNewSeason(dbCtx, season, champions)

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
	for _, c := range countries {
		if !entrants[c.champion] {
			t.Errorf("tier-1 champion %d is missing from the cup entrants %v", c.champion, entrants)
		}
		if entrants[c.promoted] {
			t.Errorf("promoted team %d must not be a cup entrant (entrants %v)", c.promoted, entrants)
		}
	}
}

func containsTeam(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
