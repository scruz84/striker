package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// SponsorshipPerMatchTier1/2/3 is a flat per-match sponsorship payment, scaled by the tier a
// team currently plays in - a bigger stage, a bigger deal. Unlike ticket revenue (home-only),
// sponsorship is paid to both sides of a match, win, draw, or lose - see
// UpdateTeamFinancesAfterMatch, which both league matches and cup ties route through.
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

func creditSponsorship(dbCtx *common.DatabaseContext, teamId int64) {
	tier, err := common.GetTeamTier(dbCtx, teamId)
	if err != nil {
		log.Errorf("error getting tier for team %d sponsorship: %s", teamId, err)
		return
	}
	amount := sponsorshipForTier(tier)
	if err := common.AdjustTeamBudget(dbCtx, teamId, amount, common.TRANSACTION_SPONSORSHIP, "Matchday sponsorship"); err != nil {
		log.Errorf("error crediting sponsorship for team %d: %s", teamId, err)
	}
}
