package engine

import (
	"errors"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// aiSponsorMinMatches is how many league matches of the current season a CPU club plays before it
// judges its sponsor deal (see chooseSponsorDealIfManager) - early results are too noisy, and last
// season's may come from another division.
const aiSponsorMinMatches = 6

// sponsorPayoutUnit is the base sponsorship the expected payouts are measured in; any positive
// value gives the same ranking since every deal is a multiple of the base.
const sponsorPayoutUnit = 1000

// expectedSponsorPayout is what deal pays per match, on average, for a side that wins with
// probability w and draws with probability d - computed with sponsorshipAmount itself so the
// CPU's choice can never drift from the real payout.
func expectedSponsorPayout(deal common.SponsorDeal, w float64, d float64) float64 {
	l := 1 - w - d
	return w*float64(sponsorshipAmount(sponsorPayoutUnit, deal, outcomeWin)) +
		d*float64(sponsorshipAmount(sponsorPayoutUnit, deal, outcomeDraw)) +
		l*float64(sponsorshipAmount(sponsorPayoutUnit, deal, outcomeLoss))
}

// bestSponsorDeal is the deal with the highest expected payout for a season record, keeping
// current on a tie, and false when the record is too short to judge (see
// openspec/specs/cpu-team-management "CPU sponsor deal").
func bestSponsorDeal(current common.SponsorDeal, record common.TeamSeasonRecord) (common.SponsorDeal, bool) {
	if record.Matches < aiSponsorMinMatches {
		return current, false
	}
	w := float64(record.Wins) / float64(record.Matches)
	d := float64(record.Draws) / float64(record.Matches)
	best, bestPayout := current, expectedSponsorPayout(current, w, d)
	for _, deal := range []common.SponsorDeal{common.SponsorDealSteady, common.SponsorDealBalanced, common.SponsorDealPerformance} {
		if payout := expectedSponsorPayout(deal, w, d); payout > bestPayout+1e-9 {
			best, bestPayout = deal, payout
		}
	}
	return best, true
}

// chooseSponsorDealIfManager has a Manager or Director switch to the sponsor deal that pays best
// for its results so far this season, through the same common.SetSponsorDeal a human uses (which
// locks the choice for the rest of the season). Nothing happens while the deal is already locked
// this season, before aiSponsorMinMatches league matches, or when the current deal is already the
// best. A Caretaker never touches its deal.
func chooseSponsorDealIfManager(dbCtx *common.DatabaseContext, team *common.Team) {
	if team.AiLevel < common.AiLevelManager {
		return
	}
	locked, err := common.IsSponsorDealLocked(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error checking the sponsor deal lock of AI-managed team %d: %s", team.Id, err)
		return
	}
	if locked {
		return
	}
	record, err := common.GetTeamSeasonRecord(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error reading the season record of AI-managed team %d: %s", team.Id, err)
		return
	}
	current, err := common.GetSponsorDeal(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error reading the sponsor deal of AI-managed team %d: %s", team.Id, err)
		return
	}
	best, ok := bestSponsorDeal(current, record)
	if !ok || best == current {
		return
	}
	if err := common.SetSponsorDeal(dbCtx, team.Id, best); err != nil && !errors.Is(err, common.ErrSponsorDealLocked) {
		log.Errorf("error setting the sponsor deal of AI-managed team %d: %s", team.Id, err)
	}
}

// aiManagerTrainerMaxSkill is the skill from which a trainer counts as elite (trainerBonus +2): a
// Manager only hires below it, leaving elite coaches to Directors and human managers.
const aiManagerTrainerMaxSkill = 80

// pickManagerTrainer is the trainer a Manager with no trainer hires from available (best first,
// see common.GetAvailableTrainers): the most skilled one below aiManagerTrainerMaxSkill, or nil.
func pickManagerTrainer(available []*common.Trainer) *common.Trainer {
	for _, t := range available {
		if t.Skill < aiManagerTrainerMaxSkill {
			return t
		}
	}
	return nil
}

// hireTrainerIfManager has a Manager with no trainer hire one (see pickManagerTrainer) through the
// same common.HireTrainer a human uses; it never fires one (see openspec/specs/cpu-team-management
// "Manager trainer"). Directors keep their own rule (upgradeTrainerIfBetterAvailable) and
// Caretakers never hire.
func hireTrainerIfManager(dbCtx *common.DatabaseContext, team *common.Team) {
	if team.AiLevel != common.AiLevelManager {
		return
	}
	current, err := common.GetTeamTrainer(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error fetching trainer for AI-managed team %d: %s", team.Id, err)
		return
	}
	if current != nil {
		return
	}
	available, err := common.GetAvailableTrainers(dbCtx)
	if err != nil {
		log.Errorf("error fetching available trainers for AI-managed team %d: %s", team.Id, err)
		return
	}
	pick := pickManagerTrainer(available)
	if pick == nil {
		return
	}
	if err := common.HireTrainer(dbCtx, pick.Id, team.Id); err != nil &&
		!errors.Is(err, common.ErrTeamAlreadyHasTrainer) && !errors.Is(err, common.ErrTrainerNotAvailable) {
		log.Errorf("error hiring trainer %d for AI-managed team %d: %s", pick.Id, team.Id, err)
	}
}
