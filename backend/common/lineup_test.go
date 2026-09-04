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
