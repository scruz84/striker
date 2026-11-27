package engine

import (
	"math"
	"testing"

	"github.com/scruz84/striker/common"
)

func TestBlendLine_EvenLineIsUnchanged(t *testing.T) {
	left, center, right := blendLine(50, 50, 1, 50)
	for _, got := range []float32{left, center, right} {
		if math.Abs(float64(got-50)) > 0.001 {
			t.Fatalf("expected an already-even line to stay at 50, got %v", got)
		}
	}
}

func TestBlendLine_FlanksPickUpAShareOfTheCenter(t *testing.T) {
	left, center, right := blendLine(0, 100, 1, 0)
	if left != right {
		t.Fatalf("expected a symmetric line to blend both flanks equally, got left=%v right=%v", left, right)
	}
	if left <= 0 {
		t.Fatalf("expected a strong center to lend some strength to an empty flank, got %v", left)
	}
	if center >= 100 {
		t.Fatalf("expected the center to give up some of its own strength to the flanks, got %v", center)
	}
}

func TestBlendLine_CenterPicksUpAShareOfBothFlanks(t *testing.T) {
	_, center, _ := blendLine(100, 0, 1, 100)
	if center <= 0 {
		t.Fatalf("expected strong flanks to lend some strength to an empty center, got %v", center)
	}
}

func TestBlendLine_TotalStrengthIsConservedForASinglePlayerCenter(t *testing.T) {
	left, center, right := blendLine(30, 60, 1, 90)
	total := left + center + right
	if math.Abs(float64(total-180)) > 0.01 {
		t.Fatalf("expected blending to conserve the line's total strength (180) for a one-player center, got %v", total)
	}
}

// A flank is always exactly one player (see formation.go), but the center can field one to three
// (DEFENSE/MIDFIELD/STRIKER), and calculateDefense/calculateMidfield/calculateAttack sum every one
// of them - so blendLine must compare the center's *per-player average* to the flank, not its raw
// (player-count-dependent) total, or picking a back three over a lone sweeper of identical quality
// would swing a flank's own rating for no skill-related reason at all.
func TestBlendLine_FormationWidthDoesNotDistortFlanks(t *testing.T) {
	// Three central defenders each worth 70 (sum 210) vs one central defender worth 70 - same
	// per-player quality, different formation width.
	leftWide, _, rightWide := blendLine(70, 210, 3, 70)
	leftNarrow, _, rightNarrow := blendLine(70, 70, 1, 70)
	if math.Abs(float64(leftWide-leftNarrow)) > 0.01 || math.Abs(float64(rightWide-rightNarrow)) > 0.01 {
		t.Fatalf("expected identical per-player center quality to blend a flank the same way regardless of center width, got wide=(%v,%v) narrow=(%v,%v)",
			leftWide, rightWide, leftNarrow, rightNarrow)
	}
}

// A wider center's greater total presence (the whole reason the game sums it) should survive
// blending unaffected when its own quality matches its flanks - only how much it gives to/takes
// from them should move.
func TestBlendLine_WiderCenterKeepsItsGreaterTotalPresence(t *testing.T) {
	_, centerWide, _ := blendLine(70, 210, 3, 70)
	_, centerNarrow, _ := blendLine(70, 70, 1, 70)
	if centerWide <= centerNarrow {
		t.Fatalf("expected a three-player center to end up stronger in total than a one-player center of the same quality, got wide=%v narrow=%v", centerWide, centerNarrow)
	}
}

// A weaker flank should be able to pull a stronger center's blended value down a little, and vice
// versa - not just the reverse, which the un-normalized version got backwards (see
// FormationWidthDoesNotDistortFlanks's doc comment).
func TestBlendLine_StrongerFlanksLiftAWeakerCenter(t *testing.T) {
	_, centerBoosted, _ := blendLine(80, 120, 3, 80) // centerAvg 40, flanks 80
	if centerBoosted <= 120 {
		t.Fatalf("expected stronger flanks to lift a weaker center above its raw total (120), got %v", centerBoosted)
	}
}

func TestBlendLine_ZeroCenterCountSkipsBlendingInsteadOfDividingByZero(t *testing.T) {
	left, center, right := blendLine(60, 0, 0, 70)
	if left != 60 || right != 70 || center != 0 {
		t.Fatalf("expected an empty center (e.g. every central player sent off) to leave the flanks untouched, got left=%v center=%v right=%v", left, center, right)
	}
}

