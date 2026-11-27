package common

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

// seasonHistoryWorld is a two-tier country at the start of season 2. Season 1 ended with U = A 1st,
// B 2nd, C 3rd (relegated) and L = D 1st (promoted), E 2nd, F 3rd; season 2 has U = A, B, D and
// L = C, E, F. Teams are inserted F first, so team ids run against the expected order.
type seasonHistoryWorld struct {
	country          string
	upper, lower     int64
	a, b, c, d, e, f int64
	exec             func(query string, args ...any)
	scanInt          func(query string, args ...any) sql.NullInt64
}

func seedSeasonHistoryWorld(t *testing.T, db *sql.DB) seasonHistoryWorld {
	t.Helper()
	w := seasonHistoryWorld{country: unusedTestCountryCode(t, db, 'y')}
	w.exec = func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	w.scanInt = func(query string, args ...any) sql.NullInt64 {
		t.Helper()
		var v sql.NullInt64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}
	id := func(query string, args ...any) int64 {
		t.Helper()
		v := w.scanInt(query, args...)
		return v.Int64
	}
	w.exec(`INSERT INTO countries (name, current_season) VALUES ($1, 2)`, w.country)
	w.upper = id(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'U', 1) RETURNING league_id`, w.country)
	w.lower = id(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L', 2) RETURNING league_id`, w.country)
	team := func(name string, league int64) int64 {
		return id(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, name, w.country, league)
	}
	w.f = team("F", w.lower)
	w.e = team("E", w.lower)
	w.c = team("C", w.lower)
	w.d = team("D", w.upper)
	w.b = team("B", w.upper)
	w.a = team("A", w.upper)

	history := func(teamId, league int64, name string, position int, promoted, relegated bool) {
		w.exec(`INSERT INTO season_history (team_id, season, league_id, league_name, final_position, promoted, relegated)
			VALUES ($1, 1, $2, $3, $4, $5, $6)`, teamId, league, name, position, promoted, relegated)
	}
	history(w.a, w.upper, "U", 1, false, false)
	history(w.b, w.upper, "U", 2, false, false)
	history(w.c, w.upper, "U", 3, false, true)
	history(w.d, w.lower, "L", 1, true, false)
	history(w.e, w.lower, "L", 2, false, false)
	history(w.f, w.lower, "L", 3, false, false)

	match := func(league, home, away int64, homeGoals, awayGoals int, played bool) {
		w.exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals, played)
			VALUES ($1, 1, 1, now() - interval '30 days', $2, $3, $4, $5, $6)`, league, home, away, homeGoals, awayGoals, played)
	}
	match(w.upper, w.a, w.b, 2, 0, true)
	match(w.upper, w.a, w.c, 1, 1, true)
	match(w.upper, w.b, w.c, 3, 1, true)
	match(w.upper, w.c, w.a, 5, 0, false) // never played: not counted
	match(w.lower, w.d, w.e, 4, 0, true)

	for _, row := range []struct{ league, teamId int64 }{
		{w.upper, w.a}, {w.upper, w.b}, {w.upper, w.d}, {w.lower, w.c}, {w.lower, w.e}, {w.lower, w.f},
	} {
		w.exec(`INSERT INTO classification (league_id, team_id, season) VALUES ($1, $2, 2)`, row.league, row.teamId)
	}
	return w
}

// TestLeagueSeasonHistoryMigration_Backfill covers league-page "Backfill" and "Deploy
// mid-preseason": init101 fills every history row's counters from its played league matches, and
// gives the season under way its starting order (relegated first, promoted last). Re-applying it
// changes nothing already set.
func TestLeagueSeasonHistoryMigration_Backfill(t *testing.T) {
	db, _ := openTransfersTestDB(t)
	w := seedSeasonHistoryWorld(t, db)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := applySchemaFile(ctx, conn, "history/init101_league_season_history.sql"); err != nil {
		t.Fatalf("applying migration: %s", err)
	}

	counters := func(teamId int64) string {
		var matches, points, wins, draws, losses, gs, gr sql.NullInt64
		if err := db.QueryRow(`SELECT matches, points, wins, draws, losses, goals_scored, goals_received
			FROM season_history WHERE team_id = $1 AND season = 1`, teamId).Scan(&matches, &points, &wins, &draws, &losses, &gs, &gr); err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf("%d %d %d %d %d %d-%d", matches.Int64, points.Int64, wins.Int64, draws.Int64, losses.Int64, gs.Int64, gr.Int64)
	}
	for _, want := range []struct {
		name   string
		teamId int64
		want   string
	}{
		{"A", w.a, "2 4 1 1 0 3-1"},
		{"B", w.b, "2 3 1 0 1 3-3"},
		{"C", w.c, "2 1 0 1 1 2-4"},
		{"D", w.d, "1 3 1 0 0 4-0"},
		{"E", w.e, "1 0 0 0 1 0-4"},
		{"F", w.f, "0 0 0 0 0 0-0"},
	} {
		if got := counters(want.teamId); got != want.want {
			t.Errorf("team %s counters = %s, want %s", want.name, got, want.want)
		}
	}

	seed := func(teamId int64) int64 {
		return w.scanInt(`SELECT seed FROM classification WHERE team_id = $1`, teamId).Int64
	}
	for _, want := range []struct {
		name   string
		teamId int64
		seed   int64
	}{
		{"A", w.a, 1}, {"B", w.b, 2}, {"D (promoted)", w.d, 3},
		{"C (relegated)", w.c, 1}, {"E", w.e, 2}, {"F", w.f, 3},
	} {
		if got := seed(want.teamId); got != want.seed {
			t.Errorf("team %s seed = %d, want %d", want.name, got, want.seed)
		}
	}

	// Re-applying (every startup does) must keep what is already set.
	w.exec(`UPDATE classification SET seed = 9 WHERE team_id = $1`, w.a)
	w.exec(`UPDATE season_history SET points = 77 WHERE team_id = $1 AND season = 1`, w.b)
	if err := applySchemaFile(ctx, conn, "history/init101_league_season_history.sql"); err != nil {
		t.Fatalf("re-applying migration: %s", err)
	}
	if got := seed(w.a); got != 9 {
		t.Errorf("after re-applying: team A seed = %d, want 9 (left as set)", got)
	}
	if got := w.scanInt(`SELECT points FROM season_history WHERE team_id = $1 AND season = 1`, w.b).Int64; got != 77 {
		t.Errorf("after re-applying: team B points = %d, want 77 (left as set)", got)
	}
}

// TestDivisionDetails_SeedBreaksExactTies covers league-page "Exact tie later": two teams level on
// points, goals, wins and draws are listed by their seed, in the table and in a team's position.
func TestDivisionDetails_SeedBreaksExactTies(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	w := seedSeasonHistoryWorld(t, db)
	// After matchday 10 of season 2 in U: A and B level on everything, D clearly ahead. A has the
	// lower seed but the higher team id.
	w.exec(`UPDATE classification SET matches = 10, points = 17, wins = 5, draws = 2, losses = 3, goals_scored = 15, goals_received = 12,
		seed = CASE team_id WHEN $1 THEN 1 WHEN $2 THEN 2 ELSE 3 END WHERE league_id = $3 AND team_id IN ($1, $2)`, w.a, w.b, w.upper)
	w.exec(`UPDATE classification SET matches = 10, points = 25, wins = 8, draws = 1, losses = 1, seed = 3 WHERE team_id = $1`, w.d)
	if w.a < w.b {
		t.Fatalf("test setup: A (%d) must have a higher team id than B (%d)", w.a, w.b)
	}

	details, err := GetDivisionDetails(dbCtx, w.upper)
	if err != nil {
		t.Fatalf("GetDivisionDetails: %s", err)
	}
	got := []int64{details.Classification.Positions[0].TeamId, details.Classification.Positions[1].TeamId, details.Classification.Positions[2].TeamId}
	if got[0] != w.d || got[1] != w.a || got[2] != w.b {
		t.Errorf("table order = %v, want D %d, A %d (seed 1), B %d (seed 2)", got, w.d, w.a, w.b)
	}
	if seed := details.Classification.Positions[1].Seed; seed == nil || *seed != 1 {
		t.Errorf("A's seed in the table = %v, want 1", seed)
	}
	if pos, err := GetTeamClassificationPosition(dbCtx, w.upper, 2, w.b); err != nil || pos != 3 {
		t.Errorf("GetTeamClassificationPosition(B) = %d, %v; want 3", pos, err)
	}
	if ranked, err := GetRankedClassification(dbCtx, w.upper); err != nil || len(ranked) != 3 || ranked[1].TeamId != w.a {
		t.Errorf("GetRankedClassification = %+v, %v; want A second", ranked, err)
	}
}
