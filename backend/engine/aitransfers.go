package engine

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// What a CPU team's transfer pass searches, and how it bids. The ceilings on how many offers a team
// or a player can have open live in common (AiMaxOpenOffers*), next to the cooldown rules.
const (
	// aiPositionCandidates is how many players per core position a market-wide search considers.
	aiPositionCandidates = 8
	// aiScoutingPoolSize is how many random youngsters a Director scouts per pass.
	aiScoutingPoolSize = 20
	// aiRadarCandidates is how many of the national teams' best a Director's radar considers.
	aiRadarCandidates = 20
	// aiFreeAgentCandidates is how many of the (best-first) free agents a pass considers.
	aiFreeAgentCandidates = 60
	// aiMaxPlayerAge keeps market searches off players too old to be worth a fee.
	aiMaxPlayerAge = 36
	// aiBaseBidMultiple is a first bid for an unlisted player, relative to the team's valuation.
	aiBaseBidMultiple = 1.1
	// aiBidBumpPerScorePoint/aiMaxBidBump raise that bid for a target the team wants more than the
	// minimum score asks for - interest buys a margin.
	aiBidBumpPerScorePoint = 0.02
	aiMaxBidBump           = 0.25
	// aiMaxBidValueMultiple is the most a team will pay for a permanent signing, relative to its
	// own valuation: a bid (or a rebid after a rejection) never goes above it, and neither does
	// the asking price of a listing or a loan's buy-option fee - a team that won't offer more than
	// this for a player nobody listed doesn't pay three times his worth because somebody did.
	aiMaxBidValueMultiple = 1.5
	// aiPassIterationsPerMove bounds how many candidates a pass tries per move it may make, so a
	// run of failures (a listing someone else just bought) can't loop forever.
	aiPassIterationsPerMove = 3
)

type candidateKind int

const (
	candidateFreeAgent candidateKind = iota
	candidateListed
	candidateUnlisted
	// candidateLoan is a listing on the loan market; candidateBuyOption a player already on loan to
	// the team whose loan carries a buy option, scored as a permanent signing at the option fee.
	candidateLoan
	candidateBuyOption
	// candidatePreContract is another club's player in his last season, offered a contract for
	// next season (openspec/specs/cpu-transfer-market "Pre-contract targets"): no fee, his Wage is
	// the one the team would offer.
	candidatePreContract
	// candidateReleaseClause is another club's player whose release clause the team can pay
	// (openspec/specs/cpu-transfer-market "Paying release clauses"): AskingPrice is the clause.
	candidateReleaseClause
)

// transferCandidate is one player a CPU team could sign, and how.
type transferCandidate struct {
	Player       common.Player
	Kind         candidateKind
	ListingId    int64
	SellerTeamId int64
	// AskingPrice is the listing's price, or a free agent's premium to his last club; unlisted
	// players have none (the team bids). MinBid is the least a rebid may be after a rejection.
	AskingPrice int64
	MinBid      int64
	// LoanSeasons, LoanWageContributionPercent and BuyOptionFee are the terms of a loan listing
	// (candidateLoan; AskingPrice is then its loan fee) or the buy option of a loan the team is
	// already in (candidateBuyOption; BuyOptionFee is then the price).
	LoanSeasons                 int
	LoanWageContributionPercent int
	BuyOptionFee                int64
	Caps                        *common.PlayerSkillCaps
	// International/NationalCaps say whether he is in a national-team pool (see
	// common.NationalRadar) and with how many caps.
	International bool
	NationalCaps  int
}

// transferState is the team-side picture a pass decides on, re-read after every signing.
type transferState struct {
	// Squad is the owned senior squad (loaned-in players excluded) and still includes whoever is
	// on the transfer list; Listed/LoanListed say which of them are.
	Squad      []*common.Player
	Listed     map[int64]bool
	LoanListed map[int64]bool
	// LoanedIn are the players on loan to the team and Loans the terms of those loans;
	// CurrentSeason is the team's country's current season (a loan returns when it reaches the
	// loan's ReturnSeason).
	LoanedIn      []*common.Player
	Loans         []common.PlayerLoan
	CurrentSeason int
	Budget        int64
	WageBill      int64
	// WageFactor is the team's division wage factor - WageBill already includes it.
	WageFactor float64
	// OpenOffers is how many offers the team currently has out.
	OpenOffers int
	// Protected are squad players the team won't sell to fund a signing - see
	// protectedFromListing.
	Protected map[int64]bool
	// SignedPreContracts is how many players have signed a pre-contract with the team - they join
	// at the next rollover, so they count toward the squad cap.
	SignedPreContracts int
}

// available is the squad the team can count on: owned and not already up for sale, on the
// transfer list or on the loan market.
func (s *transferState) available() []*common.Player {
	return withoutPlayers(withoutPlayers(s.Squad, s.Listed), s.LoanListed)
}

// playingSquad is available plus the players on loan to the team: everyone who will play for it
// this season, which is what a loan or a buy option is scored against. A transfer is scored
// against available alone, since a loanee goes back when his loan ends.
func (s *transferState) playingSquad() []*common.Player {
	return append(s.available(), s.LoanedIn...)
}

// playingSquadWithout is playingSquad minus one player - the squad a buy option's player would
// leave behind.
func (s *transferState) playingSquadWithout(playerId int64) []*common.Player {
	return withoutPlayers(s.playingSquad(), map[int64]bool{playerId: true})
}

// projectsInUse approximates how many places the team has given to young players it has not yet
// made starters of: under-24s outside the best XI.
func (s *transferState) projectsInUse() int {
	available := s.available()
	starters := map[int64]bool{}
	for _, ids := range common.SelectBestLineup(available) {
		for _, id := range ids {
			starters[id] = true
		}
	}
	projects := 0
	for _, p := range available {
		if p.Age < common.ProjectionMaxAge && !starters[p.Id] {
			projects++
		}
	}
	return projects
}

