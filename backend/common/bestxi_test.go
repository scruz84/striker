package common

import "testing"

func TestSelectBestXi(t *testing.T) {
	candidates := []BestXiCandidate{
		{PlayerId: 1, Line: BestXiLineGoalkeeper, Rating: 7},
		{PlayerId: 2, Line: BestXiLineGoalkeeper, Rating: 8},
		{PlayerId: 10, Line: BestXiLineDefence, Rating: 6},
		{PlayerId: 11, Line: BestXiLineDefence, Rating: 7},
		{PlayerId: 12, Line: BestXiLineDefence, Rating: 7, Minutes: 90},
		{PlayerId: 13, Line: BestXiLineDefence, Rating: 7, Minutes: 90, Goals: 1},
		{PlayerId: 14, Line: BestXiLineDefence, Rating: 5},
		{PlayerId: 20, Line: BestXiLineMidfield, Rating: 9},
		{PlayerId: 30, Line: BestXiLineAttack, Rating: 6},
		{PlayerId: 31, Line: BestXiLineAttack, Rating: 6},
		{PlayerId: 32, Line: BestXiLineAttack, Rating: 6},
	}
	picks := SelectBestXi(candidates)
	slots := map[int]int64{}
	for _, p := range picks {
		slots[p.Slot] = p.PlayerId
	}
	want := map[int]int64{1: 2, 2: 13, 3: 12, 4: 11, 5: 10, 6: 20, 10: 30, 11: 31}
	if len(slots) != len(want) {
		t.Fatalf("picks = %+v, want %v", picks, want)
	}
	for slot, id := range want {
		if slots[slot] != id {
			t.Errorf("slot %d = player %d, want %d (picks %+v)", slot, slots[slot], id, picks)
		}
	}
}

func TestBestXiLineOf(t *testing.T) {
	cases := map[int]int{GOALKEEPER: 1, LATERAL_DEFENSE_LEFT: 2, DEFENSE: 2, MIDFIELD: 3, LATERAL_MIDFIELD_RIGHT: 3, STRIKER: 4, LATERAL_STRIKER_LEFT: 4}
	for position, line := range cases {
		if got := BestXiLineOf(position); got != line {
			t.Errorf("BestXiLineOf(%d) = %d, want %d", position, got, line)
		}
	}
}
