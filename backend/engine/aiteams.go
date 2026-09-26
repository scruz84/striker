package engine

import (
	"errors"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"sort"
	"time"
)

// isAiManagedTeam reports whether a team is both unowned and has an AI level entitling it to any
// automated behavior at all (see common.AiLevelCaretaker/Manager/Director) - the one gate shared
// by every entry point into AI-team automation (RunAITeamManagement, RespondToAllPendingOffers,
// RunAiTeamDailyActivity), so the definition of "AI-managed" can't drift between them.
func isAiManagedTeam(team *common.Team) bool {
	return team.OwnerId <= 0 && team.AiLevel >= common.AiLevelCaretaker
}

// RunAITeamManagement drives every automatic action an unowned team's AI level entitles it to -
// called once per match for each side involved (see UpdateTeamFinancesAfterMatch), and once more
// a day (on average) for every AI-managed team regardless of match activity (see
// RunAiTeamDailyActivity/StartAiActivityScheduler) - safe to call redundantly since every action
// inside is itself a no-op once there's nothing left to do (nothing to renew, no surplus to list,
// etc.). See common.AiLevelCaretaker/Manager/Director for what each tier adds on top of the one
// below it; a no-op for any team with a real owner.
func RunAITeamManagement(dbCtx *common.DatabaseContext, teamId int64) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching team %d for AI management: %s", teamId, err)
		return
	}
	if !isAiManagedTeam(team) {
		return
	}

	refreshAiTeamLineup(dbCtx, teamId)
	// Responding to incoming offers is baseline Caretaker-level behavior (see
	// AiLevelCaretaker/Manager/Director's own doc comment: "Caretaker only reacts ... accepts
	// clearly-good incoming offers") - it belongs here, unconditional at every AI level, not
	// gated behind Manager inside manageRosterIfManager the way it used to be. Before this, a
	// human's offer on a Caretaker-level AI team's player got no automated response at all and
	// sat pending forever (issue #61).
	respondToReceivedOffers(dbCtx, team.Id)
	manageRosterIfManager(dbCtx, team)
	manageMarketIfDirector(dbCtx, team)
}

// RespondToAllPendingOffers sweeps every unowned team's pending incoming transfer offers once or
// twice a day (see engine.StartOfferResponseScheduler), independent of match activity.
// RunAITeamManagement above only fires when that team's own match is simulated, which could
// otherwise leave a human's offer on an AI-managed team unanswered for as long as that team goes
// between matches - issue #61 ("no news from the CPU team ... the offer can remain one or two
// days, then the club must respond").
func RespondToAllPendingOffers(dbCtx *common.DatabaseContext) {
	teamIds, err := common.GetAllTeamIds(dbCtx)
	if err != nil {
		log.Errorf("error fetching team ids for offer response sweep: %s", err)
		return
	}
	for _, teamId := range teamIds {
		team, err := common.GetTeamById(teamId, dbCtx)
		if err != nil {
			log.Errorf("error fetching team %d for offer response sweep: %s", teamId, err)
			continue
		}
		if !isAiManagedTeam(team) {
			continue
		}
		respondToReceivedOffers(dbCtx, teamId)
	}
}

// contractRenewalThresholdYears is how few seasons a Manager-level AI team lets a player's
// contract run down to before proactively renewing it (see renewExpiringContracts) - a real
// owner wouldn't let a useful squad player walk for nothing at season end just from inattention.
const contractRenewalThresholdYears = 1

// surplusDepthMargin is how far a core position's depth (see common.AssessPositionNeeds) must
// exceed common.MinPositionDepth before listSurplusPlayer treats it as overstocked enough to
// trim - one full backup beyond the minimum, so a position sitting exactly at minimum depth is
// never touched.
const surplusDepthMargin = 1

// manageRosterIfManager runs every Manager-level (see common.AiLevelManager) roster/business
// action for an unowned team: keep valuable players under contract, trim genuinely surplus
// depth, fill a thin position from the free-agent market, and charge what the gate will bear.
// A no-op below Manager level - see RunAITeamManagement, the only caller. Responding to incoming
// offers is handled a level below this, in RunAITeamManagement itself, not here - see its comment.
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
	listSurplusPlayer(dbCtx, team.Id, *players)
	signFreeAgentForWeakestPosition(dbCtx, team.Id, *players)
	maximizeTicketPrice(dbCtx, team.Id)
}

