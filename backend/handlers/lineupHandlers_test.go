package handlers

import (
	"github.com/scruz84/striker/common"
	"testing"
)

func rosterOf(playerIds ...int64) *[]*common.Player {
	players := make([]*common.Player, 0, len(playerIds))
	for _, id := range playerIds {
		players = append(players, &common.Player{Id: id})
	}
	return &players
}

func fullValidLineup() []common.LineupPosition {
	positions := make([]common.LineupPosition, 0, 10)
	for pos := common.GOALKEEPER; pos <= common.LATERAL_STRIKER_RIGHT; pos++ {
		positions = append(positions, common.LineupPosition{PlayerId: int64(pos + 1), Position: pos})
	}
	return positions
}

func TestValidateLineup_Valid(t *testing.T) {
	roster := rosterOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	if err := validateLineup(fullValidLineup(), roster); err != nil {
		t.Fatalf("expected valid lineup to pass, got error: %s", err)
	}
}

func TestValidateLineup_WrongCount(t *testing.T) {
	roster := rosterOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	positions := fullValidLineup()[:9]
	if err := validateLineup(positions, roster); err == nil {
		t.Fatal("expected error for lineup with fewer than 10 positions")
	}
}

func TestValidateLineup_DuplicatePosition(t *testing.T) {
	roster := rosterOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	positions := fullValidLineup()
	positions[1].Position = positions[0].Position
	if err := validateLineup(positions, roster); err == nil {
		t.Fatal("expected error for duplicate position")
	}
}

func TestValidateLineup_DuplicatePlayer(t *testing.T) {
	roster := rosterOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	positions := fullValidLineup()
	positions[1].PlayerId = positions[0].PlayerId
	if err := validateLineup(positions, roster); err == nil {
		t.Fatal("expected error for duplicate player")
	}
}

func TestValidateLineup_PlayerNotOnRoster(t *testing.T) {
	roster := rosterOf(1, 2, 3, 4, 5, 6, 7, 8, 9)
	positions := fullValidLineup()
	positions[9].PlayerId = 999
	if err := validateLineup(positions, roster); err == nil {
		t.Fatal("expected error for player not belonging to the team")
	}
}

func TestValidateLineup_InvalidPositionId(t *testing.T) {
	roster := rosterOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	positions := fullValidLineup()
	positions[0].Position = 42
	if err := validateLineup(positions, roster); err == nil {
		t.Fatal("expected error for out-of-range position id")
	}
}
