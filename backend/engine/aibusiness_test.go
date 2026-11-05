package engine

import (
	"math"
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "CPU sponsor deal": "Payout formulas" - Steady 1.3, Balanced 1 + 0.6w + 0.3d,
// Performance 0.6 + 1.2w + 0.4d, in units of the base.
func TestExpectedSponsorPayout_Formulas(t *testing.T) {
	for _, r := range []struct{ w, d float64 }{{0, 0}, {0.125, 0.25}, {0.4, 0.3}, {7.0 / 9, 1.0 / 9}, {1, 0}} {
		want := map[common.SponsorDeal]float64{
			common.SponsorDealSteady:      1.3,
			common.SponsorDealBalanced:    1 + 0.6*r.w + 0.3*r.d,
			common.SponsorDealPerformance: 0.6 + 1.2*r.w + 0.4*r.d,
		}
		for deal, unit := range want {
			if got := expectedSponsorPayout(deal, r.w, r.d) / sponsorPayoutUnit; math.Abs(got-unit) > 1e-9 {
				t.Errorf("w=%.3f d=%.3f deal %d: %.4f, want %.4f", r.w, r.d, deal, got, unit)
			}
		}
	}
}

func TestBestSponsorDeal(t *testing.T) {
	cases := []struct {
		name    string
		current common.SponsorDeal
		record  common.TeamSeasonRecord
		want    common.SponsorDeal
		judged  bool
	}{
		{"strugglers take the guarantee", common.SponsorDealBalanced, common.TeamSeasonRecord{Matches: 8, Wins: 1, Draws: 2}, common.SponsorDealSteady, true},
		{"runaway leader", common.SponsorDealBalanced, common.TeamSeasonRecord{Matches: 9, Wins: 7, Draws: 1}, common.SponsorDealPerformance, true},
		{"too early", common.SponsorDealBalanced, common.TeamSeasonRecord{Matches: 5, Wins: 0, Draws: 0}, common.SponsorDealBalanced, false},
		// 2w + d = 1 makes Balanced and Steady pay the same 1.3: the current deal stays.
		{"tie keeps the current deal", common.SponsorDealSteady, common.TeamSeasonRecord{Matches: 10, Wins: 3, Draws: 4}, common.SponsorDealSteady, true},
		{"mid-table stays balanced", common.SponsorDealBalanced, common.TeamSeasonRecord{Matches: 10, Wins: 4, Draws: 3}, common.SponsorDealBalanced, true},
	}
	for _, c := range cases {
		got, judged := bestSponsorDeal(c.current, c.record)
		if got != c.want || judged != c.judged {
			t.Errorf("%s: (%d, %v), want (%d, %v)", c.name, got, judged, c.want, c.judged)
		}
	}
}

// cpu-team-management "Manager trainer".
func TestPickManagerTrainer(t *testing.T) {
	market := []*common.Trainer{{Id: 1, Skill: 88}, {Id: 2, Skill: 79}, {Id: 3, Skill: 45}}
	if got := pickManagerTrainer(market); got == nil || got.Id != 2 {
		t.Errorf("elite coaches left alone: %+v, want the 79", got)
	}
	if got := pickManagerTrainer([]*common.Trainer{{Id: 1, Skill: 90}, {Id: 2, Skill: 80}}); got != nil {
		t.Errorf("only elite coaches: %+v, want nobody", got)
	}
	if got := pickManagerTrainer(nil); got != nil {
		t.Errorf("empty market: %+v, want nobody", got)
	}
}
