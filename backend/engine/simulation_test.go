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