// minAcceptableTransferFeeWageMultiplier is how many times a player's own market-rate wage (see
// common.PlayerWageFromSkills) an incoming transfer offer's fee must be worth before an
// AI-managed team accepts it - low enough that a genuinely competitive offer for a squad
// player gets taken, high enough that a lowball offer aimed at poaching a good player for next to
// nothing gets rejected instead. Mirrors EarlyReleaseWageMultiplier's use of wage as this file's
// common yardstick for a player's worth. A sale thinning the squad below common.SquadSize is not
// specially guarded against here: signFreeAgentForWeakestPosition (run just before this, every
// time) reactively refills whichever position ends up thinnest, the same way it already covers a
// retirement or an expired contract - there's nothing about a sale that needs different handling.
const minAcceptableTransferFeeWageMultiplier = 20

// respondToReceivedOffers judges every pending offer on an AI-managed team's own players (any AI
// level, Caretaker and up - see RunAITeamManagement/RespondToAllPendingOffers, its two callers):
// accept a fee that clears minAcceptableTransferFeeWageMultiplier (the personal terms are already
// settled - see MakeTransferOffer, which never even creates an offer the player themself wouldn't
// accept), reject anything less.
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

// minSquadSizeAfterSale is the smallest owned, unlisted senior squad an AI-managed team will
// still have once a listed player is sold - listSurplusPlayer won't list below it, and
// withdrawSurplusListings pulls listings back once the squad has shrunk under it. Deliberately a
// couple of players under common.SquadSize: every team starts at exactly SquadSize and the
// academy only tops back up to exactly SquadSize, so requiring a squad above it (as this used to)
// meant almost no CPU team ever qualified to list anyone. The per-position surplus check
// (surplusDepthMargin) is what really keeps a sale from hurting the team.
const minSquadSizeAfterSale = common.SquadSize - 2

// listSurplusPlayer keeps an AI-managed team's transfer listings in line with its squad: first
// withdraws any listing that's no longer genuine surplus (see planListingWithdrawals), then lists
// at most one more player for transfer when a core position is comfortably overstocked (see
// pickSurplusPlayerToList) - one per call, so a squad's depth erodes gradually rather than all at
// once, but listings do accumulate over successive matches while the surplus lasts. Lists rather
// than releasing so a human browsing the transfer market gets a chance to buy a CPU team's
// genuine squad surplus. The asking price is minAcceptableTransferFeeWageMultiplier times the
// player's own market-rate wage - the same yardstick this team uses to judge an incoming offer on
// that player (see respondToReceivedOffers). Loaned-in, already-listed and loan-listed players
// are filtered out up front (see common.GetTeamMarketCommitments) rather than picked and then
// silently rejected, which used to leave a team stuck retrying the same unlistable player every
// match. The transfer window being closed is an expected outcome, not a failure.
func listSurplusPlayer(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) {
	commitments, err := common.GetTeamMarketCommitments(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching market commitments for AI-managed team %d: %s", teamId, err)
		return
	}

	owned := ownedPlayers(players, commitments.LoanedIn)
	for _, p := range planListingWithdrawals(owned, commitments.TransferListed) {
		if err := common.CancelTransferListing(dbCtx, p.Id, teamId); err != nil &&
			!errors.Is(err, common.ErrPlayerNotListedByTeam) {
			log.Errorf("error withdrawing listing for player %d of AI-managed team %d: %s", p.Id, teamId, err)
			continue
		}
		delete(commitments.TransferListed, p.Id)
	}

	target := pickSurplusPlayerToList(owned, commitments.TransferListed, commitments.LoanListed)
	if target == nil {
		return
	}

	expectedWage := common.PlayerWageFromSkills(target.GoalKeeper, target.Defense, target.Lateral, target.Pass,
		target.Midfield, target.Dribbling, target.Shot)
	askingPrice := int64(expectedWage * minAcceptableTransferFeeWageMultiplier)

	if err := common.CreateTransferListing(dbCtx, target.Id, teamId, askingPrice); err != nil &&
		!errors.Is(err, common.ErrPlayerAlreadyListed) && !errors.Is(err, common.ErrTransferWindowClosed) &&
		!errors.Is(err, common.ErrPlayerOnLoan) && !errors.Is(err, common.ErrPlayerAlreadyLoanListed) {
		log.Errorf("error listing surplus player %d for AI-managed team %d: %s", target.Id, teamId, err)
	}
}

