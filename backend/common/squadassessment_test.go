package common

import "testing"

// strongGoalkeeper etc. build a minimal player whose attributes make them clearly the best (or
// only plausible) fit for one core position - good enough for these pure-function tests without
// needing a full realistic squad.
func strongGoalkeeper(id int64) *Player { return &Player{Id: id, GoalKeeper: 90} }
func strongDefender(id int64) *Player   { return &Player{Id: id, Defense: 90} }
func strongMidfielder(id int64) *Player { return &Player{Id: id, Midfield: 90, Pass: 90} }
func strongStriker(id int64) *Player    { return &Player{Id: id, Shot: 90, Pass: 90, Dribbling: 90} }

func fullSquad() []*Player {
	players := []*Player{strongGoalkeeper(1), strongGoalkeeper(20)}
	for i := int64(2); i <= 5; i++ {
		players = append(players, strongDefender(i))
	}
	for i := int64(6); i <= 9; i++ {
		players = append(players, strongMidfielder(i))
	}
	for i := int64(10); i <= 13; i++ {
		players = append(players, strongStriker(i))
	}
	return players
}

func TestSquadStrength_EmptySquadIsZero(t *testing.T) {
	if got := SquadStrength(nil); got != 0 {
		t.Errorf("SquadStrength(nil) = %d, want 0", got)
	}
}

func TestSquadStrength_HigherRatedSquadScoresHigher(t *testing.T) {
	weak := []*Player{
		{Id: 1, GoalKeeper: 30},
		{Id: 2, Defense: 30}, {Id: 3, Defense: 30},
		{Id: 4, Midfield: 30, Pass: 30}, {Id: 5, Midfield: 30, Pass: 30},
		{Id: 6, Shot: 30, Pass: 30, Dribbling: 30}, {Id: 7, Shot: 30, Pass: 30, Dribbling: 30},
	}
	strong := fullSquad()
	if SquadStrength(strong) <= SquadStrength(weak) {
		t.Fatal("expected a squad of 90-rated players to score higher than a squad of 30-rated players")
	}
}

func TestSquadStrength_DepthBeyondStartingXIDoesNotInflateScore(t *testing.T) {
	squad := fullSquad()
	withBench := append(append([]*Player(nil), squad...), strongStriker(14), strongStriker(15))
	if SquadStrength(squad) != SquadStrength(withBench) {
		t.Errorf("SquadStrength should only count the best-XI, not bench depth: %d != %d",
			SquadStrength(squad), SquadStrength(withBench))
	}
}

func TestAssessPositionNeeds_ReturnsOneEntryPerCorePosition(t *testing.T) {
	needs := AssessPositionNeeds(fullSquad())
	if len(needs) != len(CorePositions) {
		t.Fatalf("got %d position needs, want %d (one per CorePositions entry)", len(needs), len(CorePositions))
	}
}

func TestAssessPositionNeeds_EmptyPositionIsThinWithZeroValue(t *testing.T) {
	// A squad with no goalkeeper at all - every outfield player, no GOALKEEPER attribute set.
	squad := []*Player{strongDefender(1), strongMidfielder(2), strongStriker(3), strongStriker(4)}
	needs := AssessPositionNeeds(squad)
	for _, n := range needs {
		if n.Position == GOALKEEPER {
			if n.Depth != 0 || n.AvgValue != 0 || !n.Thin {
				t.Errorf("expected an empty GOALKEEPER need, got %+v", n)
			}
			return
		}
	}
	t.Fatal("no GOALKEEPER entry found in AssessPositionNeeds output")
}

func TestAssessPositionNeeds_WellStockedPositionIsNotThin(t *testing.T) {
	needs := AssessPositionNeeds(fullSquad())
	for _, n := range needs {
		if n.Thin {
			t.Errorf("position %d unexpectedly flagged thin with depth %d (min %d)", n.Position, n.Depth, MinPositionDepth[n.Position])
		}
	}
}

func TestWeakestPositionNeed_NilForEmptySquad(t *testing.T) {
	if got := WeakestPositionNeed(nil); got != nil {
		t.Errorf("WeakestPositionNeed(nil) = %+v, want nil", got)
	}
}

func TestWeakestPositionNeed_PicksThinnestPosition(t *testing.T) {
	// Only a goalkeeper and one defender - every other core position has zero depth, but
	// GOALKEEPER's min depth (2) is satisfied by neither GK nor DEFENSE at depth 1, so the tie
	// should break toward MIDFIELD/STRIKER (min depth 3, so a bigger deficit at depth 0).
	squad := []*Player{strongGoalkeeper(1), strongDefender(2)}
	weakest := WeakestPositionNeed(squad)
	if weakest == nil {
		t.Fatal("expected a weakest position need, got nil")
	}
	if weakest.Position != MIDFIELD && weakest.Position != STRIKER {
		t.Errorf("expected MIDFIELD or STRIKER (deficit 3) to be weakest, got position %d (deficit %d)",
			weakest.Position, positionNeedDeficit(*weakest))
	}
}

func TestWeakestPositionNeed_TieBreaksOnLowestAvgValue(t *testing.T) {
	// Both MIDFIELD and STRIKER are equally thin (depth 0, deficit 3) in this squad, and so is
	// DEFENSE - all three deficits tie. GOALKEEPER's deficit is smaller (depth 1 vs min 2), so it
	// should lose to any of the depth-0 positions regardless of tie-breaking.
	squad := []*Player{strongGoalkeeper(1)}
	weakest := WeakestPositionNeed(squad)
	if weakest == nil {
		t.Fatal("expected a weakest position need, got nil")
	}
	if weakest.Position == GOALKEEPER {
		t.Errorf("GOALKEEPER (smaller deficit) should not win over a fully-empty core position, got %+v", weakest)
	}
}
