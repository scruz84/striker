package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// SeasonTicketRevenuePerSeat is a lump sum paid once per season, at rollover, for every seat in a
// team's stadium - modeling season-ticket holders paying up front rather than match by match.
// Independent of the existing per-match attendance-driven ticket revenue in engine/stadium.go: a
// bigger stadium earns more here even if it's rarely full on any given matchday.
const SeasonTicketRevenuePerSeat = 5

// MerchandisingPoolTier1/2/3 mirrors PrizeMoneyPoolTier1/2/3's tier scaling but pays out on a much
// flatter curve (see merchandisingForPosition) - shirt sales are a small, steady income stream,
// not a competitive reward for finishing well.
const MerchandisingPoolTier1 = 100000
const MerchandisingPoolTier2 = 40000
const MerchandisingPoolTier3 = 10000

func merchandisingPoolForTier(tier int) int64 {
	switch tier {
	case 1:
		return MerchandisingPoolTier1
	case 2:
		return MerchandisingPoolTier2
	default:
		return MerchandisingPoolTier3
	}
}

// merchandisingForPosition splits a tier's merchandising pool across every team in the league,
// weighted lightly by final position: 1st earns 1.5x the per-team average share, last earns 0.5x
// - unlike prizeMoneyForPosition's 2x-to-0 spread, a struggling team's shirts still sell.
func merchandisingForPosition(pool int64, position int, teamCount int) int64 {
	if teamCount <= 1 {
		return pool
	}
	share := 1.5 - 1.0*float64(position-1)/float64(teamCount-1)
	perTeamAverage := float64(pool) / float64(teamCount)
	return int64(perTeamAverage * share)
}

// awardSeasonTicketsAndMerchandising pays every team in one league's final standings its season-
// ticket lump sum (stadium capacity x SeasonTicketRevenuePerSeat) and merchandising share (see
// merchandisingForPosition) - called from applyPromotionRelegation alongside
// awardLeaguePrizeMoney, which already has each league's ranked final standings on hand.
func awardSeasonTicketsAndMerchandising(dbCtx *common.DatabaseContext, tier int, standings []common.TeamStanding) {
	merchandisingPool := merchandisingPoolForTier(tier)
	for i, team := range standings {
		awardSeasonTickets(dbCtx, team.TeamId)

		merchandisingAmount := merchandisingForPosition(merchandisingPool, i+1, len(standings))
		if merchandisingAmount == 0 {
			continue
		}
		if err := common.AdjustTeamBudget(dbCtx, team.TeamId, merchandisingAmount, common.TRANSACTION_MERCHANDISING, "Merchandising revenue"); err != nil {
			log.Errorf("error paying merchandising revenue to team %d: %s", team.TeamId, err)
		}
	}
}

func awardSeasonTickets(dbCtx *common.DatabaseContext, teamId int64) {
	stadium, err := common.GetStadium(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching stadium for team %d season tickets: %s", teamId, err)
		return
	}
	amount := int64(stadium.Capacity * SeasonTicketRevenuePerSeat)
	if err := common.AdjustTeamBudget(dbCtx, teamId, amount, common.TRANSACTION_SEASON_TICKETS, "Season ticket sales"); err != nil {
		log.Errorf("error paying season tickets to team %d: %s", teamId, err)
	}
}