// ownedPlayers drops every player on loan to this team from another club - they count towards
// neither the squad size nor the position depth listSurplusPlayer judges surplus by, since they
// aren't this team's to keep or sell.
func ownedPlayers(players []*common.Player, loanedIn map[int64]bool) []*common.Player {
	owned := make([]*common.Player, 0, len(players))
	for _, p := range players {
		if !loanedIn[p.Id] {
			owned = append(owned, p)
		}
	}
	return owned
}

// withoutPlayers returns players minus every id in excluded.
func withoutPlayers(players []*common.Player, excluded map[int64]bool) []*common.Player {
	kept := make([]*common.Player, 0, len(players))
	for _, p := range players {
		if !excluded[p.Id] {
			kept = append(kept, p)
		}
	}
	return kept
}

// positionDepth counts players whose best core position is position.
func positionDepth(players []*common.Player, position int) int {
	depth := 0
	for _, p := range players {
		if common.BestCorePosition(*p) == position {
			depth++
		}
	}
	return depth
}

// planListingWithdrawals picks which of an AI-managed team's current transfer listings to take
// back off the market: any listed player whose position would be thin without them (depth among
// the owned, unlisted squad below common.MinPositionDepth - e.g. after a sale via an accepted
// offer or a retirement at that position), or any listing while the unlisted squad has
// shrunk below minSquadSizeAfterSale. Each withdrawn player rejoins the unlisted squad before the
// next listing is judged, so only as many come back as are needed. The thresholds sit one player
// below pickSurplusPlayerToList's, so a listing it just made is never immediately withdrawn.
func planListingWithdrawals(owned []*common.Player, listed map[int64]bool) []*common.Player {
	available := withoutPlayers(owned, listed)
	withdrawals := make([]*common.Player, 0)
	for _, p := range owned {
		if !listed[p.Id] {
			continue
		}
		position := common.BestCorePosition(*p)
		if positionDepth(available, position) < common.MinPositionDepth[position] ||
			len(available) < minSquadSizeAfterSale {
			withdrawals = append(withdrawals, p)
			available = append(available, p)
		}
	}
	return withdrawals
}

