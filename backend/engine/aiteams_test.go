package engine

import (
	"math/rand"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

func goalkeeper(id int64, skill int) *common.Player { return &common.Player{Id: id, GoalKeeper: skill} }
func defender(id int64, skill int) *common.Player   { return &common.Player{Id: id, Defense: skill} }
func midfielder(id int64, skill int) *common.Player {
	return &common.Player{Id: id, Midfield: skill, Pass: skill}
}
func striker(id int64, skill int) *common.Player {
	return &common.Player{Id: id, Shot: skill, Pass: skill, Dribbling: skill}
}

// squadOf builds a squad with the given number of players at each core position, ids assigned
// sequentially from 1 and skill descending within a position (so the last one is the weakest).
func squadOf(goalkeepers, defenders, midfielders, strikers int) []*common.Player {
	players := make([]*common.Player, 0)
	id := int64(1)
	add := func(n int, build func(int64, int) *common.Player) {
		for i := 0; i < n; i++ {
			players = append(players, build(id, 90-i*5))
			id++
		}
	}
	add(goalkeepers, goalkeeper)
	add(defenders, defender)
	add(midfielders, midfielder)
	add(strikers, striker)
	return players
}

func none() map[int64]bool { return map[int64]bool{} }

func TestPickSurplusPlayerToList_ListsAtStartingSquadSize(t *testing.T) {
	// 18 players, exactly common.SquadSize, with 6 defenders - the old "> SquadSize" rule never
	// listed anyone here.
	squad := squadOf(2, 6, 5, 5)
	got := pickSurplusPlayerToList(squad, none(), none(), none())
	if got == nil {
		t.Fatal("expected a surplus defender to be listed from a starting-size squad")
	}
	if want := squad[7]; got.Id != want.Id {
		t.Errorf("listed player %d, want the weakest defender %d", got.Id, want.Id)
	}
}

func TestPickSurplusPlayerToList_NoSurplusNoListing(t *testing.T) {
	squad := squadOf(3, 4, 4, 4) // every position exactly min+1
	if got := pickSurplusPlayerToList(squad, none(), none(), none()); got != nil {
		t.Errorf("listed player %d, want nobody without a surplus beyond surplusDepthMargin", got.Id)
	}
}

func TestPickSurplusPlayerToList_RespectsMinSquadSizeAfterSale(t *testing.T) {
	squad := squadOf(2, 6, 4, 4) // 16 players: selling one would drop below minSquadSizeAfterSale
	if got := pickSurplusPlayerToList(squad, none(), none(), none()); got != nil {
		t.Errorf("listed player %d, want nobody when the sale would leave fewer than %d", got.Id, minSquadSizeAfterSale)
	}
}

func TestPickSurplusPlayerToList_SkipsLoanListedPlayer(t *testing.T) {
	squad := squadOf(2, 6, 5, 5)
	weakestDefender := squad[7]
	got := pickSurplusPlayerToList(squad, none(), map[int64]bool{weakestDefender.Id: true}, none())
	if got == nil {
		t.Fatal("expected another defender to be listed instead of the loan-listed one")
	}
	if got.Id == weakestDefender.Id {
		t.Error("listed a player who is already loan-listed")
	}
	if want := squad[6]; got.Id != want.Id {
		t.Errorf("listed player %d, want the next weakest defender %d", got.Id, want.Id)
	}
}

func TestPickSurplusPlayerToList_ListingsAccumulateWhileSurplusLasts(t *testing.T) {
	squad := squadOf(2, 7, 5, 5) // 19 players, defense surplus of 4
	listed := none()
	for i := 0; i < 2; i++ {
		got := pickSurplusPlayerToList(squad, listed, none(), none())
		if got == nil {
			t.Fatalf("round %d: expected another defender to be listed", i)
		}
		if listed[got.Id] {
			t.Fatalf("round %d: re-picked already-listed player %d", i, got.Id)
		}
		listed[got.Id] = true
	}
	// Defense now has 5 unlisted (min+2) but 17 unlisted overall: one more is still allowed...
	if got := pickSurplusPlayerToList(squad, listed, none(), none()); got == nil {
		t.Fatal("expected a third listing while defense still has min+2 unlisted")
	} else {
		listed[got.Id] = true
	}
	// ...but then defense is at min+1 and the squad at 16, so it stops.
	if got := pickSurplusPlayerToList(squad, listed, none(), none()); got != nil {
		t.Errorf("listed player %d, want listing to stop once the surplus is used up", got.Id)
	}
}

func TestOwnedPlayers_ExcludesLoanedIn(t *testing.T) {
	squad := squadOf(2, 6, 5, 5)
	loanedIn := map[int64]bool{squad[7].Id: true}
	owned := ownedPlayers(squad, loanedIn)
	if len(owned) != len(squad)-1 {
		t.Fatalf("got %d owned players, want %d", len(owned), len(squad)-1)
	}
	// With the loaned-in defender gone, the weakest *owned* defender is the one listed.
	got := pickSurplusPlayerToList(owned, none(), none(), none())
	if got == nil || got.Id != squad[6].Id {
		t.Errorf("expected the weakest owned defender %d to be listed, got %v", squad[6].Id, got)
	}
}

func TestPlanListingWithdrawals_KeepsGenuineSurplusListing(t *testing.T) {
	squad := squadOf(2, 6, 5, 5)
	listed := map[int64]bool{squad[7].Id: true}
	if got := planListingWithdrawals(squad, listed); len(got) != 0 {
		t.Errorf("withdrew %d listings, want none right after a valid listing", len(got))
	}
}

func TestPlanListingWithdrawals_WithdrawsWhenPositionTurnsThin(t *testing.T) {
	// A defender was listed, then other defenders left - only 2 unlisted defenders remain.
	squad := squadOf(3, 3, 6, 6)
	listed := map[int64]bool{squad[5].Id: true}
	got := planListingWithdrawals(squad, listed)
	if len(got) != 1 || got[0].Id != squad[5].Id {
		t.Errorf("expected the listed defender %d to be withdrawn, got %v", squad[5].Id, got)
	}
}

func TestPlanListingWithdrawals_WithdrawsOnlyEnoughWhenSquadTooSmall(t *testing.T) {
	squad := squadOf(2, 7, 4, 4) // 17 players
	listed := map[int64]bool{squad[7].Id: true, squad[8].Id: true}
	// 15 unlisted < minSquadSizeAfterSale (16): exactly one listing should come back.
	if got := planListingWithdrawals(squad, listed); len(got) != 1 {
		t.Errorf("withdrew %d listings, want exactly 1", len(got))
	}
}

func TestIsAiManagedTeam(t *testing.T) {
	tests := []struct {
		name    string
		ownerId int64
		aiLevel int
		want    bool
	}{
		{"unowned Caretaker", 0, common.AiLevelCaretaker, true},
		{"unowned Director", 0, common.AiLevelDirector, true},
		{"unowned but no AI level yet", 0, 0, false},
		{"owned, even with a leftover AI level", 42, common.AiLevelDirector, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := &common.Team{OwnerId: tt.ownerId, AiLevel: tt.aiLevel}
			if got := isAiManagedTeam(team); got != tt.want {
				t.Errorf("isAiManagedTeam(%+v) = %v, want %v", team, got, tt.want)
			}
		})
	}
}

