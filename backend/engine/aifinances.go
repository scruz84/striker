package engine

import (
	"errors"
	"math"
	"sort"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// The CPU's financial discipline (see openspec/specs/cpu-team-management "CPU financial crisis",
// "Contract renewals" and "Releasing unsold veterans").
const (
	// aiCrisisBudget: a club whose budget is below it is in financial crisis.
	aiCrisisBudget = 0
	// aiCrisisMaxOpenListings: a club in crisis lists nobody else while it has this many players
	// on the transfer list already.
	aiCrisisMaxOpenListings = 2
	// aiCrisisPriceShare is the share of the usual asking price a crisis sale is listed at.
	aiCrisisPriceShare = 0.8
	// aiVeteranAge, aiVeteranMinContractYears and aiVeteranUnsoldFor pick who a club releases: a
	// veteran with this many seasons or more left who has gone this long without a taker.
	aiVeteranAge              = 30
	aiVeteranMinContractYears = 2
	aiVeteranUnsoldFor        = 14 * 24 * time.Hour
	// aiReleaseBudgetCushion: a club only releases when its budget covers this many severances.
	aiReleaseBudgetCushion = 3
	// aiMaxRenewalRaise caps, as a multiple of the skill-based wage, the terms a CPU club meets
	// on its second renewal offer to a player who is not a starter.
	aiMaxRenewalRaise = 1.25
)

// inFinancialCrisis reports whether a club with this budget is in financial crisis.
func inFinancialCrisis(budget int64) bool {
	return budget < aiCrisisBudget
}

// teamInCrisis reads a team's budget and reports whether it is in financial crisis; a budget that
// can't be read counts as no crisis (the human rules still guard every spend).
func teamInCrisis(dbCtx *common.DatabaseContext, teamId int64) bool {
	budget, err := common.GetTeamBudget(dbCtx, teamId)
	if err != nil {
		log.Errorf("error reading the budget of AI-managed team %d: %s", teamId, err)
		return false
	}
	return inFinancialCrisis(budget)
}

// pickCrisisSale is the player a club in crisis lists to raise money, or nil: its highest-wage
// owned player (ties to the lowest id) who is not transfer- or loan-listed, not protected (a recent
// signing or an international), not a key player, and whose sale keeps his position at minimum
// depth and the squad at MinSquadSizeAfterSale owned, unlisted players. Pure.
func pickCrisisSale(owned []*common.Player, transferListed map[int64]bool, loanListed map[int64]bool, protected map[int64]bool) *common.Player {
	available := withoutPlayers(owned, transferListed)
	if len(available)-1 < minSquadSizeAfterSale {
		return nil
	}
	candidates := make([]*common.Player, 0, len(available))
	for _, p := range available {
		if loanListed[p.Id] || protected[p.Id] || common.IsKeyPlayer(owned, p.Id) {
			continue
		}
		position := common.BestCorePosition(*p)
		if positionDepth(available, position)-1 < common.MinPositionDepth[position] {
			continue
		}
		candidates = append(candidates, p)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Wage != candidates[j].Wage {
			return candidates[i].Wage > candidates[j].Wage
		}
		return candidates[i].Id < candidates[j].Id
	})
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0]
}

// listForFundsIfInCrisis has a Manager or Director in financial crisis list one player for sale at
// aiCrisisPriceShare of his asking price (see pickCrisisSale), through the same
// common.CreateTransferListing a human uses, while it has fewer than aiCrisisMaxOpenListings on
// the transfer list. A closed window is an expected outcome.
func listForFundsIfInCrisis(dbCtx *common.DatabaseContext, team *common.Team, players []*common.Player, crisis bool) {
	if !crisis || team.AiLevel < common.AiLevelManager {
		return
	}
	commitments, err := common.GetTeamMarketCommitments(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error fetching market commitments for AI-managed team %d: %s", team.Id, err)
		return
	}
	if len(commitments.TransferListed) >= aiCrisisMaxOpenListings {
		return
	}
	owned := ownedPlayers(players, commitments.LoanedIn)
	target := pickCrisisSale(owned, commitments.TransferListed, commitments.LoanListed, recentlySignedOrInternational(dbCtx, team, owned))
	if target == nil {
		return
	}
	international, caps := nationalStanding(dbCtx, target.Id)
	asking := int64(math.Round(float64(common.ListingAskingPrice(*target, common.AiTransferProfileFor(team.AiLevel), international, caps)) * aiCrisisPriceShare))
	err = common.CreateTransferListing(dbCtx, target.Id, team.Id, asking)
	if err == nil {
		recordAiMarketAction(team, "listed_in_crisis", target.Id, asking)
		return
	}
	if !errors.Is(err, common.ErrPlayerAlreadyListed) && !errors.Is(err, common.ErrTransferWindowClosed) &&
		!errors.Is(err, common.ErrPlayerOnLoan) && !errors.Is(err, common.ErrPlayerAlreadyLoanListed) {
		log.Errorf("error listing player %d of AI-managed team %d in crisis: %s", target.Id, team.Id, err)
	}
}

