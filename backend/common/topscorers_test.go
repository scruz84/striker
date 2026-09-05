package common

import "testing"

func TestTallyGoalsByPlayer_CountsGoalsPerPlayer(t *testing.T) {
	rows := []string{"[10]", "[20]", "[10]", "[10]"}

	goals := tallyGoalsByPlayer(rows)

	if goals[10] != 3 {
		t.Fatalf("expected player 10 to have 3 goals, got %d", goals[10])
	}
	if goals[20] != 1 {
		t.Fatalf("expected player 20 to have 1 goal, got %d", goals[20])
	}
}

func TestTallyGoalsByPlayer_SkipsUnparseableOrEmptyRows(t *testing.T) {
	rows := []string{"not json", "[]", "[10]", "null"}

	goals := tallyGoalsByPlayer(rows)

	if len(goals) != 1 || goals[10] != 1 {
		t.Fatalf("expected only player 10's single goal to be tallied, got %+v", goals)
	}
}

func TestRankScorerStandings_SortsByGoalsDescending(t *testing.T) {
	standings := []ScorerStanding{
		{PlayerId: 1, Goals: 3},
		{PlayerId: 2, Goals: 7},
		{PlayerId: 3, Goals: 5},
	}

	ranked := rankScorerStandings(standings, 0)

	if ranked[0].PlayerId != 2 || ranked[1].PlayerId != 3 || ranked[2].PlayerId != 1 {
		t.Fatalf("expected order [2,3,1] by descending goals, got %v", []int64{ranked[0].PlayerId, ranked[1].PlayerId, ranked[2].PlayerId})
	}
}

func TestRankScorerStandings_TiesBrokenByPlayerIdForStableOrder(t *testing.T) {
	standings := []ScorerStanding{
		{PlayerId: 5, Goals: 4},
		{PlayerId: 2, Goals: 4},
	}

	ranked := rankScorerStandings(standings, 0)

	if ranked[0].PlayerId != 2 || ranked[1].PlayerId != 5 {
		t.Fatalf("expected tie broken by lower player id first, got %v", []int64{ranked[0].PlayerId, ranked[1].PlayerId})
	}
}

func TestRankScorerStandings_CapsAtLimit(t *testing.T) {
	standings := []ScorerStanding{
		{PlayerId: 1, Goals: 9},
		{PlayerId: 2, Goals: 8},
		{PlayerId: 3, Goals: 7},
	}

	ranked := rankScorerStandings(standings, 2)

	if len(ranked) != 2 {
		t.Fatalf("expected 2 results with limit 2, got %d", len(ranked))
	}
}

func TestRankScorerStandings_ZeroOrNegativeLimitMeansNoCap(t *testing.T) {
	standings := []ScorerStanding{{PlayerId: 1, Goals: 1}, {PlayerId: 2, Goals: 1}, {PlayerId: 3, Goals: 1}}

	if len(rankScorerStandings(standings, 0)) != 3 {
		t.Fatalf("expected limit 0 to mean no cap")
	}
	if len(rankScorerStandings(standings, -1)) != 3 {
		t.Fatalf("expected a negative limit to mean no cap")
	}
}