func TestFilterAiManagedTeams(t *testing.T) {
	teams := []*common.Team{
		{Id: 1, OwnerId: 0, AiLevel: common.AiLevelManager},
		{Id: 2, OwnerId: 7, AiLevel: common.AiLevelDirector}, // owned - excluded
		{Id: 3, OwnerId: 0, AiLevel: 0},                      // no AI level yet - excluded
		{Id: 4, OwnerId: 0, AiLevel: common.AiLevelCaretaker},
	}
	got := filterAiManagedTeams(teams)
	if len(got) != 2 || got[0].Id != 1 || got[1].Id != 4 {
		t.Errorf("expected teams [1 4], got %v", got)
	}
}

// ProbAiLocalActivityPerTick must stay a valid probability (0,1] - a regression here (e.g. the
// activity window shrinking to zero width) would otherwise silently turn every roll into a
// division-by-zero NaN/Inf instead of failing loudly.
func TestProbAiLocalActivityPerTick_IsValidProbability(t *testing.T) {
	if ProbAiLocalActivityPerTick <= 0 || ProbAiLocalActivityPerTick > 1 {
		t.Fatalf("ProbAiLocalActivityPerTick = %v, want a value in (0, 1]", ProbAiLocalActivityPerTick)
	}
}

// awakeTeamIds must drop a team whose own country's local clock is outside the activity window
// at the given instant, even though the team is otherwise AI-managed and eligible - this is what
// stops RunAiTeamDailyActivity from matching an awake challenger into a friendly against an
// opponent whose own local time is the middle of the night (the gap the original timezone PR
// left open: only the challenger's own window was checked, never the candidate opponent's).
func TestAwakeTeamIds_ExcludesTeamsOutsideTheirOwnLocalWindow(t *testing.T) {
	// 03:00 UTC: night in Spain (Europe/Madrid, UTC+1/+2), afternoon in Japan (Asia/Tokyo, UTC+9).
	now := time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC)
	teams := []*common.Team{
		{Id: 1, Country: "es"},
		{Id: 2, Country: "jp"},
	}

	got := awakeTeamIds(now, teams)

	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("expected only the Japanese team (id 2), got %v", got)
	}
}