// calculateLateralAttack used to only ever read its first ("striker") argument - fine for
// formations like 4-4-2 whose wide attacking outlet is the lateral midfielder passed into that
// slot at the call site, but a 4-3-3 fields its wide attacker at LATERAL_STRIKER_LEFT/RIGHT
// instead (see formation.go), which that same call site passed into the unused "mid" argument -
// silently zeroing a 4-3-3's entire flank attack rating. This pins the fix: whichever of the two
// positions is actually occupied should count.
func TestCalculateLateralAttack_FallsBackBetweenStrikerAndMidfieldSlots(t *testing.T) {
	players := map[int64]common.Player{
		1: {Id: 1, Lateral: 80, Shot: 60, Dribbling: 60}, // a 4-3-3 wide forward (LATERAL_STRIKER_LEFT/RIGHT)
		2: {Id: 2, Lateral: 40, Shot: 20, Dribbling: 20}, // a 4-4-2 winger (LATERAL_MIDFIELD_LEFT/RIGHT)
	}

	// 4-3-3 shape: only the striker slot is populated.
	if got := calculateLateralAttack([]int64{1}, nil, &players); got != 70 {
		t.Fatalf("expected the wide forward's own Lateral/attacking-value blend (70) when only the striker slot is filled, got %v", got)
	}

	// 4-4-2 shape: only the midfield slot is populated.
	if got := calculateLateralAttack(nil, []int64{2}, &players); got != 30 {
		t.Fatalf("expected a fallback to the winger's Lateral/attacking-value blend (30) when the striker slot is empty, got %v", got)
	}
}

// evenFormation lays out a formation's slots with players 1..11 who all have every skill at 50.
func evenFormation(formation int) (map[int][]int64, map[int64]common.Player) {
	pbp := map[int][]int64{}
	players := map[int64]common.Player{}
	id := int64(1)
	for _, position := range []int{common.GOALKEEPER, common.LATERAL_DEFENSE_LEFT, common.DEFENSE, common.LATERAL_DEFENSE_RIGHT,
		common.LATERAL_MIDFIELD_LEFT, common.MIDFIELD, common.LATERAL_MIDFIELD_RIGHT,
		common.LATERAL_STRIKER_LEFT, common.STRIKER, common.LATERAL_STRIKER_RIGHT} {
		for i := 0; i < common.FormationPositionCounts(formation)[position]; i++ {
			pbp[position] = append(pbp[position], id)
			players[id] = common.Player{Id: id, GoalKeeper: 50, Defense: 50, Lateral: 50, Pass: 50, Midfield: 50, Dribbling: 50, Shot: 50}
			id++
		}
	}
	return pbp, players
}

func evenQua(formation int) common.TeamQualification {
	pbp, players := evenFormation(formation)
	qua := common.TeamQualification{}
	calculateQua(&qua, nil, &pbp, &players)
	return qua
}

func assertZone(t *testing.T, name string, got float32, want float64) {
	t.Helper()
	if math.Abs(float64(got)-want) > 0.05 {
		t.Errorf("%s: got %.2f, want %.2f", name, got, want)
	}
}

// openspec match-simulation "Identical 4-4-2 and 4-3-3": with every skill at 50, line cover and the
// tucked-in wide players give a 4-4-2 central lines of 110/185/110 and a 4-3-3 115/165/115 - a
// 4-3-3's wide forwards count only LateralStrikerCentralWeight toward the central attack, so it no
// longer out-rates every other shape (its zones used to add up to 700 against everyone else's 600).
func TestCalculateQua_IdenticalFourFourTwoAndFourThreeThree(t *testing.T) {
	q := evenQua(common.FormationBalanced)
	assertZone(t, "4-4-2 defense", q.Defense, 110)
	assertZone(t, "4-4-2 midfield", q.Midfield, 185)
	assertZone(t, "4-4-2 attack", q.Attack, 110)
	q = evenQua(common.FormationFourThreeThree)
	assertZone(t, "4-3-3 defense", q.Defense, 115)
	assertZone(t, "4-3-3 midfield", q.Midfield, 165)
	assertZone(t, "4-3-3 attack", q.Attack, 115)
}

