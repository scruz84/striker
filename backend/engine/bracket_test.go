package engine

import (
	"math/rand"
	"testing"
)

func TestNextPowerOfTwo(t *testing.T) {
	cases := map[int]int{1: 1, 2: 2, 3: 4, 4: 4, 5: 8, 10: 16, 16: 16, 17: 32}
	for n, want := range cases {
		if got := NextPowerOfTwo(n); got != want {
			t.Errorf("NextPowerOfTwo(%d) = %d, want %d", n, got, want)
		}
	}
}

func TestBuildFirstRoundPairings_TenTeams(t *testing.T) {
	teamIds := make([]int64, 10)
	for i := range teamIds {
		teamIds[i] = int64(i + 1)
	}
	r := rand.New(rand.NewSource(1))

	pairs := BuildFirstRoundPairings(teamIds, r)

	if len(pairs) != 8 {
		t.Fatalf("expected 8 pairs (16-slot bracket for 10 teams), got %d", len(pairs))
	}

	seen := make(map[int64]int)
	byeCount := 0
	for _, pair := range pairs {
		for _, team := range pair {
			if team == ByeTeamId {
				byeCount++
				continue
			}
			seen[team]++
		}
		if pair[0] == ByeTeamId && pair[1] == ByeTeamId {
			t.Fatalf("two byes paired together: %v", pair)
		}
	}

	if byeCount != 6 {
		t.Fatalf("expected 6 byes (16 slots - 10 teams), got %d", byeCount)
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
