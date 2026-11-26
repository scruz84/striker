package common

import "testing"

// openspec player-development "Slot weights": every row sums to 100, so a slot value stays on the
// skill scale, and a player with the same value in every skill is worth exactly that everywhere.
func TestSlotWeights_RowsSumToOneHundred(t *testing.T) {
	for position, weights := range SlotWeights {
		total := 0
		for _, w := range weights {
			total += w
		}
		if total != 100 {
			t.Errorf("slot %d weights sum to %d, want 100", position, total)
		}
		if got := weights.Value(Player{GoalKeeper: 63, Defense: 63, Lateral: 63, Pass: 63, Midfield: 63, Dribbling: 63, Shot: 63}); got != 63 {
			t.Errorf("slot %d values an all-63 player at %d, want 63", position, got)
		}
	}
}

// PositionValue of a fixed player at every slot, computed by hand from the "Slot weights" table
// (GoalKeeper/Defense/Lateral/Pass/Midfield/Dribbling/Shot = 30/70/60/40/50/45/80), rounding halves up.
func TestPositionValue_EverySlot(t *testing.T) {
	p := Player{GoalKeeper: 30, Defense: 70, Lateral: 60, Pass: 40, Midfield: 50, Dribbling: 45, Shot: 80}
	cases := []struct {
		position int
		want     int
	}{
		{GOALKEEPER, 30},
		{DEFENSE, 68},                // 0.9 x 70 + 0.1 x 50 = 68
		{LATERAL_DEFENSE_LEFT, 62},   // 0.35 x 70 + 0.5 x 60 + 0.15 x 50 = 62
		{LATERAL_DEFENSE_RIGHT, 62},  //
		{MIDFIELD, 54},               // 0.1 x 70 + 0.2 x 40 + 0.5 x 50 + 0.05 x 45 + 0.15 x 80 = 54.25
		{LATERAL_MIDFIELD_LEFT, 60},  // 0.4 x 60 + 0.1 x 40 + 0.2 x 50 + 0.05 x 45 + 0.25 x 80 = 60.25
		{LATERAL_MIDFIELD_RIGHT, 60}, //
		{STRIKER, 68},                // 0.05 x 60 + 0.1 x 40 + 0.2 x 45 + 0.65 x 80 = 68
		{LATERAL_STRIKER_LEFT, 67},   // 0.3 x 60 + 0.05 x 40 + 0.15 x 45 + 0.5 x 80 = 66.75
		{LATERAL_STRIKER_RIGHT, 67},  //
		{-1, 0},                      // not a slot
	}
	for _, c := range cases {
		if got := PositionValue(c.position, p); got != c.want {
			t.Errorf("PositionValue(%d) = %d, want %d", c.position, got, c.want)
		}
	}
}