// openspec match-simulation "Crowded back line": a back five shuts its flanks, a back three leaves
// them a little more exposed (56.2 / 51.1 / 47.2 with every skill at 50).
func TestCalculateQua_CrowdedBackLine(t *testing.T) {
	assertZone(t, "5-3-2 left defense", evenQua(common.FormationDefensive).LeftDefense, 56.1875)
	assertZone(t, "4-4-2 left defense", evenQua(common.FormationBalanced).LeftDefense, 51.125)
	assertZone(t, "3-4-3 right defense", evenQua(common.FormationAttacking).RightDefense, 47.1875)
}

// openspec match-simulation "Back-three wing-backs": with one DEFENSE slot each full-back adds
// BackThreeWingBackCentralWeight of his Defense to the central defense; in a back four he adds none.
func TestCalculateQua_BackThreeWingBacksTuckIn(t *testing.T) {
	pbp, players := evenFormation(common.FormationAttacking) // 3-4-3
	before := common.TeamQualification{}
	calculateQua(&before, nil, &pbp, &players)
	for _, pos := range []int{common.LATERAL_DEFENSE_LEFT, common.LATERAL_DEFENSE_RIGHT} {
		p := players[pbp[pos][0]]
		p.Defense = 90
		players[p.Id] = p
	}
	after := common.TeamQualification{}
	calculateQua(&after, nil, &pbp, &players)
	// The central weight sum is 1 (CB) + 2 x 0.1 (CM cover) + 2 x 0.15 (wing-backs) = 1.5. The two
	// wing-backs' +40 Defense adds 2 x 0.15 x 40 = 12 to the raw total (+8 per-player average) and
	// 0.65 x 40 = 26 to each flank, which spills back into the centre.
	const weight = 1.5
	want := ((1-2*ZoneSpillover)*(2*BackThreeWingBackCentralWeight*40/weight) + ZoneSpillover*2*0.65*40) * weight
	if got := after.Defense - before.Defense; math.Abs(float64(got)-want) > 0.05 {
		t.Errorf("back three: wing-backs' Defense moved the central defense by %.2f, want %.2f", got, want)
	}

	pbp, players = evenFormation(common.FormationBalanced) // 4-4-2
	before = common.TeamQualification{}
	calculateQua(&before, nil, &pbp, &players)
	p := players[pbp[common.LATERAL_DEFENSE_LEFT][0]]
	p.Defense = 90
	players[p.Id] = p
	after = common.TeamQualification{}
	calculateQua(&after, nil, &pbp, &players)
	centralDelta := after.Defense - before.Defense
	if want := float32(ZoneSpillover * 0.5 * 40 * 2.2); math.Abs(float64(centralDelta-want)) > 0.05 {
		t.Errorf("back four: a full-back's Defense should reach the centre only through spillover (%.2f), got %.2f", want, centralDelta)
	}
}

// openspec match-simulation "Flank midfield": with a lateral midfielder the flank's midfield is half
// the full-back's flank-midfield value and half the wide midfielder's; without one it is the
// full-back's alone.
func TestCalculateLateralMidfield_SharesTheFlankWithTheWideMidfielder(t *testing.T) {
	players := map[int64]common.Player{
		1: {Id: 1, Lateral: 60, Midfield: 40},           // full-back: 0.5 x 60 + 0.5 x 40 = 50
		2: {Id: 2, Lateral: 80, Midfield: 70, Pass: 40}, // wide midfielder: 0.5 x 80 + 0.5 x (0.7 x 70 + 0.3 x 40) = 70.5
	}
	if got := calculateLateralMidfield([]int64{1}, []int64{2}, &players, 2); math.Abs(float64(got)-60.25) > 0.01 {
		t.Errorf("flank with a wide midfielder: got %.2f, want 60.25", got)
	}
	if got := calculateLateralMidfield([]int64{1}, nil, &players, 2); got != 50 {
		t.Errorf("flank without a wide midfielder: got %.2f, want 50", got)
	}
}

