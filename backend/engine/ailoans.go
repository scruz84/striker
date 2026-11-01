package engine

import (
	"errors"
	"math"
	"sort"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// What a CPU team's loan-out pass offers, and on what terms - see manageLoanListings.
const (
	// aiLoanDurationSeasons is how long every loan a CPU team offers lasts.
	aiLoanDurationSeasons = 1
	// aiSurplusLoanAfter is how long a surplus player stays on the transfer list, unsold, before
	// his team offers him on loan instead.
	aiSurplusLoanAfter = 7 * 24 * time.Hour
	// aiSurplusLoanFeeShare is the loan fee of a surplus loan, as a share of the player's market
	// value; the lender then covers none of the wage, and the buy option is the old asking price.
	aiSurplusLoanFeeShare = 0.10
	// aiDevelopmentLoanMaxAge is the oldest a benched player can be to be loaned out to play.
	aiDevelopmentLoanMaxAge = 21
	// aiDevelopmentLoanWageContribution is the share of a developing player's wage his club keeps
	// paying while he is away: no fee, so that borrowing him is attractive.
	aiDevelopmentLoanWageContribution = 50
)

type loanOutKind int

const (
	loanOutSurplus loanOutKind = iota
	loanOutDevelopment
)

// loanOutPlan is a decision to offer one player on loan, with the terms to list him at.
type loanOutPlan struct {
	Player *common.Player
	Kind   loanOutKind
	// Fee/WageContributionPercent/BuyOptionFee are the terms of the loan listing.
	Fee                     int64
	WageContributionPercent int
	BuyOptionFee            int64
	// RelistAt is the asking price of the transfer listing a surplus loan replaces (0 for a
	// development loan, which comes from the squad) - kept so a failed loan listing can put the
	// player back on the transfer list at the price he had.
	RelistAt int64
}

// loanOutRoom reports whether a team may start another loan listing: at most
// common.AiMaxOpenLoanListings open, and the players out on loan plus the ones about to be (open
// listings) at most common.AiMaxLoanedOut, so that the cap holds even when every listing is taken.
func loanOutRoom(loanListed, loanedOut int) bool {
	return loanListed < common.AiMaxOpenLoanListings && loanedOut+loanListed < common.AiMaxLoanedOut
}

// loanOutEligible reports whether a team can lose p for a loan without hurting itself: he isn't one
// of the excluded (recently signed, international), his contract outlives the loan by a season so
// the parent club still has him afterwards, and without him the unlisted squad keeps at least
// minSquadSizeAfterSale players and his position its minimum depth. available is the squad the
// team counts on (owned, not on the transfer list, not on the loan market); p may or may not be in
// it - a surplus player already listed for transfer isn't.
func loanOutEligible(p *common.Player, available []*common.Player, excluded map[int64]bool) bool {
	if excluded[p.Id] || p.ContractYearsRemaining < aiLoanDurationSeasons+1 {
		return false
	}
	after := withoutPlayers(available, map[int64]bool{p.Id: true})
	position := common.BestCorePosition(*p)
	return len(after) >= minSquadSizeAfterSale && positionDepth(after, position) >= common.MinPositionDepth[position]
}

// startersOf is the ids of the best XI of squad.
func startersOf(squad []*common.Player) map[int64]bool {
	starters := map[int64]bool{}
	for _, ids := range common.SelectBestLineup(squad) {
		for _, id := range ids {
			starters[id] = true
		}
	}
	return starters
}

// loanOutInputs is what pickPlayerToLoanOut decides on. Owned is the team's owned senior squad
// (players on loan to it excluded); TransferListings its open transfer listings; LoanListed and
// LoanedOut the players it already has on the loan market / out on loan; Excluded who it never
// lends (recently signed, internationals).
type loanOutInputs struct {
	Owned            []*common.Player
	TransferListings map[int64]common.TeamTransferListing
	LoanListed       map[int64]bool
	LoanedOut        map[int64]bool
	Excluded         map[int64]bool
	// DevelopmentLoans lets the team loan out benched youngsters too (a Director).
	DevelopmentLoans bool
	Now              time.Time
}

func (in loanOutInputs) available() []*common.Player {
	listed := make(map[int64]bool, len(in.TransferListings))
	for id := range in.TransferListings {
		listed[id] = true
	}
	return withoutPlayers(withoutPlayers(in.Owned, listed), in.LoanListed)
}

// pickPlayerToLoanOut chooses the one player a team should offer on loan next, or nil. Limits
// first (loanOutRoom); then a surplus player left unsold on the transfer list for aiSurplusLoanAfter
// (the longest-listed first) is turned into a loan - 1 season, a fee of aiSurplusLoanFeeShare of his
// market value, the borrower pays the whole wage, and a buy option at the price he was listed at;
// failing that, a team that loans out youth offers its youngest benched player aged at most
// aiDevelopmentLoanMaxAge on a free loan with half his wage covered and no option. Pure.
func pickPlayerToLoanOut(in loanOutInputs) *loanOutPlan {
	if !loanOutRoom(len(in.LoanListed), len(in.LoanedOut)) {
		return nil
	}
	available := in.available()
	byId := make(map[int64]*common.Player, len(in.Owned))
	for _, p := range in.Owned {
		byId[p.Id] = p
	}

	unsold := make([]common.TeamTransferListing, 0)
	for _, l := range in.TransferListings {
		if p, ok := byId[l.PlayerId]; ok && !in.LoanListed[p.Id] && !in.LoanedOut[p.Id] && in.Now.Sub(l.ListedAt) >= aiSurplusLoanAfter {
			unsold = append(unsold, l)
		}
	}
	sort.SliceStable(unsold, func(i, j int) bool {
		if !unsold[i].ListedAt.Equal(unsold[j].ListedAt) {
			return unsold[i].ListedAt.Before(unsold[j].ListedAt)
		}
		return unsold[i].PlayerId < unsold[j].PlayerId
	})
	for _, l := range unsold {
		p := byId[l.PlayerId]
		if loanOutEligible(p, available, in.Excluded) {
			return &loanOutPlan{
				Player: p, Kind: loanOutSurplus, RelistAt: l.AskingPrice, BuyOptionFee: l.AskingPrice,
				Fee: int64(math.Round(float64(common.MarketValue(*p)) * aiSurplusLoanFeeShare)),
			}
		}
	}

	if !in.DevelopmentLoans {
		return nil
	}
	starters := startersOf(available)
	var best *common.Player
	for _, p := range available {
		if p.Age > aiDevelopmentLoanMaxAge || starters[p.Id] || !loanOutEligible(p, available, in.Excluded) {
			continue
		}
		if best == nil || p.Age < best.Age || (p.Age == best.Age && p.Id < best.Id) {
			best = p
		}
	}
	if best == nil {
		return nil
	}
	return &loanOutPlan{Player: best, Kind: loanOutDevelopment, WageContributionPercent: aiDevelopmentLoanWageContribution}
}

// planLoanListingWithdrawals picks which of a team's loan listings to take back: any listed player
// whose position would be under its minimum depth without him, any while the unlisted squad is
// under minSquadSizeAfterSale, and any who has become a starter of the best XI he would be part
// of. Each withdrawn player rejoins the squad before the next listing is judged, so only as many
// come back as are needed.
func planLoanListingWithdrawals(owned []*common.Player, transferListed, loanListed map[int64]bool) []*common.Player {
	available := withoutPlayers(withoutPlayers(owned, transferListed), loanListed)
	withdrawals := make([]*common.Player, 0)
	for _, p := range owned {
		if !loanListed[p.Id] {
			continue
		}
		position := common.BestCorePosition(*p)
		returning := append(append([]*common.Player(nil), available...), p)
		if positionDepth(available, position) < common.MinPositionDepth[position] ||
			len(available) < minSquadSizeAfterSale || startersOf(returning)[p.Id] {
			withdrawals = append(withdrawals, p)
			available = returning
		}
	}
	return withdrawals
}

// manageLoanListings keeps a Manager's or Director's loan listings in line with its squad: first
// takes back any listing that stopped being right (see planLoanListingWithdrawals), then offers at
// most common.AiMaxNewLoanListingsPerTick more player on loan (see pickPlayerToLoanOut). A surplus
// loan replaces the player's transfer listing; if the loan can't be listed the transfer listing is
// put back at its old price. Every step goes through the same functions a human uses, so a closed
// transfer window stops it - which is an expected outcome, not a failure.
func manageLoanListings(dbCtx *common.DatabaseContext, team *common.Team, players []*common.Player) {
	profile := common.AiTransferProfileFor(team.AiLevel)
	if !profile.LendsPlayers {
		return
	}
	commitments, err := common.GetTeamMarketCommitments(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error fetching market commitments for AI-managed team %d: %s", team.Id, err)
		return
	}
	owned := ownedPlayers(players, commitments.LoanedIn)

	for _, p := range planLoanListingWithdrawals(owned, commitments.TransferListed, commitments.LoanListed) {
		if err := common.CancelLoanListing(dbCtx, p.Id, team.Id); err != nil {
			if !errors.Is(err, common.ErrPlayerNotLoanListedByTeam) {
				log.Errorf("error withdrawing loan listing of player %d for AI-managed team %d: %s", p.Id, team.Id, err)
			}
			continue
		}
		delete(commitments.LoanListed, p.Id)
		recordAiMarketAction(team, "loan_listing_withdrawn", p.Id, 0)
	}

	if !common.IsTransferWindowOpen(dbCtx) || !loanOutRoom(len(commitments.LoanListed), len(commitments.LoanedOut)) {
		return
	}
	listings, err := common.GetTeamTransferListings(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error reading the transfer listings of AI-managed team %d: %s", team.Id, err)
		return
	}
	plan := pickPlayerToLoanOut(loanOutInputs{
		Owned: owned, TransferListings: listings, LoanListed: commitments.LoanListed, LoanedOut: commitments.LoanedOut,
		Excluded: recentlySignedOrInternational(dbCtx, team, owned), DevelopmentLoans: profile.LoansOutYouth, Now: time.Now(),
	})
	if plan == nil {
		return
	}

	if plan.Kind == loanOutSurplus {
		if err := common.CancelTransferListing(dbCtx, plan.Player.Id, team.Id); err != nil {
			if !errors.Is(err, common.ErrPlayerNotListedByTeam) {
				log.Errorf("error taking player %d off the transfer list for AI-managed team %d: %s", plan.Player.Id, team.Id, err)
			}
			return
		}
	}
	err = common.CreateLoanListing(dbCtx, plan.Player.Id, team.Id, aiLoanDurationSeasons, plan.Fee, plan.WageContributionPercent, plan.BuyOptionFee)
	if err != nil {
		if !errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrPlayerOnLoan) &&
			!errors.Is(err, common.ErrPlayerAlreadyListed) && !errors.Is(err, common.ErrPlayerAlreadyLoanListed) {
			log.Errorf("error listing player %d for loan for AI-managed team %d: %s", plan.Player.Id, team.Id, err)
		}
		if plan.Kind == loanOutSurplus {
			if err := common.CreateTransferListing(dbCtx, plan.Player.Id, team.Id, plan.RelistAt); err != nil &&
				!errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrPlayerAlreadyListed) {
				log.Errorf("error putting player %d back on the transfer list for AI-managed team %d: %s", plan.Player.Id, team.Id, err)
			}
		}
		return
	}
	recordAiMarketAction(team, "listed_for_loan", plan.Player.Id, plan.Fee)
}
