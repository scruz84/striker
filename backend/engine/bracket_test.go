package engine

import (
	"math/rand"
	"testing"
)

func TestBuildFirstRoundPairings_TenTeams(t *testing.T) {
	teamIds := make([]int64, 10)
	for i := range teamIds {
		teamIds[i] = int64(i + 1)
	}
	r := rand.New(rand.NewSource(1))

	pairs := BuildFirstRoundPairings(teamIds, r)

	// 10 is even, so every team should get a real opponent immediately - no byes at all (see
	// issue #20: padding up to a 16-slot bracket used to leave 6 of these 10 teams with an
	// unplayed bye instead).
	if len(pairs) != 5 {
		t.Fatalf("expected 5 pairs (10 teams, no byes), got %d", len(pairs))
	}

	seen := make(map[int64]int)
	for _, pair := range pairs {
		for _, team := range pair {
			if team == ByeTeamId {
				t.Fatalf("did not expect any byes for an even entrant count, got pair %v", pair)
			}
			seen[team]++
		}
	}
	if len(seen) != 10 {
		t.Fatalf("expected all 10 teams to appear exactly once, got %d distinct teams: %v", len(seen), seen)
	}
	for team, count := range seen {
		if count != 1 {
			t.Errorf("team %d appears %d times, want exactly 1", team, count)
		}
	}
}

func TestBuildFirstRoundPairings_OddCountGetsExactlyOneBye(t *testing.T) {
	teamIds := make([]int64, 9)
	for i := range teamIds {
		teamIds[i] = int64(i + 1)
	}
	r := rand.New(rand.NewSource(1))

	pairs := BuildFirstRoundPairings(teamIds, r)

	if len(pairs) != 5 {
		t.Fatalf("expected 5 pairs (4 real matches + 1 bye for 9 teams), got %d", len(pairs))
	}
	byeCount := 0
	seen := make(map[int64]int)
	for _, pair := range pairs {
		for _, team := range pair {
			if team == ByeTeamId {
				byeCount++
				continue
			}
			seen[team]++
		}
	}
	if byeCount != 1 {
		t.Fatalf("expected exactly 1 bye for an odd entrant count, got %d", byeCount)
	}
	if len(seen) != 9 {
		t.Fatalf("expected all 9 teams to appear exactly once, got %d distinct teams: %v", len(seen), seen)
	}
}

func TestBuildFirstRoundPairings_PowerOfTwoHasNoByes(t *testing.T) {
	teamIds := make([]int64, 8)
	for i := range teamIds {
		teamIds[i] = int64(i + 1)
	}
	r := rand.New(rand.NewSource(1))

	pairs := BuildFirstRoundPairings(teamIds, r)

	if len(pairs) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}
	for _, pair := range pairs {
		if pair[0] == ByeTeamId || pair[1] == ByeTeamId {
			t.Fatalf("did not expect any byes for an exact power-of-two entrant count, got pair %v", pair)
		}
	}
}

func TestBuildNextRoundPairings(t *testing.T) {
	winners := []int64{10, 20, 30, 40, 50, 60, 70, 80}
	pairs := BuildNextRoundPairings(winners)

	want := [][2]int64{{10, 20}, {30, 40}, {50, 60}, {70, 80}}
	if len(pairs) != len(want) {
		t.Fatalf("expected %d pairs, got %d", len(want), len(pairs))
	}
	for i, pair := range pairs {
		if pair != want[i] {
			t.Errorf("pair %d = %v, want %v", i, pair, want[i])
		}
	}
}

func TestBuildNextRoundPairings_OddWinnersGetsOneBye(t *testing.T) {
	winners := []int64{10, 20, 30, 40, 50}
	pairs := BuildNextRoundPairings(winners)

	want := [][2]int64{{10, 20}, {30, 40}, {50, ByeTeamId}}
	if len(pairs) != len(want) {
		t.Fatalf("expected %d pairs, got %d", len(want), len(pairs))
	}
	for i, pair := range pairs {
		if pair != want[i] {
			t.Errorf("pair %d = %v, want %v", i, pair, want[i])
		}
	}
}
