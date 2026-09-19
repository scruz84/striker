package engine

import (
	"errors"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// RunAITeamManagement drives every automatic action an unowned team's AI level entitles it to -
// called once per match for each side involved (see UpdateTeamFinancesAfterMatch) instead of a
// separate scheduler job, so AI upkeep happens exactly as often as game time actually advances.
// See common.AiLevelCaretaker/Manager/Director for what each tier adds on top of the one below
// it; a no-op for any team with a real owner.
func RunAITeamManagement(dbCtx *common.DatabaseContext, teamId int64) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching team %d for AI management: %s", teamId, err)
		return
	}
	if team.OwnerId > 0 || team.AiLevel < common.AiLevelCaretaker {
		return
	}

	refreshAiTeamLineup(dbCtx, teamId)
	manageRosterIfManager(dbCtx, team)
	manageMarketIfDirector(dbCtx, team)
}

// contractRenewalThresholdYears is how few seasons a Manager-level AI team lets a player's
// contract run down to before proactively renewing it (see renewExpiringContracts) - a real
// owner wouldn't let a useful squad player walk for nothing at season end just from inattention.
const contractRenewalThresholdYears = 1

// surplusDepthMargin is how far a core position's depth (see common.AssessPositionNeeds) must
// exceed common.MinPositionDepth before releaseSurplusPlayer treats it as overstocked enough to
// trim - one full backup beyond the minimum, so a position sitting exactly at minimum depth is
// never touched.
const surplusDepthMargin = 1

// manageRosterIfManager runs every Manager-level (see common.AiLevelManager) roster/business
// action for an unowned team: keep valuable players under contract, trim genuinely surplus
// depth, fill a thin position from the free-agent market, and charge what the gate will bear.
// A no-op below Manager level - see RunAITeamManagement, the only caller.
func manageRosterIfManager(dbCtx *common.DatabaseContext, team *common.Team) {
	if team.AiLevel < common.AiLevelManager {
		return
	}

	players, err := common.GetPlayersFromTeamId(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error fetching squad for AI-managed team %d roster management: %s", team.Id, err)
		return
	}

	renewExpiringContracts(dbCtx, team.Id, *players)
	releaseSurplusPlayer(dbCtx, team.Id, *players)
	signFreeAgentForWeakestPosition(dbCtx, team.Id, *players)
	respondToReceivedOffers(dbCtx, team.Id)
	maximizeTicketPrice(dbCtx, team.Id)
}

// minAcceptableTransferFeeWageMultiplier is how many times a player's own market-rate wage (see
// common.PlayerWageFromSkills) an incoming transfer offer's fee must be worth before a
// Manager-level AI team accepts it - low enough that a genuinely competitive offer for a squad
// player gets taken, high enough that a lowball offer aimed at poaching a good player for next to
// nothing gets rejected instead. Mirrors EarlyReleaseWageMultiplier's use of wage as this file's
// common yardstick for a player's worth. A sale thinning the squad below common.SquadSize is not
// specially guarded against here: signFreeAgentForWeakestPosition (run just before this, every
// time) reactively refills whichever position ends up thinnest, the same way it already covers a
// retirement or an expired contract - there's nothing about a sale that needs different handling.
const minAcceptableTransferFeeWageMultiplier = 20

// respondToReceivedOffers judges every pending offer on a Manager-level AI team's own players:
// accept a fee that clears minAcceptableTransferFeeWageMultiplier (the personal terms are already
// settled - see MakeTransferOffer, which never even creates an offer the player themself wouldn't
// accept), reject anything less. Before this, a human's offer on an AI-managed team's player had
// no automated response at all and would sit pending forever - only interacting with the open
// market as a buyer/lister was covered.
func respondToReceivedOffers(dbCtx *common.DatabaseContext, teamId int64) {
	offers, err := common.GetOffersReceived(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching received offers for AI-managed team %d: %s", teamId, err)
		return
	}

	for _, offer := range offers {
		p := offer.Player
		expectedWage := common.PlayerWageFromSkills(p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot)
		goodOffer := offer.Amount >= int64(expectedWage*minAcceptableTransferFeeWageMultiplier)

		if !goodOffer {
			if err := common.RejectOffer(dbCtx, offer.OfferId, teamId); err != nil {
				log.Errorf("error rejecting offer %d for AI-managed team %d: %s", offer.OfferId, teamId, err)
			}
			continue
		}

		if err := common.AcceptOffer(dbCtx, offer.OfferId, teamId); err != nil &&
			!errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrOfferNoLongerValid) &&
			!errors.Is(err, common.ErrInsufficientBudget) {
			log.Errorf("error accepting offer %d for AI-managed team %d: %s", offer.OfferId, teamId, err)
		}
	}
}

