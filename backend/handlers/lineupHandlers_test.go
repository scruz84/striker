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

// squadWithBench is fullRoster() (players 1-11, the starting XI in balancedLineup()) plus
// players 12+ available on the bench for substitution tests.
func squadWithBench(benchIds ...int64) *[]*common.Player {
	ids := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	ids = append(ids, benchIds...)
	return rosterOf(ids...)
}

func TestValidateSubstitutions_Valid(t *testing.T) {
	subs := []common.LineupSubstitution{
		{PlayerOutId: 1, PlayerInId: 12, Minute: 46},
		{PlayerOutId: 2, PlayerInId: 13, Minute: 60},
		{PlayerOutId: 3, PlayerInId: 14, Minute: 75},
	}
	if err := validateSubstitutions(balancedLineup(), subs, squadWithBench(12, 13, 14), nil); err != nil {
		t.Fatalf("expected valid substitutions to pass, got error: %s", err)
	}
}

func TestValidateSubstitutions_EmptyIsValid(t *testing.T) {
	if err := validateSubstitutions(balancedLineup(), nil, squadWithBench(), nil); err != nil {
		t.Fatalf("expected no scheduled substitutions to be valid, got error: %s", err)
	}
}

func TestValidateSubstitutions_RejectsMoreThanMax(t *testing.T) {
	subs := []common.LineupSubstitution{
		{PlayerOutId: 1, PlayerInId: 12, Minute: 46},
		{PlayerOutId: 2, PlayerInId: 13, Minute: 50},
		{PlayerOutId: 3, PlayerInId: 14, Minute: 60},
		{PlayerOutId: 4, PlayerInId: 15, Minute: 70},
	}
	if err := validateSubstitutions(balancedLineup(), subs, squadWithBench(12, 13, 14, 15), nil); err == nil {
		t.Fatal("expected error for more than the maximum allowed substitutions")
	}
}

func TestValidateSubstitutions_RejectsMinuteOutOfRange(t *testing.T) {
	for _, minute := range []int{0, 91, -1} {
		subs := []common.LineupSubstitution{{PlayerOutId: 1, PlayerInId: 12, Minute: minute}}
		if err := validateSubstitutions(balancedLineup(), subs, squadWithBench(12), nil); err == nil {
			t.Fatalf("expected error for out-of-range minute %d", minute)
		}
	}
}

func TestValidateSubstitutions_RejectsOutgoingPlayerNotInLineup(t *testing.T) {
	subs := []common.LineupSubstitution{{PlayerOutId: 999, PlayerInId: 12, Minute: 46}}
	if err := validateSubstitutions(balancedLineup(), subs, squadWithBench(12), nil); err == nil {
		t.Fatal("expected error when the outgoing player isn't a starter")
	}
}

func TestValidateSubstitutions_RejectsIncomingPlayerAlreadyStarting(t *testing.T) {
	subs := []common.LineupSubstitution{{PlayerOutId: 1, PlayerInId: 2, Minute: 46}}
	if err := validateSubstitutions(balancedLineup(), subs, fullRoster(), nil); err == nil {
		t.Fatal("expected error when the incoming player is already in the starting lineup")
	}
}

func TestValidateSubstitutions_RejectsIncomingPlayerNotOnRoster(t *testing.T) {
	subs := []common.LineupSubstitution{{PlayerOutId: 1, PlayerInId: 999, Minute: 46}}
	if err := validateSubstitutions(balancedLineup(), subs, fullRoster(), nil); err == nil {
		t.Fatal("expected error when the incoming player doesn't belong to the team")
	}
}

func TestValidateSubstitutions_RejectsUnavailableIncomingPlayer(t *testing.T) {
	subs := []common.LineupSubstitution{{PlayerOutId: 1, PlayerInId: 12, Minute: 46}}
	unavailable := map[int64]bool{12: true}
	if err := validateSubstitutions(balancedLineup(), subs, squadWithBench(12), unavailable); err == nil {
		t.Fatal("expected error for a suspended/injured incoming player")
	}
}

func TestValidateSubstitutions_RejectsSamePlayerSubbedOutTwice(t *testing.T) {
	subs := []common.LineupSubstitution{
		{PlayerOutId: 1, PlayerInId: 12, Minute: 46},
		{PlayerOutId: 1, PlayerInId: 13, Minute: 60},
	}
	if err := validateSubstitutions(balancedLineup(), subs, squadWithBench(12, 13), nil); err == nil {
		t.Fatal("expected error when the same player is scheduled to be substituted out twice")
	}
}

func TestValidateSubstitutions_RejectsSamePlayerSubbedInTwice(t *testing.T) {
	subs := []common.LineupSubstitution{
		{PlayerOutId: 1, PlayerInId: 12, Minute: 46},
		{PlayerOutId: 2, PlayerInId: 12, Minute: 60},
	}
	if err := validateSubstitutions(balancedLineup(), subs, squadWithBench(12), nil); err == nil {
		t.Fatal("expected error when the same player is scheduled to be substituted in twice")
	}
}
