package engine

import (
	"math"
	"math/rand"
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

// cpu-team-management "Prospects by potential": "Gifted prospect", "Ordinary prospect" and
// "Balanced Manager".
func TestPickProspectByPotential(t *testing.T) {
	squad := squadOf(2, 5, 6, 5) // ids 14-18 strikers
	for _, p := range squad[13:] {
		p.Shot, p.Pass, p.Dribbling = 60, 60, 60
	}
	prospect := func(id int64, skill int) *common.Player {
		return &common.Player{Id: id, Age: 17, Shot: skill, Pass: skill, Dribbling: skill}
	}
	caps := func(ids ...int64) map[int64]common.PlayerSkillCaps {
		m := map[int64]common.PlayerSkillCaps{}
		for _, id := range ids {
			m[id] = common.PlayerSkillCaps{GoalKeeperMax: 99, DefenseMax: 99, LateralMax: 99, PassMax: 99, MidfieldMax: 99, DribblingMax: 99, ShotMax: 99}
		}
		return m
	}
	director := common.AiTransferProfileFor(common.AiLevelDirector)
	gifted, ordinary := prospect(101, 52), prospect(102, 38)

	if got := pickProspectByPotential([]*common.Player{ordinary, gifted}, caps(101, 102), squad, director, rand.New(rand.NewSource(1))); got == nil || got.Id != gifted.Id {
		t.Errorf("picked %v, want the gifted 17-year-old (52 now, about 64 in 3 seasons) over a weakest starter of 60", got)
	}
	if got := pickProspectByPotential([]*common.Player{ordinary}, caps(102), squad, director, rand.New(rand.NewSource(1))); got != nil {
		t.Errorf("promoted the ordinary prospect %d (about 50 in 3 seasons)", got.Id)
	}
	if balanced := common.ApplyClubStyle(common.AiTransferProfileFor(common.AiLevelManager), common.AiStyleBalanced); balanced.SignsProjects {
		t.Error("a balanced Manager signs projects, so it would promote for potential")
	}
	if youth := common.ApplyClubStyle(common.AiTransferProfileFor(common.AiLevelManager), common.AiStyleYouth); !youth.SignsProjects {
		t.Error("a youth Manager does not promote for potential")
	}
}
