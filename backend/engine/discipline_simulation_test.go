package engine

import (
	"github.com/scruz84/striker/common"
	"math/rand"
	"testing"
)

func TestRemovePlayerFromPosition(t *testing.T) {
	positions := map[int][]int64{
		common.DEFENSE: {10, 20, 30},
	}

	removePlayerFromPosition(&positions, common.DEFENSE, 20)

	remaining := positions[common.DEFENSE]
	if len(remaining) != 2 {
		t.Fatalf("expected 2 players left, got %d: %v", len(remaining), remaining)
	}
	for _, id := range remaining {
		if id == 20 {
			t.Fatalf("player 20 should have been removed, still present: %v", remaining)
		}
	}
}

func TestPickOutfieldPlayer_ExcludesGoalkeeper(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	positions := map[int][]int64{
		common.GOALKEEPER: {1},
		common.STRIKER:    {2},
	}

	for i := 0; i < 50; i++ {
		player, position := pickOutfieldPlayer(r, &positions)
		if player == 1 || position == common.GOALKEEPER {
			t.Fatalf("goalkeeper must never be picked for a card, got player %d position %d", player, position)
		}
	}
}

func TestPickOutfieldPlayer_NoEligiblePlayers(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	positions := map[int][]int64{
		common.GOALKEEPER: {1},
	}

	player, _ := pickOutfieldPlayer(r, &positions)
	if player != -1 {
		t.Fatalf("expected -1 when only the goalkeeper remains, got %d", player)
	}
}

func TestCalculateLateralFunctions_EmptySliceDoesNotPanic(t *testing.T) {
	players := map[int64]common.Player{}

	if v := calculateLateralDefense(nil, nil, &players, 2); v != 0 {
		t.Fatalf("expected 0 for empty lateral defense, got %v", v)
	}
	if v := calculateLateralMidfield(nil, nil, &players, 2); v != 0 {
		t.Fatalf("expected 0 for empty lateral midfield, got %v", v)
	}
	if v := calculateLateralAttack(nil, nil, &players); v != 0 {
		t.Fatalf("expected 0 for empty lateral attack, got %v", v)
	}
}
