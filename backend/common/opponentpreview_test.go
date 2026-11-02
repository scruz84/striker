package common

import (
	"reflect"
	"testing"
	"time"
)

func TestRecentFinishedCompetitive_KeepsFinishedLeagueAndCupNewestFirst(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-LiveBroadcastWindow)
	matches := []TeamMatchSummary{
		{Kind: MatchKindLeague, MatchId: 1, Played: true, MatchDate: now.Add(-72 * time.Hour)},
		{Kind: MatchKindFriendly, MatchId: 2, Played: true, MatchDate: now.Add(-60 * time.Hour)},
		{Kind: MatchKindCup, MatchId: 3, Played: true, MatchDate: now.Add(-48 * time.Hour)},
		{Kind: MatchKindLeague, MatchId: 4, Played: true, MatchDate: now.Add(-30 * time.Minute)}, // still being replayed
		{Kind: MatchKindLeague, MatchId: 5, Played: false, MatchDate: now.Add(24 * time.Hour)},
		{Kind: MatchKindLeague, MatchId: 6, Played: true, MatchDate: now.Add(-24 * time.Hour)},
	}

	got := recentFinishedCompetitive(matches, cutoff, 5)

	var ids []int64
	for _, m := range got {
		ids = append(ids, m.MatchId)
	}
	if want := []int64{6, 3, 1}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("got matches %v, want %v", ids, want)
	}
}

func TestRecentFinishedCompetitive_CapsToTheLatestFive(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	var matches []TeamMatchSummary
	for i := 0; i < 8; i++ {
		matches = append(matches, TeamMatchSummary{Kind: MatchKindLeague, MatchId: int64(i + 1), Played: true, MatchDate: now.Add(-time.Duration(8-i) * 24 * time.Hour)})
	}

	got := recentFinishedCompetitive(matches, now.Add(-LiveBroadcastWindow), OpponentPreviewFormMatches)

	if len(got) != 5 || got[0].MatchId != 8 || got[4].MatchId != 4 {
		t.Fatalf("got %+v, want matches 8..4", got)
	}
}

func rated(teamId, playerId int64, rating float64, position int) PlayerRating {
	return PlayerRating{TeamId: teamId, PlayerId: playerId, Rated: true, Rating: rating, Position: position}
}

func TestAggregateInForm_TopThreeWithAtLeastThreeAppearances(t *testing.T) {
	const team = 7
	byMatch := [][]PlayerRating{ // newest first
		{rated(team, 1, 8.0, 4), rated(team, 2, 7.0, 5), rated(team, 3, 6.0, 6), rated(team, 4, 9.5, 7), rated(99, 50, 10, 1)},
		{rated(team, 1, 7.0, 3), rated(team, 2, 7.0, 5), rated(team, 3, 6.5, 6), rated(team, 4, 9.5, 7)},
		{rated(team, 1, 7.5, 3), rated(team, 2, 6.5, 5), rated(team, 3, 6.5, 6), {TeamId: team, PlayerId: 5, Rated: false, Rating: 9, Position: 8}},
		{rated(team, 5, 9.0, 8)},
	}

	got := aggregateInForm(team, byMatch, 3, 3)

	if len(got) != 3 {
		t.Fatalf("got %d players, want 3: %+v", len(got), got)
	}
	// player 4 has 2 appearances, player 5 has 1 rated one, player 50 plays for another team
	if got[0].PlayerId != 1 || got[0].AverageRating != 7.5 || got[0].Appearances != 3 {
		t.Fatalf("best = %+v, want player 1 averaging 7.5 over 3", got[0])
	}
	if got[0].Position != 4 {
		t.Fatalf("position = %d, want the most recent appearance's 4", got[0].Position)
	}
	if got[1].PlayerId != 2 || got[2].PlayerId != 3 {
		t.Fatalf("order = %d, %d, want 2 then 3", got[1].PlayerId, got[2].PlayerId)
	}
}

func TestAggregateInForm_NobodyReachesTheMinimum(t *testing.T) {
	got := aggregateInForm(7, [][]PlayerRating{{rated(7, 1, 8, 1)}, {rated(7, 1, 8, 1)}}, 3, 3)
	if len(got) != 0 {
		t.Fatalf("got %+v, want none", got)
	}
}

func TestLikelyLineupFromSaved_DropsPlayersWhoLeft(t *testing.T) {
	squad := map[int64]*Player{1: {Id: 1, FirstName: "Ana", LastName: "Gil"}, 2: {Id: 2, FirstName: "Bo", LastName: "Ruiz"}}
	lineup := Lineup{Formation: 3, Positions: []LineupPosition{{PlayerId: 1, Position: GOALKEEPER}, {PlayerId: 9, Position: DEFENSE}, {PlayerId: 2, Position: STRIKER}}}

	got := likelyLineupFromSaved(MatchKindCup, 11, lineup, squad)

	if got == nil || got.Kind != MatchKindCup || got.MatchId != 11 || got.Formation != 3 || len(got.Players) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got.Players[0].Name != "Ana Gil" {
		t.Fatalf("name = %q", got.Players[0].Name)
	}
	if likelyLineupFromSaved(MatchKindLeague, 1, Lineup{Positions: []LineupPosition{{PlayerId: 9, Position: DEFENSE}}}, squad) != nil {
		t.Fatal("a lineup with nobody left must be nil")
	}
}
