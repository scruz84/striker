package engine

import (
	"github.com/scruz84/striker/common"
	"testing"
)

func TestFilterOutfieldPlayers_RemovesExcludedIds(t *testing.T) {
	positions := map[int][]int64{
		common.GOALKEEPER: {1},
		common.DEFENSE:    {10, 20},
		common.MIDFIELD:   {30},
		common.STRIKER:    {40, 50},
	}

	filterOutfieldPlayers(&positions, map[int64]bool{20: true, 40: true})

	if got := positions[common.DEFENSE]; len(got) != 1 || got[0] != 10 {
		t.Fatalf("expected only player 10 left in defense, got %v", got)
	}
	if got := positions[common.STRIKER]; len(got) != 1 || got[0] != 50 {
		t.Fatalf("expected only player 50 left at striker, got %v", got)
	}
	if got := positions[common.MIDFIELD]; len(got) != 1 || got[0] != 30 {
		t.Fatalf("expected midfield untouched, got %v", got)
	}
}

func TestFilterOutfieldPlayers_NeverTouchesGoalkeeper(t *testing.T) {
	positions := map[int][]int64{
		common.GOALKEEPER: {1},
	}

	// Even if (somehow) the goalkeeper is flagged unavailable, filterOutfieldPlayers only
	// walks ALL_OUTFIELD_POSITIONS, so the goalkeeper slot must survive untouched.
	filterOutfieldPlayers(&positions, map[int64]bool{1: true})

	if got := positions[common.GOALKEEPER]; len(got) != 1 || got[0] != 1 {
		t.Fatalf("goalkeeper must never be filtered out, got %v", got)
	}
}

func TestFilterOutfieldPlayers_NoExclusionsIsNoOp(t *testing.T) {
	positions := map[int][]int64{
		common.DEFENSE: {10, 20},
	}

	filterOutfieldPlayers(&positions, nil)

	if got := positions[common.DEFENSE]; len(got) != 2 {
		t.Fatalf("expected no change with an empty exclusion set, got %v", got)
	}
}

func benchPlayer(id int64, skill int) *common.Player {
	return &common.Player{
		Id: id, GoalKeeper: skill, Defense: skill, Lateral: skill, Pass: skill,
		Midfield: skill, Dribbling: skill, Shot: skill,
	}
}

func TestBackfillEmptyPositions_FillsBackUpToOriginalCount(t *testing.T) {
	// Defense started with 2 players; one (id 20) was excluded as unavailable, leaving 1.
	positions := map[int][]int64{
		common.DEFENSE: {10},
	}
	wantCounts := map[int]int{common.DEFENSE: 2}
	squad := []*common.Player{
		benchPlayer(10, 50), // already fielded
		benchPlayer(20, 40), // the unavailable starter - must not be picked
		benchPlayer(30, 30), // bench option
		benchPlayer(40, 60), // best bench option
	}
	unavailable := map[int64]bool{20: true}

	backfillEmptyPositions(&positions, wantCounts, squad, unavailable)

	got := positions[common.DEFENSE]
	if len(got) != 2 {
		t.Fatalf("expected defense restored to 2 players, got %v", got)
	}
	if got[0] != 10 || got[1] != 40 {
		t.Fatalf("expected [10 40] (original starter plus the best available bench option), got %v", got)
	}
}

func TestBackfillEmptyPositions_NeverPicksAPlayerTwice(t *testing.T) {
	positions := map[int][]int64{
		common.DEFENSE:  {},
		common.MIDFIELD: {},
	}
	wantCounts := map[int]int{common.DEFENSE: 1, common.MIDFIELD: 1}
	// Only one eligible player exists - it can fill exactly one of the two empty positions,
	// not both.
	squad := []*common.Player{benchPlayer(1, 50)}

	backfillEmptyPositions(&positions, wantCounts, squad, nil)

	total := len(positions[common.DEFENSE]) + len(positions[common.MIDFIELD])
	if total != 1 {
		t.Fatalf("expected exactly one position filled from a single eligible player, got defense=%v midfield=%v",
			positions[common.DEFENSE], positions[common.MIDFIELD])
	}
}

func TestBackfillEmptyPositions_LeavesPositionShortWhenSquadExhausted(t *testing.T) {
	positions := map[int][]int64{common.STRIKER: {}}
	wantCounts := map[int]int{common.STRIKER: 2}
	squad := []*common.Player{benchPlayer(1, 50)}

	backfillEmptyPositions(&positions, wantCounts, squad, nil)

	if got := positions[common.STRIKER]; len(got) != 1 {
		t.Fatalf("expected striker left short at 1 (only one eligible player existed), got %v", got)
	}
}

func TestBackfillEmptyPositions_AlreadyAtWantCountIsNoOp(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {1, 2}}
	wantCounts := map[int]int{common.MIDFIELD: 2}
	squad := []*common.Player{benchPlayer(1, 50), benchPlayer(2, 50), benchPlayer(3, 99)}

	backfillEmptyPositions(&positions, wantCounts, squad, nil)

	if got := positions[common.MIDFIELD]; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("expected midfield untouched when already at its original count, got %v", got)
	}
}
