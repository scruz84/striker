package engine

import (
	"math/rand"
	"testing"
)

func makeTeamIds(n int) []int64 {
	teamIds := make([]int64, n)
	for i := range teamIds {
		teamIds[i] = int64(i + 1)
	}
	return teamIds
}

func TestBuildFirstRoundPairings_TenTeams(t *testing.T) {
	teamIds := makeTeamIds(10)
	r := rand.New(rand.NewSource(1))

	pairs := BuildFirstRoundPairings(teamIds, r, nil)

	// 10 isn't a power of two, so this round doubles as a play-in: trimming 10 down to the next
	// power of two below it (8) needs exactly 2 real matches (10-8) plus 6 byes (2*8-10) - see
	// BuildFirstRoundPairings's own doc comment for the general formula. That's a very different
	// shape from the old "just one leftover bye since 10 is even anyway" behavior (issue #20's
	// fix), which is exactly the point: every round from here on (round of 8, 4, 2) now needs no
	// bye at all, instead of byes trickling in one at a time across several later rounds.
	if len(pairs) != 8 {
		t.Fatalf("expected 8 ties (2 real matches + 6 byes), got %d", len(pairs))
	}
	assertByeCountAndFullCoverage(t, pairs, teamIds, 6)
}

func TestBuildFirstRoundPairings_OddCountGetsExactlyOneBye(t *testing.T) {
	teamIds := makeTeamIds(9)
	r := rand.New(rand.NewSource(1))

	pairs := BuildFirstRoundPairings(teamIds, r, nil)

	// 9 trims to the power of two below it (8) via a single real match (9-8=1) plus 7 byes
	// (2*8-9=7) - same formula as the 10-team case above, just happening to need only one match.
	if len(pairs) != 8 {
		t.Fatalf("expected 8 ties (1 real match + 7 byes), got %d", len(pairs))
	}
	assertByeCountAndFullCoverage(t, pairs, teamIds, 7)
}

func TestBuildFirstRoundPairings_PowerOfTwoHasNoByes(t *testing.T) {
	teamIds := makeTeamIds(8)
	r := rand.New(rand.NewSource(1))

	pairs := BuildFirstRoundPairings(teamIds, r, nil)

	if len(pairs) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}
	for _, pair := range pairs {
		if pair[0] == ByeTeamId || pair[1] == ByeTeamId {
			t.Fatalf("did not expect any byes for an exact power-of-two entrant count, got pair %v", pair)
		}
	}
}

// TestBuildFirstRoundPairings_TrimsAnyFieldToAPowerOfTwo is issue 6's "extend coverage to every
// odd/non-power-of-two field size, not just the first round's own" ask, applied to
// BuildFirstRoundPairings: for a range of field sizes, the round this produces must always trim
// the field down to exactly the next power of two below it (or leave it untouched if it's already
// one), with every entrant appearing exactly once and the right number of byes for the formula in
// BuildFirstRoundPairings's own doc comment (byeCount = 2*p - n, matchCount = n - p).
func TestBuildFirstRoundPairings_TrimsAnyFieldToAPowerOfTwo(t *testing.T) {
	for n := 2; n <= 20; n++ {
		teamIds := makeTeamIds(n)
		r := rand.New(rand.NewSource(int64(n)))

		pairs := BuildFirstRoundPairings(teamIds, r, nil)

		p := largestPowerOfTwoAtMost(n)
		wantByes := 0
		if p != n {
			wantByes = 2*p - n
		}
		assertByeCountAndFullCoverage(t, pairs, teamIds, wantByes)

		// Whether or not this round needed trimming, it must leave behind a winner count that's
		// itself a power of two - the whole point being that no round after this one ever needs
		// a bye again.
		winners := wantByes + (n-wantByes)/2
		if largestPowerOfTwoAtMost(winners) != winners {
			t.Errorf("n=%d: winners after this round = %d, want a power of two", n, winners)
		}
	}
}

