package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

func TestSponsorshipForTier_HigherTiersPayMore(t *testing.T) {
	if sponsorshipForTier(1) <= sponsorshipForTier(2) {
		t.Fatal("expected tier 1 sponsorship to exceed tier 2")
	}
	if sponsorshipForTier(2) <= sponsorshipForTier(3) {
		t.Fatal("expected tier 2 sponsorship to exceed tier 3")
	}
	// An unrecognized tier falls back to the smallest payment rather than paying nothing.
	if sponsorshipForTier(99) != SponsorshipPerMatchTier3 {
		t.Fatal("expected an unknown tier to fall back to the tier-3 payment")
	}
}

func TestOutcomeFor(t *testing.T) {
	if got := outcomeFor(2, 1); got != outcomeWin {
		t.Errorf("outcomeFor(2, 1) = %v, want outcomeWin", got)
	}
	if got := outcomeFor(1, 1); got != outcomeDraw {
		t.Errorf("outcomeFor(1, 1) = %v, want outcomeDraw", got)
	}
	if got := outcomeFor(0, 1); got != outcomeLoss {
		t.Errorf("outcomeFor(0, 1) = %v, want outcomeLoss", got)
	}
}

func TestSponsorshipAmount_SteadyIsFlatRegardlessOfResult(t *testing.T) {
	base := int64(10000)
	win := sponsorshipAmount(base, common.SponsorDealSteady, outcomeWin)
	draw := sponsorshipAmount(base, common.SponsorDealSteady, outcomeDraw)
	loss := sponsorshipAmount(base, common.SponsorDealSteady, outcomeLoss)
	if win != draw || draw != loss {
		t.Fatalf("expected SponsorDealSteady to pay the same regardless of result, got win=%d draw=%d loss=%d", win, draw, loss)
	}
}

func TestSponsorshipAmount_BalancedAndPerformanceRewardBetterResults(t *testing.T) {
	base := int64(10000)
	for _, deal := range []common.SponsorDeal{common.SponsorDealBalanced, common.SponsorDealPerformance} {
		win := sponsorshipAmount(base, deal, outcomeWin)
		draw := sponsorshipAmount(base, deal, outcomeDraw)
		loss := sponsorshipAmount(base, deal, outcomeLoss)
		if !(win > draw && draw > loss) {
			t.Errorf("deal %v: expected win (%d) > draw (%d) > loss (%d)", deal, win, draw, loss)
		}
	}
}

func TestSponsorshipAmount_PerformanceBeatsSteadyOnAWinButNotOnALoss(t *testing.T) {
	base := int64(10000)
	steadyLoss := sponsorshipAmount(base, common.SponsorDealSteady, outcomeLoss)
	performanceLoss := sponsorshipAmount(base, common.SponsorDealPerformance, outcomeLoss)
	if performanceLoss >= steadyLoss {
		t.Fatalf("expected Performance's guaranteed floor (%d) to be below Steady's flat payment (%d) on a loss", performanceLoss, steadyLoss)
	}

	steadyWin := sponsorshipAmount(base, common.SponsorDealSteady, outcomeWin)
	performanceWin := sponsorshipAmount(base, common.SponsorDealPerformance, outcomeWin)
	if performanceWin <= steadyWin {
		t.Fatalf("expected Performance's win bonus (%d) to exceed Steady's flat payment (%d) on a win", performanceWin, steadyWin)
	}
}
