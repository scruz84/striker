package common

import "testing"

// TestMatchViewerPayloads_CupTieAndFriendly runs against a real Postgres (STRIKER_TEST_DB_DSN, see
// openTransfersTestDB; skipped when unset). It pins what the shared match viewer reads: a played
// cup tie comes back with its instants, flavor and home stadium, an unplayed one with none of the
// played data, and a friendly carries its flavor and - only for a World Cup group fixture - its
// group name and matchday.
func TestMatchViewerPayloads_CupTieAndFriendly(t *testing.T) {
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

	var leagueId, teamA, teamB, cupId, playedTie, openTie, groupFriendly, plainFriendly, worldCupId, groupId int64
	mustExec(`INSERT INTO countries (name, current_season) VALUES ('zmv', 1) ON CONFLICT (name) DO NOTHING`)
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zmv', 1, 'L') RETURNING league_id`)
	mustScan(&teamA, `INSERT INTO teams (name, country, stadium_name) VALUES ('ViewerA', 'zmv', 'Estadio Norte') RETURNING team_id`)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('ViewerB', 'zmv') RETURNING team_id`)
	mustScan(&cupId, InsertCupStmtSql, "zmv", 1, "Cup")
	mustScan(&playedTie, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3) RETURNING tie_id`, cupId, teamA, teamB)
	mustScan(&openTie, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 2, $2, $3) RETURNING tie_id`, cupId, teamA, teamB)

	// A played tie: stored the way engine.simulateCupTie does (common.SaveCupTieResultTx).
	instants := []MatchInstant{{Minute: 1, HomeQua: TeamQualification{Midfield: 12.5}}, {Minute: 2, HomeGoals: 1}}
	if err := SaveCupTieResultTx(dbCtx, playedTie, false, teamA, MatchResult{HomeGoals: 1, Instants: instants}); err != nil {
		t.Fatalf("SaveCupTieResultTx: %s", err)
	}
	if err := UpdateCupTieFlavor(dbCtx, playedTie, "rain", 8412, "Ref One", "Assistant A", "Assistant B"); err != nil {
		t.Fatalf("UpdateCupTieFlavor: %s", err)
	}
	// The flavor of a tie that isn't played yet must not be exposed.
	if err := UpdateCupTieFlavor(dbCtx, openTie, "sunny", 100, "Ref Two", "Assistant C", "Assistant D"); err != nil {
		t.Fatalf("UpdateCupTieFlavor (open tie): %s", err)
	}

	details, err := GetCupTieDetails(dbCtx, playedTie)
	if err != nil {
		t.Fatalf("GetCupTieDetails: %s", err)
	}
	if details.MatchInstants == nil || len(*details.MatchInstants) != 2 || (*details.MatchInstants)[0].HomeQua.Midfield != 12.5 ||
		(*details.MatchInstants)[1].HomeGoals != 1 {
		t.Errorf("played tie instants: got %+v", details.MatchInstants)
	}
	if details.Weather != "rain" || details.Attendance != 8412 || details.RefereeName != "Ref One" ||
		details.Assistant1Name != "Assistant A" || details.Assistant2Name != "Assistant B" || details.HomeStadiumName != "Estadio Norte" {
		t.Errorf("played tie flavor: got %+v", details)
	}

	open, err := GetCupTieDetails(dbCtx, openTie)
	if err != nil {
		t.Fatalf("GetCupTieDetails (open tie): %s", err)
	}
	if open.MatchInstants != nil || open.Weather != "" || open.Attendance != 0 || open.RefereeName != "" {
		t.Errorf("an unplayed tie must not expose played data, got %+v", open)
	}
	if open.HomeStadiumName != "Estadio Norte" {
		t.Errorf("the stadium name is shown before kickoff too, got %q", open.HomeStadiumName)
	}

	// Friendlies: one linked to a World Cup group, one ordinary.
	mustScan(&groupFriendly, InsertNationalTeamFriendlyMatchStmtSql, teamA, teamB, "2026-01-01T00:00:00Z")
	mustScan(&plainFriendly, InsertNationalTeamFriendlyMatchStmtSql, teamA, teamB, "2026-01-02T00:00:00Z")
	mustScan(&worldCupId, `INSERT INTO world_cups (season, name) VALUES (1, 'World Cup') RETURNING world_cup_id`)
	mustScan(&groupId, `INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, 'B') RETURNING group_id`, worldCupId)
	mustExec(`INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, 2, $2)`, groupId, groupFriendly)
	for _, id := range []int64{groupFriendly, plainFriendly} {
		if err := SaveFriendlyResultTx(dbCtx, id, teamA, teamB, MatchResult{HomeGoals: 2, AwayGoals: 1}); err != nil {
			t.Fatalf("SaveFriendlyResultTx: %s", err)
		}
		if err := UpdateFriendlyFlavor(dbCtx, id, "windy", 5000, "Ref Three", "Assistant E", "Assistant F"); err != nil {
			t.Fatalf("UpdateFriendlyFlavor: %s", err)
		}
	}

	group, err := GetFriendlyMatchById(dbCtx, groupFriendly)
	if err != nil {
		t.Fatalf("GetFriendlyMatchById: %s", err)
	}
	if group.WorldCupGroupName != "B" || group.WorldCupRound != 2 {
		t.Errorf("World Cup group fixture: got group %q round %d, want B / 2", group.WorldCupGroupName, group.WorldCupRound)
	}
	if group.Weather != "windy" || group.Attendance != 5000 || group.RefereeName != "Ref Three" || group.HomeStadiumName != "Estadio Norte" {
		t.Errorf("friendly flavor: got %+v", group)
	}

	plain, err := GetFriendlyMatchById(dbCtx, plainFriendly)
	if err != nil {
		t.Fatalf("GetFriendlyMatchById (plain): %s", err)
	}
	if plain.WorldCupGroupName != "" || plain.WorldCupRound != 0 {
		t.Errorf("an ordinary friendly has no World Cup label, got %q / %d", plain.WorldCupGroupName, plain.WorldCupRound)
	}
}
