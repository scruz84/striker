package engine

import (
	"math"
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

// Rivalry points of a match (openspec rivalries "Rivalry events").
func TestRivalryPointsForMatch(t *testing.T) {
	for _, c := range []struct {
		name             string
		home, away, reds int
		cup, final       bool
		want             float64
	}{
		{"tight league win with a red card", 2, 1, 1, false, false, 2},
		{"thrashing", 4, 0, 0, false, false, 1.5},
		{"comfortable win", 2, 0, 0, false, false, 0},
		{"goalless draw", 0, 0, 0, false, false, 1},
		{"red cards capped at 2", 1, 1, 4, false, false, 3},
		{"cup elimination", 3, 1, 0, true, false, 3},
		{"cup final decided on penalties", 1, 1, 0, true, true, 6},
		{"cup final 2-1", 2, 1, 0, true, true, 6},
	} {
		if got := rivalryPointsForMatch(c.home, c.away, c.reds, c.cup, c.final); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: %v points, want %v", c.name, got, c.want)
		}
	}
}

// Rival match effects (openspec match-simulation "Discipline", fanbase "Result drift", socios
// "Attendance factors").
func TestRivalryEffects(t *testing.T) {
	if p := cardIncidentProbability(common.RIVALRY_NONE); p != 0.03 {
		t.Errorf("normal card rate %v, want 0.03", p)
	}
	if p := cardIncidentProbability(common.RIVALRY_RIVAL); p != 0.036 {
		t.Errorf("Rival card rate %v, want 0.036", p)
	}
	if p := cardIncidentProbability(common.RIVALRY_CLASICO); p != 0.042 {
		t.Errorf("Clásico card rate %v, want 0.042", p)
	}
	// Clásico league match: a Fanbase-50 visitor gives (0.85 + 0.30 × 50/99) × 1.20 ≈ 1.20.
	want := (RivalFactorMin + RivalFactorSpan*50/99.0) * 1.20
	if got := rivalFactor(50, 0, common.RIVALRY_CLASICO); math.Abs(got-want) > 1e-9 || math.Abs(got-1.2) > 0.01 {
		t.Errorf("Clásico rival factor %v, want %v (about 1.20)", got, want)
	}
	if got, base := rivalFactor(50, 2, common.RIVALRY_RIVAL), rivalFactor(50, 2, common.RIVALRY_NONE); math.Abs(got-base*1.10) > 1e-9 {
		t.Errorf("Rival cup factor %v, want %v × 1.10", got, base)
	}
	// Clásico defeat: the Fanbase drops by 1 with probability 45%.
	r := rand.New(rand.NewSource(3))
	const n = 20000
	drops := 0
	for i := 0; i < n; i++ {
		if driftFanbaseForResult(50, outcomeLoss, common.RIVALRY_CLASICO, r) == 49 {
			drops++
		}
	}
	if rate := float64(drops) / n; math.Abs(rate-0.45) > 0.015 {
		t.Errorf("Clásico defeat drift rate %.3f, want 0.45", rate)
	}
}

// Five simulated seasons of a 20-team division with realistic score lines (two meetings a pair a
// season, no cup or title events) leave at most 5% of the pairs Rival or above and at most 1%
// Clásico - ordinary close games alone must not turn every pairing into a rivalry.
func TestRivalryCalibration_OrdinarySeasons(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	const teams = 20
	poisson := func(lambda float64) int {
		l, k, p := math.Exp(-lambda), 0, 1.0
		for {
			p *= r.Float64()
			if p <= l {
				return k
			}
			k++
		}
	}
	scores := make(map[[2]int]float64)
	for season := 0; season < 5; season++ {
		for a := 0; a < teams; a++ {
			for b := a + 1; b < teams; b++ {
				for leg := 0; leg < 2; leg++ {
					points := rivalryPointsForMatch(poisson(1.45), poisson(1.15), poisson(0.2), false, false)
					scores[[2]int{a, b}] = math.Min(common.RivalryMaxScore, scores[[2]int{a, b}]+points)
				}
			}
		}
		for pair, s := range scores {
			scores[pair] = common.DecayRivalryScore(s)
		}
	}
	rival, clasico := 0, 0
	for _, s := range scores {
		switch common.RivalryLevel(s) {
		case common.RIVALRY_RIVAL:
			rival++
		case common.RIVALRY_CLASICO:
			rival++
			clasico++
		}
	}
	pairs := float64(len(scores))
	if float64(rival)/pairs > 0.05 || float64(clasico)/pairs > 0.01 {
		t.Errorf("after 5 ordinary seasons %d of %d pairs are Rival or above and %d Clásico, want at most 5%% and 1%%", rival, len(scores), clasico)
	}
}
