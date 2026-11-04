package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// seedsInOrder lists a league's teams by ascending seed, failing if the seeds are not exactly 1..n.
func seedsInOrder(t *testing.T, seeds map[int64]int) []int64 {
	t.Helper()
	order := make([]int64, len(seeds))
	for teamId, seed := range seeds {
		if seed < 1 || seed > len(seeds) || order[seed-1] != 0 {
			t.Fatalf("seeds %v are not exactly 1..%d", seeds, len(seeds))
		}
		order[seed-1] = teamId
	}
	return order
}

func assertOrder(t *testing.T, what string, got []int64, want ...int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", what, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", what, got, want)
		}
	}
}

// tenTeamUpperAndLower is a 10-team top division (ids 100..109 in final order) over a 10-team
// second division (ids 200..209), with two relegation slots and two promotion slots.
func tenTeamUpperAndLower() ([]LeagueStandings, []LeagueMove) {
	upper := rankedGroup(1, 50, 45, 40, 35, 30, 25, 20, 15, 10, 5)
	lower := rankedGroup(2, 50, 45, 40, 35, 30, 25, 20, 15, 10, 5)
	moves := ComputePromotionRelegation(map[int][]LeagueStandings{1: {upper}, 2: {lower}},
		common.PromotionRelegationConfig{PromotionSlotsPerGroup: 2, RelegationSlotsPerGroup: 2})
	return []LeagueStandings{upper, lower}, moves
}

func TestComputeSeasonSeeds_PromotedTeamsStartAtTheBottom(t *testing.T) {
	leagues, moves := tenTeamUpperAndLower()
	seeds := ComputeSeasonSeeds(leagues, moves)

	// Last season's 1st to 8th keep places 1-8, then P1 (200) 9th and P2 (201) 10th.
	assertOrder(t, "top division", seedsInOrder(t, seeds[1]), 100, 101, 102, 103, 104, 105, 106, 107, 200, 201)
}

func TestComputeSeasonSeeds_RelegatedTeamsStartAtTheTop(t *testing.T) {
	leagues, moves := tenTeamUpperAndLower()
	seeds := ComputeSeasonSeeds(leagues, moves)

	// R9 (108) 1st, R10 (109) 2nd, then the teams that stayed (202..209) in their final order.
	assertOrder(t, "second division", seedsInOrder(t, seeds[2]), 108, 109, 202, 203, 204, 205, 206, 207, 208, 209)
}

func TestComputeSeasonSeeds_PromotedFromTwoGroups(t *testing.T) {
	upper := rankedGroup(1, 50, 45, 40, 35)
	groupA := LeagueStandings{LeagueId: 2, Standings: []common.TeamStanding{standing(201, 64, 20, 40), standing(202, 50, 0, 20)}}
	groupB := LeagueStandings{LeagueId: 3, Standings: []common.TeamStanding{standing(301, 70, 25, 45), standing(302, 40, 0, 20)}}
	moves := ComputePromotionRelegation(map[int][]LeagueStandings{1: {upper}, 2: {groupA, groupB}}, defaultConfig)
	seeds := ComputeSeasonSeeds([]LeagueStandings{upper, groupA, groupB}, moves)

	// Both group winners finished 1st: the 70-point team starts second to last, the 64-point one last.
	assertOrder(t, "top division", seedsInOrder(t, seeds[1]), 100, 101, 301, 201)
}

func TestComputeSeasonSeeds_LeagueWithNoPreviousSeason(t *testing.T) {
	upper := rankedGroup(1, 30, 20, 10)
	brandNew := LeagueStandings{LeagueId: 9}
	seeds := ComputeSeasonSeeds([]LeagueStandings{upper, brandNew}, nil)

	if _, ok := seeds[9]; ok {
		t.Errorf("a league with no standings got seeds %v, want none (its teams sort after every seeded team)", seeds[9])
	}
	assertOrder(t, "single tier", seedsInOrder(t, seeds[1]), 100, 101, 102)
}