// renewExpiringContracts extends every squad player's contract once it's down to
// contractRenewalThresholdYears seasons or fewer remaining - a Manager tier team always prefers
// keeping a player on the books over losing them to free agency for nothing, so it offers
// exactly what the player is guaranteed to accept (their own market wage and their own
// age-appropriate maximum term - see common.PlayerWageFromSkills/MaxAcceptableRenewalYears)
// rather than lowballing and risking a decline. ErrPlayerOnLoan (the parent club's contract, not
// this team's to renew) is an expected outcome, not a failure, and is silently skipped.
func renewExpiringContracts(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) {
	for _, p := range players {
		if p.ContractYearsRemaining > contractRenewalThresholdYears {
			continue
		}
		wage := common.PlayerWageFromSkills(p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot)
		years := common.MaxAcceptableRenewalYears(p.Age)
		if _, err := common.MakeContractRenewalOffer(dbCtx, p.Id, teamId, wage, years); err != nil &&
			!errors.Is(err, common.ErrPlayerOnLoan) {
			log.Errorf("error renewing contract for AI-managed team %d player %d: %s", teamId, p.Id, err)
		}
	}
}

// releaseSurplusPlayer frees up one roster spot (and its wage) when a core position is
// comfortably overstocked (see surplusDepthMargin), releasing the weakest player at the single
// most overstocked position - at most one release per call, so a squad's depth erodes gradually
// rather than all at once. Never shrinks the squad below common.SquadSize, the size every team
// starts with. ErrPlayerOnLoan and ErrInsufficientBudget (can't afford the severance right now)
// are expected outcomes, not failures.
func releaseSurplusPlayer(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) {
	if len(players) <= common.SquadSize {
		return
	}

	overstockedPosition := -1
	maxSurplus := surplusDepthMargin
	for _, need := range common.AssessPositionNeeds(players) {
		if surplus := need.Depth - common.MinPositionDepth[need.Position]; surplus > maxSurplus {
			maxSurplus = surplus
			overstockedPosition = need.Position
		}
	}
	if overstockedPosition == -1 {
		return
	}

	var weakest *common.Player
	weakestValue := 0
	for _, p := range players {
		if common.BestCorePosition(*p) != overstockedPosition {
			continue
		}
		if v := common.PositionValue(overstockedPosition, *p); weakest == nil || v < weakestValue {
			weakest, weakestValue = p, v
		}
	}
	if weakest == nil {
		return
	}

	if err := common.ReleasePlayer(dbCtx, weakest.Id, teamId); err != nil &&
		!errors.Is(err, common.ErrPlayerOnLoan) && !errors.Is(err, common.ErrInsufficientBudget) {
		log.Errorf("error releasing surplus player %d for AI-managed team %d: %s", weakest.Id, teamId, err)
	}
}

// signFreeAgentForWeakestPosition signs the best available free agent for a team's single
// neediest core position (see common.WeakestPositionNeed), but only when that position is
// actually thin - a squad that merely has below-average depth everywhere shouldn't go shopping.
// ErrTransferWindowClosed and ErrPlayerNotFree (someone else signed them first) are expected
// outcomes, not failures.
func signFreeAgentForWeakestPosition(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) {
	weakest := common.WeakestPositionNeed(players)
	if weakest == nil || !weakest.Thin {
		return
	}

	freeAgents, err := common.GetFreeAgents(dbCtx)
	if err != nil {
		log.Errorf("error fetching free agents for AI-managed team %d: %s", teamId, err)
		return
	}
	var best *common.Player
	bestValue := 0
	for _, p := range freeAgents {
		if common.BestCorePosition(*p) != weakest.Position {
			continue
		}
		if v := common.PositionValue(weakest.Position, *p); best == nil || v > bestValue {
			best, bestValue = p, v
		}
	}
	if best == nil {
		return
	}

	if _, err := common.SignFreeAgentAtMarketRate(dbCtx, best.Id, teamId); err != nil &&
		!errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrPlayerNotFree) &&
		!errors.Is(err, common.ErrInsufficientBudget) && !errors.Is(err, common.ErrTeamInsolvent) {
		log.Errorf("error signing free agent %d for AI-managed team %d: %s", best.Id, teamId, err)
	}
}

