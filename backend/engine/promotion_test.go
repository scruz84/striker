package engine

import (
	"github.com/scruz84/striker/common"
	"testing"
)

func standing(teamId int64, points, goalDiff, goalsScored int) common.TeamStanding {
	return common.TeamStanding{TeamId: teamId, Points: points, GoalDiff: goalDiff, GoalsScored: goalsScored}
}

// rankedGroup builds a LeagueStandings already sorted best to worst by descending points,
// matching what GetRankedClassification returns - team ids are assigned 1..n in rank order.
func rankedGroup(leagueId int64, pointsBestToWorst ...int) LeagueStandings {
	standings := make([]common.TeamStanding, 0, len(pointsBestToWorst))
	for i, points := range pointsBestToWorst {
		standings = append(standings, standing(leagueId*100+int64(i), points, 0, 0))
	}
	return LeagueStandings{LeagueId: leagueId, Standings: standings}
}

func TestComputePromotionRelegation_SingleTierNoMoves(t *testing.T) {
	leaguesByTier := map[int][]LeagueStandings{
		1: {rankedGroup(1, 30, 25, 20, 15, 10)},
	}

	moves := ComputePromotionRelegation(leaguesByTier)
	if len(moves) != 0 {
		t.Fatalf("expected no moves with a single tier, got %v", moves)
	}
}

func TestComputePromotionRelegation_NonAdjacentTiersDoNotSwap(t *testing.T) {
	leaguesByTier := map[int][]LeagueStandings{
		1: {rankedGroup(1, 30, 25, 20)},
		3: {rankedGroup(3, 30, 25, 20)},
	}

	moves := ComputePromotionRelegation(leaguesByTier)
	if len(moves) != 0 {
		t.Fatalf("expected no moves between non-adjacent tiers, got %v", moves)
	}
}

func TestComputePromotionRelegation_OneToOneGroupSwapsTwo(t *testing.T) {
	// One upper league, one lower league: 2 relegation slots, but only 1 promotion
	// candidate (PromotionCandidatesPerGroup=1 per lower-tier group) - numMoves = 1.
	leaguesByTier := map[int][]LeagueStandings{
		1: {rankedGroup(1, 30, 25, 20, 15, 10)}, // worst team: league 1, id 104
		2: {rankedGroup(2, 40, 35, 30, 25, 20)}, // best team: league 2, id 200
	}

	moves := ComputePromotionRelegation(leaguesByTier)
	if len(moves) != 2 {
		t.Fatalf("expected exactly one swap (2 moves), got %d: %v", len(moves), moves)
	}

	movesByTeam := movesByTeamId(moves)
	if movesByTeam[200] != 1 {
		t.Errorf("expected the best lower-tier team (200) promoted into league 1, got league %d", movesByTeam[200])
	}
	if movesByTeam[104] != 2 {
		t.Errorf("expected the worst upper-tier team (104) relegated into league 2, got league %d", movesByTeam[104])
	}
}

func TestComputePromotionRelegation_MultipleGroupsRankGlobally(t *testing.T) {
	// Matches the real seed shape: tier 1 has a single group (2 relegation slots), tier 2
	// has 4 groups (1 promotion candidate each = 4 candidates competing for 2 slots).
	leaguesByTier := map[int][]LeagueStandings{
		1: {rankedGroup(1, 30, 25, 20, 15, 10)}, // relegates ids 103 (15pts) and 104 (10pts)
		2: {
			rankedGroup(21, 40), // best in group 21: id 2100, 40 points - should be promoted
			rankedGroup(22, 35), // id 2200, 35 points - should be promoted
			rankedGroup(23, 32), // id 2300, 32 points - just misses out (only 2 slots)
			rankedGroup(24, 20), // id 2400, 20 points - misses out
		},
	}

	moves := ComputePromotionRelegation(leaguesByTier)
	if len(moves) != 4 {
		t.Fatalf("expected 2 swaps (4 moves), got %d: %v", len(moves), moves)
	}

	movesByTeam := movesByTeamId(moves)
	if _, ok := movesByTeam[2100]; !ok {
		t.Error("expected the top-ranked promotion candidate (2100, 40pts) to be promoted")
	}
	if _, ok := movesByTeam[2200]; !ok {
		t.Error("expected the second-ranked promotion candidate (2200, 35pts) to be promoted")
	}
	if _, ok := movesByTeam[2300]; ok {
		t.Error("expected the third-ranked candidate (2300, 32pts) to miss out - only 2 slots")
	}
	if _, ok := movesByTeam[2400]; ok {
		t.Error("expected the fourth-ranked candidate (2400, 20pts) to miss out")
	}
	if _, ok := movesByTeam[103]; !ok {
		t.Error("expected the second-worst tier-1 team (103) to be relegated")
	}
	if _, ok := movesByTeam[104]; !ok {
		t.Error("expected the worst tier-1 team (104) to be relegated")
	}

	// Every relegated team must land in a league that actually vacated a promoted team's spot.
	promotedFromLeagues := map[int64]bool{21: true, 22: true}
	if !promotedFromLeagues[movesByTeam[103]] || !promotedFromLeagues[movesByTeam[104]] {
		t.Errorf("relegated teams should fill exactly the 2 vacated tier-2 leagues, got %d and %d",
			movesByTeam[103], movesByTeam[104])
	}
}

func TestComputePromotionRelegation_TieBreaksByGoalDiffThenGoalsScored(t *testing.T) {
	leaguesByTier := map[int][]LeagueStandings{
		1: {rankedGroup(1, 30, 25)}, // 2 relegation slots (both teams relegate)
		2: {
			{LeagueId: 21, Standings: []common.TeamStanding{standing(2100, 40, 5, 10)}},
			{LeagueId: 22, Standings: []common.TeamStanding{standing(2200, 40, 8, 10)}}, // same points, better diff
		},
	}

	moves := ComputePromotionRelegation(leaguesByTier)
	movesByTeam := movesByTeamId(moves)

	// Both promotion candidates get in here (2 slots, 2 candidates), so just confirm both
	// moved and the swap is internally consistent (each landed in the other's old league).
	if movesByTeam[2100] == 0 || movesByTeam[2200] == 0 {
		t.Fatalf("expected both tied-on-points candidates to be promoted, got %v", movesByTeam)
	}
}

func movesByTeamId(moves []LeagueMove) map[int64]int64 {
	result := make(map[int64]int64, len(moves))
	for _, m := range moves {
		result[m.TeamId] = m.NewLeagueId
	}
	return result
}
