package common

import "testing"

func TestPickRandomLiveMatch_Empty(t *testing.T) {
	if _, ok := PickRandomLiveMatch(nil); ok {
		t.Fatalf("expected no pick from an empty match list")
	}
}

func TestPickRandomLiveMatch_PrefersTopDivision(t *testing.T) {
	matches := []LiveMatchSummary{
		{Kind: MatchKindLeague, MatchId: 1, Tier: 3},
		{Kind: MatchKindLeague, MatchId: 2, Tier: 1},
		{Kind: MatchKindCup, MatchId: 3},
		{Kind: MatchKindFriendly, MatchId: 4},
	}

	for i := 0; i < 20; i++ {
		picked, ok := PickRandomLiveMatch(matches)
		if !ok {
			t.Fatalf("expected a pick")
		}
		if picked.MatchId != 2 {
			t.Fatalf("expected the tier-1 league match to always be picked over lower-priority matches, got match %d", picked.MatchId)
		}
	}
}

func TestPickRandomLiveMatch_FallsBackWhenNoLeagueMatchIsLive(t *testing.T) {
	matches := []LiveMatchSummary{
		{Kind: MatchKindCup, MatchId: 1},
		{Kind: MatchKindFriendly, MatchId: 2},
	}

	seen := make(map[int64]bool)
	for i := 0; i < 40; i++ {
		picked, ok := PickRandomLiveMatch(matches)
		if !ok {
			t.Fatalf("expected a pick")
		}
		seen[picked.MatchId] = true
	}

	if !seen[1] || !seen[2] {
		t.Fatalf("expected both non-league matches to be picked at least once across attempts, got %v", seen)
	}
}

func TestPickRandomLiveMatch_OnlyPicksAmongLowestTier(t *testing.T) {
	matches := []LiveMatchSummary{
		{Kind: MatchKindLeague, MatchId: 1, Tier: 2},
		{Kind: MatchKindLeague, MatchId: 2, Tier: 2},
		{Kind: MatchKindLeague, MatchId: 3, Tier: 4},
	}

	seen := make(map[int64]bool)
	for i := 0; i < 40; i++ {
		picked, ok := PickRandomLiveMatch(matches)
		if !ok {
			t.Fatalf("expected a pick")
		}
		if picked.MatchId == 3 {
			t.Fatalf("expected the higher-tier-number (lower priority) match never to be picked while a tier-2 match is live")
		}
		seen[picked.MatchId] = true
	}

	if !seen[1] || !seen[2] {
		t.Fatalf("expected both tier-2 matches to be picked at least once across attempts, got %v", seen)
	}
}