// scoredCandidate is a candidate together with the team's verdict on him.
type scoredCandidate struct {
	transferCandidate
	Score common.TargetScore
}

// scoreOne scores one candidate for the team as it stands. A bid for an unlisted player starts at
// aiBaseBidMultiple of the team's valuation; if he already rejected an offer it must at least beat
// the rebid floor, and a floor above aiMaxBidValueMultiple of the valuation rules the player out.
// The same ceiling rules out a listing, or a buy option, priced above aiMaxBidValueMultiple of the
// valuation: the value-for-money term alone is capped, so without it a big enough upgrade would
// be bought at any price.
func scoreOne(profile common.AiTransferProfile, state *transferState, c transferCandidate, rng *rand.Rand) (common.TargetScore, bool) {
	squad := state.available()
	switch c.Kind {
	case candidateLoan:
		squad = state.playingSquad()
	case candidateBuyOption:
		squad = state.playingSquadWithout(c.Player.Id)
	}
	ctx := common.TargetContext{
		Squad:         squad,
		Target:        c.Player,
		Caps:          c.Caps,
		Price:         c.AskingPrice,
		Budget:        state.Budget,
		WageBill:      state.WageBill,
		WageFactor:    state.WageFactor,
		ProjectsInUse: state.projectsInUse(),

		International:     c.International,
		InternationalCaps: c.NationalCaps,
	}
	switch c.Kind {
	case candidateUnlisted:
		ctx.BidMultiple = aiBaseBidMultiple
	case candidateLoan:
		ctx.Loan, ctx.LoanSeasons = true, c.LoanSeasons
		ctx.LoanWageContributionPercent, ctx.BuyOptionFee = c.LoanWageContributionPercent, c.BuyOptionFee
	case candidateBuyOption:
		ctx.Price = c.BuyOptionFee
	}
	score := common.ScoreTarget(profile, ctx, rng)
	if (c.Kind == candidateListed || c.Kind == candidateBuyOption || c.Kind == candidateReleaseClause) &&
		float64(score.Price) > float64(score.Value)*aiMaxBidValueMultiple {
		return score, false
	}
	if c.Kind == candidateUnlisted && c.MinBid > score.Price {
		if float64(c.MinBid) > float64(score.Value)*aiMaxBidValueMultiple {
			return score, false
		}
		ctx.BidMultiple = 0
		ctx.Price = c.MinBid
		score = common.ScoreTarget(profile, ctx, rng)
	}
	return score, true
}

// fillsNeed reports whether signing p plugs a hole in the team: its squad (owned and available,
// plus players on loan to it) is under common.SquadSize, or p's position is under
// common.MinPositionDepth - the only signings a profile that BuysOnlyForNeeds makes.
func fillsNeed(state *transferState, p common.Player) bool {
	available := state.available()
	if len(available)+len(state.LoanedIn) < common.SquadSize {
		return true
	}
	position := common.BestCorePosition(p)
	return positionDepth(available, position) < common.MinPositionDepth[position]
}

