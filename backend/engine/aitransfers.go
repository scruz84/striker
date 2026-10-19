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
	// aiMaxBidValueMultiple is the most a team will bid, relative to its own valuation - a rebid
	// after a rejection only goes ahead if its valuation allows it.
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
	Caps        *common.PlayerSkillCaps
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
	Budget     int64
	WageBill   int64
	// WageFactor is the team's division wage factor - WageBill already includes it.
	WageFactor float64
	// OpenOffers is how many offers the team currently has out.
	OpenOffers int
	// Protected are squad players the team won't sell to fund a signing - see
	// protectedFromListing.
	Protected map[int64]bool
}

// available is the squad the team can count on: owned and not already up for sale.
func (s *transferState) available() []*common.Player {
	return withoutPlayers(s.Squad, s.Listed)
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
func scoreOne(profile common.AiTransferProfile, state *transferState, c transferCandidate, rng *rand.Rand) (common.TargetScore, bool) {
	ctx := common.TargetContext{
		Squad:         state.available(),
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
	if c.Kind == candidateUnlisted {
		ctx.BidMultiple = aiBaseBidMultiple
	}
	score := common.ScoreTarget(profile, ctx, rng)
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

// rankCandidates scores every candidate not yet attempted and keeps the ones that clear the
// profile's bar - affordable, and a score of at least MinScore - best first. Pure given the rng,
// so the choice of targets can be tested without a database.
func rankCandidates(profile common.AiTransferProfile, state *transferState, candidates []transferCandidate,
	attempted map[int64]bool, rng *rand.Rand) []scoredCandidate {
	ranked := make([]scoredCandidate, 0)
	for _, c := range candidates {
		if attempted[c.Player.Id] {
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
// effective squad is at common.MaxAiSquadSize signs nobody (it has to sell first), and only
// offersLeft more offers for unlisted players may be started this pass.
func pickNextTarget(ranked []scoredCandidate, state *transferState, offersLeft int) *scoredCandidate {
	if len(state.available()) >= common.MaxAiSquadSize {
		return nil
	}
	for i := range ranked {
		if ranked[i].Kind == candidateUnlisted &&
			(offersLeft <= 0 || state.OpenOffers >= common.AiMaxOpenOffersPerTeam) {
			continue
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
	squadFull := len(available) >= common.MaxAiSquadSize

	var best *sellPlan
	for _, c := range candidates {
		if attempted[c.Player.Id] || c.Kind == candidateFreeAgent {
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
	squad := ownedPlayers(*players, commitments.LoanedIn)
	return &transferState{
		Squad:      squad,
		Listed:     commitments.TransferListed,
		LoanListed: commitments.LoanListed,
		Budget:     budget,
		WageBill:   wageBill,
		WageFactor: wageFactor,
		OpenOffers: len(sent),
		Protected:  protectedFromListing(dbCtx, team, squad),
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
		for _, p := range loadPlayers(dbCtx, fresh) {
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
	wage := common.PlayerWageFromSkills(p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot)
	amount := bidAmount(profile, state, target)
	result, err := common.MakeTransferOffer(dbCtx, p.Id, team.Id, amount, wage, common.MaxAcceptableRenewalYears(p.Age))
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
	profile := common.AiTransferProfileFor(team.AiLevel)
	if profile.MovesPerPass == 0 {
		return
	}
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
	for i := 0; i < profile.MovesPerPass*aiPassIterationsPerMove && movesLeft > 0; i++ {
		ranked := rankCandidates(profile, state, candidates, attempted, rng)
		target := pickNextTarget(ranked, state, offersLeft)
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
			continue
		}
		movesLeft--
		if target.Kind == candidateUnlisted {
			offersLeft--
		}
		if state, err = loadTransferState(dbCtx, team); err != nil {
			log.Errorf("error re-reading the state of AI-managed team %d during its transfer pass: %s", team.Id, err)
			return
		}
	}
}
