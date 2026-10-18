package common

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

// TestGetUpcomingAndRecentNationalTeamFriendlies_ExcludeWorldCupGroupMatches reproduces the
// reported bug: a country's World Cup group stage is a real friendly_matches row under the hood
// (see engine.startWorldCupGroup), and its whole round-robin - every group opponent, several
// weeks apart, all created at once - used to show up on the national-teams page's "upcoming
// friendlies" list at the same time, looking exactly like the national-team double-booking bug
// this package already guards against elsewhere (see GetCountriesWithPendingNationalTeamFriendly)
// even though every fixture is legitimate. GetUpcomingNationalTeamFriendlies/
// GetRecentNationalTeamFriendlies must only surface a country's ordinary weekly friendlies; its
// World Cup group fixtures belong on the World Cup tab's own group cards instead (see
// GetWorldCupGroups). Runs against a real Postgres (STRIKER_TEST_DB_DSN, see
// TestEnsureSchema_RestartDoesNotTouchPlayerData; skipped when unset).
func TestGetUpcomingAndRecentNationalTeamFriendlies_ExcludeWorldCupGroupMatches(t *testing.T) {
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

	mustExec(`INSERT INTO countries (name) VALUES ('zr') ON CONFLICT DO NOTHING`)
	mustExec(`INSERT INTO countries (name) VALUES ('zq') ON CONFLICT DO NOTHING`)
	var teamZr, teamZq int64
	mustScan(&teamZr, `INSERT INTO teams (name, country, is_national_team) VALUES ('ZR', 'zr', true) RETURNING team_id`)
	mustScan(&teamZq, `INSERT INTO teams (name, country, is_national_team) VALUES ('ZQ', 'zq', true) RETURNING team_id`)

	now := time.Now().Truncate(time.Second)

	// An ordinary weekly friendly for zr, still pending - this is the only fixture either query
	// should ever return as upcoming.
	var ordinaryMatchId int64
	mustScan(&ordinaryMatchId, InsertNationalTeamFriendlyMatchStmtSql, teamZr, teamZq, now.Add(7*24*time.Hour))

	// A World Cup group fixture also involving zr, pending - linked into world_cup_group_matches
	// the same way engine.startWorldCupGroup does.
	var worldCupId, groupId, groupMatchId int64
	mustScan(&worldCupId, `INSERT INTO world_cups (season, name, stage) VALUES (4, 'FIFA World Cup', 'groups') RETURNING world_cup_id`)
	mustScan(&groupId, `INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, 'A') RETURNING group_id`, worldCupId)
	mustScan(&groupMatchId, InsertNationalTeamFriendlyMatchStmtSql, teamZr, teamZq, now.Add(14*24*time.Hour))
	mustExec(`INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, 1, $2)`, groupId, groupMatchId)

	upcoming, err := GetUpcomingNationalTeamFriendlies(dbCtx, 50)
	if err != nil {
		t.Fatalf("GetUpcomingNationalTeamFriendlies: %s", err)
	}
	for _, m := range upcoming {
		if m.MatchId == groupMatchId {
			t.Fatalf("GetUpcomingNationalTeamFriendlies returned World Cup group match %d alongside ordinary friendlies", groupMatchId)
		}
	}
	foundOrdinary := false
	for _, m := range upcoming {
		if m.MatchId == ordinaryMatchId {
			foundOrdinary = true
		}
	}
	if !foundOrdinary {
		t.Fatalf("GetUpcomingNationalTeamFriendlies did not return the ordinary friendly %d", ordinaryMatchId)
	}

	// Mark both played so GetRecentNationalTeamFriendlies has something to return, and check the
	// same exclusion holds there too.
	mustExec(`UPDATE friendly_matches SET played = true, home_goals = 1, away_goals = 0 WHERE match_id = ANY($1)`,
		[]int64{ordinaryMatchId, groupMatchId})

	recent, err := GetRecentNationalTeamFriendlies(dbCtx, 50)
	if err != nil {
		t.Fatalf("GetRecentNationalTeamFriendlies: %s", err)
	}
	for _, m := range recent {
		if m.MatchId == groupMatchId {
			t.Fatalf("GetRecentNationalTeamFriendlies returned World Cup group match %d alongside ordinary friendlies", groupMatchId)
		}
	}
	foundOrdinary = false
	for _, m := range recent {
		if m.MatchId == ordinaryMatchId {
			foundOrdinary = true
		}
	}
	if !foundOrdinary {
		t.Fatalf("GetRecentNationalTeamFriendlies did not return the ordinary friendly %d", ordinaryMatchId)
	}
}