// rankCandidates scores every candidate not yet attempted and keeps the ones that clear the
// profile's bar - affordable, a score of at least MinScore and, for a profile that
// BuysOnlyForNeeds, filling a need - best first. Pure given the rng,
// so the choice of targets can be tested without a database.
func rankCandidates(profile common.AiTransferProfile, state *transferState, candidates []transferCandidate,
	attempted map[int64]bool, rng *rand.Rand) []scoredCandidate {
	ranked := make([]scoredCandidate, 0)
	for _, c := range candidates {
		if attempted[c.Player.Id] {
			continue
		}
		if profile.BuysOnlyForNeeds && !fillsNeed(state, c.Player) {
			continue
		}
		score, ok := scoreOne(profile, state, c, rng)
		if !ok || !score.Affordable || score.Total < profile.MinScore {
			continue
		}
		ranked = append(ranked, scoredCandidate{c, score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score.Total != ranked[j].Score.Total {
			return ranked[i].Score.Total > ranked[j].Score.Total
		}
		return ranked[i].Player.Id < ranked[j].Player.Id
	})
	return ranked
}

// pickNextTarget chooses the best-ranked candidate the team can act on right now: a team whose
// effective squad - owned players it can count on plus the ones on loan to it - is at
// common.MaxAiSquadSize takes on nobody (it has to sell first; exercising a buy option adds no
// body, so that still goes ahead), a team with common.AiMaxLoanedIn players on loan accepts no
// more loans, and only offersLeft more offers for unlisted players may be started this pass.
func pickNextTarget(ranked []scoredCandidate, state *transferState, offersLeft int, preContractsLeft int) *scoredCandidate {
	return pickNextTargetWithClauses(ranked, state, offersLeft, preContractsLeft, 0)
}

// pickNextTargetWithClauses is pickNextTarget that may also pick a release clause to pay while
// clausesLeft is positive.
func pickNextTargetWithClauses(ranked []scoredCandidate, state *transferState, offersLeft int, preContractsLeft int, clausesLeft int) *scoredCandidate {
	atCap := len(state.available())+len(state.LoanedIn)+state.SignedPreContracts >= common.MaxAiSquadSize
	for i := range ranked {
		switch ranked[i].Kind {
		case candidatePreContract:
			if atCap || preContractsLeft <= 0 || state.SignedPreContracts >= common.AiMaxSignedPreContracts {
				continue
			}
		case candidateReleaseClause:
			if atCap || clausesLeft <= 0 {
				continue
			}
		case candidateBuyOption:
		case candidateLoan:
			if atCap || len(state.LoanedIn) >= common.AiMaxLoanedIn {
				continue
			}
		case candidateUnlisted:
			if atCap || offersLeft <= 0 || state.OpenOffers >= common.AiMaxOpenOffersPerTeam {
				continue
			}
		default:
			if atCap {
				continue
			}
		}
		return &ranked[i]
	}
	return nil
}

// bidAmount is what a team offers for an unlisted player: the scored bid, raised by how far the
// target's score clears the profile's bar, but never beyond aiMaxBidValueMultiple of the team's
// valuation nor the budget it may spend.
func bidAmount(profile common.AiTransferProfile, state *transferState, target scoredCandidate) int64 {
	bump := math.Min(aiMaxBidBump, aiBidBumpPerScorePoint*(target.Score.Total-profile.MinScore))
	if bump < 0 {
		bump = 0
	}
	amount := int64(math.Round(float64(target.Score.Price) * (1 + bump)))
	if ceiling := int64(float64(target.Score.Value) * aiMaxBidValueMultiple); amount > ceiling {
		amount = ceiling
	}
	if spendable := state.Budget - int64(float64(state.Budget)*profile.BudgetReserve); amount > spendable {
		amount = spendable
	}
	if amount < target.Score.Price {
		amount = target.Score.Price
	}
	return amount
}

// sellPlan is a Director's decision to put a player on the transfer list so that a target it
// can't pay for yet (or has no room for) becomes possible.
type sellPlan struct {
	Seller *common.Player
	Asking int64
	For    scoredCandidate
}

// planSellToBuy looks for the best target the team could take if it first sold one of its own:
// one that failed only for lack of money, or any target when the squad is full, replacing the
// weakest player at the same position. The sale is only planned when it leaves that position at
// its minimum depth (otherwise the listing would be withdrawn again at once) and the squad above
// the minimum size, when the team has no other listing out, and when - credited with what the
// seller asks - the target would clear the profile's bar. Only for profiles that SellsToBuy.
func planSellToBuy(profile common.AiTransferProfile, state *transferState, candidates []transferCandidate,
	attempted map[int64]bool, rng *rand.Rand) *sellPlan {
	if !profile.SellsToBuy || len(state.Listed) > 0 {
		return nil
	}
	available := state.available()
	squadFull := len(available)+len(state.LoanedIn) >= common.MaxAiSquadSize

	var best *sellPlan
	for _, c := range candidates {
		if attempted[c.Player.Id] || c.Kind == candidateFreeAgent || c.Kind == candidateLoan || c.Kind == candidateBuyOption {
			continue
		}
		base, ok := scoreOne(profile, state, c, rng)
		if !ok || (base.Affordable && !squadFull) || (!base.Affordable && !base.BudgetShort && !squadFull) {
			continue
		}
		position := common.BestCorePosition(c.Player)
		if common.PositionValue(position, c.Player) <= 0 || len(available)-1 < common.MinSquadSizeAfterSale ||
			positionDepth(available, position)-1 < common.MinPositionDepth[position] {
			continue
		}
		var seller *common.Player
		sellerValue := 0
		for _, p := range available {
			if common.BestCorePosition(*p) != position || state.LoanListed[p.Id] || state.Protected[p.Id] {
				continue
			}
			if v := common.PositionValue(position, *p); seller == nil || v < sellerValue {
				seller, sellerValue = p, v
			}
		}
		if seller == nil || sellerValue >= common.PositionValue(position, c.Player) {
			continue
		}
		asking := common.ListingAskingPrice(*seller, profile, false, 0)

		after := *state
		after.Squad = withoutPlayers(state.Squad, map[int64]bool{seller.Id: true})
		after.Budget += asking
		score, ok := scoreOne(profile, &after, c, rng)
		if !ok || !score.Affordable || score.Total < profile.MinScore {
			continue
		}
		if best == nil || score.Total > best.For.Score.Total {
			best = &sellPlan{Seller: seller, Asking: asking, For: scoredCandidate{c, score}}
		}
	}
	return best
}

// loadTransferState reads the team-side picture a pass decides on.
func loadTransferState(dbCtx *common.DatabaseContext, team *common.Team) (*transferState, error) {
	teamId := team.Id
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	commitments, err := common.GetTeamMarketCommitments(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	budget, err := common.GetTeamBudget(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	wageBill, err := common.GetTeamWageBill(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	wageFactor, err := common.GetTeamWageFactor(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	sent, err := common.GetPlayersWithOfferFromTeam(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	loans, err := common.GetLoansToTeam(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	currentSeason, err := common.GetCurrentSeasonForTeam(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	squad := ownedPlayers(*players, commitments.LoanedIn)
	loanedIn := make([]*common.Player, 0, len(commitments.LoanedIn))
	for _, p := range *players {
		if commitments.LoanedIn[p.Id] {
			loanedIn = append(loanedIn, p)
		}
	}
	signedPreContracts, err := common.CountSignedPreContracts(dbCtx.Db, teamId)
	if err != nil {
		return nil, err
	}
	return &transferState{
		SignedPreContracts: signedPreContracts,
		Squad:              squad,
		Listed:             commitments.TransferListed,
		LoanListed:         commitments.LoanListed,
		LoanedIn:           loanedIn,
		Loans:              loans,
		CurrentSeason:      currentSeason,
		Budget:             budget,
		WageBill:           wageBill,
		WageFactor:         wageFactor,
		OpenOffers:         len(sent),
		Protected:          protectedFromListing(dbCtx, team, squad),
	}, nil
}

// weakestStarterValue is the lowest rating, at position, among the best XI's players whose natural
// position it is - the bar a market search asks a candidate to clear. 0 when the XI has nobody
// there, which turns the search into "the best available".
func weakestStarterValue(squad []*common.Player, position int) int {
	byId := make(map[int64]*common.Player, len(squad))
	for _, p := range squad {
		byId[p.Id] = p
	}
	lowest := -1
	for _, ids := range common.SelectBestLineup(squad) {
		for _, id := range ids {
			p, ok := byId[id]
			if !ok || common.BestCorePosition(*p) != position {
				continue
			}
			if v := common.PositionValue(position, *p); lowest == -1 || v < lowest {
				lowest = v
			}
		}
	}
	if lowest < 0 {
		return 0
	}
	return lowest
}

// freeAgentPremium is what signing a free agent costs his previous club, if it is due.
func freeAgentPremium(p common.Player) int64 {
	if p.FreeAgentSeasons != 0 || p.LastTeamId == 0 {
		return 0
	}
	wage := common.PlayerWageFromSkills(p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot)
	return int64(wage) * common.FreeAgentPremiumWageMultiplier
}

// loadPlayers reads a batch of players by id. One unreadable row (a player whose record is
// incomplete) would sink the whole batch, so when the batch fails the players are read one by one
// and the bad ones skipped - a market search shouldn't be lost to one corrupt record.
func loadPlayers(dbCtx *common.DatabaseContext, ids []int64) []*common.Player {
	if players, err := common.GetPlayersByIds(dbCtx, ids); err == nil {
		return *players
	}
	players := make([]*common.Player, 0, len(ids))
	for _, id := range ids {
		p, err := common.GetPlayerById(dbCtx, id)
		if err != nil {
			log.Errorf("skipping unreadable market candidate %d: %s", id, err)
			continue
		}
		players = append(players, p)
	}
	return players
}

// radarCandidateIds is a Director's national-team radar: the best players of every country's
// national team, wherever they play and whether or not anyone listed them, narrowed to those who
// beat the team's weakest starter at their own position (the others can't help it) and capped at
// aiRadarCandidates, best first.
func radarCandidateIds(radar *common.NationalRadar, available []*common.Player, seen map[int64]bool) []int64 {
	type scouted struct {
		id      int64
		overall int
	}
	found := make([]scouted, 0)
	for _, e := range radar.Best(common.NationalRadarTopPerCountry) {
		if seen[e.Player.Id] || e.Player.Age > aiMaxPlayerAge {
			continue
		}
		position := common.BestCorePosition(e.Player)
		if common.PositionValue(position, e.Player) <= weakestStarterValue(available, position) {
			continue
		}
		found = append(found, scouted{e.Player.Id, common.AverageSkill(e.Player)})
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].overall != found[j].overall {
			return found[i].overall > found[j].overall
		}
		return found[i].id < found[j].id
	})
	if len(found) > aiRadarCandidates {
		found = found[:aiRadarCandidates]
	}
	ids := make([]int64, len(found))
	for i, f := range found {
		ids[i] = f.id
	}
	return ids
}

// loanListingCandidate turns a loan-market listing into a candidate: AskingPrice is its loan fee.
func loanListingCandidate(l common.LoanListing) transferCandidate {
	return transferCandidate{
		Player: l.Player, Kind: candidateLoan, ListingId: l.ListingId, SellerTeamId: l.LendingTeamId,
		AskingPrice: l.LoanFee, LoanSeasons: l.DurationSeasons,
		LoanWageContributionPercent: l.WageContributionPercent, BuyOptionFee: l.BuyOptionFee,
	}
}

// loanListingCandidates are the loan-market listings a team may take: not its own, nobody it
// already sees elsewhere, and nobody past aiMaxPlayerAge. Each player taken is marked in seen.
func loanListingCandidates(listings []common.LoanListing, teamId int64, seen map[int64]bool) []transferCandidate {
	candidates := make([]transferCandidate, 0)
	for _, l := range listings {
		if l.LendingTeamId == teamId || seen[l.Player.Id] || l.Player.Age > aiMaxPlayerAge {
			continue
		}
		seen[l.Player.Id] = true
		candidates = append(candidates, loanListingCandidate(l))
	}
	return candidates
}

// buyOptionCandidates are the players on loan to the team whose loan carries a buy option and
// ends at the coming rollover (the loan returns once the season reaches its ReturnSeason, so in
// its last season ReturnSeason - CurrentSeason is 1). Before that there is still time to see how
// the player does, and a player past aiMaxPlayerAge isn't worth a fee.
func buyOptionCandidates(state *transferState) []transferCandidate {
	loanedIn := make(map[int64]*common.Player, len(state.LoanedIn))
	for _, p := range state.LoanedIn {
		loanedIn[p.Id] = p
	}
	candidates := make([]transferCandidate, 0)
	for _, loan := range state.Loans {
		p, ok := loanedIn[loan.PlayerId]
		if !ok || loan.BuyOptionFee <= 0 || loan.ReturnSeason-state.CurrentSeason > 1 || p.Age > aiMaxPlayerAge {
			continue
		}
		candidates = append(candidates, transferCandidate{
			Player: *p, Kind: candidateBuyOption, SellerTeamId: loan.FromTeamId, BuyOptionFee: loan.BuyOptionFee,
		})
	}
	return candidates
}

// gatherCandidates collects everyone the profile lets the team look at: the best free agents, the
// transfer list, and - for a Director - the best players at each position anywhere in the world
// (human managers' squads included) plus a scouting sample of youngsters. Players the team has an
// offer out on, or that rejected it inside the cooldown, are left out.
func gatherCandidates(dbCtx *common.DatabaseContext, team *common.Team, profile common.AiTransferProfile,
	state *transferState, radar *common.NationalRadar, now time.Time) []transferCandidate {
	candidates := make([]transferCandidate, 0)
	seen := map[int64]bool{}
	for _, p := range state.Squad {
		seen[p.Id] = true
	}
	for _, p := range state.LoanedIn {
		seen[p.Id] = true
	}

	if profile.SearchesFreeAgents {
		if freeAgents, err := common.GetFreeAgents(dbCtx); err != nil {
			log.Errorf("error fetching free agents for AI-managed team %d: %s", team.Id, err)
		} else {
			if len(freeAgents) > aiFreeAgentCandidates {
				freeAgents = freeAgents[:aiFreeAgentCandidates]
			}
			for _, p := range freeAgents {
				if seen[p.Id] || p.Age > aiMaxPlayerAge {
					continue
				}
				seen[p.Id] = true
				candidates = append(candidates, transferCandidate{Player: *p, Kind: candidateFreeAgent, AskingPrice: freeAgentPremium(*p)})
			}
		}
	}

	if profile.SearchesTransferList {
		if listings, err := common.GetActiveTransferListings(dbCtx); err != nil {
			log.Errorf("error fetching transfer listings for AI-managed team %d: %s", team.Id, err)
		} else {
			for _, l := range listings {
				if l.SellingTeamId == team.Id || seen[l.Player.Id] || l.Player.Age > aiMaxPlayerAge {
					continue
				}
				seen[l.Player.Id] = true
				candidates = append(candidates, transferCandidate{
					Player: l.Player, Kind: candidateListed, ListingId: l.ListingId,
					SellerTeamId: l.SellingTeamId, AskingPrice: l.AskingPrice,
				})
			}
		}
	}

	if profile.SearchesLoanList {
		candidates = append(candidates, buyOptionCandidates(state)...)
		if listings, err := common.GetActiveLoanListings(dbCtx); err != nil {
			log.Errorf("error fetching loan listings for AI-managed team %d: %s", team.Id, err)
		} else {
			candidates = append(candidates, loanListingCandidates(listings, team.Id, seen)...)
		}
	}

	if profile.PreContractCandidates > 0 {
		candidates = append(candidates, preContractCandidates(dbCtx, team, profile, state, seen)...)
	}

	if profile.SearchesUnlisted {
		unlisted := make([]int64, 0)
		available := state.available()
		for _, position := range common.CorePositions {
			ids, err := common.GetMarketCandidatesForPosition(dbCtx, team.Id, position,
				weakestStarterValue(available, position), aiMaxPlayerAge, aiPositionCandidates)
			if err != nil {
				log.Errorf("error searching the market for AI-managed team %d: %s", team.Id, err)
				continue
			}
			unlisted = append(unlisted, ids...)
		}
		if profile.SignsProjects {
			if ids, err := common.GetYoungScoutingPool(dbCtx, team.Id, common.ProjectionMaxAge-1, aiScoutingPoolSize); err != nil {
				log.Errorf("error scouting youngsters for AI-managed team %d: %s", team.Id, err)
			} else {
				unlisted = append(unlisted, ids...)
			}
		}

		unlisted = append(unlisted, radarCandidateIds(radar, available, seen)...)

		fresh := make([]int64, 0, len(unlisted))
		for _, id := range unlisted {
			if !seen[id] && !state.Listed[id] {
				seen[id] = true
				fresh = append(fresh, id)
			}
		}
		rejections, err := common.GetRecentOfferRejections(dbCtx, team.Id, now.Add(-2*common.AiOfferCooldown))
		if err != nil {
			log.Errorf("error reading offer rejections of AI-managed team %d: %s", team.Id, err)
			rejections = map[int64]common.OfferRejection{}
		}
		sentOffers, err := common.GetPlayersWithOfferFromTeam(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error reading open offers of AI-managed team %d: %s", team.Id, err)
			sentOffers = map[int64]bool{}
		}
		clauses := payableClauses(dbCtx, fresh, now)
		for _, p := range loadPlayers(dbCtx, fresh) {
			if clause, ok := clauses[p.Id]; ok && cpuMayPayClauseOn(dbCtx, p.TeamId, now) {
				// His clause is another way to sign him: paid at once, no consent needed.
				candidates = append(candidates, transferCandidate{Player: *p, Kind: candidateReleaseClause, SellerTeamId: p.TeamId, AskingPrice: clause})
			}
			if sentOffers[p.Id] {
				continue
			}
			c := transferCandidate{Player: *p, Kind: candidateUnlisted, SellerTeamId: p.TeamId}
			if rejection, ok := rejections[p.Id]; ok {
				allowed, minBid := common.OfferRetryAllowed(&rejection, now)
				if !allowed {
					continue
				}
				c.MinBid = minBid
			}
			candidates = append(candidates, c)
		}
	}

	ids := make([]int64, len(candidates))
	for i, c := range candidates {
		ids[i] = c.Player.Id
		if entry, ok := radar.Lookup(c.Player.Id); ok {
			candidates[i].International, candidates[i].NationalCaps = true, entry.Caps
		}
	}
	if caps, err := common.GetPlayerSkillCaps(dbCtx, ids); err != nil {
		log.Errorf("error reading skill caps for AI-managed team %d: %s", team.Id, err)
	} else {
		for i := range candidates {
			if c, ok := caps[candidates[i].Player.Id]; ok {
				c := c
				candidates[i].Caps = &c
			}
		}
	}
	return candidates
}

func recordAiMarketAction(team *common.Team, action string, playerId int64, price int64) {
	common.AiMarketActionsCounter.Add(context.Background(), 1, metric.WithAttributes(attribute.String("action", action)))
	log.Infof("ai-market team=%d level=%d action=%s player=%d price=%d", team.Id, team.AiLevel, action, playerId, price)
}

// executeTransferMove carries out the signing the pass settled on and reports whether it went
// through; every failure here is an expected race (somebody else signed the player, the window
// closed, the budget moved) and just means the pass tries the next candidate.
func executeTransferMove(dbCtx *common.DatabaseContext, team *common.Team, profile common.AiTransferProfile,
	state *transferState, target scoredCandidate) bool {
	switch target.Kind {
	case candidateFreeAgent:
		result, err := common.SignFreeAgentAtMarketRate(dbCtx, target.Player.Id, team.Id)
		if err != nil || !result.Accepted {
			if err != nil && !errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrPlayerNotFree) &&
				!errors.Is(err, common.ErrInsufficientBudget) && !errors.Is(err, common.ErrTeamInsolvent) {
				log.Errorf("error signing free agent %d for AI-managed team %d: %s", target.Player.Id, team.Id, err)
			}
			return false
		}
		recordAiMarketAction(team, "signed_free_agent", target.Player.Id, result.Premium)
		return true

	case candidateListed:
		if err := common.ExecuteTransfer(dbCtx, target.ListingId, team.Id); err != nil {
			if !errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrInsufficientBudget) &&
				!errors.Is(err, common.ErrListingNotFound) && !errors.Is(err, common.ErrCannotBuyOwnPlayer) &&
				!errors.Is(err, common.ErrTeamInsolvent) {
				log.Errorf("error buying transfer-listed player for AI-managed team %d: %s", team.Id, err)
			}
			return false
		}
		recordAiMarketAction(team, "bought_listed", target.Player.Id, target.AskingPrice)
		return true

	case candidateLoan:
		if err := common.AcceptLoan(dbCtx, target.ListingId, team.Id); err != nil {
			if !errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrInsufficientBudget) &&
				!errors.Is(err, common.ErrLoanListingNotFound) && !errors.Is(err, common.ErrCannotAcceptOwnLoan) &&
				!errors.Is(err, common.ErrTeamInsolvent) {
				log.Errorf("error accepting loan listing %d for AI-managed team %d: %s", target.ListingId, team.Id, err)
			}
			return false
		}
		recordAiMarketAction(team, "loaned_in", target.Player.Id, target.AskingPrice)
		return true

	case candidatePreContract:
		return makeAiPreContractOffer(dbCtx, team, target)

	case candidateReleaseClause:
		return payAiReleaseClause(dbCtx, team, target)

	case candidateBuyOption:
		if err := common.ExerciseLoanBuyOption(dbCtx, target.Player.Id, team.Id); err != nil {
			if !errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrInsufficientBudget) &&
				!errors.Is(err, common.ErrPlayerNotOnLoan) && !errors.Is(err, common.ErrPlayerNotOnLoanToTeam) &&
				!errors.Is(err, common.ErrNoBuyOptionOnLoan) && !errors.Is(err, common.ErrTeamInsolvent) {
				log.Errorf("error exercising the buy option on player %d for AI-managed team %d: %s", target.Player.Id, team.Id, err)
			}
			return false
		}
		recordAiMarketAction(team, "exercised_buy_option", target.Player.Id, target.BuyOptionFee)
		return true

	default:
		return makeAiOffer(dbCtx, team, profile, state, target)
	}
}

// makeAiOffer opens an offer for an unlisted player, within the limits on open offers: no more
// than common.AiMaxOpenOffersPerHumanPlayer CPU offers at once on a human manager's player (who
// is told about each), and only one at a time on another CPU team's player, so CPU teams don't
// all pile onto the same target.
func makeAiOffer(dbCtx *common.DatabaseContext, team *common.Team, profile common.AiTransferProfile,
	state *transferState, target scoredCandidate) bool {
	owner, err := common.GetTeamOwner(dbCtx, target.SellerTeamId)
	if err != nil {
		log.Errorf("error looking up the owner of team %d: %s", target.SellerTeamId, err)
		return false
	}
	humanOwned := owner > 0
	open, err := common.CountOpenAiOffersOnPlayer(dbCtx, target.Player.Id)
	if err != nil {
		log.Errorf("error counting open CPU offers on player %d: %s", target.Player.Id, err)
		return false
	}
	if (humanOwned && open >= common.AiMaxOpenOffersPerHumanPlayer) || (!humanOwned && open >= 1) {
		return false
	}

	p := target.Player
	// Exactly the terms the player accepts (openspec/specs/cpu-transfer-market "CPU personal
	// terms"), so the offer is never refused on personal terms.
	wage, years, err := common.PlayerTermsExpectation(dbCtx, p, common.TermsTransfer)
	if err != nil {
		log.Errorf("error reading the personal terms of player %d for AI-managed team %d: %s", p.Id, team.Id, err)
		return false
	}
	amount := bidAmount(profile, state, target)
	result, err := common.MakeTransferOffer(dbCtx, p.Id, team.Id, amount, wage, years)
	if err != nil {
		if !errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrOfferAlreadyExists) &&
			!errors.Is(err, common.ErrPlayerOnLoan) && !errors.Is(err, common.ErrPlayerIsFreeAgent) &&
			!errors.Is(err, common.ErrTeamInsolvent) && !errors.Is(err, common.ErrCannotOfferOnOwnPlayer) {
			log.Errorf("error making offer for player %d on behalf of AI-managed team %d: %s", p.Id, team.Id, err)
		}
		return false
	}
	if !result.Accepted {
		return false
	}
	recordAiMarketAction(team, "offer_made", p.Id, amount)

	if humanOwned {
		playerName := p.FirstName + " " + p.LastName
		_ = common.NotifyTeamOwnerWithLink(dbCtx, target.SellerTeamId, common.NOTIFICATION_TRANSFER, common.NotificationMessage{
			Text:   team.TeamName + " made an offer for " + playerName,
			Key:    "notifications.messages.cpuOfferReceived",
			Params: map[string]any{"team": team.TeamName, "player": playerName},
		}, "/transfers?tab=offers")
	}
	return true
}

// runAiTransferPass is one shopping pass of a CPU team: it looks at what its profile lets it see,
// scores every candidate with common.ScoreTarget and signs the best one that clears the bar,
// repeating up to the profile's MovesPerPass - there is no quota beyond that, and a team that finds
// nothing worth having signs nobody. A Director starts by withdrawing its stale offers, and when
// nothing is affordable (or the squad is full) lists the player a target would replace, to buy
// him in a later pass. Called from RunAITeamManagement; a no-op for a profile that never shops.
func runAiTransferPass(dbCtx *common.DatabaseContext, team *common.Team) {
	profile := aiProfile(team)
	if profile.MovesPerPass == 0 {
		return
	}
	// A club fighting relegation (or a Director chasing promotion) shops with a lower bar - see
	// openspec/specs/cpu-transfer-market "Market urgency".
	profile = common.AdjustProfileForObjective(profile, common.TeamSeasonObjective(dbCtx, team.Id))
	now := time.Now()
	rng := rand.New(rand.NewSource(now.UnixNano() + team.Id))

	if profile.SearchesUnlisted {
		if expired, err := common.ExpireStaleOffers(dbCtx, team.Id, now.Add(-common.AiOfferLifetime)); err != nil {
			log.Errorf("error withdrawing stale offers of AI-managed team %d: %s", team.Id, err)
		} else if expired > 0 {
			common.AiMarketActionsCounter.Add(context.Background(), int64(expired),
				metric.WithAttributes(attribute.String("action", "offer_expired")))
		}
	}

	state, err := loadTransferState(dbCtx, team)
	if err != nil {
		log.Errorf("error reading the state of AI-managed team %d for its transfer pass: %s", team.Id, err)
		return
	}
	radar, err := common.GetNationalRadar(dbCtx)
	if err != nil {
		log.Errorf("error loading the national-team radar for AI-managed team %d: %s", team.Id, err)
	}
	candidates := gatherCandidates(dbCtx, team, profile, state, radar, now)
	if len(candidates) == 0 {
		return
	}

	attempted := map[int64]bool{}
	movesLeft := profile.MovesPerPass
	offersLeft := profile.MaxNewOffersPerPass
	preContractsLeft := aiPreContractOffersPerPass
	clausesLeft := aiReleaseClausesPerPass
	for i := 0; i < profile.MovesPerPass*aiPassIterationsPerMove && movesLeft > 0; i++ {
		ranked := rankCandidates(profile, state, candidates, attempted, rng)
		target := pickNextTargetWithClauses(ranked, state, offersLeft, preContractsLeft, clausesLeft)
		if target == nil {
			if plan := planSellToBuy(profile, state, candidates, attempted, rng); plan != nil {
				if err := common.CreateTransferListing(dbCtx, plan.Seller.Id, team.Id, plan.Asking); err == nil {
					recordAiMarketAction(team, "listed_to_fund", plan.Seller.Id, plan.Asking)
				} else if !errors.Is(err, common.ErrPlayerAlreadyListed) && !errors.Is(err, common.ErrTransferWindowClosed) &&
					!errors.Is(err, common.ErrPlayerOnLoan) && !errors.Is(err, common.ErrPlayerAlreadyLoanListed) {
					log.Errorf("error listing player %d to fund a signing for AI-managed team %d: %s", plan.Seller.Id, team.Id, err)
				}
			}
			return
		}

		attempted[target.Player.Id] = true
		if !executeTransferMove(dbCtx, team, profile, state, *target) {
			if target.Kind == candidatePreContract {
				// One failed pre-contract offer ends them for this pass, so they can't use up the
				// iterations the pass's signings need.
				preContractsLeft = 0
			}
			continue
		}
		// A pre-contract offer is not a signing (the player only decides in 72 hours and joins next
		// season), so it uses the pass's one pre-contract slot but none of its moves.
		if target.Kind == candidatePreContract {
			preContractsLeft--
		} else {
			movesLeft--
		}
		if target.Kind == candidateUnlisted {
			offersLeft--
		}
		if target.Kind == candidateReleaseClause {
			clausesLeft--
		}
		if state, err = loadTransferState(dbCtx, team); err != nil {
			log.Errorf("error re-reading the state of AI-managed team %d during its transfer pass: %s", team.Id, err)
			return
		}
	}
}

// aiPreContractOffersPerPass is how many pre-contract offers a CPU team makes per transfer pass.
const aiPreContractOffersPerPass = 1

// preContractCandidates are the players in their last season a Manager or Director scouts for a
// pre-contract: at each core position the best profile.PreContractCandidates of other clubs who
// beat its weakest starter there, nobody it already sees elsewhere or already made a pre-contract
// offer to, and nobody past aiMaxPlayerAge. Each candidate's Wage is the one the team would offer:
// common.PreContractMinimumWage of his transfer expectation. Each player taken is marked in seen.
func preContractCandidates(dbCtx *common.DatabaseContext, team *common.Team, profile common.AiTransferProfile,
	state *transferState, seen map[int64]bool) []transferCandidate {
	offered, err := common.GetPlayersWithPreContractOfferFromTeam(dbCtx, team.Id)
	if err != nil {
		log.Errorf("error reading pre-contract offers of AI-managed team %d: %s", team.Id, err)
		return nil
	}
	available := state.available()
	ids := make([]int64, 0)
	for _, position := range common.CorePositions {
		found, err := common.GetExpiringCandidatesForPosition(dbCtx, team.Id, position,
			weakestStarterValue(available, position), aiMaxPlayerAge, profile.PreContractCandidates)
		if err != nil {
			log.Errorf("error searching last-season players for AI-managed team %d: %s", team.Id, err)
			continue
		}
		ids = append(ids, found...)
	}
	fresh := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] && !offered[id] {
			fresh = append(fresh, id)
		}
	}
	candidates := make([]transferCandidate, 0, len(fresh))
	// Only players whose league is far enough along can be offered one (common.
	// PreContractSeasonOpen); one read per club.
	open := map[int64]bool{}
	for _, p := range loadPlayers(dbCtx, fresh) {
		isOpen, known := open[p.TeamId]
		if !known {
			situation, err := common.GetTeamSeasonSituation(dbCtx, p.TeamId)
			isOpen = err == nil && common.PreContractSeasonOpen(situation.Played, situation.Remaining)
			open[p.TeamId] = isOpen
		}
		if !isOpen {
			continue
		}
		expected, _, err := common.PlayerTermsExpectation(dbCtx, *p, common.TermsTransfer)
		if err != nil {
			log.Errorf("error reading the terms of player %d for AI-managed team %d: %s", p.Id, team.Id, err)
			continue
		}
		seen[p.Id] = true
		target := *p
		target.Wage = common.PreContractMinimumWage(expected)
		candidates = append(candidates, transferCandidate{Player: target, Kind: candidatePreContract, SellerTeamId: p.TeamId})
	}
	return candidates
}