func TestBuildFirstRoundPairings_SeededByePrefersHighestPoints(t *testing.T) {
	teamIds := makeTeamIds(9)
	seedPoints := map[int64]int{1: 10, 2: 90, 3: 20, 4: 15, 5: 5, 6: 1, 7: 2, 8: 3, 9: 4}
	r := rand.New(rand.NewSource(1))

	pairs := BuildFirstRoundPairings(teamIds, r, seedPoints)

	// 9 teams -> 7 byes, 1 real match (see the OddCountGetsExactlyOneBye test above). With seeding,
	// those 7 byes must go to the 7 highest-ranked teams by points, leaving only the two
	// lowest-ranked teams (6 and 7) to actually play - never the top-ranked team 2.
	byeTeams := map[int64]bool{}
	for _, pair := range pairs {
		if pair[1] == ByeTeamId {
			byeTeams[pair[0]] = true
		}
	}
	if len(byeTeams) != 7 {
		t.Fatalf("expected 7 byes, got %d", len(byeTeams))
	}
	if !byeTeams[2] {
		t.Errorf("expected the highest-seeded team (2, 90 points) to receive a bye, byes went to %v", byeTeams)
	}
	for _, lowSeed := range []int64{6, 7} {
		if byeTeams[lowSeed] {
			t.Errorf("expected the lowest-seeded teams (6, 7) to play the play-in match, not receive a bye; byes went to %v", byeTeams)
		}
	}
}

func TestBuildNextRoundPairings(t *testing.T) {
	winners := []int64{10, 20, 30, 40, 50, 60, 70, 80}
	pairs := BuildNextRoundPairings(winners, nil)

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
	pairs := BuildNextRoundPairings(winners, nil)

	byeCount := 0
	seen := make(map[int64]int)
	for _, pair := range pairs {
		if pair[1] == ByeTeamId {
			byeCount++
		}
		seen[pair[0]]++
		if pair[1] != ByeTeamId {
			seen[pair[1]]++
		}
	}
	if byeCount != 1 {
		t.Fatalf("expected exactly 1 bye for 5 winners, got %d", byeCount)
	}
	for _, team := range winners {
		if seen[team] != 1 {
			t.Errorf("team %d appears %d times, want exactly 1", team, seen[team])
		}
	}
}

// TestBuildNextRoundPairings_OddCountAtAnySize is issue 6's ask applied to BuildNextRoundPairings
// itself (an in-progress cup whose first round predates the play-in trim can still hand this an
// odd winner count) - every odd size from 3 up gets exactly one bye and every winner appears
// exactly once, not just a single hand-picked size.
func TestBuildNextRoundPairings_OddCountAtAnySize(t *testing.T) {
	for n := 3; n <= 21; n += 2 {
		winners := make([]int64, n)
		for i := range winners {
			winners[i] = int64(i + 1)
		}

		pairs := BuildNextRoundPairings(winners, nil)

		assertByeCountAndFullCoverage(t, pairs, winners, 1)
	}
}

func TestBuildNextRoundPairings_SeededByePrefersHighestPoints(t *testing.T) {
	winners := []int64{1, 2, 3, 4, 5}
	seedPoints := map[int64]int{1: 10, 2: 90, 3: 20, 4: 15, 5: 5}

	pairs := BuildNextRoundPairings(winners, seedPoints)

	gotBye := int64(-1)
	for _, pair := range pairs {
		if pair[1] == ByeTeamId {
			gotBye = pair[0]
		}
	}
	if gotBye != 2 {
		t.Errorf("expected the highest-seeded team (2, 90 points) to receive the bye, got team %d", gotBye)
	}
}

// assertByeCountAndFullCoverage checks pairs against teamIds generically: exactly wantByes byes,
// every id in teamIds appears exactly once across all pairs (as either side), and no id outside
// teamIds (or ByeTeamId itself) ever appears as a "real" entrant.
func assertByeCountAndFullCoverage(t *testing.T, pairs [][2]int64, teamIds []int64, wantByes int) {
	t.Helper()
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
	if byeCount != wantByes {
		t.Fatalf("expected %d byes, got %d", wantByes, byeCount)
	}
	if len(seen) != len(teamIds) {
		t.Fatalf("expected all %d teams to appear exactly once, got %d distinct teams: %v", len(teamIds), len(seen), seen)
	}
	for _, team := range teamIds {
		if seen[team] != 1 {
			t.Errorf("team %d appears %d times, want exactly 1", team, seen[team])
		}
	}
}
