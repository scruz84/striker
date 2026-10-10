package common

import (
	"math/rand"
	"testing"
)

func TestAssignWorldCupGroups_EveryCountryAppearsExactlyOnce(t *testing.T) {
	countries := []string{"es", "en", "fr", "de", "it", "pt", "nl", "se", "pl", "br", "ar", "mx"}
	groups := AssignWorldCupGroups(countries, rand.New(rand.NewSource(1)))

	seen := make(map[string]int)
	for _, group := range groups {
		for _, country := range group {
			seen[country]++
		}
	}
	if len(seen) != len(countries) {
		t.Fatalf("expected %d distinct countries, got %d", len(countries), len(seen))
	}
	for _, country := range countries {
		if seen[country] != 1 {
			t.Errorf("country %q appeared %d times, want 1", country, seen[country])
		}
	}
}

func TestAssignWorldCupGroups_SizesBalanced(t *testing.T) {
	countries := make([]string, 13)
	for i := range countries {
		countries[i] = string(rune('a' + i))
	}
	groups := AssignWorldCupGroups(countries, rand.New(rand.NewSource(2)))

	min, max := len(countries), 0
	for _, group := range groups {
		if len(group) < min {
			min = len(group)
		}
		if len(group) > max {
			max = len(group)
		}
	}
	if max-min > 1 {
		t.Errorf("group sizes differ by more than one: min %d, max %d", min, max)
	}
}

func TestAssignWorldCupGroups_AtLeastTwoGroups(t *testing.T) {
	countries := []string{"es", "en", "fr", "de", "it", "pt", "nl", "se"}
	groups := AssignWorldCupGroups(countries, rand.New(rand.NewSource(3)))
	if len(groups) < 2 {
		t.Errorf("expected at least 2 groups for %d countries, got %d", len(countries), len(groups))
	}
}

func TestAssignWorldCupGroups_DeterministicGivenSameSeed(t *testing.T) {
	countries := []string{"es", "en", "fr", "de", "it", "pt", "nl", "se", "pl", "br"}
	a := AssignWorldCupGroups(countries, rand.New(rand.NewSource(42)))
	b := AssignWorldCupGroups(countries, rand.New(rand.NewSource(42)))

	if len(a) != len(b) {
		t.Fatalf("group counts differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			t.Fatalf("group %d sizes differ: %d vs %d", i, len(a[i]), len(b[i]))
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				t.Errorf("group %d slot %d differs: %q vs %q", i, j, a[i][j], b[i][j])
			}
		}
	}
}

func TestBuildRoundRobinRounds_EveryPairPlaysExactlyOnce(t *testing.T) {
	for _, n := range []int{4, 5, 6, 7} {
		rounds := BuildRoundRobinRounds(n)
		played := make(map[[2]int]int)
		for _, pairs := range rounds {
			for _, pair := range pairs {
				a, b := pair[0], pair[1]
				if a > b {
					a, b = b, a
				}
				played[[2]int{a, b}]++
			}
		}

		want := n * (n - 1) / 2
		if len(played) != want {
			t.Errorf("n=%d: expected %d distinct pairs, got %d", n, want, len(played))
		}
		for pair, count := range played {
			if count != 1 {
				t.Errorf("n=%d: pair %v played %d times, want 1", n, pair, count)
			}
		}
	}
}

func TestBuildRoundRobinRounds_RoundCount(t *testing.T) {
	if got := len(BuildRoundRobinRounds(4)); got != 3 {
		t.Errorf("n=4: expected 3 rounds, got %d", got)
	}
	if got := len(BuildRoundRobinRounds(5)); got != 5 {
		t.Errorf("n=5: expected 5 rounds, got %d", got)
	}
}

func TestBuildRoundRobinRounds_EveryoneSitsOutAtMostOncePerOddN(t *testing.T) {
	n := 5
	rounds := BuildRoundRobinRounds(n)
	appearances := make([]int, n)
	for _, pairs := range rounds {
		playedThisRound := make(map[int]bool)
		for _, pair := range pairs {
			playedThisRound[pair[0]] = true
			playedThisRound[pair[1]] = true
		}
		for i := 0; i < n; i++ {
			if playedThisRound[i] {
				appearances[i]++
			}
		}
	}
	// n rounds, n participants, one sits out per round -> everyone plays in n-1 of the n rounds.
	for i, count := range appearances {
		if count != n-1 {
			t.Errorf("participant %d played %d rounds, want %d", i, count, n-1)
		}
	}
}

func TestBuildRoundRobinRounds_TooFewParticipants(t *testing.T) {
	if got := BuildRoundRobinRounds(1); got != nil {
		t.Errorf("expected nil for n=1, got %v", got)
	}
	if got := BuildRoundRobinRounds(0); got != nil {
		t.Errorf("expected nil for n=0, got %v", got)
	}
}

func TestRankGroupStandings_SortsByPointsThenGoalDifferenceThenGoalsFor(t *testing.T) {
	standings := []WorldCupGroupStanding{
		{TeamName: "Low points", Won: 1, Drawn: 0, Lost: 2, GoalsFor: 5, GoalsAgainst: 5},
		{TeamName: "High points", Won: 2, Drawn: 1, Lost: 0, GoalsFor: 4, GoalsAgainst: 2},
		{TeamName: "Same points better GD", Won: 1, Drawn: 2, Lost: 0, GoalsFor: 6, GoalsAgainst: 2},
		{TeamName: "Same points worse GD", Won: 1, Drawn: 2, Lost: 0, GoalsFor: 3, GoalsAgainst: 3},
	}
	ranked := RankGroupStandings(standings)

	want := []string{"High points", "Same points better GD", "Same points worse GD", "Low points"}
	for i, name := range want {
		if ranked[i].TeamName != name {
			t.Errorf("position %d: got %q, want %q", i, ranked[i].TeamName, name)
		}
	}
}

func TestRankGroupStandings_TiebreaksAlphabeticallyByTeamName(t *testing.T) {
	standings := []WorldCupGroupStanding{
		{TeamName: "Zebra", Won: 1, Drawn: 0, Lost: 0, GoalsFor: 2, GoalsAgainst: 0},
		{TeamName: "Alpha", Won: 1, Drawn: 0, Lost: 0, GoalsFor: 2, GoalsAgainst: 0},
	}
	ranked := RankGroupStandings(standings)
	if ranked[0].TeamName != "Alpha" || ranked[1].TeamName != "Zebra" {
		t.Errorf("expected alphabetical tiebreak, got %q then %q", ranked[0].TeamName, ranked[1].TeamName)
	}
}

func TestRankGroupStandings_DoesNotMutateInput(t *testing.T) {
	standings := []WorldCupGroupStanding{
		{TeamName: "B", Won: 0},
		{TeamName: "A", Won: 1},
	}
	_ = RankGroupStandings(standings)
	if standings[0].TeamName != "B" || standings[1].TeamName != "A" {
		t.Errorf("RankGroupStandings mutated its input: %+v", standings)
	}
}
