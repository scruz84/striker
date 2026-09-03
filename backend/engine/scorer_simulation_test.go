package engine

import (
	"github.com/scruz84/striker/common"
	"math/rand"
	"testing"
)

func TestGetPlayerForChance_ReturnsSentinelWhenNoOutfieldPlayersLeft(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	positions := map[int][]int64{
		common.GOALKEEPER: {1},
	}

	if player := getPlayerForChance(r, &positions); player != -1 {
		t.Fatalf("expected -1 when striker/midfield/defense are all empty, got %d", player)
	}
}

func TestGetPlayerForChance_FallsBackToDefenseWhenAttackIsEmpty(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	positions := map[int][]int64{
		common.DEFENSE: {42},
	}

	for i := 0; i < 20; i++ {
		if player := getPlayerForChance(r, &positions); player != 42 {
			t.Fatalf("expected the only remaining defender (42), got %d", player)
		}
	}
}
