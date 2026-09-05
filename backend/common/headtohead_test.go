package common

import "testing"

func TestSummarizeHeadToHead_EmptyMeetings(t *testing.T) {
	result := summarizeHeadToHead(1, 2, nil)
	if result.TotalMatches != 0 || result.TeamAWins != 0 || result.TeamBWins != 0 || result.Draws != 0 {
		t.Fatalf("expected an all-zero summary for no meetings, got %+v", result)
	}
	if len(result.Meetings) != 0 {
		t.Fatalf("expected no meetings, got %d", len(result.Meetings))
	}
}

func TestSummarizeHeadToHead_CountsWinsRegardlessOfHomeAway(t *testing.T) {
	// Team 1 wins as home in the first meeting, and wins again as away in the second - both
	// should count as a win for team 1 regardless of which side of the fixture it was on.
	meetings := []HeadToHeadMeeting{
		{HomeTeamId: 1, AwayTeamId: 2, HomeGoals: 3, AwayGoals: 1},
		{HomeTeamId: 2, AwayTeamId: 1, HomeGoals: 0, AwayGoals: 2},
	}

	result := summarizeHeadToHead(1, 2, meetings)

	if result.TotalMatches != 2 {
		t.Fatalf("expected 2 total matches, got %d", result.TotalMatches)
	}
	if result.TeamAWins != 2 {
		t.Fatalf("expected team A (1) to have 2 wins, got %d", result.TeamAWins)
	}
	if result.TeamBWins != 0 || result.Draws != 0 {
		t.Fatalf("expected no wins for team B and no draws, got teamBWins=%d draws=%d", result.TeamBWins, result.Draws)
	}
	if result.TeamAGoals != 5 || result.TeamBGoals != 1 {
		t.Fatalf("expected team A goals=5 team B goals=1, got %d/%d", result.TeamAGoals, result.TeamBGoals)
	}
}

func TestSummarizeHeadToHead_CountsDraws(t *testing.T) {
	meetings := []HeadToHeadMeeting{
		{HomeTeamId: 1, AwayTeamId: 2, HomeGoals: 1, AwayGoals: 1},
	}

	result := summarizeHeadToHead(1, 2, meetings)

	if result.Draws != 1 || result.TeamAWins != 0 || result.TeamBWins != 0 {
		t.Fatalf("expected a single draw, got %+v", result)
	}
}

func TestSummarizeHeadToHead_TeamBPerspectiveIsTheComplement(t *testing.T) {
	meetings := []HeadToHeadMeeting{
		{HomeTeamId: 2, AwayTeamId: 1, HomeGoals: 4, AwayGoals: 0},
	}

	result := summarizeHeadToHead(1, 2, meetings)

	if result.TeamBWins != 1 || result.TeamAWins != 0 {
		t.Fatalf("expected team B (2) to have won as home team, got %+v", result)
	}
	if result.TeamAGoals != 0 || result.TeamBGoals != 4 {
		t.Fatalf("expected team A goals=0 team B goals=4, got %d/%d", result.TeamAGoals, result.TeamBGoals)
	}
}

func TestSummarizeHeadToHead_CapsMeetingsButNotAggregateCounts(t *testing.T) {
	meetings := make([]HeadToHeadMeeting, HeadToHeadMeetingsLimit+5)
	for i := range meetings {
		meetings[i] = HeadToHeadMeeting{HomeTeamId: 1, AwayTeamId: 2, HomeGoals: 1, AwayGoals: 0}
	}

	result := summarizeHeadToHead(1, 2, meetings)

	if result.TotalMatches != HeadToHeadMeetingsLimit+5 {
		t.Fatalf("expected aggregate count to cover every meeting, got %d", result.TotalMatches)
	}
	if len(result.Meetings) != HeadToHeadMeetingsLimit {
		t.Fatalf("expected meetings list capped at %d, got %d", HeadToHeadMeetingsLimit, len(result.Meetings))
	}
}
