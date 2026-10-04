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

// calculateMidfield used to sum the DEFENSE group's Midfield attribute instead of the MIDFIELD
// group's (a copy-paste mix-up of the two parameters) - this pins the fix: it must read from the
// actual central midfielders, not whoever's passed as the first argument.
func TestCalculateMidfield_ReadsFromTheMidfieldGroupNotDefense(t *testing.T) {
	players := map[int64]common.Player{
		1: {Id: 1, Midfield: 10}, // a "defender" with a low Midfield stat
		2: {Id: 2, Midfield: 90}, // the actual central midfielder
	}
	defenders := []int64{1}
	midfielders := []int64{2}

	got := calculateMidfield(defenders, midfielders, &players)
	if got != 90 {
		t.Fatalf("expected calculateMidfield to sum the midfield group's Midfield stat (90), got %v", got)
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
		1: {Id: 1, Lateral: 80, Shot: 60}, // a 4-3-3 wide forward (LATERAL_STRIKER_LEFT/RIGHT)
		2: {Id: 2, Lateral: 40, Shot: 20}, // a 4-4-2 winger (LATERAL_MIDFIELD_LEFT/RIGHT)
	}

	// 4-3-3 shape: only the striker slot is populated.
	if got := calculateLateralAttack([]int64{1}, nil, &players); got != 70 {
		t.Fatalf("expected the wide forward's own Lateral/Shot blend (70) when only the striker slot is filled, got %v", got)
	}

	// 4-4-2 shape: only the midfield slot is populated.
	if got := calculateLateralAttack(nil, []int64{2}, &players); got != 30 {
		t.Fatalf("expected a fallback to the winger's Lateral/Shot blend (30) when the striker slot is empty, got %v", got)
	}
}
