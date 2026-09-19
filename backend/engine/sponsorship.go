package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// SponsorshipPerMatchTier1/2/3 is the base per-match sponsorship figure, scaled by the tier a
// team currently plays in - a bigger stage, a bigger deal. Unlike ticket revenue (home-only),
// sponsorship is paid to both sides of a match, win, draw, or lose - see
// UpdateTeamFinancesAfterMatch, which both league matches and cup ties route through. The actual
// amount paid also depends on the team's chosen common.SponsorDeal (see sponsorshipAmount).
const SponsorshipPerMatchTier1 = 8000
const SponsorshipPerMatchTier2 = 4000
const SponsorshipPerMatchTier3 = 1500

func sponsorshipForTier(tier int) int64 {
	switch tier {
	case 1:
		return SponsorshipPerMatchTier1
	case 2:
		return SponsorshipPerMatchTier2
	default:
		return SponsorshipPerMatchTier3
	}
}

// matchOutcome is a team's own result in one match, from that team's perspective - see
// outcomeFor. Sponsorship bonuses (see sponsorshipAmount) are computed against this, not the raw
// scoreline, so the same match yields a different outcome for each side.
type matchOutcome int

const (
	outcomeLoss matchOutcome = iota
	outcomeDraw
	outcomeWin
)

func outcomeFor(goalsFor int, goalsAgainst int) matchOutcome {
	switch {
	case goalsFor > goalsAgainst:
		return outcomeWin
	case goalsFor == goalsAgainst:
		return outcomeDraw
	default:
		return outcomeLoss
	}
}

// sponsorshipAmount applies deal's guaranteed/bonus split to base (see sponsorshipForTier) for
// one match's outcome. SponsorDealSteady always pays the same 1.3x base regardless of result -
// the safest, lowest-upside choice. SponsorDealBalanced pays base guaranteed plus a bonus on a
// win or draw; SponsorDealPerformance pays a smaller 0.6x base guaranteed but a bigger bonus on
// top - genuinely worse than Steady for a team that mostly loses, better for one that mostly wins
// or draws.
func sponsorshipAmount(base int64, deal common.SponsorDeal, outcome matchOutcome) int64 {
	switch deal {
	case common.SponsorDealPerformance:
		guaranteed := base * 6 / 10
		switch outcome {
		case outcomeWin:
			return guaranteed + base*12/10
		case outcomeDraw:
			return guaranteed + base*4/10
		default:
			return guaranteed
		}
	case common.SponsorDealSteady:
		return base * 13 / 10
	default: // common.SponsorDealBalanced
		switch outcome {
		case outcomeWin:
			return base + base*6/10
		case outcomeDraw:
			return base + base*3/10
		default:
			return base
		}
	}
}

func creditSponsorship(dbCtx *common.DatabaseContext, teamId int64, goalsFor int, goalsAgainst int) {
	tier, err := common.GetTeamTier(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting tier for team %d sponsorship: %s", teamId, err)
		return
	}
	deal, err := common.GetSponsorDeal(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting sponsor deal for team %d: %s", teamId, err)
		return
	}
	amount := sponsorshipAmount(sponsorshipForTier(tier), deal, outcomeFor(goalsFor, goalsAgainst))
	if err := common.AdjustTeamBudget(dbCtx, teamId, amount, common.TRANSACTION_SPONSORSHIP, "Matchday sponsorship"); err != nil {
		log.Errorf("error crediting sponsorship for team %d: %s", teamId, err)
	}
}