// maximizeTicketPrice charges whichever price maximizes expected matchday revenue (see
// optimalTicketPrice) - simply charging the maximum allowed price used to be this same rational
// move, back when attendance never fell as price rose, but now that demandRate makes attendance
// genuinely price-sensitive, always maxing out would just as often cost a Manager-level AI team
// revenue rather than earn it.
func maximizeTicketPrice(dbCtx *common.DatabaseContext, teamId int64) {
	stadium, err := common.GetStadium(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching stadium for AI-managed team %d ticket pricing: %s", teamId, err)
		return
	}
	optimal := optimalTicketPrice()
	if stadium.TicketPrice == optimal {
		return
	}
	if err := common.SetTicketPrice(dbCtx, teamId, optimal); err != nil {
		log.Errorf("error setting ticket price for AI-managed team %d: %s", teamId, err)
	}
}

// stadiumUpgradeBudgetCushionMultiplier is how large a Director-level team's budget must be
// relative to one stadium upgrade's cost (see common.StadiumUpgradeCostPerCurrentSeat) before
// upgradeStadiumIfAffordable will spend on it - keeps a working cash reserve afterward instead of
// spending down to the wire the moment it can technically afford one.
const stadiumUpgradeBudgetCushionMultiplier = 3

// trainerUpgradeMinSkillMargin is how much better the best available trainer's skill must be
// than a Director-level team's current one (see common.Trainer.Skill) before
// upgradeTrainerIfBetterAvailable will fire the incumbent and hire them - swapping trainers has
// no severance cost, but this still avoids constant churn over a marginal, noise-level difference.
const trainerUpgradeMinSkillMargin = 10

// manageMarketIfDirector runs every Director-level (see common.AiLevelDirector) active-market
// action for an unowned team: shop the transfer list for a genuine upgrade at its neediest core
// position, expand the stadium once it can comfortably afford to, and move to a clearly better
// trainer when one's on the open market. A no-op below Director level - see
// RunAITeamManagement, the only caller.
func manageMarketIfDirector(dbCtx *common.DatabaseContext, team *common.Team) {
	if team.AiLevel < common.AiLevelDirector {
		return
	}

	players, err := common.GetPlayersFromTeamId(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error fetching squad for AI-managed team %d market management: %s", team.Id, err)
		return
	}

	buyTransferListedPlayerForWeakestPosition(dbCtx, team.Id, *players)
	upgradeStadiumIfAffordable(dbCtx, team.Id)
	upgradeTrainerIfBetterAvailable(dbCtx, team.Id)
}

// buyTransferListedPlayerForWeakestPosition spends money on the single best transfer-listed
// upgrade available at a team's neediest core position (see common.WeakestPositionNeed) - but
// only a genuine upgrade over that position's current average quality (PositionNeed.AvgValue),
// unlike signFreeAgentForWeakestPosition which will take any free agent to fill a thin spot since
// that costs nothing. ErrTransferWindowClosed, ErrInsufficientBudget, ErrListingNotFound (someone
// else bought them first) and ErrCannotBuyOwnPlayer are expected outcomes, not failures.
func buyTransferListedPlayerForWeakestPosition(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) {
	weakest := common.WeakestPositionNeed(players)
	if weakest == nil {
		return
	}

	listings, err := common.GetActiveTransferListings(dbCtx)
	if err != nil {
		log.Errorf("error fetching transfer listings for AI-managed team %d: %s", teamId, err)
		return
	}

	var target *common.TransferListing
	bestValue := weakest.AvgValue
	for i := range listings {
		listing := &listings[i]
		if listing.SellingTeamId == teamId || common.BestCorePosition(listing.Player) != weakest.Position {
			continue
		}
		if v := common.PositionValue(weakest.Position, listing.Player); v > bestValue {
			target, bestValue = listing, v
		}
	}
	if target == nil {
		return
	}

	if err := common.ExecuteTransfer(dbCtx, target.ListingId, teamId); err != nil &&
		!errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrInsufficientBudget) &&
		!errors.Is(err, common.ErrListingNotFound) && !errors.Is(err, common.ErrCannotBuyOwnPlayer) &&
		!errors.Is(err, common.ErrTeamInsolvent) {
		log.Errorf("error buying transfer-listed player for AI-managed team %d: %s", teamId, err)
	}
}

