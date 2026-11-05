package engine

import (
	"errors"
	"math/rand"
	"time"

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

const getLeagueTierStmtSql = "SELECT tier FROM leagues WHERE league_id = $1"

// redrawAiLevelAfterMove moves a CPU club that just changed division one step towards a level
// drawn with its new tier's weights (see common.RandomAiLevelFrom / StepAiLevelToward and
// openspec/specs/cpu-team-management "AI level after a division change"). Owned clubs and clubs
// with no AI level are untouched. Runs inside the season rollover, under the country's lock.
func redrawAiLevelAfterMove(dbCtx *common.DatabaseContext, teamId int64, newLeagueId int64, rng *rand.Rand) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching team %d to redraw its AI level: %s", teamId, err)
		return
	}
	if !isAiManagedTeam(team) {
		return
	}
	var tier int
	if err := dbCtx.Db.QueryRow(getLeagueTierStmtSql, newLeagueId).Scan(&tier); err != nil {
		log.Errorf("error reading the tier of league %d to redraw team %d's AI level: %s", newLeagueId, teamId, err)
		return
	}
	next := common.StepAiLevelToward(team.AiLevel, common.RandomAiLevelFrom(tier, rng))
	if next == team.AiLevel {
		return
	}
	if err := common.SetTeamAiLevel(dbCtx, teamId, next); err != nil {
		log.Errorf("error setting team %d's AI level to %d: %s", teamId, next, err)
		return
	}
	log.Infof("ai-level team=%d from=%d to=%d tier=%d", teamId, team.AiLevel, next, tier)
}

// pickProspectByPotential is the academy prospect a club that signs projects promotes for his
// future (see openspec/specs/cpu-team-management "Prospects by potential"), or nil: the one whose
// projected value at his best core position (common.ProjectPlayer, through the profile's potential
// noise) beats the squad's weakest starter there (weakestStarterValue) by the widest margin, ties
// to the lowest id. A position with no starter of its own sets no bar. Pure given rng.
func pickProspectByPotential(cantera []*common.Player, caps map[int64]common.PlayerSkillCaps, squad []*common.Player,
	profile common.AiTransferProfile, rng *rand.Rand) *common.Player {
	var pick *common.Player
	bestMargin := 0
	for _, p := range cantera {
		var c *common.PlayerSkillCaps
		if pc, ok := caps[p.Id]; ok {
			c = &pc
		}
		projected := common.ProjectPlayer(*p, c, profile, rng)
		position := common.BestCorePosition(projected)
		bar := weakestStarterValue(squad, position)
		if bar <= 0 {
			continue
		}
		margin := common.PositionValue(position, projected) - bar
		if margin < 0 {
			continue
		}
		if pick == nil || margin > bestMargin || (margin == bestMargin && p.Id < pick.Id) {
			pick, bestMargin = p, margin
		}
	}
	return pick
}

// promoteProspectByPotential has a club that signs projects (every Director, and youth-style
// Managers - see common.ApplyClubStyle) promote its most promising academy prospect (see
// pickProspectByPotential) through the same CanPromoteFromAcademy / PromoteFromAcademy a human
// uses, so the weekly cap applies. The rng is seeded from the team and the week, so a re-run of the
// same week reads the prospects the same way.
func promoteProspectByPotential(dbCtx *common.DatabaseContext, team *common.Team, players []*common.Player) {
	profile := aiProfile(team)
	if !profile.SignsProjects {
		return
	}
	canPromote, _, err := common.CanPromoteFromAcademy(dbCtx, team.Id)
	if err != nil || !canPromote {
		return
	}
	cantera, err := common.GetCantera(dbCtx, team.Id)
	if err != nil || len(cantera) == 0 {
		return
	}
	ids := make([]int64, len(cantera))
	for i, p := range cantera {
		ids[i] = p.Id
	}
	caps, err := common.GetPlayerSkillCaps(dbCtx, ids)
	if err != nil {
		log.Errorf("error reading the academy ceilings of AI-managed team %d: %s", team.Id, err)
		return
	}
	_, week := time.Now().ISOWeek()
	rng := rand.New(rand.NewSource(team.Id*53 + int64(week)))
	prospect := pickProspectByPotential(cantera, caps, players, profile, rng)
	if prospect == nil {
		return
	}
	if err := common.PromoteFromAcademy(dbCtx, prospect.Id, team.Id); err != nil &&
		!errors.Is(err, common.ErrAcademyPromotionAlreadyUsedThisWeek) {
		log.Errorf("error promoting academy prospect %d for AI-managed team %d: %s", prospect.Id, team.Id, err)
	}
}
