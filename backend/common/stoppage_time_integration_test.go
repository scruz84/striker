package common

import (
	"testing"
	"time"
)

// TestStoppageTime_RoundTrips runs against a real Postgres (STRIKER_TEST_DB_DSN, see
// openTransfersTestDB; skipped when unset). A league match, a cup tie and a friendly saved with
// added time store both halves' added minutes on the fixture, and read back their events and
// instants in match order - 45, 45+1, 45+2, 46 - with an added-time goal and the ADDED_TIME board.
// A row written without the new columns (every row from before init107_stoppage_time.sql) reads
// back as added minute 0 with no board value.
func TestStoppageTime_RoundTrips(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)

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
	addedTimes := func(table, idColumn string, id int64) (int64, int64) {
		t.Helper()
		var first, second int64
		if err := db.QueryRow("SELECT first_half_added_time, second_half_added_time FROM "+table+" WHERE "+idColumn+" = $1", id).Scan(&first, &second); err != nil {
			t.Fatalf("reading %s added time: %s", table, err)
		}
		return first, second
	}

	var leagueId, teamA, teamB, matchId, cupId, tieId, friendlyId, oldMatchId int64
	mustExec(`INSERT INTO countries (name, current_season) VALUES ('zst1', 1) ON CONFLICT (name) DO NOTHING`)
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zst1', 1, 'L') RETURNING league_id`)
	mustScan(&teamA, `INSERT INTO teams (name, country) VALUES ('StoppageA', 'zst1') RETURNING team_id`)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('StoppageB', 'zst1') RETURNING team_id`)
	mustScan(&matchId, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, false, now(), $2, $3) RETURNING match_id`, leagueId, teamA, teamB)
	mustScan(&oldMatchId, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, true, now(), $2, $3) RETURNING match_id`, leagueId, teamA, teamB)
	mustScan(&cupId, InsertCupStmtSql, "zst1", 1, "Cup")
	mustScan(&tieId, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3) RETURNING tie_id`,
		cupId, teamA, teamB)
	mustScan(&friendlyId, InsertNationalTeamFriendlyMatchStmtSql, teamA, teamB, "2026-01-01T00:00:00Z")

	two := 2
	// Stored in simulation order: the board at 45 comes after minute 45's goal, then the goal at 45+1.
	events := []Event{
		{EventType: GOAL, Minute: 45, TeamOwner: teamA, TeamOwnerPlayers: []int64{7}, TeamOwnerPlayersNames: []string{"S"}, OtherTeam: teamB, HomeGoals: 1},
		{EventType: ADDED_TIME, Minute: 45, TeamOwner: teamA, OtherTeam: teamB, HomeGoals: 1, AddedTime: &two},
		{EventType: GOAL, Minute: 45, AddedMinute: 1, TeamOwner: teamB, TeamOwnerPlayers: []int64{8}, TeamOwnerPlayersNames: []string{"T"}, OtherTeam: teamA, HomeGoals: 1, AwayGoals: 1},
		{EventType: NO_GOAL, Minute: 46, TeamOwner: teamA, TeamOwnerPlayers: []int64{7}, TeamOwnerPlayersNames: []string{"S"}, OtherTeam: teamB, HomeGoals: 1, AwayGoals: 1},
	}
	instants := []MatchInstant{{Minute: 45}, {Minute: 45, AddedMinute: 1}, {Minute: 45, AddedMinute: 2}, {Minute: 46}}
	result := MatchResult{HomeGoals: 1, AwayGoals: 1, Events: events, Instants: instants, FirstHalfAddedTime: 2, SecondHalfAddedTime: 4}

	checkEvents := func(kind string, got []Event) {
		t.Helper()
		if len(got) != 4 {
			t.Fatalf("%s: want 4 events, got %+v", kind, got)
		}
		if got[0].EventType != GOAL || got[0].AddedMinute != 0 || got[0].AddedTime != nil {
			t.Errorf("%s: event 0 should be the regulation goal at 45, got %+v", kind, got[0])
		}
		if got[1].EventType != ADDED_TIME || got[1].AddedTime == nil || *got[1].AddedTime != 2 {
			t.Errorf("%s: event 1 should be the board announcing 2 minutes, got %+v", kind, got[1])
		}
		if got[2].EventType != GOAL || got[2].Minute != 45 || got[2].AddedMinute != 1 {
			t.Errorf("%s: event 2 should be the goal at 45+1, got %+v", kind, got[2])
		}
		if got[3].Minute != 46 || got[3].AddedMinute != 0 {
			t.Errorf("%s: event 3 should be at 46, got %+v", kind, got[3])
		}
	}
	checkInstants := func(kind string, got []MatchInstant) {
		t.Helper()
		want := [][2]int{{45, 0}, {45, 1}, {45, 2}, {46, 0}}
		if len(got) != len(want) {
			t.Fatalf("%s: want %d instants, got %+v", kind, len(want), got)
		}
		for i, w := range want {
			if got[i].Minute != w[0] || got[i].AddedMinute != w[1] {
				t.Errorf("%s: instant %d = %d+%d, want %d+%d", kind, i, got[i].Minute, got[i].AddedMinute, w[0], w[1])
			}
		}
	}

	if err := SaveLeagueMatchResultTx(dbCtx, matchId, leagueId, teamA, teamB, result); err != nil {
		t.Fatalf("SaveLeagueMatchResultTx: %s", err)
	}
	checkEvents("league", *GetMatchEvents(dbCtx, matchId))
	checkInstants("league", *GetMatchInstants(dbCtx, matchId))
	if first, second := addedTimes("matches", "match_id", matchId); first != 2 || second != 4 {
		t.Errorf("league added time: got %d and %d, want 2 and 4", first, second)
	}

	if err := SaveCupTieResultTx(dbCtx, tieId, false, teamA, result); err != nil {
		t.Fatalf("SaveCupTieResultTx: %s", err)
	}
	cupEvents, err := GetCupTieEvents(dbCtx, tieId)
	if err != nil {
		t.Fatalf("GetCupTieEvents: %s", err)
	}
	checkEvents("cup", cupEvents)
	cupInstants, err := GetCupTieInstants(dbCtx, tieId)
	if err != nil {
		t.Fatalf("GetCupTieInstants: %s", err)
	}
	checkInstants("cup", *cupInstants)
	if first, second := addedTimes("cup_ties", "tie_id", tieId); first != 2 || second != 4 {
		t.Errorf("cup added time: got %d and %d, want 2 and 4", first, second)
	}

	if err := SaveFriendlyResultTx(dbCtx, friendlyId, teamA, teamB, result); err != nil {
		t.Fatalf("SaveFriendlyResultTx: %s", err)
	}
	checkEvents("friendly", *GetFriendlyEvents(dbCtx, friendlyId))
	checkInstants("friendly", *GetFriendlyInstants(dbCtx, friendlyId))
	if first, second := addedTimes("friendly_matches", "match_id", friendlyId); first != 2 || second != 4 {
		t.Errorf("friendly added time: got %d and %d, want 2 and 4", first, second)
	}

	// A match stored before this change: no added-time columns written.
	mustExec(`INSERT INTO match_events (match_id, event_type, team_owner, other_team, minute, team_players) VALUES ($1, $2, $3, $4, 90, '[7]')`,
		oldMatchId, GOAL, teamA, teamB)
	mustExec(`INSERT INTO match_instants (match_id, minute) VALUES ($1, 90)`, oldMatchId)
	old := *GetMatchEvents(dbCtx, oldMatchId)
	if len(old) != 1 || old[0].AddedMinute != 0 || old[0].AddedTime != nil {
		t.Errorf("old event: want added minute 0 and no board, got %+v", old)
	}
	if oldInstants := *GetMatchInstants(dbCtx, oldMatchId); len(oldInstants) != 1 || oldInstants[0].AddedMinute != 0 {
		t.Errorf("old instant: want added minute 0, got %+v", oldInstants)
	}
	if first, second := addedTimes("matches", "match_id", oldMatchId); first != 0 || second != 0 {
		t.Errorf("old match added time: got %d and %d, want 0", first, second)
	}
}

