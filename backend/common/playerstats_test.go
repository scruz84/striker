package common

import (
	"database/sql"
	"testing"
)

func TestBuildPlayerMatchStatLines(t *testing.T) {
	cameOn := 60
	ratings := []PlayerRating{
		{PlayerId: 2, TeamId: 10, Rated: true, Rating: 7.5, Minutes: 90},
		{PlayerId: 1, TeamId: 10, Rated: false, Rating: 6, Minutes: 5, CameOn: &cameOn},
	}
	events := []Event{
		{EventType: GOAL, TeamOwner: 10, TeamOwnerPlayers: []int64{2}, SetPiece: SET_PIECE_PENALTY},
		{EventType: GOAL, TeamOwner: 10, TeamOwnerPlayers: []int64{2}, SetPiece: SET_PIECE_DIRECT_FREE_KICK},
		{EventType: YELLOW_CARD, TeamOwner: 10, TeamOwnerPlayers: []int64{1}},
		{EventType: NO_GOAL, TeamOwner: 10, TeamOwnerPlayers: []int64{1}},
		{EventType: SUBSTITUTION, TeamOwner: 10, TeamOwnerPlayers: []int64{1, 3}},
		{EventType: RED_CARD, TeamOwner: 20, TeamOwnerPlayers: []int64{9}},
		{EventType: GOAL, TeamOwner: 20},
	}
	lines := BuildPlayerMatchStatLines(ratings, events)
	if len(lines) != 3 {
		t.Fatalf("lines = %+v, want 3", lines)
	}
	sub, scorer, sentOff := lines[0], lines[1], lines[2]
	if sub.PlayerId != 1 || sub.Started || !sub.Appeared || sub.Rated || sub.YellowCards != 1 || sub.Goals != 0 {
		t.Errorf("substitute = %+v", sub)
	}
	if scorer.PlayerId != 2 || !scorer.Started || scorer.Goals != 2 || scorer.PenaltyGoals != 1 || scorer.Rating != 7.5 {
		t.Errorf("scorer = %+v", scorer)
	}
	// An event naming a player with no rating still counts, without an appearance.
	if sentOff.PlayerId != 9 || sentOff.TeamId != 20 || sentOff.Appeared || sentOff.RedCards != 1 {
		t.Errorf("unrated player = %+v", sentOff)
	}
}

// addLeagueGoalStatForTest records one goal of playerId (playing for teamId) in a league match a
// test inserted by hand, as replacePlayerMatchStatsTx would have for a simulated one.
func addLeagueGoalStatForTest(t *testing.T, db interface {
	Exec(query string, args ...any) (sql.Result, error)
}, matchId, playerId, teamId int64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO player_match_stats (kind, match_id, player_id, team_id, kickoff, season, competition, competition_id, goals)
		SELECT 'league', m.match_id, $2, $3, m.match_date, m.season, 1, m.league_id, 1 FROM matches m WHERE m.match_id = $1
		ON CONFLICT (kind, match_id, player_id) DO UPDATE SET goals = player_match_stats.goals + 1`, matchId, playerId, teamId); err != nil {
		t.Fatalf("adding a goal stat: %s", err)
	}
}
