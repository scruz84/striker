package common

import "testing"

// TestOpenPlayKeeper_RoundTrips runs against a real Postgres (STRIKER_TEST_DB_DSN, see
// openTransfersTestDB; skipped when unset). openspec match-simulation "Defending keeper on every
// chance": a cup tie and a friendly store the defending keeper of an open-play chance and return
// it with their events, like a league match; a row written without the init108 columns (every
// cup/friendly row from before it) reads back with no keeper.
func TestOpenPlayKeeper_RoundTrips(t *testing.T) {
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

	var leagueId, teamA, teamB, matchId, cupId, tieId, oldTieId, friendlyId, oldFriendlyId int64
	mustExec(`INSERT INTO countries (name, current_season) VALUES ('zkp1', 1) ON CONFLICT (name) DO NOTHING`)
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zkp1', 1, 'L') RETURNING league_id`)
	mustScan(&teamA, `INSERT INTO teams (name, country) VALUES ('KeeperA', 'zkp1') RETURNING team_id`)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('KeeperB', 'zkp1') RETURNING team_id`)
	mustScan(&matchId, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, false, now(), $2, $3) RETURNING match_id`, leagueId, teamA, teamB)
	mustScan(&cupId, InsertCupStmtSql, "zkp1", 1, "Cup")
	mustScan(&tieId, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3) RETURNING tie_id`,
		cupId, teamA, teamB)
	mustScan(&oldTieId, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, played) VALUES ($1, 1, 2, $2, $3, true) RETURNING tie_id`,
		cupId, teamA, teamB)
	mustScan(&friendlyId, InsertNationalTeamFriendlyMatchStmtSql, teamA, teamB, "2026-01-01T00:00:00Z")
	mustScan(&oldFriendlyId, InsertNationalTeamFriendlyMatchStmtSql, teamA, teamB, "2026-01-02T00:00:00Z")

	// An away open-play miss saved by the home keeper (id 501, "Ruiz"), then a home open-play goal
	// past the away keeper (id 602, "Lopez").
	events := []Event{
		{EventType: NO_GOAL, Minute: 12, TeamOwner: teamB, TeamOwnerPlayers: []int64{8}, TeamOwnerPlayersNames: []string{"T"},
			OtherTeam: teamA, OtherTeamPlayers: []int64{501}, OtherTeamPlayersNames: []string{"Ruiz"}},
		{EventType: GOAL, Minute: 30, TeamOwner: teamA, TeamOwnerPlayers: []int64{7}, TeamOwnerPlayersNames: []string{"S"},
			OtherTeam: teamB, OtherTeamPlayers: []int64{602}, OtherTeamPlayersNames: []string{"Lopez"}, HomeGoals: 1},
	}
	result := MatchResult{HomeGoals: 1, Events: events, Instants: []MatchInstant{{Minute: 12}, {Minute: 30}}}

	checkKeepers := func(kind string, got []Event) {
		t.Helper()
		if len(got) != 2 {
			t.Fatalf("%s: want 2 events, got %+v", kind, got)
		}
		want := []struct {
			id   int64
			name string
		}{{501, "Ruiz"}, {602, "Lopez"}}
		for i, w := range want {
			e := got[i]
			if len(e.OtherTeamPlayers) != 1 || e.OtherTeamPlayers[0] != w.id ||
				len(e.OtherTeamPlayersNames) != 1 || e.OtherTeamPlayersNames[0] != w.name {
				t.Errorf("%s: event %d should name keeper %d %q, got %+v", kind, i, w.id, w.name, e)
			}
		}
	}

	if err := SaveLeagueMatchResultTx(dbCtx, matchId, leagueId, teamA, teamB, result); err != nil {
		t.Fatalf("SaveLeagueMatchResultTx: %s", err)
	}
	checkKeepers("league", *GetMatchEvents(dbCtx, matchId))

	if err := SaveCupTieResultTx(dbCtx, tieId, false, teamA, result); err != nil {
		t.Fatalf("SaveCupTieResultTx: %s", err)
	}
	cupEvents, err := GetCupTieEvents(dbCtx, tieId)
	if err != nil {
		t.Fatalf("GetCupTieEvents: %s", err)
	}
	checkKeepers("cup", cupEvents)

	if err := SaveFriendlyResultTx(dbCtx, friendlyId, teamA, teamB, result); err != nil {
		t.Fatalf("SaveFriendlyResultTx: %s", err)
	}
	checkKeepers("friendly", *GetFriendlyEvents(dbCtx, friendlyId))

	// Rows stored before init108: the other-team columns are never written.
	mustExec(`INSERT INTO cup_tie_events (tie_id, event_type, team_owner, other_team, minute, team_players, team_player_names)
		VALUES ($1, $2, $3, $4, 20, '[8]', '["T"]')`, oldTieId, NO_GOAL, teamB, teamA)
	mustExec(`INSERT INTO friendly_events (match_id, event_type, team_owner, other_team, minute, team_players, team_player_names)
		VALUES ($1, $2, $3, $4, 20, '[8]', '["T"]')`, oldFriendlyId, NO_GOAL, teamB, teamA)
	oldCup, err := GetCupTieEvents(dbCtx, oldTieId)
	if err != nil {
		t.Fatalf("GetCupTieEvents (old row): %s", err)
	}
	oldFriendly := *GetFriendlyEvents(dbCtx, oldFriendlyId)
	for kind, got := range map[string][]Event{"old cup": oldCup, "old friendly": oldFriendly} {
		if len(got) != 1 || got[0].OtherTeamPlayers != nil || got[0].OtherTeamPlayersNames != nil || got[0].TeamOwnerPlayers[0] != 8 {
			t.Errorf("%s: want the event with no keeper, got %+v", kind, got)
		}
	}
}