// makeAiPreContractOffer offers a pre-contract at exactly the wage the player asks to leave his
// club (target.Player.Wage, see preContractCandidates) and his age's maximum term, within the
// limits on pending CPU offers: at most common.AiMaxPreContractOffersPerHumanPlayer on a human
// club's player, one on a CPU club's.
func makeAiPreContractOffer(dbCtx *common.DatabaseContext, team *common.Team, target scoredCandidate) bool {
	p := target.Player
	owner, err := common.GetTeamOwner(dbCtx, target.SellerTeamId)
	if err != nil {
		log.Errorf("error looking up the owner of team %d: %s", target.SellerTeamId, err)
		return false
	}
	pending, err := common.CountAiPreContractOffersOnPlayer(dbCtx, p.Id)
	if err != nil {
		log.Errorf("error counting CPU pre-contract offers on player %d: %s", p.Id, err)
		return false
	}
	if (owner > 0 && pending >= common.AiMaxPreContractOffersPerHumanPlayer) || (owner <= 0 && pending >= 1) {
		return false
	}
	current, err := common.GetPlayerById(dbCtx, p.Id)
	if err != nil {
		return false
	}
	_, years, err := common.PlayerTermsExpectation(dbCtx, *current, common.TermsTransfer)
	if err != nil {
		return false
	}
	result, err := common.MakePreContractOffer(dbCtx, p.Id, team.Id, p.Wage, years)
	if err != nil {
		if !errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrPreContractAlreadyOffered) &&
			!errors.Is(err, common.ErrPreContractTooEarly) && !errors.Is(err, common.ErrPreContractNotLastSeason) &&
			!errors.Is(err, common.ErrPlayerPreContracted) && !errors.Is(err, common.ErrPlayerOnLoan) &&
			!errors.Is(err, common.ErrTeamInsolvent) && !errors.Is(err, common.ErrPreContractLimit) &&
			!errors.Is(err, common.ErrPlayerIsFreeAgent) {
			log.Errorf("error making a pre-contract offer for player %d on behalf of AI-managed team %d: %s", p.Id, team.Id, err)
		}
		return false
	}
	if !result.Accepted {
		return false
	}
	recordAiMarketAction(team, "pre_contract_offered", p.Id, int64(p.Wage))
	return true
}