func TestPickRandomOpponent_NeverReturnsExcludedTeam(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	teamIds := []int64{10, 20, 30}
	for i := 0; i < 50; i++ {
		got, ok := pickRandomOpponent(r, teamIds, 20)
		if !ok {
			t.Fatalf("expected a candidate, got none")
		}
		if got == 20 {
			t.Fatalf("pickRandomOpponent returned the excluded team")
		}
	}
}

func TestPickRandomOpponent_FalseWhenOnlyExcludedCandidate(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	if _, ok := pickRandomOpponent(r, []int64{5}, 5); ok {
		t.Fatalf("expected no candidate when the only team is the excluded one")
	}
	if _, ok := pickRandomOpponent(r, []int64{}, 5); ok {
		t.Fatalf("expected no candidate for an empty team list")
	}
}

func TestExcludeTeamIds_DropsOnlyTeamsMarkedExcluded(t *testing.T) {
	got := excludeTeamIds([]int64{1, 2, 3, 4}, map[int64]bool{2: true, 4: true})
	want := []int64{1, 3}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestExcludeTeamIds_NoneExcludedReturnsAllUnchanged(t *testing.T) {
	ids := []int64{7, 8, 9}
	got := excludeTeamIds(ids, map[int64]bool{})
	if len(got) != len(ids) {
		t.Fatalf("expected %v, got %v", ids, got)
	}
	for i := range ids {
		if got[i] != ids[i] {
			t.Fatalf("expected %v, got %v", ids, got)
		}
	}
}

func TestPickSurplusPlayerToList_SkipsProtectedPlayers(t *testing.T) {
	squad := squadOf(2, 6, 5, 5)
	weakestDefender := squad[7]
	got := pickSurplusPlayerToList(squad, none(), none(), map[int64]bool{weakestDefender.Id: true})
	if got == nil || got.Id == weakestDefender.Id {
		t.Fatalf("listed %v, want the next weakest defender instead of the protected one", got)
	}
	if want := squad[6]; got.Id != want.Id {
		t.Errorf("listed player %d, want %d", got.Id, want.Id)
	}
}

// A CPU seller counters a near miss instead of rejecting it (openspec/specs/cpu-team-management,
// "Answering transfer offers").
func TestCpuCounterAmount(t *testing.T) {
	below := func(threshold int64) common.OfferDecision {
		return common.OfferDecision{Threshold: threshold, Reason: common.OfferReasonBelowValue}
	}
	cases := []struct {
		name     string
		level    int
		decision common.OfferDecision
		amount   int64
		want     int64
		ok       bool
	}{
		{"near-miss offer to a Director", common.AiLevelDirector, below(1234500), 1000000, 1235000, true},
		{"key player at exactly 70%", common.AiLevelManager, common.OfferDecision{Threshold: 2000000, Reason: common.OfferReasonKeyPlayer}, 1400000, 2000000, true},
		{"international near miss", common.AiLevelDirector, common.OfferDecision{Threshold: 1500001, Reason: common.OfferReasonInternational}, 1200000, 1501000, true},
		{"low offer to a Manager", common.AiLevelManager, below(1000000), 699999, 0, false},
		{"caretaker near miss", common.AiLevelCaretaker, below(1000000), 900000, 0, false},
		{"thin position near miss", common.AiLevelDirector, common.OfferDecision{Threshold: 1000000, Reason: common.OfferReasonThinPosition}, 950000, 0, false},
		{"squad too small near miss", common.AiLevelDirector, common.OfferDecision{Threshold: 1000000, Reason: common.OfferReasonSquadTooSmall}, 950000, 0, false},
		{"accepted offer", common.AiLevelDirector, common.OfferDecision{Accept: true, Threshold: 1000000, Reason: common.OfferReasonAccepted}, 1000000, 0, false},
	}
	for _, c := range cases {
		got, ok := cpuCounterAmount(c.level, c.decision, c.amount)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: cpuCounterAmount = %d, %v; want %d, %v", c.name, got, ok, c.want, c.ok)
		}
	}
}

