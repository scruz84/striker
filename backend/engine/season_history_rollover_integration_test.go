package engine

import (
	"database/sql"
	"os"
	"testing"

	"github.com/scruz84/striker/common"
)

// TestAdvanceCountrySeasonIfComplete_StartingOrderAndFinalTables runs a two-tier country's real
// rollover against Postgres (STRIKER_TEST_DB_DSN; skipped when unset) and checks league-page
// "Starting order of a new season" (relegated teams open the lower division, promoted teams close
// the upper one, matchday 0 reads in that order) and "Final tables are kept" ("Rollover").
func TestAdvanceCountrySeasonIfComplete_StartingOrderAndFinalTables(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer db.Close()
	dbCtx := &common.DatabaseContext{Db: db, ObjectCache: common.NewRedisCache()}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}

	country := uniqueCountryCode()
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`, country)
	upper := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'U', 1) RETURNING league_id`, country)
	lower := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L', 2) RETURNING league_id`, country)

	// Final standings, best to worst. Teams are inserted worst first, so their ids run against the
	// expected order and only the seed can produce it.
	type line struct{ matches, points, wins, draws, losses, gs, gr int }
	upperLines := []line{{18, 40, 12, 4, 2, 35, 15}, {18, 36, 11, 3, 4, 30, 18}, {18, 33, 10, 3, 5, 29, 20}, {18, 31, 9, 4, 5, 28, 20}}
	lowerLines := []line{{18, 40, 12, 4, 2, 35, 15}, {18, 30, 9, 3, 6, 25, 22}, {18, 20, 6, 2, 10, 18, 30}, {18, 10, 3, 1, 14, 10, 40}}
	insertLeague := func(league int64, name string, lines []line) []int64 {
		ids := make([]int64, len(lines))
		for i := len(lines) - 1; i >= 0; i-- {
			ids[i] = scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, name, country, league)
			l := lines[i]
			exec(`INSERT INTO classification (league_id, team_id, season, matches, points, wins, draws, losses, goals_scored, goals_received)
				VALUES ($1, $2, 1, $3, $4, $5, $6, $7, $8, $9)`, league, ids[i], l.matches, l.points, l.wins, l.draws, l.losses, l.gs, l.gr)
		}
		return ids
	}
	u := insertLeague(upper, "U", upperLines)
	l := insertLeague(lower, "L", lowerLines)
	// One played match per league, well outside the live-broadcast window, completes the season.
	exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
		VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true), ($4, 1, 1, now() - interval '3 hours', $5, $6, true)`,
		upper, u[0], u[1], lower, l[0], l[1])

	// Whatever slot counts the shared test database is configured with: n teams swap places.
	config := common.GetPromotionRelegationConfig(dbCtx)
	n := min(config.RelegationSlotsPerGroup, config.PromotionSlotsPerGroup, len(u))
	if n < 1 {
		t.Skipf("promotion/relegation config %+v moves no team", config)
	}

	advanceCountrySeasonIfComplete(dbCtx, country)

	if season, err := common.GetCurrentSeasonForCountry(dbCtx, country); err != nil || season != 2 {
		t.Fatalf("GetCurrentSeasonForCountry: got %d, %v; want 2, nil", season, err)
	}

	// Upper: the teams that stayed in their final order, then the promoted ones in theirs.
	wantUpper := append(append([]int64{}, u[:len(u)-n]...), l[:n]...)
	// Lower: the relegated teams in their final order, then the teams that stayed.
	wantLower := append(append([]int64{}, u[len(u)-n:]...), l[n:]...)
	for _, league := range []struct {
		name string
		id   int64
		want []int64
	}{{"upper", upper, wantUpper}, {"lower", lower, wantLower}} {
		rows, err := db.Query(`SELECT team_id, seed FROM classification WHERE league_id = $1 ORDER BY seed`, league.id)
		if err != nil {
			t.Fatal(err)
		}
		got := make([]int64, 0)
		for rows.Next() {
			var teamId int64
			var seed sql.NullInt64
			if err := rows.Scan(&teamId, &seed); err != nil {
				t.Fatal(err)
			}
			if !seed.Valid || int(seed.Int64) != len(got)+1 {
				t.Errorf("%s league: team %d has seed %v, want %d", league.name, teamId, seed, len(got)+1)
			}
			got = append(got, teamId)
		}
		rows.Close()
		assertOrder(t, league.name+" seeds", got, league.want...)

		// Matchday 0: every team on 0 points, so the table reads in the starting order.
		details, err := common.GetDivisionDetails(dbCtx, league.id)
		if err != nil {
			t.Fatalf("GetDivisionDetails(%s): %s", league.name, err)
		}
		table := make([]int64, 0, len(league.want))
		for _, p := range details.Classification.Positions[:len(league.want)] {
			table = append(table, p.TeamId)
		}
		assertOrder(t, league.name+" matchday 0 table", table, league.want...)
	}

	// Rollover: the 4th-placed team's history row holds its whole final line.
	var position, matches, points, wins, draws, losses, gs, gr int
	var relegated bool
	if err := db.QueryRow(`SELECT final_position, matches, points, wins, draws, losses, goals_scored, goals_received, relegated
		FROM season_history WHERE team_id = $1 AND season = 1`, u[3]).Scan(&position, &matches, &points, &wins, &draws, &losses, &gs, &gr, &relegated); err != nil {
		t.Fatalf("reading season history: %s", err)
	}
	if position != 4 || matches != 18 || points != 31 || wins != 9 || draws != 4 || losses != 5 || gs != 28 || gr != 20 || !relegated {
		t.Errorf("4th-placed team's history = pos %d, %d m, %d pts, %d-%d-%d, %d-%d, relegated %v; want 4, 18, 31, 9-4-5, 28-20, relegated",
			position, matches, points, wins, draws, losses, gs, gr, relegated)
	}
}
