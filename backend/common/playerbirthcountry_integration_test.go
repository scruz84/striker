package common

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestSummaryRowsCarryBirthCountry runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when
// unset). The UI shows a player's flag in the top scorers, search results and latest evolutions,
// so each of those rows must come back with the player's birth_country - and an empty one for a
// legacy player who has none.
func TestSummaryRowsCarryBirthCountry(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer db.Close()
	dbCtx := &DatabaseContext{Db: db}
	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	mustScan := func(dest *int64, query string, args ...any) {
		t.Helper()
		if err := db.QueryRow(query, args...).Scan(dest); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}

	// The database persists across runs, so search for a surname no earlier run used.
	surname := fmt.Sprintf("Zq%d", time.Now().UnixNano())
	var leagueId, teamId, withCountry, withoutCountry, matchId int64
	mustExec(`INSERT INTO countries (name) VALUES ('zf') ON CONFLICT DO NOTHING`)
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zf', 1, 'L') RETURNING league_id`)
	mustScan(&teamId, `INSERT INTO teams (name, country) VALUES ('Flag FC', 'zf') RETURNING team_id`)
	// Complete players, as the game generates them: 'zf' stays an active country in this shared
	// database, so later tests (national squad selection, the national radar) read this player.
	const insertPlayerSql = `INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield,
		dribbling, shot, "header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country, shirt_number)
		VALUES ($1, $2, $3, 0, 10, 40, 40, 40, 40, 40, 40, false, false, false, false, '', 1000, 25, 2, NULLIF($4, ''), $5) RETURNING player_id`
	mustScan(&withCountry, insertPlayerSql, teamId, "Flaga", surname, "zf", 1)
	mustScan(&withoutCountry, insertPlayerSql, teamId, "Flagb", surname, "", 2)

	const season = 1
	mustScan(&matchId, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
		VALUES ($1, $2, true, $3, $4, $4) RETURNING match_id`, leagueId, season, time.Now().Add(-24*time.Hour), teamId)
	for _, playerId := range []int64{withCountry, withCountry, withoutCountry} {
		mustExec(`INSERT INTO match_events (match_id, event_type, team_players) VALUES ($1, $2, $3)`,
			matchId, GOAL, fmt.Sprintf("[%d]", playerId))
	}

	scorers, err := GetTopScorers(dbCtx, leagueId, season, 0)
	if err != nil || len(scorers) != 2 {
		t.Fatalf("GetTopScorers: %v, %v", scorers, err)
	}
	if scorers[0].PlayerId != withCountry || scorers[0].BirthCountry != "zf" {
		t.Errorf("top scorer: got player %d with country %q, want %d with \"zf\"", scorers[0].PlayerId, scorers[0].BirthCountry, withCountry)
	}
	if scorers[1].BirthCountry != "" {
		t.Errorf("legacy scorer: got country %q, want empty", scorers[1].BirthCountry)
	}

	results, err := Search(dbCtx, surname)
	if err != nil || len(results.Players) != 2 {
		t.Fatalf("Search: %v, %v", results, err)
	}
	for _, p := range results.Players {
		want := ""
		if p.Id == withCountry {
			want = "zf"
		}
		if p.BirthCountry != want {
			t.Errorf("search result %d: got country %q, want %q", p.Id, p.BirthCountry, want)
		}
	}

	for _, playerId := range []int64{withCountry, withoutCountry} {
		if _, err := LogPlayerProgression(dbCtx, playerId, season, 20, 21, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2); err != nil {
			t.Fatalf("LogPlayerProgression: %s", err)
		}
	}
	evolutions, err := GetLatestPlayerEvolutionsForTeam(dbCtx, teamId)
	if err != nil || len(evolutions) != 2 {
		t.Fatalf("GetLatestPlayerEvolutionsForTeam: %v, %v", evolutions, err)
	}
	for _, e := range evolutions {
		want := ""
		if e.PlayerId == withCountry {
			want = "zf"
		}
		if e.BirthCountry != want {
			t.Errorf("evolution %d: got country %q, want %q", e.PlayerId, e.BirthCountry, want)
		}
	}
}
