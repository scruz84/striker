package engine

import (
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

func rngSeed(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// adultSquad is squadOf (2 GK, 5 D, 5 M, 5 S - skills 90, 85, 80... within a position) with every
// player 28 years old and an extra pair of weak outfield reserves ids 100+ to reach SquadSize,
// the standing assumption of the transfer pass tests.
func adultSquad() []*common.Player {
	squad := squadOf(2, 5, 5, 5)
	for len(squad) < common.SquadSize {
		squad = append(squad, defender(int64(100+len(squad)), 20))
	}
	for _, p := range squad {
		p.Age = 28
	}
	return squad
}

// exact is a profile with its valuation noise switched off, for tests of the ranking and planning
// logic that shouldn't depend on a lucky draw.
func exact(profile common.AiTransferProfile) common.AiTransferProfile {
	profile.ValuationNoise = 0
	return profile
}

func stateOf(squad []*common.Player, budget int64) *transferState {
	return &transferState{Squad: squad, Listed: map[int64]bool{}, LoanListed: map[int64]bool{}, Budget: budget, WageBill: 1000}
}

func listedCandidate(id int64, p *common.Player) transferCandidate {
	return transferCandidate{Player: *p, Kind: candidateListed, ListingId: id, SellerTeamId: 99, AskingPrice: common.MarketValue(*p)}
}

func unlistedCandidate(p *common.Player) transferCandidate {
	return transferCandidate{Player: *p, Kind: candidateUnlisted, SellerTeamId: 99}
}

func TestRankCandidates_UpgradeBeatsFillerAndFillerIsDropped(t *testing.T) {
	profile := common.AiTransferProfileFor(common.AiLevelDirector)
	state := stateOf(adultSquad(), 50_000_000)
	star := midfielder(500, 99)
	star.Age = 27
	filler := midfielder(501, 30)
	filler.Age = 27

	ranked := rankCandidates(profile, state, []transferCandidate{listedCandidate(1, filler), listedCandidate(2, star)}, map[int64]bool{}, rngSeed(1))
	if len(ranked) != 1 || ranked[0].Player.Id != 500 {
		t.Fatalf("ranked %d candidates (first %v), want only the star", len(ranked), ranked)
	}
}

func TestRankCandidates_NothingWorthSigningMeansNoSignings(t *testing.T) {
	// No quota: a squad full of better players than any candidate signs nobody.
	profile := common.AiTransferProfileFor(common.AiLevelDirector)
	state := stateOf(adultSquad(), 50_000_000)
	weak := midfielder(500, 40)
	weak.Age = 27
	if ranked := rankCandidates(profile, state, []transferCandidate{listedCandidate(1, weak)}, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Errorf("ranked %d candidates, want none", len(ranked))
	}
}

func TestRankCandidates_SkipsAttemptedAndSortsBestFirst(t *testing.T) {
	profile := common.AiTransferProfileFor(common.AiLevelDirector)
	state := stateOf(adultSquad(), 50_000_000)
	good := midfielder(500, 96)
	better := midfielder(501, 99)
	for _, p := range []*common.Player{good, better} {
		p.Age = 27
	}
	candidates := []transferCandidate{listedCandidate(1, good), listedCandidate(2, better)}

	profile = exact(profile)
	ranked := rankCandidates(profile, state, candidates, map[int64]bool{}, rngSeed(1))
	if len(ranked) != 2 || ranked[0].Player.Id != 501 {
		t.Fatalf("want best first, got %v", ranked)
	}
	ranked = rankCandidates(profile, state, candidates, map[int64]bool{501: true}, rngSeed(1))
	if len(ranked) != 1 || ranked[0].Player.Id != 500 {
		t.Errorf("an attempted candidate must be skipped, got %v", ranked)
	}
}

func TestRankCandidates_ManagerNeedsAClearerUpgradeThanDirector(t *testing.T) {
	// A small upgrade (a couple of points) clears a Director's bar but not a Manager's.
	squad := adultSquad()
	small := midfielder(500, 80) // adds 3 points to the XI: over a Director's bar of 3, under a Manager's 5
	small.Age = 27
	candidates := []transferCandidate{listedCandidate(1, small)}

	director := rankCandidates(exact(common.AiTransferProfileFor(common.AiLevelDirector)), stateOf(squad, 50_000_000), candidates, map[int64]bool{}, rngSeed(1))
	manager := rankCandidates(exact(common.AiTransferProfileFor(common.AiLevelManager)), stateOf(squad, 50_000_000), candidates, map[int64]bool{}, rngSeed(1))
	if len(director) != 1 {
		t.Fatalf("director should take a modest upgrade, ranked %d", len(director))
	}
	if len(manager) != 0 {
		t.Errorf("manager should only move for a clear upgrade, ranked %d", len(manager))
	}
}

func TestRankCandidates_ProjectsAreDirectorOnly(t *testing.T) {
	prospect := midfielder(500, 70) // too weak to start today, but with a 99 ceiling
	prospect.Age = 18
	caps := &common.PlayerSkillCaps{MidfieldMax: 99, PassMax: 99}
	c := unlistedCandidate(prospect)
	c.Caps = caps

	state := stateOf(adultSquad(), 50_000_000)
	if ranked := rankCandidates(exact(common.AiTransferProfileFor(common.AiLevelDirector)), state, []transferCandidate{c}, map[int64]bool{}, rngSeed(1)); len(ranked) != 1 || !ranked[0].Score.Project {
		t.Errorf("a Director should sign a high-ceiling teenager as a project, got %v", ranked)
	}
	if ranked := rankCandidates(exact(common.AiTransferProfileFor(common.AiLevelManager)), state, []transferCandidate{c}, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Errorf("a Manager does not buy for the future, got %v", ranked)
	}
}

func TestRankCandidates_RebidBeyondValuationAllowanceIsDropped(t *testing.T) {
	profile := common.AiTransferProfileFor(common.AiLevelDirector)
	state := stateOf(adultSquad(), 500_000_000)
	star := midfielder(500, 99)
	star.Age = 27
	c := unlistedCandidate(star)

	if ranked := rankCandidates(profile, state, []transferCandidate{c}, map[int64]bool{}, rngSeed(1)); len(ranked) != 1 {
		t.Fatalf("setup: the star should be a target, got %d", len(ranked))
	}
	c.MinBid = common.MarketValue(*star) * 5 // rejected a bid far above anything the team would pay
	if ranked := rankCandidates(profile, state, []transferCandidate{c}, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Error("a rebid floor beyond the valuation allowance must rule the player out")
	}
	c.MinBid = common.MarketValue(*star) * 11 / 10 // modest raise: fine
	ranked := rankCandidates(profile, state, []transferCandidate{c}, map[int64]bool{}, rngSeed(1))
	if len(ranked) != 1 || ranked[0].Score.Price < c.MinBid {
		t.Errorf("a modest rebid should proceed at or above the floor, got %v", ranked)
	}
}

func TestPickNextTarget_RespectsOfferAllowanceAndSquadCeiling(t *testing.T) {
	star := midfielder(500, 99)
	star.Age = 27
	listed := scoredCandidate{transferCandidate: listedCandidate(1, star)}
	unlisted := scoredCandidate{transferCandidate: unlistedCandidate(star)}
	state := stateOf(adultSquad(), 50_000_000)

	if got := pickNextTarget([]scoredCandidate{unlisted, listed}, state, 0, 1); got == nil || got.Kind != candidateListed {
		t.Errorf("with no offers left the pass must skip to the listed target, got %v", got)
	}
	state.OpenOffers = common.AiMaxOpenOffersPerTeam
	if got := pickNextTarget([]scoredCandidate{unlisted}, state, 2, 1); got != nil {
		t.Error("a team with the maximum offers out starts no more")
	}

	full := stateOf(adultSquad(), 50_000_000)
	for i := len(full.Squad); i < common.MaxAiSquadSize; i++ {
		full.Squad = append(full.Squad, defender(int64(300+i), 20))
	}
	if got := pickNextTarget([]scoredCandidate{listed}, full, 2, 1); got != nil {
		t.Error("a squad at its ceiling signs nobody")
	}
	full.Listed[full.Squad[len(full.Squad)-1].Id] = true
	if got := pickNextTarget([]scoredCandidate{listed}, full, 2, 1); got == nil {
		t.Error("a player already on the transfer list doesn't count against the ceiling")
	}
}

func TestBidAmount_RisesWithInterestWithinLimits(t *testing.T) {
	profile := common.AiTransferProfileFor(common.AiLevelDirector)
	state := stateOf(adultSquad(), 10_000_000)
	base := scoredCandidate{Score: common.TargetScore{Total: profile.MinScore, Price: 10_000, Value: 10_000}}
	keen := scoredCandidate{Score: common.TargetScore{Total: profile.MinScore + 50, Price: 10_000, Value: 10_000}}

	if got := bidAmount(profile, state, base); got != 10_000 {
		t.Errorf("a target at the bar is bid for the scored price, got %d", got)
	}
	if got := bidAmount(profile, state, keen); got <= 10_000 || got > 12_500 {
		t.Errorf("a keen bid should be raised but by at most %v, got %d", aiMaxBidBump, got)
	}
	capped := scoredCandidate{Score: common.TargetScore{Total: profile.MinScore + 50, Price: 10_000, Value: 10_100}}
	if got := bidAmount(profile, state, capped); got > 15_150 {
		t.Errorf("a bid never exceeds %vx the valuation, got %d", aiMaxBidValueMultiple, got)
	}
	poor := stateOf(adultSquad(), 10_050)
	if got := bidAmount(profile, poor, keen); got > 10_050-int64(float64(10_050)*profile.BudgetReserve) && got != 10_000 {
		t.Errorf("a bid never goes past what the budget allows, got %d", got)
	}
}

func TestPlanSellToBuy_ListsTheReplacedPlayerWhenBudgetIsShort(t *testing.T) {
	profile := common.AiTransferProfileFor(common.AiLevelDirector)
	squad := adultSquad()
	star := midfielder(500, 99)
	star.Age = 27
	candidates := []transferCandidate{listedCandidate(1, star)}
	price := candidates[0].AskingPrice

	// Not enough money for the star (nor the payroll runway for his wage), but selling the weakest
	// of the five midfielders would do.
	profile = exact(profile)
	state := stateOf(squad, price+1500)
	state.WageBill = 0
	if ranked := rankCandidates(profile, state, candidates, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Fatal("setup: the star must be unaffordable")
	}
	plan := planSellToBuy(profile, state, candidates, map[int64]bool{}, rngSeed(1))
	if plan == nil {
		t.Fatal("a Director with an unaffordable upgrade should plan to sell")
	}
	weakestMidfielder := squad[2+5+4] // squadOf: 2 GK, 5 D, then midfielders 90..70
	if plan.Seller.Id != weakestMidfielder.Id {
		t.Errorf("planned to sell player %d, want the weakest midfielder %d", plan.Seller.Id, weakestMidfielder.Id)
	}
	if plan.For.Player.Id != 500 || plan.Asking <= 0 {
		t.Errorf("plan = %+v", plan)
	}
}

func TestPlanSellToBuy_OnlyDirectorsAndOnlyWithoutOtherListings(t *testing.T) {
	squad := adultSquad()
	star := midfielder(500, 99)
	star.Age = 27
	candidates := []transferCandidate{listedCandidate(1, star)}
	state := stateOf(squad, candidates[0].AskingPrice+1500)
	state.WageBill = 0

	if plan := planSellToBuy(common.AiTransferProfileFor(common.AiLevelManager), state, candidates, map[int64]bool{}, rngSeed(1)); plan != nil {
		t.Error("a Manager doesn't sell to buy")
	}
	state.Listed[squad[0].Id] = true
	if plan := planSellToBuy(exact(common.AiTransferProfileFor(common.AiLevelDirector)), state, candidates, map[int64]bool{}, rngSeed(1)); plan != nil {
		t.Error("with a player already listed the team waits for that sale first")
	}
}

func TestPlanSellToBuy_NeverBreaksPositionDepthOrSquadMinimum(t *testing.T) {
	profile := common.AiTransferProfileFor(common.AiLevelDirector)
	star := midfielder(500, 99)
	star.Age = 27
	candidates := []transferCandidate{listedCandidate(1, star)}

	// Only the minimum three midfielders: selling one would be withdrawn straight away.
	squad := squadOf(2, 5, 3, 5)
	for len(squad) < common.SquadSize {
		squad = append(squad, defender(int64(100+len(squad)), 20))
	}
	for _, p := range squad {
		p.Age = 28
	}
	state := stateOf(squad, candidates[0].AskingPrice+1500)
	state.WageBill = 0
	if plan := planSellToBuy(exact(profile), state, candidates, map[int64]bool{}, rngSeed(1)); plan != nil {
		t.Errorf("planned to sell %d from a position at minimum depth", plan.Seller.Id)
	}
}

func TestProjectsInUse_CountsYoungPlayersOutsideTheBestXI(t *testing.T) {
	squad := adultSquad()
	kid := midfielder(600, 20)
	kid.Age = 19
	starterKid := midfielder(601, 99)
	starterKid.Age = 20
	squad = append(squad, kid, starterKid)
	state := stateOf(squad, 1)
	if got := state.projectsInUse(); got != 1 {
		t.Errorf("projects in use = %d, want only the benched teenager", got)
	}
}

func TestWeakestStarterValue(t *testing.T) {
	squad := adultSquad()
	// squadOf's midfielders are 90, 85, 80, 75, 70 (value = (midfield+pass)/2): the 4-4-2 best XI
	// plays four of them (two central, two wide), so the weakest starter is the 75.
	if got := weakestStarterValue(squad, common.MIDFIELD); got != 75 {
		t.Errorf("weakest starting midfielder = %d, want 75", got)
	}
	if got := weakestStarterValue(nil, common.MIDFIELD); got != 0 {
		t.Errorf("an empty squad has no starter to beat, got %d", got)
	}
}

func TestFreeAgentPremium_OnlyInTheFirstSeasonWithALastClub(t *testing.T) {
	p := common.Player{Midfield: 70, Pass: 70, LastTeamId: 5, FreeAgentSeasons: 0}
	if freeAgentPremium(p) <= 0 {
		t.Error("a freshly released player costs a premium")
	}
	p.FreeAgentSeasons = 1
	if freeAgentPremium(p) != 0 {
		t.Error("no premium after a season unsigned")
	}
	p.FreeAgentSeasons, p.LastTeamId = 0, 0
	if freeAgentPremium(p) != 0 {
		t.Error("no premium without a last club")
	}
}

func TestRadarCandidateIds_OnlyThoseWhoBeatTheWeakestStarterBestFirst(t *testing.T) {
	squad := adultSquad() // weakest starting midfielder is 75
	strongMid := midfielder(900, 95)
	weakMid := midfielder(901, 60)
	oldStar := midfielder(902, 99)
	oldStar.Age = 40
	midStar := midfielder(903, 90)
	for _, p := range []*common.Player{strongMid, weakMid, midStar} {
		p.Age = 27
	}
	radar := common.NewNationalRadar(map[string][]*common.Player{
		"es": {strongMid, weakMid, oldStar},
		"it": {midStar},
	}, nil)

	got := radarCandidateIds(radar, squad, map[int64]bool{})
	if len(got) != 2 || got[0] != 900 || got[1] != 903 {
		t.Errorf("radar candidates = %v, want [900 903] (the 60 can't help, the 40-year-old is too old)", got)
	}
	if got := radarCandidateIds(radar, squad, map[int64]bool{900: true}); len(got) != 1 || got[0] != 903 {
		t.Errorf("players already considered must be skipped, got %v", got)
	}
	if got := radarCandidateIds(nil, squad, map[int64]bool{}); len(got) != 0 {
		t.Errorf("no radar, no candidates; got %v", got)
	}
}

func TestPlanSellToBuy_NeverSellsAProtectedPlayer(t *testing.T) {
	squad := adultSquad()
	star := midfielder(500, 99)
	star.Age = 27
	candidates := []transferCandidate{listedCandidate(1, star)}
	state := stateOf(squad, candidates[0].AskingPrice+1500)
	state.WageBill = 0
	profile := exact(common.AiTransferProfileFor(common.AiLevelDirector))

	weakestMidfielder := squad[2+5+4]
	state.Protected = map[int64]bool{weakestMidfielder.Id: true}
	plan := planSellToBuy(profile, state, candidates, map[int64]bool{}, rngSeed(1))
	if plan == nil {
		t.Fatal("the next weakest midfielder should be sold instead")
	}
	if plan.Seller.Id == weakestMidfielder.Id {
		t.Error("planned to sell a protected player")
	}
}

func TestScoreOne_CarriesInternationalStandingIntoTheScore(t *testing.T) {
	profile := exact(common.AiTransferProfileFor(common.AiLevelDirector))
	state := stateOf(adultSquad(), 50_000_000)
	p := midfielder(500, 95)
	p.Age = 27
	domestic := listedCandidate(1, p)
	international := domestic
	international.International, international.NationalCaps = true, 12

	a, _ := scoreOne(profile, state, domestic, rngSeed(1))
	b, _ := scoreOne(profile, state, international, rngSeed(1))
	if b.Reputation <= 0 || b.Total <= a.Total {
		t.Errorf("international scored %v (reputation %v), domestic %v - want the international ahead", b.Total, b.Reputation, a.Total)
	}
}

// loanCandidate is a loan-market listing: AskingPrice is its loan fee.
func loanCandidate(id int64, p *common.Player, fee int64, seasons int, contribution int, option int64) transferCandidate {
	return transferCandidate{
		Player: *p, Kind: candidateLoan, ListingId: id, SellerTeamId: 99, AskingPrice: fee,
		LoanSeasons: seasons, LoanWageContributionPercent: contribution, BuyOptionFee: option,
	}
}

func TestPickNextTarget_LoanedInPlayersCountTowardsTheCap(t *testing.T) {
	star := midfielder(500, 99)
	star.Age = 27
	loan := scoredCandidate{transferCandidate: loanCandidate(1, star, 0, 1, 0, 0)}
	listed := scoredCandidate{transferCandidate: listedCandidate(2, star)}
	option := scoredCandidate{transferCandidate: transferCandidate{Player: *star, Kind: candidateBuyOption, BuyOptionFee: 1}}

	// SquadSize+2 owned players plus 2 loanees: at the SquadSize+4 ceiling.
	state := stateOf(adultSquad(), 50_000_000)
	for len(state.Squad) < common.MaxAiSquadSize-2 {
		state.Squad = append(state.Squad, defender(int64(300+len(state.Squad)), 20))
	}
	state.LoanedIn = []*common.Player{midfielder(600, 50), midfielder(601, 50)}

	for name, c := range map[string]scoredCandidate{"loan": loan, "transfer": listed} {
		if got := pickNextTarget([]scoredCandidate{c}, state, 2, 1); got != nil {
			t.Errorf("a squad full of owned players and loanees must take on no %s, got %v", name, got.Kind)
		}
	}
	if got := pickNextTarget([]scoredCandidate{option}, state, 2, 1); got == nil {
		t.Error("exercising a buy option adds no body, so it goes ahead at the ceiling")
	}

	// One loanee fewer leaves room for one more body - a transfer, but no loan while 2 are in.
	state.LoanedIn = state.LoanedIn[:1]
	if got := pickNextTarget([]scoredCandidate{listed}, state, 2, 1); got == nil {
		t.Error("with room under the ceiling a transfer goes ahead")
	}
}

func TestPickNextTarget_NoMoreLoansAtTheLoanLimit(t *testing.T) {
	star := midfielder(500, 99)
	star.Age = 27
	loan := scoredCandidate{transferCandidate: loanCandidate(1, star, 0, 1, 0, 0)}
	listed := scoredCandidate{transferCandidate: listedCandidate(2, star)}
	state := stateOf(adultSquad(), 50_000_000)

	state.LoanedIn = []*common.Player{midfielder(600, 50)}
	if got := pickNextTarget([]scoredCandidate{loan}, state, 2, 1); got == nil {
		t.Error("with one loanee a second loan is allowed")
	}
	state.LoanedIn = []*common.Player{midfielder(600, 50), midfielder(601, 50)}
	if got := pickNextTarget([]scoredCandidate{loan}, state, 2, 1); got != nil {
		t.Errorf("with %d players on loan no loan is accepted, got %v", common.AiMaxLoanedIn, got.Kind)
	}
	if got := pickNextTarget([]scoredCandidate{loan, listed}, state, 2, 1); got == nil || got.Kind != candidateListed {
		t.Errorf("the pass moves on to the next candidate, got %v", got)
	}
}

func TestScoreOne_LoanIsScoredAgainstPlayersAlreadyOnLoan(t *testing.T) {
	profile := exact(common.AiTransferProfileFor(common.AiLevelManager))
	target := striker(500, 97)
	target.Age = 27
	c := loanCandidate(1, target, 0, 1, 0, 0)

	alone := stateOf(adultSquad(), 50_000_000)
	without, ok := scoreOne(profile, alone, c, rngSeed(1))
	if !ok || without.Improvement <= 0 {
		t.Fatalf("test setup: a 97 striker should improve the squad, got %+v (ok=%v)", without, ok)
	}

	withLoanee := stateOf(adultSquad(), 50_000_000)
	withLoanee.LoanedIn = []*common.Player{striker(501, 99), striker(502, 98), striker(503, 98), striker(504, 98), striker(505, 98)}
	with, _ := scoreOne(profile, withLoanee, c, rngSeed(1))
	if with.Improvement >= without.Improvement {
		t.Errorf("a team whose strikers are all on loan already gains less (%v) than one without them (%v)", with.Improvement, without.Improvement)
	}

	// A transfer ignores the loanees: they go back, so it is judged against the owned squad.
	transfer := listedCandidate(2, target)
	a, _ := scoreOne(profile, alone, transfer, rngSeed(1))
	b, _ := scoreOne(profile, withLoanee, transfer, rngSeed(1))
	if a.Improvement != b.Improvement {
		t.Errorf("a transfer's improvement should not depend on loanees: %v vs %v", a.Improvement, b.Improvement)
	}
}

func TestRankCandidates_ALoanCanBeatATransfer(t *testing.T) {
	profile := exact(common.AiTransferProfileFor(common.AiLevelManager))
	state := stateOf(adultSquad(), 50_000_000)
	star := midfielder(500, 98)
	star.Age = 27
	alternative := midfielder(501, 96)
	alternative.Age = 27

	ranked := rankCandidates(profile, state, []transferCandidate{
		listedCandidate(1, alternative), loanCandidate(2, star, 0, 1, 0, 0),
	}, map[int64]bool{}, rngSeed(1))
	if len(ranked) == 0 || ranked[0].Kind != candidateLoan {
		t.Fatalf("a free loan of a better player should lead the ranking, got %+v", ranked)
	}
	if got := pickNextTarget(ranked, state, 0, 1); got == nil || got.Kind != candidateLoan {
		t.Errorf("the pass should accept the loan, got %v", got)
	}
}

func TestLoanListingCandidates_SkipsOwnListingsTheOldAndPlayersAlreadySeen(t *testing.T) {
	young := midfielder(1, 80)
	young.Age = 25
	old := midfielder(2, 80)
	old.Age = aiMaxPlayerAge + 1
	own := midfielder(3, 80)
	own.Age = 25
	seenAlready := midfielder(4, 80)
	seenAlready.Age = 25
	listings := []common.LoanListing{
		{ListingId: 11, Player: *young, LendingTeamId: 7, LoanFee: 5000, DurationSeasons: 2, WageContributionPercent: 30, BuyOptionFee: 90000},
		{ListingId: 12, Player: *old, LendingTeamId: 7},
		{ListingId: 13, Player: *own, LendingTeamId: 5},
		{ListingId: 14, Player: *seenAlready, LendingTeamId: 7},
	}
	seen := map[int64]bool{4: true}

	got := loanListingCandidates(listings, 5, seen)
	if len(got) != 1 {
		t.Fatalf("only the young player from another club qualifies, got %d candidates", len(got))
	}
	c := got[0]
	if c.Kind != candidateLoan || c.Player.Id != 1 || c.ListingId != 11 || c.SellerTeamId != 7 || c.AskingPrice != 5000 ||
		c.LoanSeasons != 2 || c.LoanWageContributionPercent != 30 || c.BuyOptionFee != 90000 {
		t.Errorf("candidate = %+v, want the listing's terms carried over", c)
	}
	if !seen[1] {
		t.Error("a player taken as a candidate is marked as seen")
	}
}

func TestBuyOptionCandidates_OnlyInTheLastSeasonOfALoanWithAnOption(t *testing.T) {
	mk := func(id int64, age int) *common.Player {
		p := midfielder(id, 90)
		p.Age = age
		return p
	}
	state := stateOf(adultSquad(), 50_000_000)
	state.CurrentSeason = 5
	state.LoanedIn = []*common.Player{mk(601, 27), mk(602, 27), mk(603, 27), mk(604, aiMaxPlayerAge+1), mk(605, 27)}
	state.Loans = []common.PlayerLoan{
		{PlayerId: 601, FromTeamId: 9, ReturnSeason: 6, BuyOptionFee: 100_000}, // last season, option: yes
		{PlayerId: 602, FromTeamId: 9, ReturnSeason: 7, BuyOptionFee: 100_000}, // two seasons left: no
		{PlayerId: 603, FromTeamId: 9, ReturnSeason: 6, BuyOptionFee: 0},       // no option: no
		{PlayerId: 604, FromTeamId: 9, ReturnSeason: 6, BuyOptionFee: 100_000}, // too old: no
		{PlayerId: 999, FromTeamId: 9, ReturnSeason: 6, BuyOptionFee: 100_000}, // not in the squad: no
	}

	got := buyOptionCandidates(state)
	if len(got) != 1 || got[0].Player.Id != 601 || got[0].Kind != candidateBuyOption || got[0].BuyOptionFee != 100_000 || got[0].SellerTeamId != 9 {
		t.Fatalf("want only player 601's option, got %+v", got)
	}
}

func TestRankCandidates_BuyOptionOnAStarterIsWorthExercising(t *testing.T) {
	profile := exact(common.AiTransferProfileFor(common.AiLevelManager))
	starter := midfielder(601, 99)
	starter.Age = 27
	starter.Wage = 1000
	option := transferCandidate{Player: *starter, Kind: candidateBuyOption, SellerTeamId: 9, BuyOptionFee: common.MarketValue(*starter)}

	// Without him the XI has nobody at that level: losing him would cost points, so keeping him scores.
	state := stateOf(adultSquad(), 50_000_000)
	state.LoanedIn = []*common.Player{starter}
	ranked := rankCandidates(profile, state, []transferCandidate{option}, map[int64]bool{}, rngSeed(1))
	if len(ranked) != 1 || ranked[0].Score.Improvement <= 0 {
		t.Fatalf("a star loanee is worth his option, got %+v", ranked)
	}

	// A loanee the squad outclasses adds nothing to the XI, so his option is not worth exercising.
	bench := midfielder(602, 30)
	bench.Age = 27
	weakOption := transferCandidate{Player: *bench, Kind: candidateBuyOption, SellerTeamId: 9, BuyOptionFee: common.MarketValue(*bench)}
	redundant := stateOf(adultSquad(), 50_000_000)
	redundant.LoanedIn = []*common.Player{bench}
	if ranked := rankCandidates(profile, redundant, []transferCandidate{weakOption}, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Errorf("an option on a player who doesn't improve the XI should be passed, got %+v", ranked)
	}

	// Too dear for a Manager's reserve: not exercised.
	poor := stateOf(adultSquad(), common.MarketValue(*starter)+1)
	poor.LoanedIn = []*common.Player{starter}
	if ranked := rankCandidates(profile, poor, []transferCandidate{option}, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Errorf("an option that would breach the budget reserve must not be exercised, got %+v", ranked)
	}
}

func TestRankCandidates_ListingFarAboveValuationIsNotBought(t *testing.T) {
	// A big upgrade clears the bar by far more than the capped value-for-money penalty, so only
	// the price ceiling keeps a CPU team from paying any asking price for him.
	star := midfielder(500, 99)
	star.Age = 27
	for _, level := range []int{common.AiLevelManager, common.AiLevelDirector} {
		profile := exact(common.AiTransferProfileFor(level))
		state := stateOf(adultSquad(), 500_000_000)

		fair := listedCandidate(1, star)
		if ranked := rankCandidates(profile, state, []transferCandidate{fair}, map[int64]bool{}, rngSeed(1)); len(ranked) != 1 {
			t.Fatalf("level %d: setup: the star at his market value should be a target, got %d", level, len(ranked))
		}
		dear := fair
		dear.AskingPrice = common.MarketValue(*star) * 3
		if ranked := rankCandidates(profile, state, []transferCandidate{dear}, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
			t.Errorf("level %d: a listing at 3x the valuation must not be bought, got %+v", level, ranked)
		}
		pricey := fair
		pricey.AskingPrice = common.MarketValue(*star) * 14 / 10
		if ranked := rankCandidates(profile, state, []transferCandidate{pricey}, map[int64]bool{}, rngSeed(1)); len(ranked) != 1 {
			t.Errorf("level %d: a listing within the bid ceiling is still worth buying, got %d", level, len(ranked))
		}
	}
}

func TestRankCandidates_BuyOptionFarAboveValuationIsNotExercised(t *testing.T) {
	profile := exact(common.AiTransferProfileFor(common.AiLevelDirector))
	starter := midfielder(601, 99)
	starter.Age = 27
	starter.Wage = 1000
	option := transferCandidate{Player: *starter, Kind: candidateBuyOption, SellerTeamId: 9, BuyOptionFee: common.MarketValue(*starter) * 3}
	state := stateOf(adultSquad(), 500_000_000)
	state.LoanedIn = []*common.Player{starter}
	if ranked := rankCandidates(profile, state, []transferCandidate{option}, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Errorf("a buy option at 3x the valuation must not be exercised, got %+v", ranked)
	}
}

func TestRankCandidates_CaretakerOnlyBuysToFillAHole(t *testing.T) {
	profile := exact(common.AiTransferProfileFor(common.AiLevelCaretaker))
	star := midfielder(500, 99)
	star.Age = 27
	candidates := []transferCandidate{listedCandidate(1, star)}

	// A full squad with every position covered: even a clear upgrade isn't bought.
	if ranked := rankCandidates(profile, stateOf(adultSquad(), 50_000_000), candidates, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Errorf("a Caretaker with no hole to fill should sign nobody, got %+v", ranked)
	}

	// Down to two midfielders (under the minimum of three): a midfielder is signed, a striker isn't.
	short := make([]*common.Player, 0)
	midfielders := 0
	for _, p := range adultSquad() {
		if common.BestCorePosition(*p) == common.MIDFIELD {
			if midfielders == 2 {
				continue
			}
			midfielders++
		}
		short = append(short, p)
	}
	for len(short) < common.SquadSize {
		extra := defender(int64(200+len(short)), 20)
		extra.Age = 28
		short = append(short, extra)
	}
	forward := striker(501, 99)
	forward.Age = 27
	ranked := rankCandidates(profile, stateOf(short, 50_000_000), []transferCandidate{listedCandidate(1, star), listedCandidate(2, forward)},
		map[int64]bool{}, rngSeed(1))
	if len(ranked) != 1 || ranked[0].Player.Id != 500 {
		t.Errorf("a Caretaker short of midfielders should sign only the midfielder, got %+v", ranked)
	}

	// Short of bodies (under SquadSize): any position will do, but not at a loss.
	small := adultSquad()[:common.SquadSize-1]
	filler := defender(502, 20)
	filler.Age = 27
	free := transferCandidate{Player: *filler, Kind: candidateFreeAgent}
	if ranked := rankCandidates(profile, stateOf(small, 50_000_000), []transferCandidate{free}, map[int64]bool{}, rngSeed(1)); len(ranked) != 1 {
		t.Errorf("a Caretaker under the squad size should sign a free agent, got %d", len(ranked))
	}
	dear := listedCandidate(3, filler)
	dear.AskingPrice = common.MarketValue(*filler) * 14 / 10
	if ranked := rankCandidates(profile, stateOf(small, 50_000_000), []transferCandidate{dear}, map[int64]bool{}, rngSeed(1)); len(ranked) != 0 {
		t.Errorf("a filler who adds nothing isn't worth paying over the odds for, got %+v", ranked)
	}
}
