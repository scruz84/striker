package common

import (
	"math/rand"
	"testing"
)

// midfielderAged builds a midfielder whose MIDFIELD and PASS (the skills GetMidfieldValue reads)
// are both skill; other skills stay zero.
func midfielderAged(id int64, skill int, age int) *Player {
	return &Player{Id: id, Midfield: skill, Pass: skill, Age: age}
}

// settledSquad is fullSquad with every starter given an age - four 28-year-olds per outfield
// line, so the best XI counts as consolidated.
func settledSquad() []*Player {
	squad := fullSquad()
	for _, p := range squad {
		p.Age = 28
	}
	// Pad to a full SquadSize with weak outfield reserves (ids 200+), so the squad sits above
	// MinSquadSizeAfterSale and no starter's position depends on them.
	builders := []func(int64, int) *Player{
		func(id int64, skill int) *Player { return strongDefender(id).withSkill(skill) },
		func(id int64, skill int) *Player { return strongMidfielder(id).withSkill(skill) },
		func(id int64, skill int) *Player { return strongStriker(id).withSkill(skill) },
	}
	for i := 0; len(squad) < SquadSize; i++ {
		p := builders[i%len(builders)](int64(200+i), 20)
		p.Age = 28
		squad = append(squad, p)
	}
	return squad
}

// withSkill rescales every non-zero skill of a strong* test player to skill.
func (p *Player) withSkill(skill int) *Player {
	for _, f := range []*int{&p.GoalKeeper, &p.Defense, &p.Pass, &p.Midfield, &p.Dribbling, &p.Shot} {
		if *f != 0 {
			*f = skill
		}
	}
	return p
}

