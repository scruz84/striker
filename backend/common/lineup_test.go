package common

import "testing"

func makePlayer(id int64, skill int) *Player {
	// A flat skill for every stat keeps the derived position values (GetLateralDefenseValue
	// etc.) proportional to it too, so "higher skill" unambiguously means "should be picked
	// first" regardless of which slot is being filled.
	return &Player{
		Id: id, GoalKeeper: skill, Defense: skill, Lateral: skill, Pass: skill,
		Midfield: skill, Dribbling: skill, Shot: skill,
	}
}

func TestSelectBestLineup_FillsEverySlotWhenEnoughPlayers(t *testing.T) {
	players := make([]*Player, 0, 18)
	for i := int64(1); i <= 18; i++ {
		players = append(players, makePlayer(i, int(i)))
	}

	lineup := SelectBestLineup(players)

	total := 0
	seen := make(map[int64]bool)
	for _, ids := range lineup {
		for _, id := range ids {
			if seen[id] {
				t.Fatalf("player %d selected more than once", id)
			}
			seen[id] = true
			total++
		}
	}
	if total != len(bestLineupSlots) {
		t.Fatalf("expected %d players selected, got %d", len(bestLineupSlots), total)
	}
}

func TestSelectBestLineup_PrefersHigherSkill(t *testing.T) {
	// Only two players and one goalkeeper slot in the pool - the better one must win it.
	best := makePlayer(1, 90)
	worst := makePlayer(2, 10)

	lineup := SelectBestLineup([]*Player{worst, best})

	if len(lineup[GOALKEEPER]) != 1 || lineup[GOALKEEPER][0] != best.Id {
		t.Fatalf("expected player %d as goalkeeper, got %v", best.Id, lineup[GOALKEEPER])
	}
}

func TestSelectBestLineup_FewerPlayersThanSlotsDoesNotPanic(t *testing.T) {
	players := []*Player{makePlayer(1, 50), makePlayer(2, 40), makePlayer(3, 30)}

	lineup := SelectBestLineup(players)

	total := 0
	for _, ids := range lineup {
		total += len(ids)
	}
	if total != len(players) {
		t.Fatalf("expected all %d players placed, got %d", len(players), total)
	}
}

func TestSelectBestLineup_EmptyPlayersDoesNotPanic(t *testing.T) {
	lineup := SelectBestLineup(nil)
	if len(lineup) != 0 {
		t.Fatalf("expected an empty lineup, got %v", lineup)
	}
}

func TestFlattenLineup(t *testing.T) {
	lineup := map[int][]int64{
		GOALKEEPER: {1},
		DEFENSE:    {2, 3},
	}

	positions := FlattenLineup(lineup)

	if len(positions) != 3 {
		t.Fatalf("expected 3 flattened positions, got %d", len(positions))
	}
	byPlayer := make(map[int64]int)
	for _, p := range positions {
		byPlayer[p.PlayerId] = p.Position
	}
	if byPlayer[1] != GOALKEEPER || byPlayer[2] != DEFENSE || byPlayer[3] != DEFENSE {
		t.Fatalf("unexpected flattened positions: %v", positions)
	}
}

func TestPositionValue_CoversEveryPosition(t *testing.T) {
	p := Player{GoalKeeper: 10, Defense: 20, Lateral: 30, Pass: 40, Midfield: 50, Dribbling: 60, Shot: 70}

	allPositions := []int{
		GOALKEEPER, DEFENSE, LATERAL_DEFENSE_LEFT, LATERAL_DEFENSE_RIGHT,
		MIDFIELD, LATERAL_MIDFIELD_LEFT, LATERAL_MIDFIELD_RIGHT,
		STRIKER, LATERAL_STRIKER_LEFT, LATERAL_STRIKER_RIGHT,
	}
	for _, position := range allPositions {
		if PositionValue(position, p) <= 0 {
			t.Errorf("expected a positive value for position %d, got %d", position, PositionValue(position, p))
		}
	}
}

func TestPositionValue_HigherSkillScoresHigher(t *testing.T) {
	weak := Player{GoalKeeper: 10, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10}
	strong := Player{GoalKeeper: 90, Defense: 90, Lateral: 90, Pass: 90, Midfield: 90, Dribbling: 90, Shot: 90}

	if PositionValue(STRIKER, strong) <= PositionValue(STRIKER, weak) {
		t.Fatal("expected the stronger player to score higher at striker")
	}
}