// aiReleaseClausesPerPass is how many release clauses a Director pays per transfer pass.
const aiReleaseClausesPerPass = 1

// payableClauses reads the release clauses of the players in ids that can be paid now (past their
// lock-in), by player id.
func payableClauses(dbCtx *common.DatabaseContext, ids []int64, now time.Time) map[int64]int64 {
	clauses := map[int64]int64{}
	for _, id := range ids {
		p, err := common.GetPlayerById(dbCtx, id)
		if err != nil || p.ReleaseClause == nil {
			continue
		}
		if p.ReleaseClausePayableAt != nil && now.Before(*p.ReleaseClausePayableAt) {
			continue
		}
		clauses[id] = *p.ReleaseClause
	}
	return clauses
}

// cpuMayPayClauseOn is common.CpuMayPayClauseOn, false when it can't be read.
func cpuMayPayClauseOn(dbCtx *common.DatabaseContext, sellerTeamId int64, now time.Time) bool {
	ok, err := common.CpuMayPayClauseOn(dbCtx, sellerTeamId, now)
	if err != nil {
		log.Errorf("error reading the release clause payments on team %d: %s", sellerTeamId, err)
		return false
	}
	return ok
}

// payAiReleaseClause pays target's release clause on the player's own transfer terms, with no
// clause in his new contract.
func payAiReleaseClause(dbCtx *common.DatabaseContext, team *common.Team, target scoredCandidate) bool {
	p := target.Player
	wage, years, err := common.PlayerTermsExpectation(dbCtx, p, common.TermsTransfer)
	if err != nil {
		log.Errorf("error reading the personal terms of player %d for AI-managed team %d: %s", p.Id, team.Id, err)
		return false
	}
	result, err := common.PayReleaseClause(dbCtx, p.Id, team.Id, wage, years, nil)
	if err != nil {
		if !errors.Is(err, common.ErrTransferWindowClosed) && !errors.Is(err, common.ErrInsufficientBudget) &&
			!errors.Is(err, common.ErrNoReleaseClause) && !errors.Is(err, common.ErrReleaseClauseLockedIn) &&
			!errors.Is(err, common.ErrPlayerOnLoan) && !errors.Is(err, common.ErrPlayerPreContracted) &&
			!errors.Is(err, common.ErrTeamInsolvent) && !errors.Is(err, common.ErrCannotOfferOnOwnPlayer) &&
			!errors.Is(err, common.ErrPlayerIsFreeAgent) {
			log.Errorf("error paying the release clause of player %d for AI-managed team %d: %s", p.Id, team.Id, err)
		}
		return false
	}
	if !result.Accepted {
		return false
	}
	recordAiMarketAction(team, "paid_release_clause", p.Id, result.Fee)
	return true
}
