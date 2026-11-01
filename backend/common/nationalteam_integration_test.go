package common

import (
	"testing"
	"time"
)

// TestNationalTeamRoundAndResults covers the national-teams page feeds against a real Postgres
// (STRIKER_TEST_DB_DSN; skipped when unset):
//   - the round lists unplayed matches, live ones and those that kicked off in the last 24 hours,
//     uncapped (the old feeds stopped at 10 out of ~100 friendlies a Saturday);
//   - a live match carries the score as of its current minute, never its final simulated score
//     (the old "En juego" box showed the final score while the match page showed the live one);
//   - World Cup group matches appear with their group and matchday;
//   - results only hold matches whose broadcast has finished, 25 per page, filterable by team;
//   - an owner's players in a saved call-up are counted per match.
func TestNationalTeamRoundAndResults(t *testing.T) {
	db, dbCtx := openEngineLocksTestDb(t)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	scanId := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}

	const countryA, countryB = "nm1", "nm2"
	for _, country := range []string{countryA, countryB} {
		exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country)
	}
	teamA, err := ensureNationalTeam(dbCtx, countryA)
	if err != nil {
		t.Fatal(err)
	}
	teamB, err := ensureNationalTeam(dbCtx, countryB)
	if err != nil {
		t.Fatal(err)
	}
	teams := []int64{teamA, teamB}
	exec(`DELETE FROM friendly_lineup_positions WHERE lineup_id IN (SELECT fl.lineup_id FROM friendly_lineup fl JOIN friendly_matches fm ON fm.match_id = fl.match_id WHERE fm.home_team_id = ANY($1) OR fm.away_team_id = ANY($1))`, teams)
	exec(`DELETE FROM friendly_lineup WHERE match_id IN (SELECT match_id FROM friendly_matches WHERE home_team_id = ANY($1) OR away_team_id = ANY($1))`, teams)
	exec(`DELETE FROM friendly_events WHERE match_id IN (SELECT match_id FROM friendly_matches WHERE home_team_id = ANY($1) OR away_team_id = ANY($1))`, teams)
	exec(`DELETE FROM world_cup_group_matches WHERE match_id IN (SELECT match_id FROM friendly_matches WHERE home_team_id = ANY($1) OR away_team_id = ANY($1))`, teams)
	exec(`DELETE FROM friendly_matches WHERE home_team_id = ANY($1) OR away_team_id = ANY($1)`, teams)

	now := time.Now().Truncate(time.Second)
	insert := func(kickoff time.Time, played bool, home, away int) int64 {
		return scanId(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, played, home_goals, away_goals) VALUES ($1, $2, $3, $4, $5, $6) RETURNING match_id`,
			teamA, teamB, kickoff, played, home, away)
	}
	scheduled := insert(now.Add(7*24*time.Hour), false, 0, 0)
	live := insert(now.Add(-30*time.Minute), true, 2, 3)
	exec(`INSERT INTO friendly_events (match_id, event_type, minute, home_goals, away_goals) VALUES ($1, 0, 10, 1, 0), ($1, 0, 80, 2, 3)`, live)
	finished := insert(now.Add(-3*time.Hour), true, 1, 1)
	old := make([]int64, 0, 26)
	for i := 0; i < 26; i++ {
		old = append(old, insert(now.Add(-time.Duration(72+i)*time.Hour), true, 0, i%3))
	}

	worldCupId := scanId(`INSERT INTO world_cups (season, name, stage, completed) VALUES (0, 'Test World Cup', 'groups', true) RETURNING world_cup_id`)
	groupId := scanId(`INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, 'C') RETURNING group_id`, worldCupId)
	groupMatch := insert(now.Add(8*24*time.Hour), false, 0, 0)
	exec(`INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, 2, $2)`, groupId, groupMatch)

	round, err := GetNationalTeamRound(dbCtx)
	if err != nil {
		t.Fatalf("GetNationalTeamRound: %s", err)
	}
	byId := make(map[int64]NationalMatch)
	for _, m := range round {
		if m.HomeTeamId == teamA && m.AwayTeamId == teamB {
			byId[m.Id] = m
		}
	}
	if len(byId) != 4 {
		t.Fatalf("round has %d of this test's matches, want 4 (scheduled, live, finished today, group match): %+v", len(byId), byId)
	}
	if m := byId[scheduled]; m.Status != NationalMatchScheduled || m.HomeGoals != nil || m.Kind != NationalMatchKindFriendly || m.HomeCountry != countryA {
		t.Errorf("scheduled match = %+v, want status scheduled, no score, kind friendly, home country %s", m, countryA)
	}
	if m := byId[live]; m.Status != NationalMatchLive || m.Minute == nil || *m.HomeGoals != 1 || *m.AwayGoals != 0 {
		t.Errorf("live match = %+v, want status live with the minute-%v score 1-0, not the final 2-3", m, m.Minute)
	}
	if m := byId[finished]; m.Status != NationalMatchFinished || *m.HomeGoals != 1 || *m.AwayGoals != 1 {
		t.Errorf("finished match = %+v, want finished 1-1", m)
	}
	if m := byId[groupMatch]; m.Kind != NationalMatchKindWorldCupGroup || m.GroupName != "C" || m.Round != 2 {
		t.Errorf("group match = %+v, want kind worldCupGroup, group C, matchday 2", m)
	}

	page0, err := GetNationalTeamResults(dbCtx, 0, teamA)
	if err != nil {
		t.Fatalf("GetNationalTeamResults: %s", err)
	}
	if len(page0.Matches) != NationalTeamResultsPageSize || !page0.HasMore {
		t.Fatalf("page 0 has %d matches (hasMore %v), want %d and more", len(page0.Matches), page0.HasMore, NationalTeamResultsPageSize)
	}
	if page0.Matches[0].Id != finished {
		t.Errorf("newest result is %d, want the finished match %d (the live one must not be listed)", page0.Matches[0].Id, finished)
	}
	for _, m := range page0.Matches {
		if m.Id == live {
			t.Errorf("results include match %d whose broadcast has not finished", live)
		}
	}
	page1, err := GetNationalTeamResults(dbCtx, 1, teamB)
	if err != nil {
		t.Fatalf("GetNationalTeamResults page 1: %s", err)
	}
	if len(page1.Matches) != 2 || page1.HasMore || page1.Matches[1].Id != old[len(old)-1] {
		t.Errorf("page 1 = %+v, want the 2 oldest results and no more", page1)
	}

	club := scanId(`INSERT INTO teams (name, country) VALUES ('Club NM', 'nm1') RETURNING team_id`)
	player1 := scanId(`INSERT INTO players (birth_day, team_id, first_name, last_name) VALUES (0, $1, 'Uno', 'NM') RETURNING player_id`, club)
	player2 := scanId(`INSERT INTO players (birth_day, team_id, first_name, last_name) VALUES (0, $1, 'Dos', 'NM') RETURNING player_id`, club)
	other := scanId(`INSERT INTO players (birth_day, team_id, first_name, last_name) VALUES (0, $1, 'Otro', 'NM') RETURNING player_id`, teamB)
	lineup := scanId(`INSERT INTO friendly_lineup (match_id, team_id) VALUES ($1, $2) RETURNING lineup_id`, scheduled, teamA)
	exec(`INSERT INTO friendly_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, 1), ($1, $3, 2), ($1, $4, 3)`, lineup, player1, player2, other)

	mine := []NationalMatch{byId[scheduled], byId[groupMatch]}
	if err := SetMyPlayersInNationalMatches(dbCtx, club, mine); err != nil {
		t.Fatalf("SetMyPlayersInNationalMatches: %s", err)
	}
	if mine[0].MyPlayers != 2 || mine[1].MyPlayers != 0 {
		t.Errorf("my players = %d and %d, want 2 (called up) and 0 (no lineup saved yet)", mine[0].MyPlayers, mine[1].MyPlayers)
	}
}