// Pass now counts toward a midfielder's midfield value and Dribbling toward a striker's attacking
// value (match-simulation "Core ratings"), so a better passer or dribbler raises his line.
func TestCalculateQua_PassAndDribblingRaiseTheirLines(t *testing.T) {
	pbp, players := evenFormation(common.FormationBalanced)
	base := common.TeamQualification{}
	calculateQua(&base, nil, &pbp, &players)

	cm := players[pbp[common.MIDFIELD][0]]
	cm.Pass += 10
	players[cm.Id] = cm
	st := players[pbp[common.STRIKER][0]]
	st.Dribbling += 10
	players[st.Id] = st
	got := common.TeamQualification{}
	calculateQua(&got, nil, &pbp, &players)

	if got.Midfield <= base.Midfield {
		t.Errorf("+10 Pass on a central midfielder should raise the midfield: %.2f vs %.2f", got.Midfield, base.Midfield)
	}
	if got.Attack <= base.Attack {
		t.Errorf("+10 Dribbling on a striker should raise the attack: %.2f vs %.2f", got.Attack, base.Attack)
	}
}

// A back-three's lone centre-back (DEFENSE count 1, as in FormationAttacking's 3-4-3 and
// FormationThreeFiveTwo's 3-5-2 - see formation.go) left the central Defense score far below a
// back-four's, since the lateral defenders never fed it beyond blendLine's fixed 15% zone
// spillover. Wing-backs in that shape tuck in to cover centrally rather than push on like a
// back-four's fullback, so calculateLateralDefense/calculateLateralMidfield now weight a lateral
// defender's attributes more toward Defense (and less toward Midfield) specifically when the
// central defender count is 1. This pins both halves of that shift.
func TestCalculateLateralFunctions_BackThreeWeightsLateralsTowardDefense(t *testing.T) {
	players := map[int64]common.Player{
		1: {Id: 1, Lateral: 40, Defense: 80, Midfield: 80},
	}
	def := []int64{1}

	backFourDefense := calculateLateralDefense(def, nil, &players, 2)
	backThreeDefense := calculateLateralDefense(def, nil, &players, 1)
	if backThreeDefense <= backFourDefense {
		t.Fatalf("expected a back-three's lateral defender to rate higher on the flank-defense blend (more Defense weight) than a back-four's, got backThree=%v backFour=%v", backThreeDefense, backFourDefense)
	}

	backFourMidfield := calculateLateralMidfield(def, nil, &players, 2)
	backThreeMidfield := calculateLateralMidfield(def, nil, &players, 1)
	if backThreeMidfield >= backFourMidfield {
		t.Fatalf("expected a back-three's lateral defender to rate lower on the flank-midfield blend (less Midfield weight) than a back-four's, got backThree=%v backFour=%v", backThreeMidfield, backFourMidfield)
	}
}

// End-to-end through calculateQua: a 3-4-3's central Defense score should benefit from the
// lateral defenders leaning more defensive, relative to simply halving the central defender count
// with unchanged lateral weights.
func TestCalculateQua_BackThreeCentralDefenseBenefitsFromDefensiveLaterals(t *testing.T) {
	players := map[int64]common.Player{}
	for _, id := range []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} {
		players[id] = common.Player{Id: id, Shot: 60, Lateral: 40, Defense: 80, Midfield: 60, Pass: 60, Dribbling: 60}
	}

	pbp := map[int][]int64{ // 3-4-3: one central defender, two lateral defenders
		common.GOALKEEPER:             {1},
		common.DEFENSE:                {2},
		common.LATERAL_DEFENSE_LEFT:   {3},
		common.LATERAL_DEFENSE_RIGHT:  {4},
		common.MIDFIELD:               {5, 6},
		common.LATERAL_MIDFIELD_LEFT:  {7},
		common.LATERAL_MIDFIELD_RIGHT: {8},
		common.STRIKER:                {9, 10, 11},
	}
	qua := &common.TeamQualification{}
	calculateQua(qua, nil, &pbp, &players)

	// Reference point using the old, formation-blind 50/50 lateral weighting.
	oldRawLateralDefense := float32(players[3].Lateral)*0.5 + float32(players[3].Defense)*0.5
	oldRawDefense := float32(players[2].Defense)
	_, oldCenter, _ := blendLine(oldRawLateralDefense, oldRawDefense, 1, oldRawLateralDefense)

	if qua.Defense <= oldCenter {
		t.Fatalf("expected the back-three's reweighted laterals to lift the central Defense score above the old 50/50 blend (%v), got %v", oldCenter, qua.Defense)
	}
}
