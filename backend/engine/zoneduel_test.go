package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// zoneDuelSquad builds a squad whose goalkeeper is player 1 and whose other ten players all carry
// skill in every outfield skill, with the 4-4-2 slots filled in order.
func zoneDuelSquad(firstId int64, skill int) ZoneDuelSide {
	slots := []int{
		common.GOALKEEPER,
		common.LATERAL_DEFENSE_LEFT, common.DEFENSE, common.DEFENSE, common.LATERAL_DEFENSE_RIGHT,
		common.LATERAL_MIDFIELD_LEFT, common.MIDFIELD, common.MIDFIELD, common.LATERAL_MIDFIELD_RIGHT,
		common.STRIKER, common.STRIKER,
	}
	side := ZoneDuelSide{Lineup: &common.Lineup{Formation: 1}}
	for i, slot := range slots {
		id := firstId + int64(i)
		side.Squad = append(side.Squad, &common.Player{
			Id: id, GoalKeeper: 60, Defense: skill, Lateral: skill, Pass: skill, Midfield: skill, Dribbling: skill, Shot: skill, Stamina: 80,
		})
		side.Lineup.Positions = append(side.Lineup.Positions, common.LineupPosition{PlayerId: id, Position: slot})
	}
	return side
}

func TestProjectZoneDuel_StrongerMidfieldShowsInTheRatings(t *testing.T) {
	own := zoneDuelSquad(1, 99)
	opponent := zoneDuelSquad(100, 81)

	got := ProjectZoneDuel(own, opponent)

	// two central midfielders at 99 vs 81 per player: 198 vs 162, a 55% / 45% share
	if got.Own.Midfield <= got.Opponent.Midfield {
		t.Fatalf("own midfield %v should beat opponent %v", got.Own.Midfield, got.Opponent.Midfield)
	}
	ratio := got.Own.Midfield / (got.Own.Midfield + got.Opponent.Midfield)
	if ratio < 0.545 || ratio > 0.555 {
		t.Fatalf("midfield share = %.3f, want 198/360 = 0.55", ratio)
	}
	if got.OwnGoalkeeper != 60 || got.OpponentGoalkeeper != 60 {
		t.Fatalf("goalkeeper ratings = %v / %v, want 60 / 60", got.OwnGoalkeeper, got.OpponentGoalkeeper)
	}
}

func TestProjectZoneDuel_DoesNotChangeItsInputs(t *testing.T) {
	own := zoneDuelSquad(1, 90)
	opponent := zoneDuelSquad(100, 80)
	before := len(own.Lineup.Positions)

	ProjectZoneDuel(own, opponent)

	if len(own.Lineup.Positions) != before || own.Squad[1].Defense != 90 {
		t.Fatal("ProjectZoneDuel must not modify the lineup or the squad")
	}
}
