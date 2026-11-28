package common

import (
	"testing"
	"time"
)

func TestClubMatchRecords(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 1, d, 18, 0, 0, 0, time.UTC) }
	crowd := func(n int) *int { return &n }
	const us, them = 1, 2
	fixtures := []RecordMatch{
		{Kind: SimulationKindLeague, MatchId: 1, MatchDate: day(1), HomeTeamId: us, AwayTeamId: them, HomeGoals: 3, AwayGoals: 0, Attendance: crowd(9000)},
		{Kind: SimulationKindLeague, MatchId: 2, MatchDate: day(2), HomeTeamId: them, AwayTeamId: us, HomeGoals: 1, AwayGoals: 4},
		{Kind: SimulationKindCup, MatchId: 3, MatchDate: day(3), HomeTeamId: us, AwayTeamId: them, HomeGoals: 5, AwayGoals: 2, Attendance: crowd(12000)},
		{Kind: SimulationKindLeague, MatchId: 4, MatchDate: day(4), HomeTeamId: us, AwayTeamId: them, HomeGoals: 1, AwayGoals: 1, Attendance: crowd(12000)},
		{Kind: SimulationKindLeague, MatchId: 5, MatchDate: day(5), HomeTeamId: them, AwayTeamId: us, HomeGoals: 4, AwayGoals: 0},
		{Kind: SimulationKindLeague, MatchId: 6, MatchDate: day(6), HomeTeamId: us, AwayTeamId: them, HomeGoals: 0, AwayGoals: 2},
	}
	var r ClubRecords
	ClubMatchRecords(us, fixtures, &r)

	// 3-0 and 4-1 and 5-2 share the margin: the most goals scored wins.
	if r.BiggestWin == nil || r.BiggestWin.MatchId != 3 {
		t.Errorf("biggest win = %+v, want the 5-2 cup tie", r.BiggestWin)
	}
	if r.BiggestDefeat == nil || r.BiggestDefeat.MatchId != 5 {
		t.Errorf("biggest defeat = %+v, want the 4-0", r.BiggestDefeat)
	}
	if r.HighestScoring == nil || r.HighestScoring.MatchId != 3 {
		t.Errorf("highest scoring = %+v, want the 5-2", r.HighestScoring)
	}
	// Two crowds of 12000: the earlier one holds the record.
	if r.RecordAttendance == nil || r.RecordAttendance.MatchId != 3 {
		t.Errorf("record attendance = %+v, want the first 12000", r.RecordAttendance)
	}
	// League matches only: W W D L L.
	if s := r.LongestWinningStreak; s == nil || s.Length != 2 || !s.From.Equal(day(1)) || !s.To.Equal(day(2)) || s.Ongoing {
		t.Errorf("winning streak = %+v", s)
	}
	if s := r.LongestUnbeatenStreak; s == nil || s.Length != 3 || !s.To.Equal(day(4)) {
		t.Errorf("unbeaten streak = %+v", s)
	}
	if s := r.LongestLosingStreak; s == nil || s.Length != 2 || !s.From.Equal(day(5)) || !s.Ongoing {
		t.Errorf("losing streak = %+v, want 2 and ongoing", s)
	}
}

func TestLeagueTitleCounts(t *testing.T) {
	counts := LeagueTitleCounts([]LeagueChampion{
		{Season: 5, TeamId: 3}, {Season: 4, TeamId: 2}, {Season: 3, TeamId: 3}, {Season: 2, TeamId: 2}, {Season: 1, TeamId: 1},
	})
	if len(counts) != 3 || counts[0].TeamId != 3 || counts[0].Titles != 2 || counts[0].LastSeason != 5 ||
		counts[1].TeamId != 2 || counts[2].TeamId != 1 {
		t.Errorf("title counts = %+v", counts)
	}
}
