package engine

import (
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

func TestDriftFanbaseForResult_WinRaisesOverManyTrials(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	fanbase := 50
	rises := 0
	const trials = 4000
	for i := 0; i < trials; i++ {
		if driftFanbaseForResult(fanbase, outcomeWin, r) > fanbase {
			rises++
		}
	}
	// A loose sanity check on the roll rate, not an exact-probability test - see
	// TestCalculateAttendance_AveragesNearBaseRate for the same style of tolerance elsewhere in
	// this package.
	got := float64(rises) / float64(trials)
	if got < FanbaseResultDriftProbability*0.5 || got > FanbaseResultDriftProbability*1.5 {
		t.Fatalf("win rise rate %.3f too far from expected ~%.3f", got, FanbaseResultDriftProbability)
	}
}

func TestDriftFanbaseForResult_LossLowersOverManyTrials(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	fanbase := 50
	drops := 0
	const trials = 4000
	for i := 0; i < trials; i++ {
		if driftFanbaseForResult(fanbase, outcomeLoss, r) < fanbase {
			drops++
		}
	}
	got := float64(drops) / float64(trials)
	if got < FanbaseResultDriftProbability*0.5 || got > FanbaseResultDriftProbability*1.5 {
		t.Fatalf("loss drop rate %.3f too far from expected ~%.3f", got, FanbaseResultDriftProbability)
	}
}

func TestDriftFanbaseForResult_DrawNeverMoves(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	fanbase := 50
	for i := 0; i < 2000; i++ {
		if got := driftFanbaseForResult(fanbase, outcomeDraw, r); got != fanbase {
			t.Fatalf("draw should never move fanbase, got %d from %d", got, fanbase)
		}
	}
}

func TestDriftFanbaseForResult_ClampsAtCeiling(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		if got := driftFanbaseForResult(99, outcomeWin, r); got > 99 {
			t.Fatalf("expected fanbase to clamp at 99, got %d", got)
		}
	}
}

func TestDriftFanbaseForResult_ClampsAtFloor(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		if got := driftFanbaseForResult(1, outcomeLoss, r); got < 1 {
			t.Fatalf("expected fanbase to clamp at 1, got %d", got)
		}
	}
}

// TestDriftFanbaseTierPull_PullsTowardTierTarget runs enough trials that even a very unlucky RNG
// streak still produces far more than the ~30 successful 5%-rolls needed to climb from
// common.DefaultFanbase (50) up to tier 1's target (80, see FanbaseTierTarget) - checking along
// the way that it never overshoots or moves the wrong direction.
func TestDriftFanbaseTierPull_PullsTowardTierTarget(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	fanbase := common.DefaultFanbase
	for i := 0; i < 5000; i++ {
		next := driftFanbaseTierPull(fanbase, 1, r)
		if next < fanbase {
			t.Fatalf("expected tier-1 pull to never lower fanbase, moved %d -> %d", fanbase, next)
		}
		fanbase = next
	}
	if fanbase <= common.DefaultFanbase {
		t.Fatalf("expected fanbase to have climbed from %d toward tier 1's target %d, got %d", common.DefaultFanbase, FanbaseTierTarget[1], fanbase)
	}
	if fanbase > FanbaseTierTarget[1] {
		t.Fatalf("expected fanbase to never overshoot tier 1's target %d, got %d", FanbaseTierTarget[1], fanbase)
	}
}

func TestDriftFanbaseTierPull_NoOpAtTarget(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	target := FanbaseTierTarget[2]
	for i := 0; i < 2000; i++ {
		if got := driftFanbaseTierPull(target, 2, r); got != target {
			t.Fatalf("expected fanbase already at its tier target %d to stay there, got %d", target, got)
		}
	}
}

func TestFanbaseTierTarget_FallsBackToLowestTierBeyondKnownTiers(t *testing.T) {
	if got := fanbaseTierTarget(4); got != FanbaseTierTarget[3] {
		t.Errorf("fanbaseTierTarget(4) = %d, want fallback to tier 3's target %d", got, FanbaseTierTarget[3])
	}
}

func TestFanbaseCommercialFactor_BoundsAndDefault(t *testing.T) {
	const tolerance = 0.0001
	if got := fanbaseCommercialFactor(0); got != FanbaseCommercialMinFactor {
		t.Errorf("fanbaseCommercialFactor(0) = %v, want %v", got, FanbaseCommercialMinFactor)
	}
	// Floating-point division/multiplication (99.0/99.0) doesn't land on exactly
	// FanbaseCommercialMaxFactor bit-for-bit, so this needs a tolerance rather than ==.
	if got := fanbaseCommercialFactor(99); got < FanbaseCommercialMaxFactor-tolerance || got > FanbaseCommercialMaxFactor+tolerance {
		t.Errorf("fanbaseCommercialFactor(99) = %v, want ~%v", got, FanbaseCommercialMaxFactor)
	}
	// common.DefaultFanbase should land close to a neutral 1.0x, so a brand new team's
	// sponsorship/season-ticket/merchandising income isn't noticeably different from before
	// Fanbase existed.
	if got := fanbaseCommercialFactor(common.DefaultFanbase); got < 0.99 || got > 1.01 {
		t.Errorf("fanbaseCommercialFactor(DefaultFanbase) = %v, want ~1.0", got)
	}
}
