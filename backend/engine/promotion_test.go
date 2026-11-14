package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

func standing(teamId int64, points, goalDiff, goalsScored int) common.TeamStanding {
	return common.TeamStanding{TeamId: teamId, Points: points, GoalDiff: goalDiff, GoalsScored: goalsScored}
}

// rankedGroup builds a LeagueStandings already sorted best to worst by descending points,
// matching what GetRankedClassification returns - team ids are assigned leagueId*100 + 0..n-1 in
// rank order.
func rankedGroup(leagueId int64, pointsBestToWorst ...int) LeagueStandings {
	standings := make([]common.TeamStanding, 0, len(pointsBestToWorst))
	for i, points := range pointsBestToWorst {
		standings = append(standings, standing(leagueId*100+int64(i), points, 0, 0))
	}
	return LeagueStandings{LeagueId: leagueId, Standings: standings}
}

// group is rankedGroup with ids leagueId*100 + 1..n, so a team's id names its league and its final
// position.
func group(leagueId int64, pointsBestToWorst ...int) LeagueStandings {
	standings := make([]common.TeamStanding, 0, len(pointsBestToWorst))
	for i, points := range pointsBestToWorst {
		standings = append(standings, standing(leagueId*100+int64(i+1), points, 0, 0))
	}
	return LeagueStandings{LeagueId: leagueId, Standings: standings}
}

func movesByTeamId(moves []LeagueMove) map[int64]int64 {
	result := make(map[int64]int64, len(moves))
	for _, m := range moves {
		result[m.TeamId] = m.NewLeagueId
	}
	return result
}

// ten is a 10-team group's points table, best to worst.
var ten = []int{50, 45, 40, 35, 30, 25, 20, 15, 10, 5}

func TestComputeDirectMoves_SingleTierNoMoves(t *testing.T) {
	moves := ComputeDirectMoves(map[int][]LeagueStandings{1: {group(1, ten...)}})
	if len(moves) != 0 {
		t.Fatalf("expected no moves with a single tier, got %v", moves)
	}
}

func TestComputeDirectMoves_NonAdjacentTiersDoNotSwap(t *testing.T) {
	moves := ComputeDirectMoves(map[int][]LeagueStandings{1: {group(1, ten...)}, 3: {group(3, ten...)}})
	if len(moves) != 0 {
		t.Fatalf("expected no moves between non-adjacent tiers, got %v", moves)
	}
}

// Spec scenario "Tier 1 and tier 2": winners A (50) and B (47); tier 1's 10th X and 9th Y.
func TestComputeDirectMoves_TierOneAndTwo(t *testing.T) {
	groupA := group(21, 50, 45, 40, 35, 30, 25, 20, 15, 10, 5) // A = 2101
	groupB := group(22, 47, 44, 40, 35, 30, 25, 20, 15, 10, 5) // B = 2201
	moves := ComputeDirectMoves(map[int][]LeagueStandings{1: {group(1, ten...)}, 2: {groupA, groupB}})
	if len(moves) != 4 {
		t.Fatalf("expected 2 swaps (4 moves), got %v", moves)
	}
	byTeam := movesByTeamId(moves)
	const a, b, x, y = 2101, 2201, 110, 109
	if byTeam[a] != 1 || byTeam[b] != 1 {
		t.Errorf("A and B should go up to league 1: %v", byTeam)
	}
	// A (best) takes X's (worst) place: X goes to A's group, Y to B's.
	if byTeam[x] != 21 || byTeam[y] != 22 {
		t.Errorf("X should go to A's group 21 and Y to B's group 22, got X->%d Y->%d", byTeam[x], byTeam[y])
	}
	for _, m := range moves {
		if m.ViaPlayoff {
			t.Errorf("a direct move is marked as a playoff move: %+v", m)
		}
	}
}

func TestComputeDirectMoves_BottomTierRelegatesNobody(t *testing.T) {
	leagues := map[int][]LeagueStandings{1: {group(1, ten...)}, 2: {group(21, ten...), group(22, ten...)}}
	for _, m := range ComputeDirectMoves(leagues) {
		if !m.Promoted && (m.TeamId/100 == 21 || m.TeamId/100 == 22) {
			t.Errorf("a bottom-tier team was relegated: %+v", m)
		}
	}
}

// Spec scenario "Tier 1 against tier 2": tier 1's 8th P and 7th Q; runners-up R (45) and S (41).
func TestComputePlayoffPairings_TierOneAgainstTierTwo(t *testing.T) {
	groupR := group(21, 50, 45, 40, 35, 30, 25, 20, 15, 10, 5) // R = 2102 (45)
	groupS := group(22, 47, 41, 40, 35, 30, 25, 20, 15, 10, 5) // S = 2202 (41)
	pairings := ComputePlayoffPairings(map[int][]LeagueStandings{1: {group(1, ten...)}, 2: {groupS, groupR}})
	const p, q, r, s = 108, 107, 2102, 2202
	if len(pairings) != 2 {
		t.Fatalf("expected 2 ties, got %v", pairings)
	}
	if pairings[0].HomeTeamId != p || pairings[0].AwayTeamId != r || pairings[1].HomeTeamId != q || pairings[1].AwayTeamId != s {
		t.Errorf("expected P vs R then Q vs S, got %+v", pairings)
	}
	if pairings[0].HomeLeagueId != 1 || pairings[0].AwayLeagueId != 21 || pairings[0].UpperTier != 1 {
		t.Errorf("tie leagues/tier wrong: %+v", pairings[0])
	}
}

