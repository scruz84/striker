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

// balancedLineup builds a valid FormationBalanced (4-4-2) lineup: 11 players, one per player id
// 1..11, matching common.FormationPositionCounts(common.FormationBalanced) exactly.
func balancedLineup() []common.LineupPosition {
	counts := common.FormationPositionCounts(common.FormationBalanced)
	order := []int{
		common.GOALKEEPER, common.DEFENSE, common.LATERAL_DEFENSE_LEFT, common.LATERAL_DEFENSE_RIGHT,
		common.MIDFIELD, common.LATERAL_MIDFIELD_LEFT, common.LATERAL_MIDFIELD_RIGHT, common.STRIKER,
	}
	positions := make([]common.LineupPosition, 0, 11)
	nextPlayerId := int64(1)
	for _, position := range order {
		for i := 0; i < counts[position]; i++ {
			positions = append(positions, common.LineupPosition{PlayerId: nextPlayerId, Position: position})
			nextPlayerId++
		}
	}
	return positions
}

func fullRoster() *[]*common.Player {
	return rosterOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
}

func TestValidateLineup_Valid(t *testing.T) {
	if err := validateLineup(common.FormationBalanced, balancedLineup(), fullRoster(), nil); err != nil {
		t.Fatalf("expected valid lineup to pass, got error: %s", err)
	}
}

func TestValidateLineup_InvalidFormation(t *testing.T) {
	if err := validateLineup(99, balancedLineup(), fullRoster(), nil); err == nil {
		t.Fatal("expected error for an unrecognized formation")
	}
}

func TestValidateLineup_EveryFormationAccepts11DistinctPlayers(t *testing.T) {
	for _, formation := range []int{common.FormationBalanced, common.FormationDefensive, common.FormationAttacking} {
		counts := common.FormationPositionCounts(formation)
		order := []int{
			common.GOALKEEPER, common.DEFENSE, common.LATERAL_DEFENSE_LEFT, common.LATERAL_DEFENSE_RIGHT,
			common.MIDFIELD, common.LATERAL_MIDFIELD_LEFT, common.LATERAL_MIDFIELD_RIGHT, common.STRIKER,
		}
		positions := make([]common.LineupPosition, 0, 11)
		nextPlayerId := int64(1)
		for _, position := range order {
			for i := 0; i < counts[position]; i++ {
				positions = append(positions, common.LineupPosition{PlayerId: nextPlayerId, Position: position})
				nextPlayerId++
			}
		}
		if len(positions) != 11 {
			t.Fatalf("formation %d: expected 11 total players, got %d", formation, len(positions))
		}
		if err := validateLineup(formation, positions, fullRoster(), nil); err != nil {
			t.Fatalf("formation %d: expected valid lineup to pass, got error: %s", formation, err)
		}
	}
}

func TestValidateLineup_WrongCount(t *testing.T) {
	positions := balancedLineup()[:10]
	if err := validateLineup(common.FormationBalanced, positions, fullRoster(), nil); err == nil {
		t.Fatal("expected error for a lineup with fewer than 11 positions")
	}
}

func TestValidateLineup_DoesNotMatchChosenFormationShape(t *testing.T) {
	// A defensive (5-3-2) shape submitted while claiming the balanced (4-4-2) formation must
	// be rejected even though it also totals 11 players.
	defensivePositions := make([]common.LineupPosition, 0, 11)
	counts := common.FormationPositionCounts(common.FormationDefensive)
	order := []int{
		common.GOALKEEPER, common.DEFENSE, common.LATERAL_DEFENSE_LEFT, common.LATERAL_DEFENSE_RIGHT,
		common.MIDFIELD, common.LATERAL_MIDFIELD_LEFT, common.LATERAL_MIDFIELD_RIGHT, common.STRIKER,
	}
	nextPlayerId := int64(1)
	for _, position := range order {
		for i := 0; i < counts[position]; i++ {
			defensivePositions = append(defensivePositions, common.LineupPosition{PlayerId: nextPlayerId, Position: position})
			nextPlayerId++
		}
	}
	if err := validateLineup(common.FormationBalanced, defensivePositions, fullRoster(), nil); err == nil {
		t.Fatal("expected error when the submitted shape doesn't match the declared formation")
	}
}

func TestValidateLineup_DuplicatePlayer(t *testing.T) {
	positions := balancedLineup()
	positions[1].PlayerId = positions[0].PlayerId
	if err := validateLineup(common.FormationBalanced, positions, fullRoster(), nil); err == nil {
		t.Fatal("expected error for duplicate player")
	}
}

func TestValidateLineup_PlayerNotOnRoster(t *testing.T) {
	roster := rosterOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	positions := balancedLineup()
	positions[10].PlayerId = 999
	if err := validateLineup(common.FormationBalanced, positions, roster, nil); err == nil {
		t.Fatal("expected error for player not belonging to the team")
	}
}

func TestValidateLineup_InvalidPositionId(t *testing.T) {
	positions := balancedLineup()
	positions[0].Position = 42
	if err := validateLineup(common.FormationBalanced, positions, fullRoster(), nil); err == nil {
		t.Fatal("expected error for out-of-range position id")
	}
}

func TestValidateLineup_UnavailablePlayerRejected(t *testing.T) {
	positions := balancedLineup()
	unavailable := map[int64]bool{positions[3].PlayerId: true}
	if err := validateLineup(common.FormationBalanced, positions, fullRoster(), unavailable); err == nil {
		t.Fatal("expected error for a suspended/injured player")
	}
}
