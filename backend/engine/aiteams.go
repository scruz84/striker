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
// by every entry point into AI-team automation (RunAITeamManagement, RunAITeamMatchReaction,
// RespondToAllPendingOffers, RunAiTeamDailyActivity), so the definition of "AI-managed" can't
// drift between them.
func isAiManagedTeam(team *common.Team) bool {
	return team.OwnerId <= 0 && team.AiLevel >= common.AiLevelCaretaker
}

// aiTeamBaselineReaction fetches teamId and, if it's AI-managed, runs the baseline reaction every
// AI-managed team takes regardless of when it runs: refresh its default lineup from the current
// squad, and answer any offer sitting on its own players. Shared by RunAITeamMatchReaction and
// RunAITeamManagement so the two can't silently drift apart on this shared baseline - each adds
// its own extra behavior on top (or none, for RunAITeamMatchReaction). Returns the fetched team
// and whether it actually ran (false, nil-safe to use, for a fetch error or a non-AI-managed
// team) so a caller that needs the team for more than this baseline doesn't have to fetch it
// twice.
func aiTeamBaselineReaction(dbCtx *common.DatabaseContext, teamId int64) (*common.Team, bool) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching team %d for AI reaction: %s", teamId, err)
		return nil, false
	}
	if !isAiManagedTeam(team) {
		return team, false
	}

	refreshAiTeamLineup(dbCtx, teamId, team.AiLevel)
	respondToReceivedOffers(dbCtx, team.Id)
	return team, true
}

// RunAITeamMatchReaction runs the baseline, time-insensitive reaction every AI-managed team takes
// right after one of its own matches simulates (see UpdateTeamFinancesAfterMatch, its only
// caller) - reactive to something that just happened to this team specifically, so it doesn't
// need to wait for the local-hour-gated daily tick the way initiating new market/roster action
// does (see RunAITeamManagement). Responding to incoming offers is baseline Caretaker-level
// behavior (see AiLevelCaretaker/Manager/Director's own doc comment: "Caretaker only reacts ...
// accepts clearly-good incoming offers") - unconditional at every AI level, not gated behind
// Manager. Before this, a human's offer on a Caretaker-level AI team's player got no automated
// response at all and sat pending forever (issue #61). A no-op for any team with a real owner.
func RunAITeamMatchReaction(dbCtx *common.DatabaseContext, teamId int64) {
	aiTeamBaselineReaction(dbCtx, teamId)
}

// RunAITeamManagement drives every proactive market/roster action an unowned team's AI level
// entitles it to - called only from the local-hour-gated daily tick (see
// RunAiTeamDailyActivity/StartAiActivityScheduler), never synchronously off a match result, so
// that every AI-managed team's own country's local awake hours decide when it initiates new
// market activity rather than all of them firing together at whatever single hour the admin-
// configured match schedule happens to use (see common.GetMatchSchedule). Safe to call
// redundantly since every action inside is itself a no-op once there's nothing left to do
// (nothing to renew, no surplus to list, etc.). See common.AiLevelCaretaker/Manager/Director for
// what each tier adds on top of the one below it - for transfers, common.AiTransferProfile
// spells out exactly what - and a no-op for any team with a real owner.
func RunAITeamManagement(dbCtx *common.DatabaseContext, teamId int64) {
	team, ran := aiTeamBaselineReaction(dbCtx, teamId)
	if !ran {
		return
	}

	manageRosterIfManager(dbCtx, team)
	runAiTransferPass(dbCtx, team)
	manageMarketIfDirector(dbCtx, team)
}

// RespondToAllPendingOffers sweeps every unowned team's pending incoming transfer offers once or
// twice a day (see engine.StartOfferResponseScheduler), independent of match activity.
// RunAITeamMatchReaction above only fires when that team's own match is simulated, and
// RunAITeamManagement only within its own local-hour window, either of which could otherwise
// leave a human's offer on an AI-managed team unanswered for as long as that team goes between
// matches and never happens to roll its daily tick - issue #61 ("no news from the CPU team ...
// the offer can remain one or two days, then the club must respond").
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
// depth, promote a cantera prospect into a thin position, and charge what the gate will bear.
// Signing from outside (free agents, the transfer list, unlisted players) is the transfer pass's
// job - see runAiTransferPass. A no-op below Manager level - see RunAITeamManagement, the only caller. Responding to incoming
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
	listSurplusPlayer(dbCtx, team, *players)
	manageLoanListings(dbCtx, team, *players)
	promoteFromAcademyForWeakestPosition(dbCtx, team.Id, *players)
	maximizeTicketPrice(dbCtx, team.Id)
}