func newRng(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

func TestAiTransferProfileFor_EachLevelIncludesTheOnesBelow(t *testing.T) {
	caretaker := AiTransferProfileFor(AiLevelCaretaker)
	manager := AiTransferProfileFor(AiLevelManager)
	director := AiTransferProfileFor(AiLevelDirector)

	if caretaker.MovesPerPass != 0 || manager.MovesPerPass != 1 || director.MovesPerPass < 2 || director.MovesPerPass > 3 {
		t.Errorf("moves per pass = %d/%d/%d, want 0/1/2-3", caretaker.MovesPerPass, manager.MovesPerPass, director.MovesPerPass)
	}
	if !(caretaker.ValuationNoise > manager.ValuationNoise && manager.ValuationNoise > director.ValuationNoise) {
		t.Errorf("valuation noise should shrink with level: %v %v %v", caretaker.ValuationNoise, manager.ValuationNoise, director.ValuationNoise)
	}
	if caretaker.EstimatesPotential || !manager.EstimatesPotential || !director.EstimatesPotential {
		t.Error("only Manager and above estimate potential")
	}
	if manager.PotentialNoise <= director.PotentialNoise {
		t.Errorf("a Manager's read of potential (%v) should be noisier than a Director's (%v)", manager.PotentialNoise, director.PotentialNoise)
	}
	if caretaker.ProtectsKeyPlayers || !manager.ProtectsKeyPlayers || !director.ProtectsKeyPlayers {
		t.Error("only Manager and above protect key players")
	}
	if caretaker.SearchesFreeAgents || caretaker.SearchesTransferList || caretaker.SearchesUnlisted {
		t.Error("a Caretaker never shops")
	}
	if !manager.SearchesFreeAgents || !manager.SearchesTransferList || manager.SearchesUnlisted {
		t.Error("a Manager shops free agents and the transfer list only")
	}
	if !director.SearchesFreeAgents || !director.SearchesTransferList || !director.SearchesUnlisted {
		t.Error("a Director shops the whole market")
	}
	if manager.SignsProjects || !director.SignsProjects || manager.SellsToBuy || !director.SellsToBuy {
		t.Error("projects and selling-to-buy are Director-only")
	}
	if !(manager.BudgetReserve > director.BudgetReserve) || manager.ConsidersWageBill || !director.ConsidersWageBill {
		t.Error("a Manager keeps a wider reserve; only a Director watches payroll")
	}
	if got := AiTransferProfileFor(0); got.Level != AiLevelCaretaker {
		t.Errorf("unknown level got profile %d, want the Caretaker one", got.Level)
	}
}

func TestMarketValue_FallsWithAge(t *testing.T) {
	young := Player{Midfield: 70, Pass: 70, Age: 21}
	prime := young
	prime.Age = 26
	veteran := young
	veteran.Age = 35
	if !(MarketValue(young) > MarketValue(prime) && MarketValue(prime) > MarketValue(veteran)) {
		t.Errorf("values young/prime/veteran = %d/%d/%d, want strictly decreasing", MarketValue(young), MarketValue(prime), MarketValue(veteran))
	}
}

func TestListingAskingPrice_InternationalPremiumOnlyWhenProfileHasOne(t *testing.T) {
	p := Player{Midfield: 70, Pass: 70, Age: 26}
	base := MarketValue(p)
	if got := ListingAskingPrice(p, AiTransferProfileFor(AiLevelManager), true, 10); got != base {
		t.Errorf("manager asks %d for an international, want the plain %d", got, base)
	}
	if got := ListingAskingPrice(p, AiTransferProfileFor(AiLevelDirector), true, 0); got <= base {
		t.Errorf("director asks %d for an international, want more than %d", got, base)
	}
	if got := ListingAskingPrice(p, AiTransferProfileFor(AiLevelDirector), false, 0); got != base {
		t.Errorf("director asks %d for a domestic player, want %d", got, base)
	}
}

func TestProjectPlayer_YoungPlayerGrowsButNotPastEstimatedCeiling(t *testing.T) {
	director := AiTransferProfileFor(AiLevelDirector)
	p := Player{Id: 1, Midfield: 50, Pass: 50, Age: 19}
	caps := &PlayerSkillCaps{MidfieldMax: 60, PassMax: 90}
	got := ProjectPlayer(p, caps, director, newRng(1))
	if got.Midfield <= p.Midfield || got.Midfield > 60+2 {
		t.Errorf("midfield projected to %d, want growth capped near the 60 ceiling", got.Midfield)
	}
	if got.Pass <= got.Midfield {
		t.Errorf("pass %d should grow further than the capped midfield %d", got.Pass, got.Midfield)
	}
}

func TestProjectPlayer_NoProjectionForOldPlayersOrCaretakers(t *testing.T) {
	old := Player{Midfield: 50, Pass: 50, Age: 30}
	if got := ProjectPlayer(old, nil, AiTransferProfileFor(AiLevelDirector), newRng(1)); got != old {
		t.Error("a 30-year-old projects to himself")
	}
	young := Player{Midfield: 50, Pass: 50, Age: 19}
	if got := ProjectPlayer(young, nil, AiTransferProfileFor(AiLevelCaretaker), newRng(1)); got != young {
		t.Error("a Caretaker doesn't try to project potential")
	}
}

func TestPerceivedValue_CaretakerMissesAWonderkid(t *testing.T) {
	wonderkid := Player{Midfield: 40, Pass: 40, Age: 18}
	caps := &PlayerSkillCaps{MidfieldMax: 90, PassMax: 90}
	var caretaker, director int64
	const runs = 200
	for i := int64(0); i < runs; i++ {
		caretaker += PerceivedValue(wonderkid, caps, AiTransferProfileFor(AiLevelCaretaker), newRng(i))
		director += PerceivedValue(wonderkid, caps, AiTransferProfileFor(AiLevelDirector), newRng(i))
	}
	if director <= caretaker {
		t.Errorf("director values the wonderkid at %d on average, caretaker at %d - want the director to see more", director/runs, caretaker/runs)
	}
}

func TestIsKeyPlayer(t *testing.T) {
	squad := settledSquad() // 2 GK, 4 D, 4 M, 4 S - every position at or above its minimum depth
	if IsKeyPlayer(squad, 20) {
		t.Error("the backup goalkeeper is not a key player")
	}
	// Losing the only starting goalkeeper leaves the second one, equally good - strength holds
	// and the position keeps one keeper short of MinPositionDepth, so he is key.
	if !IsKeyPlayer(squad, 1) {
		t.Error("a goalkeeper whose sale leaves the position under minimum depth is key")
	}
	if IsKeyPlayer(squad, 999) {
		t.Error("an unknown player id is not key")
	}
}

func TestIsKeyPlayer_StarThatCannotBeReplaced(t *testing.T) {
	// A 99 goalkeeper with two 60 deputies: the position keeps its minimum depth without him,
	// but the XI loses 39 points.
	squad := settledSquad()
	star := strongGoalkeeper(50)
	star.GoalKeeper = 99
	squad = append(squad, star)
	for _, p := range squad {
		if p.GoalKeeper == 90 {
			p.GoalKeeper = 60
		}
	}
	if !IsKeyPlayer(squad, star.Id) {
		t.Errorf("a starter worth far more than his replacement should be key (drop threshold %d)", keyPlayerStrengthDrop)
	}
	if IsKeyPlayer(squad, 20) {
		t.Error("his deputy is not")
	}
}

func offerContext(squad []*Player, p *Player, amount int64) OfferContext {
	return OfferContext{Squad: squad, Player: *p, Amount: amount}
}

func TestEvaluateIncomingOffer_AcceptsAtOrAboveValue(t *testing.T) {
	squad := settledSquad()
	benchwarmer := midfielderAged(60, 50, 26)
	squad = append(squad, benchwarmer)
	profile := AiTransferProfileFor(AiLevelDirector)
	value := PerceivedValue(*benchwarmer, nil, profile, newRng(7))

	if d := EvaluateIncomingOffer(profile, offerContext(squad, benchwarmer, value), newRng(7)); !d.Accept {
		t.Errorf("rejected an offer equal to the valuation (%+v)", d)
	}
	if d := EvaluateIncomingOffer(profile, offerContext(squad, benchwarmer, value/2), newRng(7)); d.Accept || d.Reason != OfferReasonBelowValue {
		t.Errorf("accepted a lowball (%+v)", d)
	}
}

func TestEvaluateIncomingOffer_ManagerProtectsKeyPlayerButCaretakerDoesNot(t *testing.T) {
	squad := settledSquad()
	key := squad[10]                               // a striker (id 11)...
	key.Shot, key.Pass, key.Dribbling = 99, 99, 99 // ...far above the 90s who would replace him
	offer := MarketValue(*key) * 3 / 2

	caretaker := EvaluateIncomingOffer(AiTransferProfileFor(AiLevelCaretaker), offerContext(squad, key, offer), newRng(3))
	manager := EvaluateIncomingOffer(AiTransferProfileFor(AiLevelManager), offerContext(squad, key, offer), newRng(3))
	if !caretaker.Accept {
		t.Errorf("caretaker should take 1.5x value for anyone: %+v", caretaker)
	}
	if manager.Accept || manager.Reason != OfferReasonKeyPlayer {
		t.Errorf("manager should refuse 1.5x for a key starter: %+v", manager)
	}
	// ...but a bid that clears the protection multiplier does buy him.
	huge := MarketValue(*key) * 4
	if d := EvaluateIncomingOffer(AiTransferProfileFor(AiLevelManager), offerContext(squad, key, huge), newRng(3)); !d.Accept {
		t.Errorf("manager should sell a key starter for 4x value: %+v", d)
	}
}

func TestEvaluateIncomingOffer_NeverLeavesAPositionUnderMinimumDepth(t *testing.T) {
	squad := settledSquad() // two goalkeepers: MinPositionDepth[GOALKEEPER] is 2
	keeper := squad[0]
	for _, level := range []int{AiLevelCaretaker, AiLevelManager, AiLevelDirector} {
		d := EvaluateIncomingOffer(AiTransferProfileFor(level), offerContext(squad, keeper, 1<<40), newRng(1))
		if d.Accept || d.Reason != OfferReasonThinPosition {
			t.Errorf("level %d sold a goalkeeper from a two-keeper squad: %+v", level, d)
		}
	}
}

func TestEvaluateIncomingOffer_DirectorAsksMoreForInternationals(t *testing.T) {
	squad := settledSquad()
	bench := midfielderAged(60, 55, 26)
	squad = append(squad, bench)
	profile := AiTransferProfileFor(AiLevelDirector)

	domestic := EvaluateIncomingOffer(profile, offerContext(squad, bench, 0), newRng(5))
	ctx := offerContext(squad, bench, 0)
	ctx.International = true
	international := EvaluateIncomingOffer(profile, ctx, newRng(5))
	if international.Threshold <= domestic.Threshold {
		t.Errorf("international threshold %d, want above the domestic %d", international.Threshold, domestic.Threshold)
	}
	if manager := EvaluateIncomingOffer(AiTransferProfileFor(AiLevelManager), ctx, newRng(5)); manager.Reason == OfferReasonInternational {
		t.Error("a Manager does not charge an international premium")
	}
}

func TestEvaluateIncomingOffer_RefusesToShrinkSquadBelowMinimum(t *testing.T) {
	squad := settledSquad()[:MinSquadSizeAfterSale] // exactly the floor: one sale breaks it
	player := squad[len(squad)-1]
	d := EvaluateIncomingOffer(AiTransferProfileFor(AiLevelCaretaker), offerContext(squad, player, 1<<40), newRng(1))
	if d.Accept || d.Reason != OfferReasonSquadTooSmall {
		t.Errorf("sold a player from a squad at the minimum size: %+v", d)
	}
}

func scoreContext(squad []*Player, target *Player) TargetContext {
	return TargetContext{Squad: squad, Target: *target, Price: MarketValue(*target), Budget: 10_000_000, WageBill: 5_000}
}

func TestScoreTarget_BetterPlayerScoresHigher(t *testing.T) {
	squad := settledSquad()
	profile := AiTransferProfileFor(AiLevelDirector)
	good := midfielderAged(70, 95, 27)
	mediocre := midfielderAged(71, 60, 27)

	g := ScoreTarget(profile, scoreContext(squad, good), newRng(1))
	m := ScoreTarget(profile, scoreContext(squad, mediocre), newRng(1))
	if !g.Affordable || !m.Affordable {
		t.Fatal("candidates should be affordable")
	}
	if g.Improvement <= 0 {
		t.Errorf("a clear upgrade improves the XI, got %v", g.Improvement)
	}
	if m.Improvement > 0 && m.Improvement >= g.Improvement {
		t.Errorf("mediocre improvement %v should be below the good one %v", m.Improvement, g.Improvement)
	}
	if g.Total <= m.Total {
		t.Errorf("good player scored %v, mediocre %v", g.Total, m.Total)
	}
}

func TestScoreTarget_BenchPlayerAddsNothing(t *testing.T) {
	squad := settledSquad()
	weak := midfielderAged(70, 30, 27)
	s := ScoreTarget(AiTransferProfileFor(AiLevelDirector), scoreContext(squad, weak), newRng(1))
	if s.Improvement != 0 {
		t.Errorf("a player weaker than every starter improves nothing, got %v", s.Improvement)
	}
	if s.Total >= AiTransferProfileFor(AiLevelDirector).MinScore {
		t.Errorf("a bench filler scored %v, should stay under the action threshold", s.Total)
	}
}

func TestScoreTarget_FillingAThinPositionIsWorthIt(t *testing.T) {
	// Keep only two strikers (MinPositionDepth is 3), then offer a weak third one.
	squad := make([]*Player, 0)
	strikers := 0
	for _, p := range settledSquad() {
		if BestCorePosition(*p) == STRIKER {
			if strikers == 2 {
				continue
			}
			strikers++
		}
		squad = append(squad, p)
	}
	weakStriker := striker(80, 30, 24)
	s := ScoreTarget(AiTransferProfileFor(AiLevelManager), scoreContext(squad, weakStriker), newRng(1))
	if s.Improvement < thinPositionBonus {
		t.Errorf("improvement %v should include the thin-position bonus %v", s.Improvement, thinPositionBonus)
	}
}

func striker(id int64, skill int, age int) *Player {
	return &Player{Id: id, Shot: skill, Pass: skill, Dribbling: skill, Age: age}
}

func TestScoreTarget_UnaffordableWhenBudgetReserveBreached(t *testing.T) {
	squad := settledSquad()
	target := midfielderAged(70, 95, 27)
	ctx := scoreContext(squad, target)
	ctx.Price = 6_000_000 // 60% of a 10M budget

	if s := ScoreTarget(AiTransferProfileFor(AiLevelManager), ctx, newRng(1)); s.Affordable {
		t.Error("a Manager keeps half its budget in reserve and can't spend 60% of it")
	}
	if s := ScoreTarget(AiTransferProfileFor(AiLevelDirector), ctx, newRng(1)); !s.Affordable {
		t.Error("a Director's tighter reserve allows spending 60% of the budget")
	}
	ctx.Price = 11_000_000
	if s := ScoreTarget(AiTransferProfileFor(AiLevelDirector), ctx, newRng(1)); s.Affordable {
		t.Error("nobody can spend more than the whole budget")
	}
}

func TestScoreTarget_DirectorWatchesPayroll(t *testing.T) {
	squad := settledSquad()
	target := midfielderAged(70, 95, 27)
	ctx := scoreContext(squad, target)
	ctx.Price = 0
	ctx.Budget = 100_000
	ctx.WageBill = 10_000 // 20 rounds of it alone would need 200k
	if s := ScoreTarget(AiTransferProfileFor(AiLevelDirector), ctx, newRng(1)); s.Affordable {
		t.Error("a Director whose budget can't carry the payroll shouldn't sign anyone")
	}
	if s := ScoreTarget(AiTransferProfileFor(AiLevelManager), ctx, newRng(1)); !s.Affordable {
		t.Error("a Manager doesn't look at the wage bill")
	}
}

func TestScoreTarget_OnlyDirectorsSignProjects(t *testing.T) {
	squad := settledSquad()
	prospect := midfielderAged(70, 85, 18)
	caps := &PlayerSkillCaps{MidfieldMax: 99, PassMax: 99}
	ctx := scoreContext(squad, prospect)
	ctx.Caps = caps

	manager := ScoreTarget(AiTransferProfileFor(AiLevelManager), ctx, newRng(1))
	if manager.Projection != 0 || manager.Project {
		t.Errorf("a Manager doesn't sign for the future: %+v", manager)
	}
	director := ScoreTarget(AiTransferProfileFor(AiLevelDirector), ctx, newRng(1))
	if director.Projection <= 0 {
		t.Fatalf("a Director sees growth in an 18-year-old with a 95 ceiling: %+v", director)
	}
	ctx.ProjectsInUse = MaxProjects
	if full := ScoreTarget(AiTransferProfileFor(AiLevelDirector), ctx, newRng(1)); full.Projection != 0 {
		t.Error("a Director with all its project places taken signs no more")
	}
}

func TestScoreTarget_ProjectionWorthMoreToSettledTeams(t *testing.T) {
	prospect := midfielderAged(70, 85, 18)
	caps := &PlayerSkillCaps{MidfieldMax: 99, PassMax: 99}
	profile := AiTransferProfileFor(AiLevelDirector)

	settled := settledSquad()
	young := settledSquad()
	for _, p := range young {
		p.Age = 20
	}
	a := ScoreTarget(profile, TargetContext{Squad: settled, Target: *prospect, Caps: caps, Price: 1000, Budget: 10_000_000}, newRng(2))
	b := ScoreTarget(profile, TargetContext{Squad: young, Target: *prospect, Caps: caps, Price: 1000, Budget: 10_000_000}, newRng(2))
	if a.Projection <= b.Projection {
		t.Errorf("projection value for a settled squad (%v) should exceed an unsettled one (%v)", a.Projection, b.Projection)
	}
}

func TestScoreTarget_BargainBeatsOverpay(t *testing.T) {
	squad := settledSquad()
	target := midfielderAged(70, 90, 27)
	profile := AiTransferProfileFor(AiLevelDirector)
	cheap := scoreContext(squad, target)
	cheap.Price = MarketValue(*target) / 2
	dear := scoreContext(squad, target)
	dear.Price = MarketValue(*target) * 2
	if a, b := ScoreTarget(profile, cheap, newRng(1)), ScoreTarget(profile, dear, newRng(1)); a.Total <= b.Total {
		t.Errorf("cheap deal scored %v, overpay %v", a.Total, b.Total)
	}
}

func TestScoreTarget_IsDeterministicForTheSameRng(t *testing.T) {
	squad := settledSquad()
	ctx := scoreContext(squad, midfielderAged(70, 80, 21))
	profile := AiTransferProfileFor(AiLevelManager)
	if a, b := ScoreTarget(profile, ctx, newRng(9)), ScoreTarget(profile, ctx, newRng(9)); a != b {
		t.Errorf("same seed, different scores: %+v vs %+v", a, b)
	}
}

// TestScoreTarget_HigherLevelsPickBetterTargets is the profile test from the plan: with the same
// squad, budget and pool of candidates, on average a Director's pick adds more to the XI than a
// Manager's, and a Manager's more than a Caretaker's - the noise in the valuation is what
// separates them.
func TestScoreTarget_HigherLevelsPickBetterTargets(t *testing.T) {
	squad := settledSquad()
	pick := func(profile AiTransferProfile, seed int64, candidates []*Player) *Player {
		rng := newRng(seed)
		var best *Player
		bestTotal := -1e9
		for _, c := range candidates {
			s := ScoreTarget(profile, TargetContext{Squad: squad, Target: *c, Price: MarketValue(*c), Budget: 100_000_000}, rng)
			if s.Affordable && s.Total > bestTotal {
				best, bestTotal = c, s.Total
			}
		}
		return best
	}
	trueGain := func(c *Player) int {
		return SquadStrength(append(append([]*Player(nil), squad...), c)) - SquadStrength(squad)
	}

	totals := map[int]int{}
	const trials = 300
	for trial := int64(0); trial < trials; trial++ {
		gen := newRng(1000 + trial)
		candidates := make([]*Player, 0, 12)
		for i := 0; i < 12; i++ {
			skill := 55 + gen.Intn(40)
			candidates = append(candidates, midfielderAged(int64(100+i), skill, 24+gen.Intn(6)))
		}
		for _, level := range []int{AiLevelCaretaker, AiLevelManager, AiLevelDirector} {
			if chosen := pick(AiTransferProfileFor(level), trial, candidates); chosen != nil {
				totals[level] += trueGain(chosen)
			}
		}
	}
	if !(totals[AiLevelDirector] > totals[AiLevelManager] && totals[AiLevelManager] > totals[AiLevelCaretaker]) {
		t.Errorf("average true improvement of the pick: caretaker %d, manager %d, director %d - want strictly increasing",
			totals[AiLevelCaretaker]/trials, totals[AiLevelManager]/trials, totals[AiLevelDirector]/trials)
	}
}

func TestListingAskingPrice_PremiumGrowsWithCaps(t *testing.T) {
	p := Player{Midfield: 70, Pass: 70, Age: 26}
	director := AiTransferProfileFor(AiLevelDirector)
	none := ListingAskingPrice(p, director, true, 0)
	some := ListingAskingPrice(p, director, true, 10)
	many := ListingAskingPrice(p, director, true, internationalCapsCap)
	beyond := ListingAskingPrice(p, director, true, 5*internationalCapsCap)
	if !(MarketValue(p) < none && none < some && some < many) {
		t.Errorf("asking price should rise with caps: value %d, 0 caps %d, 10 caps %d, max caps %d", MarketValue(p), none, some, many)
	}
	if beyond != many {
		t.Errorf("caps stop counting at %d: %d vs %d", internationalCapsCap, beyond, many)
	}
}

func TestEvaluateIncomingOffer_InternationalPremiumGrowsWithCaps(t *testing.T) {
	squad := settledSquad()
	bench := midfielderAged(60, 55, 26)
	squad = append(squad, bench)
	profile := AiTransferProfileFor(AiLevelDirector)
	ctx := offerContext(squad, bench, 0)
	ctx.International = true
	rookie := EvaluateIncomingOffer(profile, ctx, newRng(5))
	ctx.InternationalCaps = 15
	veteran := EvaluateIncomingOffer(profile, ctx, newRng(5))
	if veteran.Threshold <= rookie.Threshold {
		t.Errorf("a capped international (%d) should cost more than an uncapped one (%d)", veteran.Threshold, rookie.Threshold)
	}
}

func TestScoreTarget_InternationalsEarnAReputationBonusOnlyForDirectors(t *testing.T) {
	squad := settledSquad()
	target := midfielderAged(70, 90, 27)
	ctx := scoreContext(squad, target)
	plain := ScoreTarget(AiTransferProfileFor(AiLevelDirector), ctx, newRng(1))

	ctx.International = true
	ctx.InternationalCaps = 10
	capped := ScoreTarget(AiTransferProfileFor(AiLevelDirector), ctx, newRng(1))
	if plain.Reputation != 0 {
		t.Errorf("a domestic player earns no reputation bonus, got %v", plain.Reputation)
	}
	if want := internationalBonusBase + 10*internationalBonusPerCap; capped.Reputation != want {
		t.Errorf("reputation = %v, want %v", capped.Reputation, want)
	}
	if capped.Total != plain.Total+capped.Reputation {
		t.Errorf("total %v should include the bonus on top of %v", capped.Total, plain.Total)
	}
	if manager := ScoreTarget(AiTransferProfileFor(AiLevelManager), ctx, newRng(1)); manager.Reputation != 0 {
		t.Errorf("a Manager doesn't value internationals, got %v", manager.Reputation)
	}
	ctx.InternationalCaps = 500
	if huge := ScoreTarget(AiTransferProfileFor(AiLevelDirector), ctx, newRng(1)); huge.Reputation != internationalBonusBase+internationalCapsCap*internationalBonusPerCap {
		t.Errorf("caps stop counting at %d, got reputation %v", internationalCapsCap, huge.Reputation)
	}
}