// Spec scenario "Two upper groups": tier 2's relegation-playoff teams 8th (20), 8th (18), 7th (22)
// and 7th (25) meet tier 3's four runners-up; the 8th on 18 faces the best runner-up.
func TestComputePlayoffPairings_TwoUpperGroups(t *testing.T) {
	g1 := group(21, 60, 55, 50, 45, 40, 35, 25, 20, 10, 5) // 7th 25 (2107), 8th 20 (2108)
	g2 := group(22, 60, 55, 50, 45, 40, 35, 22, 18, 10, 5) // 7th 22 (2207), 8th 18 (2208)
	lower := []LeagueStandings{
		group(31, 60, 30, 20, 19, 18, 17, 16, 15, 14, 13), // runner-up 3102 (30)
		group(32, 60, 44, 20, 19, 18, 17, 16, 15, 14, 13), // runner-up 3202 (44) - best
		group(33, 60, 40, 20, 19, 18, 17, 16, 15, 14, 13), // 3302 (40)
		group(34, 60, 35, 20, 19, 18, 17, 16, 15, 14, 13), // 3402 (35)
	}
	pairings := ComputePlayoffPairings(map[int][]LeagueStandings{1: {group(1, ten...)}, 2: {g1, g2}, 3: lower})
	tier2 := make([]PlayoffPairing, 0)
	for _, p := range pairings {
		if p.UpperTier == 2 {
			tier2 = append(tier2, p)
		}
	}
	if len(tier2) != 4 {
		t.Fatalf("expected 4 tier-2/3 ties, got %+v", tier2)
	}
	wantHome := []int64{2208, 2108, 2207, 2107}
	wantAway := []int64{3202, 3302, 3402, 3102}
	for i := range tier2 {
		if tier2[i].HomeTeamId != wantHome[i] || tier2[i].AwayTeamId != wantAway[i] {
			t.Errorf("tie %d = %d vs %d, want %d vs %d", i, tier2[i].HomeTeamId, tier2[i].AwayTeamId, wantHome[i], wantAway[i])
		}
	}
}

func TestApplyPlayoffOutcomes(t *testing.T) {
	pairing := PlayoffPairing{UpperTier: 1, HomeTeamId: 108, HomeLeagueId: 1, AwayTeamId: 2102, AwayLeagueId: 21}
	// Spec scenario "Lower team wins": R takes P's league and P goes to R's group.
	moves := ApplyPlayoffOutcomes([]PlayoffResult{{PlayoffPairing: pairing, WinnerTeamId: 2102}})
	byTeam := movesByTeamId(moves)
	if len(moves) != 2 || byTeam[2102] != 1 || byTeam[108] != 21 {
		t.Fatalf("lower team win: %+v", moves)
	}
	for _, m := range moves {
		if !m.ViaPlayoff || m.Promoted != (m.TeamId == 2102) {
			t.Errorf("playoff move flags wrong: %+v", m)
		}
	}
	// Spec scenario "Upper team wins": both stay.
	if moves := ApplyPlayoffOutcomes([]PlayoffResult{{PlayoffPairing: pairing, WinnerTeamId: 108}}); len(moves) != 0 {
		t.Errorf("upper team win should move nobody, got %+v", moves)
	}
}

// Spec scenario "Season in progress at deploy": a 1/4/16 country swaps tier 1's 9th and 10th
// with the best 2 of the 4 tier-2 winners, and plays no playoff.
func TestComputeSeasonMoves_Legacy(t *testing.T) {
	leagues := map[int][]LeagueStandings{
		1: {group(1, ten...)},
		2: {group(21, 40), group(22, 35), group(23, 32), group(24, 20)},
	}
	moves := ComputeSeasonMoves(leagues, true, []PlayoffResult{{PlayoffPairing: PlayoffPairing{HomeTeamId: 108, AwayTeamId: 2101}, WinnerTeamId: 2101}})
	byTeam := movesByTeamId(moves)
	if len(moves) != 4 {
		t.Fatalf("expected 2 swaps, got %+v", moves)
	}
	if byTeam[2101] != 1 || byTeam[2201] != 1 {
		t.Errorf("the best two winners (2101, 2201) should go up: %v", byTeam)
	}
	if byTeam[110] != 21 || byTeam[109] != 22 {
		t.Errorf("10th should take the best winner's group and 9th the second's: %v", byTeam)
	}
	if _, moved := byTeam[2301]; moved {
		t.Error("the third-best winner should stay")
	}
}

func TestComputeDirectMoves_TieBreaksByGoalDiffThenGoalsScored(t *testing.T) {
	g1 := LeagueStandings{LeagueId: 21, Standings: []common.TeamStanding{standing(1, 30, 5, 20)}}
	g2 := LeagueStandings{LeagueId: 22, Standings: []common.TeamStanding{standing(2, 30, 8, 10)}}
	upper := group(1, ten[:9]...) // only 9 teams: the 10th place is missing, so only 9th goes down
	upper.Standings = upper.Standings[:9]
	moves := ComputeDirectMoves(map[int][]LeagueStandings{1: {upper}, 2: {g1, g2}})
	byTeam := movesByTeamId(moves)
	// Two relegation places (8th, 9th of 9) and two winners: team 2 (better goal difference)
	// takes the worst (9th) team's place.
	if byTeam[2] != 1 || byTeam[109] != 22 {
		t.Errorf("tie-break by goal difference failed: %v", byTeam)
	}
}