// promoteFromAcademyForWeakestPosition has a Manager-level AI-managed team weigh its own cantera
// (see common.GetCantera) against its single neediest core position (see
// common.WeakestPositionNeed), exactly the choice CanPromoteFromAcademy/PromoteFromAcademy exist
// to offer a human manager - a CPU team should take the option into account too rather than let
// prospects sit in the academy forever. Same "only when actually thin" trigger as
// signFreeAgentForWeakestPosition (both are a free source of bodies, so neither should fire just
// because a position has below-average depth), and deliberately tried first: promoting one of the
// team's own prospects costs nothing extra beyond the weekly cap, so it's preferred over reaching
// for an outside free agent when both are on the table for the same thin spot. Reports whether it
// promoted anyone, so manageRosterIfManager can skip the free-agent path this tick for the same
// position it just filled - the stale `players` snapshot both act on wouldn't otherwise reflect
// the promotion, and filling one thin position twice in the same pass has no benefit. The weekly
// cap (ErrAcademyPromotionAlreadyUsedThisWeek) is an expected outcome, not a failure - it just
// means the option was already taken this week, by this call or an earlier one.
func promoteFromAcademyForWeakestPosition(dbCtx *common.DatabaseContext, teamId int64, players []*common.Player) bool {
	weakest := common.WeakestPositionNeed(players)
	if weakest == nil || !weakest.Thin {
		return false
	}

	canPromote, _, err := common.CanPromoteFromAcademy(dbCtx, teamId)
	if err != nil {
		log.Errorf("error checking academy promotion availability for AI-managed team %d: %s", teamId, err)
		return false
	}
	if !canPromote {
		return false
	}

	cantera, err := common.GetCantera(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching cantera for AI-managed team %d: %s", teamId, err)
		return false
	}
	var best *common.Player
	bestValue := 0
	for _, p := range cantera {
		if common.BestCorePosition(*p) != weakest.Position {
			continue
		}
		if v := common.PositionValue(weakest.Position, *p); best == nil || v > bestValue {
			best, bestValue = p, v
		}
	}
	if best == nil {
		return false
	}

	if err := common.PromoteFromAcademy(dbCtx, best.Id, teamId); err != nil {
		if !errors.Is(err, common.ErrAcademyPromotionAlreadyUsedThisWeek) {
			log.Errorf("error promoting academy prospect %d for AI-managed team %d: %s", best.Id, teamId, err)
		}
		return false
	}
	return true
}

// respondToReceivedOffers judges every pending offer on an AI-managed team's own players (any AI
// level, Caretaker and up - see RunAITeamMatchReaction/RunAITeamManagement/
// RespondToAllPendingOffers, its three callers): accept a fee that clears the team's valuation of
// the player, reject anything less. How the valuation is formed, and how much a key starter or a
// national-team player is worth on top of it, is the team's AiTransferProfile - see
// common.EvaluateIncomingOffer. The personal terms are already settled (see MakeTransferOffer,
// which never even creates an offer the player themself wouldn't accept). The highest offers are
// judged first, and each offer's noise is seeded from its own id so a pending offer that is
// waiting out its negotiation window gets the same answer on every sweep instead of being
// re-rolled. An offer that is good but still inside the negotiation window is simply left pending.
func respondToReceivedOffers(dbCtx *common.DatabaseContext, teamId int64) {
	team, err := common.GetTeamById(teamId, dbCtx)
	if err != nil {
		log.Errorf("error fetching AI-managed team %d to respond to offers: %s", teamId, err)
		return
	}
	offers, err := common.GetOffersReceived(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching received offers for AI-managed team %d: %s", teamId, err)
		return
	}
	if len(offers) == 0 {
		return
	}
	sort.SliceStable(offers, func(i, j int) bool { return offers[i].Amount > offers[j].Amount })
	profile := common.AiTransferProfileFor(team.AiLevel)

	for _, offer := range offers {
		squad, err := ownedSquad(dbCtx, teamId)
		if err != nil {
			log.Errorf("error fetching squad for AI-managed team %d offer response: %s", teamId, err)
			return
		}
		international, nationalCaps := nationalStanding(dbCtx, offer.Player.Id)
		decision := common.EvaluateIncomingOffer(profile, common.OfferContext{
			Squad:             squad,
			Player:            offer.Player,
			Caps:              skillCapsFor(dbCtx, offer.Player.Id),
			Amount:            offer.Amount,
			International:     international,
			InternationalCaps: nationalCaps,
		}, rand.New(rand.NewSource(offer.OfferId)))

		if !decision.Accept {
			if err := common.RejectOffer(dbCtx, offer.OfferId, teamId); err != nil {
				log.Errorf("error rejecting offer %d for AI-managed team %d: %s", offer.OfferId, teamId, err)
			}
			continue
		}

		if err := common.AcceptOffer(dbCtx, offer.OfferId, teamId); err != nil &&
			!errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrOfferNoLongerValid) &&
			!errors.Is(err, common.ErrInsufficientBudget) && !errors.Is(err, common.ErrOfferNegotiationWindow) {
			log.Errorf("error accepting offer %d for AI-managed team %d: %s", offer.OfferId, teamId, err)
		}
	}
}

