package common

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestGetTopScorersFinal_CountsLiveWindowMatches runs against a real Postgres (STRIKER_TEST_DB_DSN,
// see TestEnsureSchema_RestartDoesNotTouchPlayerData; skipped when unset). The season's last match
// is simulated but still inside its live-broadcast window - exactly where the season closes - and
// only its goals put player B ahead of player A: the public leaderboard must still hide it, while
// the season-award variant must count it.
func TestGetTopScorersFinal_CountsLiveWindowMatches(t *testing.T) {
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

	var leagueId, teamA, teamB, playerA, playerB, oldMatch, liveMatch int64
	mustExec(`INSERT INTO countries (name) VALUES ('zy') ON CONFLICT DO NOTHING`)
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zy', 1, 'L') RETURNING league_id`)
	mustScan(&teamA, `INSERT INTO teams (name, country) VALUES ('A', 'zy') RETURNING team_id`)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('B', 'zy') RETURNING team_id`)
	mustScan(&playerA, `INSERT INTO players (birth_day, team_id, first_name, last_name) VALUES (0, $1, 'A', 'A') RETURNING player_id`, teamA)
	mustScan(&playerB, `INSERT INTO players (birth_day, team_id, first_name, last_name) VALUES (0, $1, 'B', 'B') RETURNING player_id`, teamB)

	const season = 1
	insertMatch := func(dest *int64, date time.Time) {
		mustScan(dest, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
			VALUES ($1, $2, true, $3, $4, $5) RETURNING match_id`, leagueId, season, date, teamA, teamB)
	}
	// An older match, then the last round: simulated 15 minutes before its match_date (see
	// GetSimulableMatches), so the season has closed while it is still live.
	insertMatch(&oldMatch, time.Now().Add(-24*time.Hour))
	insertMatch(&liveMatch, time.Now().Add(15*time.Minute))

	goal := func(matchId, playerId int64) {
		mustExec(`INSERT INTO match_events (match_id, event_type, team_players) VALUES ($1, $2, $3)`,
			matchId, GOAL, fmt.Sprintf("[%d]", playerId))
	}
	goal(oldMatch, playerA)
	goal(oldMatch, playerA)
	goal(oldMatch, playerB)
	goal(liveMatch, playerB)
	goal(liveMatch, playerB)

	public, err := GetTopScorers(dbCtx, leagueId, season, 1)
	if err != nil || len(public) != 1 {
		t.Fatalf("GetTopScorers: %v, %v", public, err)
	}
	if public[0].PlayerId != playerA || public[0].Goals != 2 {
		t.Errorf("public leaderboard: got player %d with %d goals, want player %d with 2 (live match hidden)",
			public[0].PlayerId, public[0].Goals, playerA)
	}

	final, err := GetTopScorersFinal(dbCtx, leagueId, season, 1)
	if err != nil || len(final) != 1 {
		t.Fatalf("GetTopScorersFinal: %v, %v", final, err)
	}
	if final[0].PlayerId != playerB || final[0].Goals != 3 {
		t.Errorf("final leaderboard: got player %d with %d goals, want player %d with 3",
			final[0].PlayerId, final[0].Goals, playerB)
	}
}
