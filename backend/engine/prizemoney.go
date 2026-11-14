package engine

import (
	"fmt"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// PrizeMoneyPoolTier1/2/3 is each tier's total season prize pool, split across every team in
// one of that tier's leagues by final position (see prizeMoneyForPosition). Bigger tiers pay
// more, the same direction stadium capacity already scales by tier - promotion is worth more
// than prestige on both counts.
const PrizeMoneyPoolTier1 = 500000
const PrizeMoneyPoolTier2 = 200000
const PrizeMoneyPoolTier3 = 50000

// CupPrizePerRound is the base cup prize for winning a tie, multiplied by the round number so
// later rounds pay more. CupChampionBonus is paid once, on top, to the eventual winner.
const CupPrizePerRound = 5000
const CupChampionBonus = 100000

func prizePoolForTier(tier int) int64 {
	switch tier {
	case 1:
		return PrizeMoneyPoolTier1
	case 2:
		return PrizeMoneyPoolTier2
	default:
		return PrizeMoneyPoolTier3
	}
}

// prizeMoneyForPosition scales a league's prize pool linearly by finishing position: 1st place
// earns roughly twice the per-team average share of the pool, last place earns roughly nothing,
// straight-line in between. position is 1-based; teamCount must be >= 1.
func prizeMoneyForPosition(pool int64, position int, teamCount int) int64 {
	if teamCount <= 1 {
		return pool
	}
	share := 2.0 - 2.0*float64(position-1)/float64(teamCount-1)
	if share < 0 {
		share = 0
	}
	perTeamAverage := float64(pool) / float64(teamCount)
	return int64(perTeamAverage * share)
}

// ordinal renders 1 -> "1st", 2 -> "2nd", 3 -> "3rd", 4 -> "4th", 11-13 -> "th" (not "1st/2nd/3rd").
func ordinal(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return fmt.Sprintf("%dth", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// awardLeaguePrizeMoney pays every team in one league's final standings its share of that
// tier's prize pool. Called from applyPromotionRelegation, which already has each league's
// ranked standings on hand for computing promotion/relegation moves.
func awardLeaguePrizeMoney(dbCtx *common.DatabaseContext, tier int, standings []common.TeamStanding) {
	pool := prizePoolForTier(tier)
	for i, team := range standings {
		position := i + 1
		amount := prizeMoneyForPosition(pool, position, len(standings))
		if amount == 0 {
			continue
		}
		description := fmt.Sprintf("%s place finish", ordinal(position))
		if err := common.AdjustTeamBudget(dbCtx, team.TeamId, amount, common.TRANSACTION_PRIZE_MONEY, description); err != nil {
			log.Errorf("error awarding league prize money to team %d: %s", team.TeamId, err)
		}
	}
}

// awardCupRoundPrize pays a cup tie's winner for advancing - see simulateCupTie. A promotion
// playoff tie pays nothing of its own (see openspec/specs/league-pyramid "Playoff match"): its
// winner earns only the gate and sponsorship every match pays.
func awardCupRoundPrize(dbCtx *common.DatabaseContext, cupId int64, winnerTeamId int64, round int) {
	if playoff, err := common.IsPlayoffCup(dbCtx, cupId); err != nil {
		log.Errorf("error checking whether cup %d is a playoff before its round prize: %s", cupId, err)
		return
	} else if playoff {
		return
	}
	amount := int64(CupPrizePerRound * round)
	if amount <= 0 {
		// No round is numbered below 1; a 0 here means the caller lost the tie's round, and a
		// "Cup round 0 prize money" row of 0 would only clutter the ledger.
		log.Errorf("cup %d: no round prize for round %d", cupId, round)
		return
	}
	description := fmt.Sprintf("Cup round %d prize money", round)
	if err := common.AdjustTeamBudget(dbCtx, winnerTeamId, amount, common.TRANSACTION_PRIZE_MONEY, description); err != nil {
		log.Errorf("error awarding cup round prize money to team %d: %s", winnerTeamId, err)
	}
}

// awardCupChampionBonus pays the cup's eventual winner a one-off bonus on top of what it
// already earned advancing round by round - see advanceCup.
func awardCupChampionBonus(dbCtx *common.DatabaseContext, championTeamId int64) {
	if err := common.AdjustTeamBudget(dbCtx, championTeamId, CupChampionBonus, common.TRANSACTION_PRIZE_MONEY, "Cup champion bonus"); err != nil {
		log.Errorf("error awarding cup champion bonus to team %d: %s", championTeamId, err)
	}
}
