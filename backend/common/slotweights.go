package common

// SkillWeights is how much each of a player's seven skills counts toward his value at one lineup
// slot, in percent, in the order GoalKeeper, Defense, Lateral, Pass, Midfield, Dribbling, Shot.
// Every row sums to 100, so a slot value stays on the 0-99 skill scale.
type SkillWeights [7]int

// Indexes into SkillWeights and skillValues.
const (
	skillGoalKeeper = iota
	skillDefense
	skillLateral
	skillPass
	skillMidfield
	skillDribbling
	skillShot
)

// SlotWeights is the value of each skill at each slot (openspec player-development "Slot
// weights"): the rounded shares of what +10 in that skill for one player at that slot is worth in
// projected goal difference to the match engine (see engine/tactics_study_test.go, the "weights"
// block, which prints the measured shares next to this table). PositionValue, the market search's
// SQL pre-selection (positionSkillSQL), skill drift (engine.driftSkillsForPosition) and the
// client's best-position helper (frontend/src/utils/playerRating.ts) all read it, so a slot is
// valued the way the engine rewards it.
var SlotWeights = map[int]SkillWeights{
	GOALKEEPER:             {100, 0, 0, 0, 0, 0, 0},
	DEFENSE:                {0, 90, 0, 0, 10, 0, 0},
	LATERAL_DEFENSE_LEFT:   {0, 35, 50, 0, 15, 0, 0},
	LATERAL_DEFENSE_RIGHT:  {0, 35, 50, 0, 15, 0, 0},
	MIDFIELD:               {0, 10, 0, 20, 50, 5, 15},
	LATERAL_MIDFIELD_LEFT:  {0, 0, 40, 10, 20, 5, 25},
	LATERAL_MIDFIELD_RIGHT: {0, 0, 40, 10, 20, 5, 25},
	STRIKER:                {0, 0, 5, 10, 0, 20, 65},
	LATERAL_STRIKER_LEFT:   {0, 0, 30, 5, 0, 15, 50},
	LATERAL_STRIKER_RIGHT:  {0, 0, 30, 5, 0, 15, 50},
}

// skillValues is a player's seven skills in SkillWeights order.
func skillValues(p Player) [7]int {
	return [7]int{p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot}
}

// Value is p's weighted value under w, rounded to the nearest integer (halves round up).
func (w SkillWeights) Value(p Player) int {
	skills := skillValues(p)
	total := 0
	for i, weight := range w {
		total += skills[i] * weight
	}
	return (total + 50) / 100
}
