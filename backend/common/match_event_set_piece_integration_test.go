package common

import "testing"

// TestMatchEventSetPiece_RoundTrips runs against a real Postgres (STRIKER_TEST_DB_DSN, see
// openTransfersTestDB; skipped when unset). A set-piece event written for a league match, a cup
// tie and a friendly reads back with its set_piece kind, and an event inserted without the column
// (as every row from before init82_match_event_set_piece.sql was) reads back as open play.
func TestMatchEventSetPiece_RoundTrips(t *testing.T) {
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

	var leagueId, teamA, teamB, matchId, cupId, tieId, friendlyId int64
	mustExec(`INSERT INTO countries (name, current_season) VALUES ('zsp1', 1) ON CONFLICT (name) DO NOTHING`)
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zsp1', 1, 'L') RETURNING league_id`)
	mustScan(&teamA, `INSERT INTO teams (name, country) VALUES ('SetPieceA', 'zsp1') RETURNING team_id`)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('SetPieceB', 'zsp1') RETURNING team_id`)
	mustScan(&matchId, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, true, now(), $2, $3) RETURNING match_id`, leagueId, teamA, teamB)
	mustScan(&cupId, InsertCupStmtSql, "zsp1", 1, "Cup")
	mustScan(&tieId, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3) RETURNING tie_id`,
		cupId, teamA, teamB)
	mustScan(&friendlyId, InsertNationalTeamFriendlyMatchStmtSql, teamA, teamB, "2026-01-01T00:00:00Z")

	penalty := Event{EventType: GOAL, Minute: 16, TeamOwner: teamA, TeamOwnerPlayers: []int64{7}, TeamOwnerPlayersNames: []string{"Taker"},
		OtherTeam: teamB, OtherTeamPlayers: []int64{1}, OtherTeamPlayersNames: []string{"Keeper"}, HomeGoals: 1, SetPiece: SET_PIECE_PENALTY}

	// League: written the way engine.UpdatePlayedMatch does.
	mustExec(InsertMatchEventStmtSql, matchId, penalty.EventType, penalty.TeamOwner, penalty.OtherTeam, penalty.Minute,
		`[7]`, `["Taker"]`, `[1]`, `["Keeper"]`, penalty.HomeGoals, penalty.AwayGoals, penalty.SetPiece)
	mustExec(`INSERT INTO match_events (match_id, event_type, team_owner, other_team, minute, team_players)
		VALUES ($1, $2, $3, $4, 40, '[8]')`, matchId, NO_GOAL, teamA, teamB)
	events := *GetMatchEvents(dbCtx, matchId)
	if len(events) != 2 || events[0].SetPiece != SET_PIECE_PENALTY || events[1].SetPiece != SET_PIECE_NONE {
		t.Fatalf("league events: want penalty then open play, got %+v", events)
	}
	if len(events[0].OtherTeamPlayers) != 1 || events[0].OtherTeamPlayers[0] != 1 {
		t.Errorf("league penalty should name the keeper, got %+v", events[0].OtherTeamPlayers)
	}

	// Cup tie.
	freeKick := penalty
	freeKick.SetPiece = SET_PIECE_DIRECT_FREE_KICK
	if err := SetCupTiePlayed(dbCtx, tieId, 1, 0, false, teamA, []Event{freeKick}, nil); err != nil {
		t.Fatalf("SetCupTiePlayed: %s", err)
	}
	cupEvents, err := GetCupTieEvents(dbCtx, tieId)
	if err != nil || len(cupEvents) != 1 || cupEvents[0].SetPiece != SET_PIECE_DIRECT_FREE_KICK {
		t.Fatalf("cup events: %+v, %v", cupEvents, err)
	}

	// Friendly.
	if err := SetFriendlyPlayed(dbCtx, friendlyId, 1, 0, []Event{penalty}, nil); err != nil {
		t.Fatalf("SetFriendlyPlayed: %s", err)
	}
	friendlyEvents := *GetFriendlyEvents(dbCtx, friendlyId)
	if len(friendlyEvents) != 1 || friendlyEvents[0].SetPiece != SET_PIECE_PENALTY {
		t.Fatalf("friendly events: %+v", friendlyEvents)
	}
}