// pickVeteranToRelease is the player a club releases, or nil: among owned players aged
// aiVeteranAge or more with aiVeteranMinContractYears or more left who have been transfer- or
// loan-listed (listedSince) for aiVeteranUnsoldFor, the highest-wage one (ties to the lowest id)
// whose severance the budget covers aiReleaseBudgetCushion times - and only while the club owns
// more than common.SquadSize players. Pure.
func pickVeteranToRelease(owned []*common.Player, listedSince map[int64]time.Time, now time.Time, budget int64) *common.Player {
	if len(owned) <= common.SquadSize {
		return nil
	}
	var pick *common.Player
	for _, p := range owned {
		since, listed := listedSince[p.Id]
		if !listed || now.Sub(since) < aiVeteranUnsoldFor || p.Age < aiVeteranAge || p.ContractYearsRemaining < aiVeteranMinContractYears {
			continue
		}
		if budget < int64(p.Wage*common.EarlyReleaseWageMultiplier)*aiReleaseBudgetCushion {
			continue
		}
		if pick == nil || p.Wage > pick.Wage || (p.Wage == pick.Wage && p.Id < pick.Id) {
			pick = p
		}
	}
	return pick
}

// releaseUnsoldVeteranIfManager has a Manager or Director that is not in crisis release one
// unsold veteran (see pickVeteranToRelease): his listing is withdrawn, then common.ReleasePlayer
// pays the severance under the same rules a human's release goes through.
func releaseUnsoldVeteranIfManager(dbCtx *common.DatabaseContext, team *common.Team, players []*common.Player, budget int64) {
	if team.AiLevel < common.AiLevelManager || inFinancialCrisis(budget) {
		return
	}
	commitments, err := common.GetTeamMarketCommitments(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error fetching market commitments for AI-managed team %d: %s", team.Id, err)
		return
	}
	listedSince, err := common.GetTeamLoanListingTimes(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error reading the loan listings of AI-managed team %d: %s", team.Id, err)
		return
	}
	transfers, err := common.GetTeamTransferListings(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error reading the transfer listings of AI-managed team %d: %s", team.Id, err)
		return
	}
	for id, l := range transfers {
		listedSince[id] = l.ListedAt
	}
	veteran := pickVeteranToRelease(ownedPlayers(players, commitments.LoanedIn), listedSince, time.Now(), budget)
	if veteran == nil {
		return
	}
	if commitments.TransferListed[veteran.Id] {
		if err := common.CancelTransferListing(dbCtx, veteran.Id, team.Id); err != nil && !errors.Is(err, common.ErrPlayerNotListedByTeam) {
			log.Errorf("error withdrawing the listing of veteran %d of AI-managed team %d: %s", veteran.Id, team.Id, err)
			return
		}
	}
	if commitments.LoanListed[veteran.Id] {
		if err := common.CancelLoanListing(dbCtx, veteran.Id, team.Id); err != nil && !errors.Is(err, common.ErrPlayerNotLoanListedByTeam) {
			log.Errorf("error withdrawing the loan listing of veteran %d of AI-managed team %d: %s", veteran.Id, team.Id, err)
			return
		}
	}
	if err := common.ReleasePlayer(dbCtx, veteran.Id, team.Id); err != nil {
		if !errors.Is(err, common.ErrInsufficientBudget) && !errors.Is(err, common.ErrPlayerOnLoan) {
			log.Errorf("error releasing veteran %d of AI-managed team %d: %s", veteran.Id, team.Id, err)
		}
		return
	}
	recordAiMarketAction(team, "released_veteran", veteran.Id, int64(veteran.Wage*common.EarlyReleaseWageMultiplier))
}
