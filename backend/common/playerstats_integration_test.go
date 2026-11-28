package common

import (
	"testing"
	"time"
)

// TestPlayerMatchStats_TotalsAcrossTheReaders runs against a real Postgres (STRIKER_TEST_DB_DSN,
// skipped when unset) and pins openspec/specs/player-stats: a saved result stores one stats line
// per player, the public totals leave out a match still on air and club friendlies, the leaders
// and the awards count the league only, and a retired player keeps his lines and still wins.
func TestPlayerMatchStats_TotalsAcrossTheReaders(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	mustScan := func(dest any, query string, args ...any) {
		t.Helper()
		if err := db.QueryRow(query, args...).Scan(dest); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}

	country := unusedTestCountryCode(t, db, 's')
	seedTransfersCountry(t, db, country)
	if _, err := db.Exec(`UPDATE countries SET current_season = 3 WHERE name = $1`, country); err != nil {
		t.Fatalf("season: %s", err)
	}
	var leagueId, teamA, teamB int64
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'StatsL', 1) RETURNING league_id`, country)
	mustScan(&teamA, `INSERT INTO teams (name, country, league_id) VALUES ('Stats A', $1, $2) RETURNING team_id`, country, leagueId)
	mustScan(&teamB, `INSERT INTO teams (name, country, league_id) VALUES ('Stats B', $1, $2) RETURNING team_id`, country, leagueId)
	a1 := seedTransfersPlayer(t, db, teamA, country)
	a2 := seedTransfersPlayer(t, db, teamA, country)
	b1 := seedTransfersPlayer(t, db, teamB, country)

	newMatch := func(kickoff time.Time) int64 {
		var id int64
		mustScan(&id, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
			VALUES ($1, 3, false, $2, $3, $4) RETURNING match_id`, leagueId, kickoff, teamA, teamB)
		return id
	}
	cameOn := 70
	ratings := []PlayerRating{
		{PlayerId: a1, TeamId: teamA, Rated: true, Rating: 8, Minutes: 90, Position: STRIKER},
		{PlayerId: a2, TeamId: teamA, Rated: true, Rating: 6.5, Minutes: 20, Position: MIDFIELD, CameOn: &cameOn},
		{PlayerId: b1, TeamId: teamB, Rated: true, Rating: 5.5, Minutes: 90, Position: DEFENSE},
	}
	events := []Event{
		{EventType: GOAL, TeamOwner: teamA, OtherTeam: teamB, TeamOwnerPlayers: []int64{a1}, SetPiece: SET_PIECE_PENALTY},
		{EventType: GOAL, TeamOwner: teamA, OtherTeam: teamB, TeamOwnerPlayers: []int64{a1}},
		{EventType: YELLOW_CARD, TeamOwner: teamB, OtherTeam: teamA, TeamOwnerPlayers: []int64{b1}},
		{EventType: RED_CARD, TeamOwner: teamB, OtherTeam: teamA, TeamOwnerPlayers: []int64{b1}},
		{EventType: SUBSTITUTION, TeamOwner: teamA, OtherTeam: teamB, TeamOwnerPlayers: []int64{a2}},
	}
	past := newMatch(time.Now().Add(-3 * time.Hour))
	if err := SaveLeagueMatchResultTx(dbCtx, past, leagueId, teamA, teamB,
		MatchResult{HomeGoals: 2, Events: events, PlayerRatings: ratings}); err != nil {
		t.Fatalf("saving the past match: %s", err)
	}
	// A match still inside its live window: its goal counts for the final tallies only.
	live := newMatch(time.Now())
	if err := SaveLeagueMatchResultTx(dbCtx, live, leagueId, teamA, teamB, MatchResult{HomeGoals: 1,
		Events:        []Event{{EventType: GOAL, TeamOwner: teamA, OtherTeam: teamB, TeamOwnerPlayers: []int64{a1}}},
		PlayerRatings: []PlayerRating{{PlayerId: a1, TeamId: teamA, Rated: true, Rating: 7, Minutes: 90}}}); err != nil {
		t.Fatalf("saving the live match: %s", err)
	}
	// A club friendly: on the profile's season lines, never in the career totals.
	var friendly int64
	mustScan(&friendly, InsertNationalTeamFriendlyMatchStmtSql, teamA, teamB, time.Now().Add(-5*time.Hour).Format(time.RFC3339))
	if err := SaveFriendlyResultTx(dbCtx, friendly, teamA, teamB, MatchResult{HomeGoals: 1,
		Events:        []Event{{EventType: GOAL, TeamOwner: teamA, OtherTeam: teamB, TeamOwnerPlayers: []int64{a1}}},
		PlayerRatings: []PlayerRating{{PlayerId: a1, TeamId: teamA, Rated: true, Rating: 7, Minutes: 90}}}); err != nil {
		t.Fatalf("saving the friendly: %s", err)
	}

	var goals, pens, season, competition int
	var started bool
	if err := db.QueryRow(`SELECT goals, penalty_goals, season, competition, started FROM player_match_stats
		WHERE kind = 'league' AND match_id = $1 AND player_id = $2`, past, a1).Scan(&goals, &pens, &season, &competition, &started); err != nil {
		t.Fatalf("reading a1's line: %s", err)
	}
	if goals != 2 || pens != 1 || season != 3 || competition != StatsCompetitionLeague || !started {
		t.Errorf("a1 line = goals %d pens %d season %d competition %d started %v", goals, pens, season, competition, started)
	}
	if err := db.QueryRow(`SELECT started FROM player_match_stats WHERE kind = 'league' AND match_id = $1 AND player_id = $2`,
		past, a2).Scan(&started); err != nil || started {
		t.Errorf("a2 started = %v, %v; want a substitute", started, err)
	}
	var friendlyCompetition int
	mustScan(&friendlyCompetition, `SELECT competition FROM player_match_stats WHERE kind = 'friendly' AND match_id = $1`, friendly)
	if friendlyCompetition != StatsCompetitionClubFriendly {
		t.Errorf("friendly competition = %d, want club friendly", friendlyCompetition)
	}

	career, err := GetPlayerCareerStats(dbCtx, a1, teamA)
	if err != nil {
		t.Fatalf("GetPlayerCareerStats: %s", err)
	}
	if career.Goals != 2 || career.GoalsForTeam != 2 || career.Appearances != 1 || career.Minutes != 90 {
		t.Errorf("career = %+v, want 2 goals in 1 appearance (live match and friendly left out)", career)
	}

	lines, err := GetPlayerSeasonStats(dbCtx, a1)
	if err != nil {
		t.Fatalf("GetPlayerSeasonStats: %s", err)
	}
	if len(lines) != 2 || lines[0].Competition != StatsCompetitionLeague || lines[0].CompetitionName != "StatsL" ||
		lines[0].Goals != 2 || lines[0].AverageRating == nil || *lines[0].AverageRating != 8 ||
		lines[1].Competition != StatsCompetitionClubFriendly || lines[1].Goals != 1 {
		t.Errorf("season lines = %+v", lines)
	}

	scorers, err := GetTopScorers(dbCtx, leagueId, 3, 10)
	if err != nil || len(scorers) != 1 || scorers[0].PlayerId != a1 || scorers[0].Goals != 2 || scorers[0].TeamName != "Stats A" {
		t.Errorf("GetTopScorers = %+v, %v", scorers, err)
	}
	final, err := GetTopScorersFinal(dbCtx, leagueId, 3, 10)
	if err != nil || len(final) != 1 || final[0].Goals != 3 {
		t.Errorf("GetTopScorersFinal = %+v, %v; want the live match's goal counted", final, err)
	}

	cutoff := LiveBroadcastCutoff()
	cards, err := GetLeagueLeaders(dbCtx, leagueId, 3, LeadersCards, 10, &cutoff)
	if err != nil || len(cards) != 1 || cards[0].PlayerId != b1 || cards[0].YellowCards != 1 || cards[0].RedCards != 1 {
		t.Errorf("card leaders = %+v, %v", cards, err)
	}
	rated, err := GetLeagueLeaders(dbCtx, leagueId, 3, LeadersRating, 10, &cutoff)
	if err != nil || len(rated) != 2 || rated[0].PlayerId != a1 || rated[1].PlayerId != b1 {
		t.Errorf("rating leaders = %+v, %v; want a1 then b1, the 20-minute substitute left out", rated, err)
	}

	// He retires in the rollover, before the awards are handed out: still Player of the Season.
	if err := RetirePlayer(dbCtx, a1); err != nil {
		t.Fatalf("RetirePlayer: %s", err)
	}
	award, err := GetPlayerOfTheSeason(dbCtx, leagueId, 3)
	if err != nil || award == nil || award.PlayerId != a1 || award.Score != 9 || award.LastName != "Player" || award.TeamId != teamA {
		t.Errorf("GetPlayerOfTheSeason = %+v, %v; want the retired a1 with 9", award, err)
	}

	squad, err := GetTeamSeasonStats(dbCtx, teamA, 0)
	if err != nil {
		t.Fatalf("GetTeamSeasonStats: %s", err)
	}
	if squad.Season != 3 || len(squad.Seasons) != 1 || len(squad.Players) != 2 {
		t.Fatalf("squad stats = %+v", squad)
	}
	if squad.Players[0].PlayerId != a1 || !squad.Players[0].Retired || squad.Players[0].InSquad || squad.Players[0].Goals != 2 {
		t.Errorf("first squad line = %+v, want the retired a1", squad.Players[0])
	}
	if squad.Players[1].PlayerId != a2 || !squad.Players[1].InSquad || squad.Players[1].Starts != 0 || squad.Players[1].Appearances != 1 {
		t.Errorf("second squad line = %+v, want a2 off the bench", squad.Players[1])
	}
}
