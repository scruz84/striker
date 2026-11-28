package common

import (
	"testing"
	"time"
)

// TestRecords_FromTheStoredHistory runs against a real Postgres (STRIKER_TEST_DB_DSN, skipped when
// unset) and pins openspec/specs/records: the club and league records read the played fixtures
// (never one still on air), the final tables, the transfers and the player stats.
func TestRecords_FromTheStoredHistory(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	mustScan := func(dest any, query string, args ...any) {
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

	country := unusedTestCountryCode(t, db, 'q')
	seedTransfersCountry(t, db, country)
	var leagueId, us, them int64
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'RecL', 1) RETURNING league_id`, country)
	mustScan(&us, `INSERT INTO teams (name, country, league_id) VALUES ('Rec Us', $1, $2) RETURNING team_id`, country, leagueId)
	mustScan(&them, `INSERT INTO teams (name, country, league_id) VALUES ('Rec Them', $1, $2) RETURNING team_id`, country, leagueId)
	scorer := seedTransfersPlayer(t, db, us, country)

	match := func(daysAgo int, home, away int64, hg, ag int, attendance int) int64 {
		var id int64
		mustScan(&id, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id, home_goals, away_goals, attendance)
			VALUES ($1, 1, true, $2, $3, $4, $5, $6, $7) RETURNING match_id`,
			leagueId, time.Now().Add(-time.Duration(daysAgo)*24*time.Hour), home, away, hg, ag, attendance)
		return id
	}
	win := match(3, us, them, 4, 0, 20000)
	match(2, them, us, 1, 1, 15000)
	match(1, us, them, 2, 1, 18000)
	// Still on air: a record-breaking 9-0 that must not show yet.
	var live int64
	mustScan(&live, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id, home_goals, away_goals, attendance)
		VALUES ($1, 1, true, now(), $2, $3, 9, 0, 99999) RETURNING match_id`, leagueId, us, them)
	addLeagueGoalStatForTest(t, db, win, scorer, us)
	addLeagueGoalStatForTest(t, db, win, scorer, us)

	mustExec(`INSERT INTO season_history (team_id, season, league_id, league_name, final_position, points) VALUES ($1, 1, $2, 'RecL', 2, 51)`, us, leagueId)
	mustExec(`INSERT INTO season_history (team_id, season, league_id, league_name, final_position, points) VALUES ($1, 2, $2, 'RecL', 1, 48)`, us, leagueId)
	mustExec(`INSERT INTO season_history (team_id, season, league_id, league_name, final_position, points) VALUES ($1, 1, $2, 'RecL', 1, 55)`, them, leagueId)
	mustExec(InsertPlayerTransferStmtSql, scorer, them, us, 750000, PLAYER_TRANSFER_OFFER, STRIKER, 25, 60)
	if err := RecordPlayerSeasonAward(dbCtx, leagueId, 1, AwardGoldenBoot, scorer, 2); err != nil {
		t.Fatalf("award: %s", err)
	}

	club, err := GetClubRecords(dbCtx, us)
	if err != nil {
		t.Fatalf("GetClubRecords: %s", err)
	}
	if club.BiggestWin == nil || club.BiggestWin.MatchId != win {
		t.Errorf("biggest win = %+v, want the 4-0 (the live 9-0 hidden)", club.BiggestWin)
	}
	if club.RecordAttendance == nil || *club.RecordAttendance.Attendance != 20000 {
		t.Errorf("record attendance = %+v, want 20000", club.RecordAttendance)
	}
	if club.LongestUnbeatenStreak == nil || club.LongestUnbeatenStreak.Length != 3 || !club.LongestUnbeatenStreak.Ongoing {
		t.Errorf("unbeaten streak = %+v, want 3 and ongoing", club.LongestUnbeatenStreak)
	}
	if club.BestFinish == nil || club.BestFinish.Season != 2 || club.BestFinish.Position != 1 {
		t.Errorf("best finish = %+v, want the season 2 title", club.BestFinish)
	}
	if club.MostPoints == nil || club.MostPoints.Points != 51 {
		t.Errorf("most points = %+v, want 51", club.MostPoints)
	}
	if club.RecordSigning == nil || club.RecordSigning.Fee != 750000 || club.RecordSigning.OtherTeamName != "Rec Them" ||
		club.RecordSigning.PlayerName != "Test Player" || club.RecordSale != nil {
		t.Errorf("transfers = %+v / %+v", club.RecordSigning, club.RecordSale)
	}
	if len(club.TopScorers) != 1 || club.TopScorers[0].PlayerId != scorer || club.TopScorers[0].Goals != 2 {
		t.Errorf("top scorers = %+v", club.TopScorers)
	}

	league, err := GetLeagueRecords(dbCtx, leagueId)
	if err != nil {
		t.Fatalf("GetLeagueRecords: %s", err)
	}
	if len(league.Champions) != 2 || league.Champions[0].Season != 2 || league.Champions[0].TeamId != us {
		t.Errorf("champions = %+v", league.Champions)
	}
	if len(league.MostTitles) != 2 || league.MostTitles[0].TeamId != us {
		t.Errorf("most titles = %+v, want us first (the latest title breaks the tie)", league.MostTitles)
	}
	if league.MostPoints == nil || league.MostPoints.Points != 55 || league.MostPoints.TeamId != them {
		t.Errorf("most points = %+v", league.MostPoints)
	}
	if league.BiggestWin == nil || league.BiggestWin.MatchId != win || league.HighestScoring == nil || league.HighestScoring.MatchId != win {
		t.Errorf("league matches = %+v / %+v, want the 4-0", league.BiggestWin, league.HighestScoring)
	}
	if len(league.GoldenBoots) != 1 || league.GoldenBoots[0].PlayerId != scorer || league.GoldenBoots[0].Goals != 2 {
		t.Errorf("golden boots = %+v", league.GoldenBoots)
	}
	if len(league.TopScorers) != 1 || league.TopScorers[0].Goals != 2 {
		t.Errorf("league top scorers = %+v", league.TopScorers)
	}
	_ = live
}