// ownedSquad is a team's senior squad minus anyone on loan to it from another club - the players
// it can actually keep or sell, which is what every transfer decision weighs.
func ownedSquad(dbCtx *common.DatabaseContext, teamId int64) ([]*common.Player, error) {
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	commitments, err := common.GetTeamMarketCommitments(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	return ownedPlayers(*players, commitments.LoanedIn), nil
}

// nationalStanding reports whether a player is in his country's national-team pool and his caps,
// per the (cached) national radar; a radar that can't be loaded just means no internationals.
func nationalStanding(dbCtx *common.DatabaseContext, playerId int64) (bool, int) {
	radar, err := common.GetNationalRadar(dbCtx)
	if err != nil {
		log.Errorf("error loading the national-team radar: %s", err)
		return false, 0
	}
	entry, ok := radar.Lookup(playerId)
	return ok, entry.Caps
}

// skillCapsFor reads one player's hidden skill ceilings for a transfer valuation, nil when they
// can't be read - the valuation then assumes a typical headroom.
func skillCapsFor(dbCtx *common.DatabaseContext, playerId int64) *common.PlayerSkillCaps {
	caps, err := common.GetPlayerSkillCaps(dbCtx, []int64{playerId})
	if err != nil {
		log.Errorf("error reading skill caps of player %d: %s", playerId, err)
		return nil
	}
	if c, ok := caps[playerId]; ok {
		return &c
	}
	return nil
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
const minSquadSizeAfterSale = common.MinSquadSizeAfterSale

// listSurplusPlayer keeps an AI-managed team's transfer listings in line with its squad: first
// withdraws any listing that's no longer genuine surplus (see planListingWithdrawals), then lists
// at most one more player for transfer when a core position is comfortably overstocked (see
// pickSurplusPlayerToList) - one per call, so a squad's depth erodes gradually rather than all at
// once, but listings do accumulate over successive matches while the surplus lasts. Lists rather
// than releasing so a human browsing the transfer market gets a chance to buy a CPU team's
// genuine squad surplus. The asking price is the player's market value (see
// common.ListingAskingPrice) - the same yardstick this team starts from when judging an incoming
// offer on that player (see respondToReceivedOffers). Loaned-in, already-listed and loan-listed players
// are filtered out up front (see common.GetTeamMarketCommitments) rather than picked and then
// silently rejected, which used to leave a team stuck retrying the same unlistable player every
// match. The transfer window being closed is an expected outcome, not a failure.
func listSurplusPlayer(dbCtx *common.DatabaseContext, team *common.Team, players []*common.Player) {
	if !common.AiTransferProfileFor(team.AiLevel).ListsSurplus {
		return
	}
	teamId := team.Id
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

	target := pickSurplusPlayerToList(owned, commitments.TransferListed, commitments.LoanListed, protectedFromListing(dbCtx, team, owned))
	if target == nil {
		return
	}

	international, nationalCaps := nationalStanding(dbCtx, target.Id)
	askingPrice := common.ListingAskingPrice(*target, common.AiTransferProfileFor(team.AiLevel), international, nationalCaps)

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

// recentlySignedOrInternational is who a CPU team never sells or lends however surplus they look:
// anyone it signed within common.AiMinHoldPeriod and every international (a player in his
// country's national-team pool - see common.NationalRadar).
func recentlySignedOrInternational(dbCtx *common.DatabaseContext, team *common.Team, owned []*common.Player) map[int64]bool {
	protected, err := common.GetPlayerIdsAcquiredSince(dbCtx, team.Id, time.Now().Add(-common.AiMinHoldPeriod))
	if err != nil {
		log.Errorf("error reading recent signings of AI-managed team %d: %s", team.Id, err)
		protected = map[int64]bool{}
	}
	if radar, err := common.GetNationalRadar(dbCtx); err != nil {
		log.Errorf("error loading the national-team radar for AI-managed team %d: %s", team.Id, err)
	} else {
		for _, p := range owned {
			if _, ok := radar.Lookup(p.Id); ok {
				protected[p.Id] = true
			}
		}
	}
	return protected
}

// protectedFromListing is who a CPU team won't list as surplus: anyone it signed within
// common.AiMinHoldPeriod, every international, and - for a profile that signs projects - every
// player young enough to still be one, since a prospect on the bench of a deep position is
// exactly what such a team bought him to be.
func protectedFromListing(dbCtx *common.DatabaseContext, team *common.Team, owned []*common.Player) map[int64]bool {
	protected := recentlySignedOrInternational(dbCtx, team, owned)
	if common.AiTransferProfileFor(team.AiLevel).SignsProjects {
		for _, p := range owned {
			if p.Age < common.ProjectionMaxAge {
				protected[p.Id] = true
			}
		}
	}
	return protected
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
// owned, unlisted players. A loan-listed player can't also be transfer-listed, and a protected one
// (see protectedFromListing) isn't for sale, so both are skipped - falling through to the next
// most overstocked position if nobody else at this one qualifies.
func pickSurplusPlayerToList(owned []*common.Player, listed map[int64]bool, loanListed map[int64]bool, protected map[int64]bool) *common.Player {
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
			if loanListed[p.Id] || protected[p.Id] || common.BestCorePosition(*p) != need.Position {
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

// maximizeTicketPrice charges whichever price maximizes expected matchday revenue for this team's
// own stadium, socios and recent form (see optimalTicketPrice) - the price its crowd will accept,
// not simply the maximum allowed: demand falls to nothing at SociosPriceZeroAt, so always maxing
// out would empty the ground.
func maximizeTicketPrice(dbCtx *common.DatabaseContext, teamId int64) {
	stadium, err := common.GetStadium(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching stadium for AI-managed team %d ticket pricing: %s", teamId, err)
		return
	}
	optimal, err := RecommendedTicketPrice(dbCtx, teamId)
	if err != nil {
		log.Errorf("error computing ticket price for AI-managed team %d: %s", teamId, err)
		return
	}
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
// action for an unowned team that isn't a signing: expand the stadium once it can comfortably
// afford to, and move to a clearly better trainer when one's on the open market. (A Director's
// transfer shopping is part of the transfer pass every Manager-and-up team runs, widened by its
// profile - see runAiTransferPass.) A no-op below Director level - see RunAITeamManagement, the
// only caller.
func manageMarketIfDirector(dbCtx *common.DatabaseContext, team *common.Team) {
	if team.AiLevel < common.AiLevelDirector {
		return
	}

	upgradeStadiumIfAffordable(dbCtx, team.Id)
	upgradeTrainerIfBetterAvailable(dbCtx, team.Id)
}

// stadiumUpgradeMinRecentSelloutShare/stadiumUpgradeMinRecentHomeMatches: a Director only expands
// when demand really outgrew the ground - at least this share of its latest home league matches
// (needing at least this many on record) drew a full house (see shouldUpgradeStadium).
const stadiumUpgradeMinRecentSelloutShare = 0.4
const stadiumUpgradeMinRecentHomeMatches = 5

// shouldUpgradeStadium is the pure demand half of the expansion rule: enough recent home matches
// on record, and a big enough share of them sold out.
func shouldUpgradeStadium(soldOut int, total int) bool {
	return total >= stadiumUpgradeMinRecentHomeMatches &&
		float64(soldOut) >= stadiumUpgradeMinRecentSelloutShare*float64(total)
}

// upgradeStadiumIfAffordable expands a team's stadium (see common.UpgradeStadium) once its
// budget comfortably covers the cost - see stadiumUpgradeBudgetCushionMultiplier - and its
// members have been filling the ground (see shouldUpgradeStadium): having the cash is no longer
// reason enough to build seats nobody turns up to fill.
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

	soldOut, total, err := common.GetRecentHomeSellouts(dbCtx, teamId, stadium.Capacity)
	if err != nil {
		log.Errorf("error fetching recent attendance for AI-managed team %d stadium upgrade: %s", teamId, err)
		return
	}
	if !shouldUpgradeStadium(soldOut, total) {
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

// refreshAiTeamLineup re-picks and persists an unowned team's formation and that formation's
// best-XI from its current squad - the Caretaker-level baseline every AI-managed team gets, every
// AI level and up. Which formation it settles for isn't fixed: common.SelectBestFormationAndLineup
// compares the squad's best-XI strength across every formation and picks the rank its own AI level
// (aiLevel) settles for - only Director reliably finds the genuinely strongest shape, Manager
// settles for the second-best, Caretaker further still (see that function and
// common.formationRankForAiLevel) - so a CPU team's formation varies with its squad and its
// sharpness instead of every unowned team defaulting to FormationBalanced's 4-4-2 forever. Runs
// every match rather than only at season rollover (see autoManageLineupIfUnowned in
// progression.go, which still covers the retirement/academy-intake refresh at season end) so a
// mid-season squad change - a Level 2/3 signing, release or transfer buy - shows up in the
// default lineup (and, now, potentially a reconsidered formation) right away instead of waiting
// for the next season.
func refreshAiTeamLineup(dbCtx *common.DatabaseContext, teamId int64, aiLevel int) {
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching squad for AI-managed team %d lineup refresh: %s", teamId, err)
		return
	}
	formation, lineup := common.SelectBestFormationAndLineup(*players, aiLevel, rand.New(rand.NewSource(time.Now().UnixNano())))
	if err := common.SetTeamFormation(dbCtx, teamId, formation); err != nil {
		log.Errorf("error setting formation for AI-managed team %d: %s", teamId, err)
	}
	if err := common.SetDefaultLineup(dbCtx, teamId, common.FlattenLineup(lineup)); err != nil {
		log.Errorf("error refreshing lineup for AI-managed team %d: %s", teamId, err)
	}
}

// localActivityWindowTicksPerDay is how many of StartAiActivityScheduler's 5-minute ticks fall
// inside a team's own local awake-hours window (see common.LocalActivityStartHour/EndHour) on any
// given day - the denominator ProbAiLocalActivityPerTick is built from.
const localActivityWindowTicksPerDay = (common.LocalActivityEndHour - common.LocalActivityStartHour) * 60 / 5

// ProbAiLocalActivityPerTick is calibrated so each eligible team's chance of triggering on any
// given tick that falls inside its own local awake-hours window (see
// common.IsWithinLocalActivityWindow) averages out to once a day - spread randomly across that
// window (whichever tick happens to roll true first) rather than every team acting in lockstep at
// the same fixed hour, the same way a single shared probability already spreads out per-minute
// match events (see e.g. engine.ProbChance) instead of scheduling them deterministically. Gating
// on the window first, rather than just lowering this probability to cover the full 24h, is what
// keeps a team's own activity out of its local small hours.
const ProbAiLocalActivityPerTick = float32(1) / float32(localActivityWindowTicksPerDay)

// aiFriendlyChallengeAttemptsPerDay is how many expected dice rolls a team's own local
// awake-hours window gives it to challenge another AI-managed team to a friendly -
// ProbAiLocalActivityPerTick averaging out to a single attempt a day is fine for market
// management (skipping a day there costs nothing), but a skipped day here is a wasted friendly
// slot: CPU teams sitting on an unused weekly allowance instead of actually playing through it.
// At this rate the chance of a whole window passing with zero attempts is negligible (~(1 -
// aiFriendlyChallengeAttemptsPerDay/localActivityWindowTicksPerDay)^localActivityWindowTicksPerDay
// ≈ e^-10, well under 0.01%), while still landing at a randomly-timed tick rather than
// deterministically at the window's first one. challengeAiTeamToFriendly's own "already has one
// today" guard is what actually keeps this to one per day despite the many rolls, and what lets
// the days a team's weekly cap isn't yet spent actually get used, one per day, instead of a few
// lucky days exhausting the whole week's allowance while the rest go without.
const aiFriendlyChallengeAttemptsPerDay = 10

// ProbAiFriendlyChallengePerTick is the per-tick dice an eligible team rolls, while inside its own
// local awake-hours window, to attempt a friendly challenge - see
// aiFriendlyChallengeAttemptsPerDay for why it's set much higher than ProbAiLocalActivityPerTick.
const ProbAiFriendlyChallengePerTick = float32(aiFriendlyChallengeAttemptsPerDay) / float32(localActivityWindowTicksPerDay)

// RunAiTeamDailyActivity gives every AI-managed team (see isAiManagedTeam) an independent,
// randomly-timed chance - once its own country's local clock is inside common.LocalActivityStart/
// EndHour (see common.IsWithinLocalActivityWindow) - to run its full market management pass and
// to challenge another AI-managed team to a friendly, both entirely decoupled from match
// simulation (see RunAITeamManagement's doc comment: it's only ever called from here). Before
// this existed at all, a freshly created world's CPU teams sat completely idle in the transfer
// market (and never played each other) until their very first league/cup match was simulated;
// this guarantees market activity - and a friendly habit - from day one, whether or not any
// competitive match has been played yet, and keeps it landing at a plausible local hour for each
// team's own country rather than any UTC instant at random (or, before RunAITeamManagement moved
// here exclusively, in lockstep with every other team at the single global match-schedule hour -
// see common.GetMatchSchedule). A friendly opponent is deliberately only ever another AI-managed
// team, never a human's - a human still only ever gets challenged by another human (see
// ChallengeToFriendly), so this can't spam a manager's inbox with unsolicited CPU challenges.
//
// Teams that already have a friendly kicking off today (see common.GetTeamsWithFriendlyToday) are
// left out of the friendly round entirely - neither offered as an opponent nor allowed to
// challenge - so a team's weekly allowance of friendlies (see common.GetMaxFriendliesPerWeek)
// spreads out one per day across the week instead of a few lucky days using it all up while the
// rest go unplayed.
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

	playedToday, err := common.GetTeamsWithFriendlyToday(dbCtx)
	if err != nil {
		log.Errorf("error checking today's friendlies for AI daily activity: %s", err)
		playedToday = map[int64]bool{}
	}

	now := time.Now()
	// Narrowed once per tick, up front, to teams currently inside their own country's local
	// window - so a team on the awake side of this check is never matched into a friendly
	// against one that isn't (the opponent's own plausible local hour matters exactly as much as
	// the challenger's own) - then further excludes anyone already playing a friendly today.
	awakeForFriendly := excludeTeamIds(awakeTeamIds(now, eligible), playedToday)

	r := rand.New(rand.NewSource(now.UnixNano()))
	for _, team := range eligible {
		if !common.IsWithinLocalActivityWindow(now, team.Country) {
			continue
		}
		if r.Float32() < ProbAiLocalActivityPerTick {
			RunAITeamManagement(dbCtx, team.Id)
		}
		if !playedToday[team.Id] && r.Float32() < ProbAiFriendlyChallengePerTick {
			challengeAiTeamToFriendly(dbCtx, r, team.Id, awakeForFriendly)
		}
	}
}

// awakeTeamIds returns the ids of every team currently inside its own country's local activity
// window (see common.IsWithinLocalActivityWindow) at the instant now - the pool
// RunAiTeamDailyActivity draws a friendly opponent from, so a team never gets matched into a
// friendly against one whose own local clock reads the middle of the night.
func awakeTeamIds(now time.Time, teams []*common.Team) []int64 {
	ids := make([]int64, 0, len(teams))
	for _, team := range teams {
		if common.IsWithinLocalActivityWindow(now, team.Country) {
			ids = append(ids, team.Id)
		}
	}
	return ids
}

// excludeTeamIds returns ids minus every id present (and true) in excluded.
func excludeTeamIds(ids []int64, excluded map[int64]bool) []int64 {
	kept := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !excluded[id] {
			kept = append(kept, id)
		}
	}
	return kept
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
// candidateIds, which RunAiTeamDailyActivity has already narrowed down to teams currently inside
// their own country's local activity window, with no friendly kicking off today) to a friendly.
// Both sides being unowned means ChallengeToFriendly auto-accepts on the spot (see its own doc
// comment) - there's nobody who'd otherwise need to act on it.
// ErrFriendlyLimitReached (either side already at its weekly cap - the default is just 1) and
// ErrFriendlyChallengeAlreadyExists (a pending challenge between this exact pair still
// outstanding, which shouldn't happen between two auto-accepting sides but costs nothing to guard
// against) are expected outcomes, not failures.
func challengeAiTeamToFriendly(dbCtx *common.DatabaseContext, r *rand.Rand, teamId int64, candidateIds []int64) {
	opponent, ok := pickRandomOpponent(r, candidateIds, teamId)
	if !ok {
		return
	}
	if err := common.ChallengeToFriendly(dbCtx, teamId, opponent, nil, true); err != nil &&
		!errors.Is(err, common.ErrFriendlyLimitReached) && !errors.Is(err, common.ErrFriendlyChallengeAlreadyExists) {
		log.Errorf("error challenging team %d to a friendly for AI-managed team %d: %s", opponent, teamId, err)
	}
}