// A CPU buyer takes a counter up to 1.5 times its valuation, if it can afford it
// (openspec/specs/cpu-transfer-market, "Answering counter-offers").
func TestAcceptsCounter(t *testing.T) {
	valued := common.TargetScore{Value: 1000000, Affordable: true}
	if !acceptsCounter(valued, 1300000) {
		t.Error("a 1,300,000 counter on a 1,000,000 valuation should be accepted")
	}
	if !acceptsCounter(valued, 1500000) {
		t.Error("a counter at exactly 1.5 times the valuation should be accepted")
	}
	if acceptsCounter(valued, 1600000) {
		t.Error("a 1,600,000 counter on a 1,000,000 valuation should be declined")
	}
	if acceptsCounter(common.TargetScore{Value: 1000000}, 1300000) {
		t.Error("an unaffordable counter should be declined")
	}
	// After declining, the original 1,100,000 bid is the rejection: 3 days of cooldown, then
	// 1,210,000 or more.
	declined := common.OfferRejection{Amount: 1100000, RejectedAt: time.Now()}
	if allowed, _ := common.OfferRetryAllowed(&declined, time.Now().Add(71*time.Hour)); allowed {
		t.Error("rebid allowed during the cooldown")
	}
	if allowed, minBid := common.OfferRetryAllowed(&declined, time.Now().Add(73*time.Hour)); !allowed || minBid != 1210000 {
		t.Errorf("after the cooldown: allowed %v, min bid %d; want true, 1,210,000", allowed, minBid)
	}
}

func TestContractsToRenew_ManagerRenewsEveryExpiringContract(t *testing.T) {
	squad := squadOf(2, 5, 6, 5)
	squad[0].ContractYearsRemaining, squad[3].ContractYearsRemaining, squad[10].ContractYearsRemaining = 1, 0, 3
	for _, level := range []int{common.AiLevelManager, common.AiLevelDirector} {
		got := contractsToRenew(level, squad, nil)
		if len(got) != 17 {
			t.Errorf("level %d renews %d players, want the 17 with one season or less left", level, len(got))
		}
	}
}

func TestContractsToRenew_CaretakerRenewsOnlyExpiringStarters(t *testing.T) {
	squad := squadOf(2, 5, 6, 5)
	for _, p := range squad {
		p.ContractYearsRemaining = 1
	}
	squad[2].ContractYearsRemaining = 4 // a starter who isn't due yet
	starters := map[int64]bool{squad[0].Id: true, squad[2].Id: true, squad[5].Id: true}
	got := contractsToRenew(common.AiLevelCaretaker, squad, starters)
	if len(got) != 2 || got[0].Id != squad[0].Id || got[1].Id != squad[5].Id {
		t.Errorf("caretaker renews %v, want only the expiring starters %d and %d", got, squad[0].Id, squad[5].Id)
	}
}