// TestStoppageTime_LongestMatchStaysHiddenUntilTheWindowCloses (stoppage-time "Results stay hidden
// until the final whistle", tasks 3.5) runs against a real Postgres (STRIKER_TEST_DB_DSN). The
// time-gated readers rely on LiveBroadcastWindow, never on a 90-minute match: a league match won at
// 90+5, read 108 minutes after kickoff (its final whistle), is still left out of the head-to-head
// and the top scorers, and its post-match effects (injuries, skill growth, the result notification)
// are not due for the recovery job - only the replay's own completion applies them. Once kickoff is
// more than LiveBroadcastWindow ago it counts everywhere.
func TestStoppageTime_LongestMatchStaysHiddenUntilTheWindowCloses(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
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

	var leagueId, teamA, teamB, scorer, matchId int64
	mustExec(`INSERT INTO countries (name, current_season) VALUES ('zst2', 1) ON CONFLICT (name) DO NOTHING`)
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zst2', 1, 'L') RETURNING league_id`)
	mustScan(&teamA, `INSERT INTO teams (name, country) VALUES ('LateA', 'zst2') RETURNING team_id`)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('LateB', 'zst2') RETURNING team_id`)
	mustScan(&scorer, `INSERT INTO players (birth_day, team_id, first_name, last_name) VALUES (0, $1, 'Late', 'Winner') RETURNING player_id`, teamA)
	const season = 1
	kickoff := time.Now().Add(-108 * time.Minute)
	mustScan(&matchId, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
		VALUES ($1, $2, false, $3, $4, $5) RETURNING match_id`, leagueId, season, kickoff, teamA, teamB)

	five := 5
	winner := Event{EventType: GOAL, Minute: 90, AddedMinute: 5, TeamOwner: teamA, TeamOwnerPlayers: []int64{scorer},
		TeamOwnerPlayersNames: []string{"Late Winner"}, OtherTeam: teamB, HomeGoals: 1}
	board := Event{EventType: ADDED_TIME, Minute: 90, TeamOwner: teamA, OtherTeam: teamB, AddedTime: &five}
	result := MatchResult{HomeGoals: 1, Events: []Event{board, winner}, FirstHalfAddedTime: 2, SecondHalfAddedTime: 5,
		Effects: &PostMatchPayload{Home: teamA, Away: teamB, HomeGoals: 1}}
	if err := SaveLeagueMatchResultTx(dbCtx, matchId, leagueId, teamA, teamB, result); err != nil {
		t.Fatalf("SaveLeagueMatchResultTx: %s", err)
	}

	visible := func() (meetings int, goals int) {
		t.Helper()
		h2h, err := GetHeadToHead(dbCtx, teamA, teamB)
		if err != nil {
			t.Fatalf("GetHeadToHead: %s", err)
		}
		scorers, err := GetTopScorers(dbCtx, leagueId, season, 10)
		if err != nil {
			t.Fatalf("GetTopScorers: %s", err)
		}
		for _, s := range scorers {
			if s.PlayerId == scorer {
				goals = s.Goals
			}
		}
		return h2h.TotalMatches, goals
	}

	if meetings, goals := visible(); meetings != 0 || goals != 0 {
		t.Errorf("at the final whistle (kickoff + 108m): head-to-head has %d meetings and the scorer %d goals, want both hidden", meetings, goals)
	}
	var dueAt time.Time
	if err := db.QueryRow(`SELECT due_at FROM pending_post_match_effects WHERE kind = $1 AND match_id = $2`, SimulationKindLeague, matchId).Scan(&dueAt); err != nil {
		t.Fatalf("reading the pending effects: %s", err)
	}
	if gap := dueAt.Sub(kickoff) - LiveBroadcastWindow; !dueAt.After(time.Now()) || gap > time.Millisecond || gap < -time.Millisecond {
		t.Errorf("post-match effects due at %s, want kickoff + %s, still in the future", dueAt, LiveBroadcastWindow)
	}

	// Time passes: the match and its player stats (which carry the kickoff) move into the past.
	mustExec(`UPDATE matches SET match_date = $1 WHERE match_id = $2`, time.Now().Add(-LiveBroadcastWindow-time.Minute), matchId)
	mustExec(`UPDATE player_match_stats SET kickoff = $1 WHERE kind = 'league' AND match_id = $2`, time.Now().Add(-LiveBroadcastWindow-time.Minute), matchId)
	if meetings, goals := visible(); meetings != 1 || goals != 1 {
		t.Errorf("after the live-broadcast window: head-to-head has %d meetings and the scorer %d goals, want 1 and 1", meetings, goals)
	}
}