// pickSurplusPlayerToList chooses the one player an AI-managed team should put up for transfer
// next, or nil if none: the weakest listable player at the most overstocked core position, where
// overstocked means the owned, unlisted depth there exceeds common.MinPositionDepth by more than
// surplusDepthMargin, and only while selling them still leaves at least minSquadSizeAfterSale
// owned, unlisted players. A loan-listed player can't also be transfer-listed, so they're skipped
// - falling through to the next most overstocked position if nobody else at this one qualifies.
func pickSurplusPlayerToList(owned []*common.Player, listed map[int64]bool, loanListed map[int64]bool) *common.Player {
	available := withoutPlayers(owned, listed)
	if len(available)-1 < minSquadSizeAfterSale {
		return nil
	}

	needs := common.AssessPositionNeeds(available)
	sort.SliceStable(needs, func(i, j int) bool {
		return needs[i].Depth-common.MinPositionDepth[needs[i].Position] >
			needs[j].Depth-common.MinPositionDepth[needs[j].Position]
	})
	for _, need := range needs {
		if need.Depth-common.MinPositionDepth[need.Position] <= surplusDepthMargin {
			break
		}
		var weakest *common.Player
		weakestValue := 0
		for _, p := range available {
			if loanListed[p.Id] || common.BestCorePosition(*p) != need.Position {
				continue
			}
			if v := common.PositionValue(need.Position, *p); weakest == nil || v < weakestValue {
				weakest, weakestValue = p, v
			}
		}
		if weakest != nil {
			return weakest
		}
	}
	return nil
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

	if _, err := common.UpgradeStadium(dbCtx, teamId, common.StadiumUpgradeSeats); err != nil &&
		!errors.Is(err, common.ErrInsufficientBudget) && !errors.Is(err, common.ErrStadiumUpgradeInProgress) {
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

// aiActivityTicksPerDay is how many times StartAiActivityScheduler's job fires per day (every 5
// minutes - see its own comment) - the denominator ProbAiDailyActivityPerTick is built from.
const aiActivityTicksPerDay = 24 * 60 / 5

// ProbAiDailyActivityPerTick is calibrated so each eligible team's chance of triggering on any
// given tick averages out to once a day - spread randomly across it (whichever tick happens to
// roll true first) rather than every team acting in lockstep at the same fixed hour, the same way
// a single shared probability already spreads out per-minute match events (see e.g.
// engine.ProbChance) instead of scheduling them deterministically.
const ProbAiDailyActivityPerTick = float32(1) / float32(aiActivityTicksPerDay)

// RunAiTeamDailyActivity gives every AI-managed team (see isAiManagedTeam) an independent,
// randomly-timed chance each tick (see StartAiActivityScheduler) to run its full market
// management pass and to challenge another AI-managed team to a friendly - both entirely
// decoupled from match simulation, unlike RunAITeamManagement's per-match trigger. Before this, a
// freshly created world's CPU teams sat completely idle in the transfer market (and never played
// each other) until their very first league/cup match was simulated; this guarantees market
// activity - and a friendly habit - from day one, whether or not any competitive match has been
// played yet. A friendly opponent is deliberately only ever another AI-managed team, never a
// human's - a human still only ever gets challenged by another human (see ChallengeToFriendly),
// so this can't spam a manager's inbox with unsolicited CPU challenges.
func RunAiTeamDailyActivity(dbCtx *common.DatabaseContext) {
	teamIds, err := common.GetAllTeamIds(dbCtx)
	if err != nil {
		log.Errorf("error listing teams for AI daily activity: %s", err)
		return
	}

	teams := make([]*common.Team, 0, len(teamIds))
	for _, teamId := range teamIds {
		team, err := common.GetTeamById(teamId, dbCtx)
		if err != nil {
			log.Errorf("error fetching team %d for AI daily activity: %s", teamId, err)
			continue
		}
		teams = append(teams, team)
	}

	eligible := filterAiManagedTeams(teams)
	eligibleIds := make([]int64, len(eligible))
	for i, team := range eligible {
		eligibleIds[i] = team.Id
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, team := range eligible {
		if r.Float32() < ProbAiDailyActivityPerTick {
			RunAITeamManagement(dbCtx, team.Id)
		}
		if r.Float32() < ProbAiDailyActivityPerTick {
			challengeAiTeamToFriendly(dbCtx, r, team.Id, eligibleIds)
		}
	}
}

// filterAiManagedTeams narrows teams down to the ones isAiManagedTeam accepts, preserving order.
func filterAiManagedTeams(teams []*common.Team) []*common.Team {
	eligible := make([]*common.Team, 0, len(teams))
	for _, team := range teams {
		if isAiManagedTeam(team) {
			eligible = append(eligible, team)
		}
	}
	return eligible
}

// pickRandomOpponent picks a uniformly random id from teamIds other than exclude, or false if
// exclude is the only (or not a genuine) candidate in there.
func pickRandomOpponent(r *rand.Rand, teamIds []int64, exclude int64) (int64, bool) {
	candidates := make([]int64, 0, len(teamIds))
	for _, id := range teamIds {
		if id != exclude {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		return 0, false
	}
	return candidates[r.Intn(len(candidates))], true
}

// challengeAiTeamToFriendly has teamId challenge a random other AI-managed team (from
// eligibleIds) to a friendly. Both sides being unowned means ChallengeToFriendly auto-accepts on
// the spot (see its own doc comment) - there's nobody who'd otherwise need to act on it.
// ErrFriendlyLimitReached (either side already at its weekly cap - the default is just 1) and
// ErrFriendlyChallengeAlreadyExists (a pending challenge between this exact pair still
// outstanding, which shouldn't happen between two auto-accepting sides but costs nothing to guard
// against) are expected outcomes, not failures.
func challengeAiTeamToFriendly(dbCtx *common.DatabaseContext, r *rand.Rand, teamId int64, eligibleIds []int64) {
	opponent, ok := pickRandomOpponent(r, eligibleIds, teamId)
	if !ok {
		return
	}
	if err := common.ChallengeToFriendly(dbCtx, teamId, opponent, nil); err != nil &&
		!errors.Is(err, common.ErrFriendlyLimitReached) && !errors.Is(err, common.ErrFriendlyChallengeAlreadyExists) {
		log.Errorf("error challenging team %d to a friendly for AI-managed team %d: %s", opponent, teamId, err)
	}
}
