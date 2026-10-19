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

	if got := pickNextTarget([]scoredCandidate{unlisted, listed}, state, 0); got == nil || got.Kind != candidateListed {
		t.Errorf("with no offers left the pass must skip to the listed target, got %v", got)
	}
	state.OpenOffers = common.AiMaxOpenOffersPerTeam
	if got := pickNextTarget([]scoredCandidate{unlisted}, state, 2); got != nil {
		t.Error("a team with the maximum offers out starts no more")
	}

	full := stateOf(adultSquad(), 50_000_000)
	for i := len(full.Squad); i < common.MaxAiSquadSize; i++ {
		full.Squad = append(full.Squad, defender(int64(300+i), 20))
	}
	if got := pickNextTarget([]scoredCandidate{listed}, full, 2); got != nil {
		t.Error("a squad at its ceiling signs nobody")
	}
	full.Listed[full.Squad[len(full.Squad)-1].Id] = true
	if got := pickNextTarget([]scoredCandidate{listed}, full, 2); got == nil {
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