// upgradeStadiumIfAffordable expands a team's stadium (see common.UpgradeStadium) once its
// budget comfortably covers the cost - see stadiumUpgradeBudgetCushionMultiplier.
// ErrInsufficientBudget is an expected outcome (the cushion check is a stricter version of the
// same one UpgradeStadium itself makes), not a failure.
func upgradeStadiumIfAffordable(dbCtx *common.DatabaseContext, teamId int64) {
	stadium, err := common.GetStadium(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching stadium for AI-managed team %d stadium upgrade: %s", teamId, err)
		return
	}
	cost := int64(stadium.Capacity) * common.StadiumUpgradeCostPerCurrentSeat

	budget, err := common.GetTeamBudget(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching budget for AI-managed team %d stadium upgrade: %s", teamId, err)
		return
	}
	if budget < cost*stadiumUpgradeBudgetCushionMultiplier {
		return
	}

	if _, err := common.UpgradeStadium(dbCtx, teamId); err != nil && !errors.Is(err, common.ErrInsufficientBudget) {
		log.Errorf("error upgrading stadium for AI-managed team %d: %s", teamId, err)
	}
}

// upgradeTrainerIfBetterAvailable hires the best available trainer (see
// common.GetAvailableTrainers, best-skilled first) when the team has none yet, or fires its
// current one to hire a clearly better one - see trainerUpgradeMinSkillMargin. Trainers have no
// signing fee (same convention as a free agent - see common.HireTrainer), so the only thing
// gating a swap is that margin, not budget.
func upgradeTrainerIfBetterAvailable(dbCtx *common.DatabaseContext, teamId int64) {
	current, err := common.GetTeamTrainer(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching trainer for AI-managed team %d: %s", teamId, err)
		return
	}
	available, err := common.GetAvailableTrainers(dbCtx)
	if err != nil {
		log.Errorf("error fetching available trainers for AI-managed team %d: %s", teamId, err)
		return
	}
	if len(available) == 0 {
		return
	}

	best := available[0]
	if current != nil && best.Skill < current.Skill+trainerUpgradeMinSkillMargin {
		return
	}

	if current != nil {
		if err := common.FireTrainer(dbCtx, current.Id, teamId); err != nil {
			if !errors.Is(err, common.ErrTrainerNotOwnedByTeam) {
				log.Errorf("error firing trainer %d for AI-managed team %d: %s", current.Id, teamId, err)
			}
			return
		}
	}
	if err := common.HireTrainer(dbCtx, best.Id, teamId); err != nil &&
		!errors.Is(err, common.ErrTeamAlreadyHasTrainer) && !errors.Is(err, common.ErrTrainerNotAvailable) {
		log.Errorf("error hiring trainer %d for AI-managed team %d: %s", best.Id, teamId, err)
	}
}

// refreshAiTeamLineup re-picks and persists an unowned team's best-XI default lineup from its
// current squad - the Caretaker-level baseline every AI-managed team gets, every AI level and up.
// Runs every match rather than only at season rollover (see autoManageLineupIfUnowned in
// progression.go, which still covers the retirement/academy-intake refresh at season end) so a
// mid-season squad change - a Level 2/3 signing, release or transfer buy - shows up in the
// default lineup right away instead of waiting for the next season.
func refreshAiTeamLineup(dbCtx *common.DatabaseContext, teamId int64) {
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching squad for AI-managed team %d lineup refresh: %s", teamId, err)
		return
	}
	if err := common.SetDefaultLineup(dbCtx, teamId, common.FlattenLineup(common.SelectBestLineup(*players))); err != nil {
		log.Errorf("error refreshing lineup for AI-managed team %d: %s", teamId, err)
	}
}
