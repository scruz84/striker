package common

import (
	"testing"
	"time"
)

// TestWorldCupGroups_StandingsWaitForTheBroadcast reproduces the leak found while fixing the
// national-teams page: world_cup_group_teams is updated when a group match is simulated, up to
// two hours before its broadcast ends, and the World Cup tab used to show it as is. The table
// must ignore a match still being broadcast (and show its live score, not the final one), then
// count it once the broadcast is over - matching the stored totals. Runs against a real Postgres
// (STRIKER_TEST_DB_DSN; skipped when unset).
func TestWorldCupGroups_StandingsWaitForTheBroadcast(t *testing.T) {
	db, dbCtx := openEngineLocksTestDb(t)
	scanId := func(query string, args ...any) int64 {
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

	for _, country := range []string{"wb1", "wb2"} {
		exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country)
	}
	spain, err := ensureNationalTeam(dbCtx, "wb1")
	if err != nil {
		t.Fatal(err)
	}
	portugal, err := ensureNationalTeam(dbCtx, "wb2")
	if err != nil {
		t.Fatal(err)
	}

	worldCupId := scanId(`INSERT INTO world_cups (season, name, stage, completed) VALUES (0, 'Broadcast World Cup', 'groups', true) RETURNING world_cup_id`)
	groupId := scanId(`INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, 'A') RETURNING group_id`, worldCupId)
	exec(`INSERT INTO world_cup_group_teams (group_id, team_id) VALUES ($1, $2), ($1, $3)`, groupId, spain, portugal)

	// Simulated 20 minutes ago: final 2-0, but at minute ~20 the broadcast still shows 0-0.
	kickoff := time.Now().Add(-20 * time.Minute).Truncate(time.Second)
	matchId := scanId(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, played, home_goals, away_goals) VALUES ($1, $2, $3, true, 2, 0) RETURNING match_id`,
		spain, portugal, kickoff)
	exec(`INSERT INTO friendly_events (match_id, event_type, minute, home_goals, away_goals) VALUES ($1, 0, 60, 1, 0), ($1, 0, 75, 2, 0)`, matchId)
	exec(`INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, 1, $2)`, groupId, matchId)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := UpdateWorldCupStandingsAfterMatchTx(tx, matchId, spain, portugal, 2, 0); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	groups, err := GetWorldCupGroups(dbCtx, worldCupId)
	if err != nil {
		t.Fatalf("GetWorldCupGroups: %s", err)
	}
	group := groups[0]
	for _, s := range group.Standings {
		if s.Played != 0 || s.GoalsFor != 0 || s.Won != 0 {
			t.Errorf("during the broadcast %s shows %+v, want no match counted", s.TeamName, s)
		}
	}
	if m := group.Matches[0]; m.Status != NationalMatchLive || *m.HomeGoals != 0 || *m.AwayGoals != 0 {
		t.Errorf("group match during the broadcast = status %s score %v-%v, want live 0-0", m.Status, *m.HomeGoals, *m.AwayGoals)
	}

	exec(`UPDATE friendly_matches SET match_date = $2 WHERE match_id = $1`, matchId, time.Now().Add(-LiveBroadcastWindow-time.Minute))
	groups, err = GetWorldCupGroups(dbCtx, worldCupId)
	if err != nil {
		t.Fatalf("GetWorldCupGroups: %s", err)
	}
	stored, err := getWorldCupGroupStandings(dbCtx, groupId)
	if err != nil {
		t.Fatal(err)
	}
	derived := groups[0].Standings
	if len(derived) != len(stored) {
		t.Fatalf("derived %d rows, stored %d", len(derived), len(stored))
	}
	for i := range stored {
		if derived[i] != stored[i] {
			t.Errorf("after the broadcast row %d derived %+v, stored %+v - want equal", i, derived[i], stored[i])
		}
	}
	if derived[0].TeamId != spain || derived[0].Points() != 3 || derived[0].GoalDifference() != 2 {
		t.Errorf("leader = %+v, want Spain with 3 points and +2", derived[0])
	}
	if m := groups[0].Matches[0]; m.Status != NationalMatchFinished || *m.HomeGoals != 2 {
		t.Errorf("group match after the broadcast = %+v, want finished 2-0", m)
	}

	history, err := GetWorldCupHistory(dbCtx)
	if err != nil {
		t.Fatalf("GetWorldCupHistory: %s", err)
	}
	if len(history) == 0 || history[0].WorldCupId != worldCupId {
		t.Errorf("history does not start with the newest World Cup %d: %+v", worldCupId, history)
	}
}
