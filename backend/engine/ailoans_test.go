package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

var loanTestNow = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

// loanSquad is adultSquad (18 players, 28 years old) with every contract running 3 more seasons -
// long enough to outlive a 1-season loan.
func loanSquad() []*common.Player {
	squad := adultSquad()
	for _, p := range squad {
		p.ContractYearsRemaining = 3
	}
	return squad
}

// listedFor is a transfer listing of p that has been up for age, at his market value.
func listedFor(p *common.Player, age time.Duration) map[int64]common.TeamTransferListing {
	return map[int64]common.TeamTransferListing{
		p.Id: {PlayerId: p.Id, AskingPrice: common.MarketValue(*p), ListedAt: loanTestNow.Add(-age)},
	}
}

func loanInputs(squad []*common.Player, listings map[int64]common.TeamTransferListing) loanOutInputs {
	return loanOutInputs{
		Owned: squad, TransferListings: listings, LoanListed: map[int64]bool{}, LoanedOut: map[int64]bool{},
		Excluded: map[int64]bool{}, Now: loanTestNow,
	}
}

// weakestDefender is the surplus defender of loanSquad (id 100+, skill 20).
func weakestDefender(squad []*common.Player) *common.Player {
	for _, p := range squad {
		if p.Id >= 100 {
			return p
		}
	}
	return nil
}

func TestPickPlayerToLoanOut_UnsoldSurplusBecomesASurplusLoan(t *testing.T) {
	squad := loanSquad()
	surplus := weakestDefender(squad)
	in := loanInputs(squad, listedFor(surplus, 8*24*time.Hour))

	plan := pickPlayerToLoanOut(in)
	if plan == nil || plan.Player.Id != surplus.Id || plan.Kind != loanOutSurplus {
		t.Fatalf("a surplus player unsold for 8 days should be loaned out, got %+v", plan)
	}
	value := common.MarketValue(*surplus)
	if want := (value + 5) / 10; plan.Fee < want-1 || plan.Fee > want {
		t.Errorf("fee = %d, want 10%% of his market value %d (about %d)", plan.Fee, value, want)
	}
	if plan.WageContributionPercent != 0 || plan.BuyOptionFee != value || plan.RelistAt != value {
		t.Errorf("a surplus loan has no wage contribution and a buy option at the old asking price %d, got %+v", value, plan)
	}
}

func TestPickPlayerToLoanOut_ListedForSixDaysStaysOnTheTransferList(t *testing.T) {
	squad := loanSquad()
	in := loanInputs(squad, listedFor(weakestDefender(squad), 6*24*time.Hour))
	if plan := pickPlayerToLoanOut(in); plan != nil {
		t.Errorf("a listing under 7 days old stays where it is, got %+v", plan)
	}
}

func TestPickPlayerToLoanOut_TheLongestListedGoesFirst(t *testing.T) {
	squad := loanSquad()
	// Two surplus defenders: the extra reserve (id 100+) and the weakest regular one (id 7).
	var second *common.Player
	for _, p := range squad {
		if p.Id == 7 {
			second = p
		}
	}
	surplus := []*common.Player{weakestDefender(squad), second}
	if surplus[0] == nil || surplus[1] == nil || common.BestCorePosition(*second) != common.DEFENSE {
		t.Fatalf("test setup: need two defenders to list, got %+v", surplus)
	}
	listings := listedFor(surplus[0], 8*24*time.Hour)
	for id, l := range listedFor(surplus[1], 12*24*time.Hour) {
		listings[id] = l
	}
	if plan := pickPlayerToLoanOut(loanInputs(squad, listings)); plan == nil || plan.Player.Id != surplus[1].Id {
		t.Errorf("the player listed for 12 days should go before the one listed for 8, got %+v", plan)
	}
}

func TestPickPlayerToLoanOut_ContractMustOutliveTheLoan(t *testing.T) {
	squad := loanSquad()
	surplus := weakestDefender(squad)
	surplus.ContractYearsRemaining = 1
	if plan := pickPlayerToLoanOut(loanInputs(squad, listedFor(surplus, 9*24*time.Hour))); plan != nil {
		t.Errorf("a player with one season left can't be loaned for one, got %+v", plan)
	}
	surplus.ContractYearsRemaining = 2
	if plan := pickPlayerToLoanOut(loanInputs(squad, listedFor(surplus, 9*24*time.Hour))); plan == nil {
		t.Error("a player with two seasons left outlives a 1-season loan")
	}
}

func TestPickPlayerToLoanOut_NeverLeavesAPositionUnderItsMinimumDepth(t *testing.T) {
	squad := loanSquad() // 2 goalkeepers: the minimum depth
	keeper := squad[0]
	if plan := pickPlayerToLoanOut(loanInputs(squad, listedFor(keeper, 9*24*time.Hour))); plan != nil {
		t.Errorf("loaning a goalkeeper would leave one, under the minimum of 2: got %+v", plan)
	}
}

func TestPickPlayerToLoanOut_NeverShrinksTheSquadUnderItsMinimum(t *testing.T) {
	squad := loanSquad()[:common.MinSquadSizeAfterSale] // exactly the floor: nobody can leave
	for len(squad) < common.MinSquadSizeAfterSale {
		squad = append(squad, defender(int64(400+len(squad)), 20))
	}
	surplus := squad[len(squad)-1]
	if plan := pickPlayerToLoanOut(loanInputs(squad, listedFor(surplus, 9*24*time.Hour))); plan != nil {
		t.Errorf("the squad is at its minimum size, nobody leaves, got %+v", plan)
	}
}

func TestPickPlayerToLoanOut_NeverLendsRecentSigningsOrInternationals(t *testing.T) {
	squad := loanSquad()
	surplus := weakestDefender(squad)
	in := loanInputs(squad, listedFor(surplus, 9*24*time.Hour))
	in.Excluded = map[int64]bool{surplus.Id: true}
	if plan := pickPlayerToLoanOut(in); plan != nil {
		t.Errorf("an excluded player is never lent, got %+v", plan)
	}
}

func TestPickPlayerToLoanOut_DirectorLoansOutABenchedTeenager(t *testing.T) {
	squad := loanSquad()
	teen := striker(300, 20)
	teen.Age = 19
	teen.ContractYearsRemaining = 3
	older := striker(301, 20)
	older.Age = 21
	older.ContractYearsRemaining = 3
	squad = append(squad, older, teen)
	in := loanInputs(squad, map[int64]common.TeamTransferListing{})
	in.DevelopmentLoans = true

	plan := pickPlayerToLoanOut(in)
	if plan == nil || plan.Player.Id != teen.Id || plan.Kind != loanOutDevelopment {
		t.Fatalf("the youngest benched youth should be loaned out, got %+v", plan)
	}
	if plan.Fee != 0 || plan.WageContributionPercent != 50 || plan.BuyOptionFee != 0 {
		t.Errorf("a development loan is free with 50%% of the wage covered and no option, got %+v", plan)
	}
}

func TestPickPlayerToLoanOut_ManagerKeepsItsYoungsters(t *testing.T) {
	squad := loanSquad()
	teen := striker(300, 20)
	teen.Age = 19
	teen.ContractYearsRemaining = 3
	in := loanInputs(append(squad, teen), map[int64]common.TeamTransferListing{})
	in.DevelopmentLoans = false
	if plan := pickPlayerToLoanOut(in); plan != nil {
		t.Errorf("only a Director loans youngsters out, got %+v", plan)
	}
}

func TestPickPlayerToLoanOut_AStarterYoungsterStaysToPlay(t *testing.T) {
	squad := loanSquad()
	prodigy := striker(300, 99)
	prodigy.Age = 18
	prodigy.ContractYearsRemaining = 3
	in := loanInputs(append(squad, prodigy), map[int64]common.TeamTransferListing{})
	in.DevelopmentLoans = true
	if plan := pickPlayerToLoanOut(in); plan != nil && plan.Player.Id == prodigy.Id {
		t.Errorf("a youngster in the best XI is not loaned out to play, got %+v", plan)
	}
}

func TestPickPlayerToLoanOut_RespectsTheLoanLimits(t *testing.T) {
	squad := loanSquad()
	surplus := weakestDefender(squad)
	listings := listedFor(surplus, 9*24*time.Hour)

	in := loanInputs(squad, listings)
	in.LoanedOut = map[int64]bool{901: true, 902: true, 903: true}
	if plan := pickPlayerToLoanOut(in); plan != nil {
		t.Errorf("with 3 players out on loan nobody else goes, got %+v", plan)
	}
	in = loanInputs(squad, listings)
	in.LoanListed = map[int64]bool{901: true, 902: true}
	if plan := pickPlayerToLoanOut(in); plan != nil {
		t.Errorf("with 2 loan listings open no third is started, got %+v", plan)
	}
	in = loanInputs(squad, listings)
	in.LoanedOut = map[int64]bool{901: true, 902: true}
	in.LoanListed = map[int64]bool{903: true}
	if plan := pickPlayerToLoanOut(in); plan != nil {
		t.Errorf("2 out plus 1 listed already reaches the cap of 3 players out on loan, got %+v", plan)
	}
	in = loanInputs(squad, listings)
	in.LoanedOut = map[int64]bool{901: true, 902: true}
	if plan := pickPlayerToLoanOut(in); plan == nil {
		t.Error("with 2 players out there is still room for one more")
	}
}

func TestPlanLoanListingWithdrawals_PositionFallingUnderItsMinimum(t *testing.T) {
	squad := loanSquad()
	// 3 defenders left in the unlisted squad would be the minimum; list a 4th defender and
	// drop the pool to 2 by moving two others out of the picture (transfer-listed).
	var defenders []*common.Player
	for _, p := range squad {
		if common.BestCorePosition(*p) == common.DEFENSE {
			defenders = append(defenders, p)
		}
	}
	loanListed := map[int64]bool{defenders[len(defenders)-1].Id: true}
	transferListed := map[int64]bool{}
	for _, d := range defenders[:len(defenders)-3] { // leave only 2 unlisted defenders
		transferListed[d.Id] = true
	}

	got := planLoanListingWithdrawals(squad, transferListed, loanListed)
	if len(got) != 1 || got[0].Id != defenders[len(defenders)-1].Id {
		t.Errorf("the loan-listed defender is needed back, got %+v", got)
	}
	if got := planLoanListingWithdrawals(squad, map[int64]bool{}, loanListed); len(got) != 0 {
		t.Errorf("with defenders to spare the listing stays, got %+v", got)
	}
}

func TestPlanLoanListingWithdrawals_ListedYouthWhoBecameAStarterComesBack(t *testing.T) {
	squad := loanSquad()
	prodigy := striker(300, 99)
	prodigy.Age = 19
	prodigy.ContractYearsRemaining = 3
	squad = append(squad, prodigy)

	got := planLoanListingWithdrawals(squad, map[int64]bool{}, map[int64]bool{prodigy.Id: true})
	if len(got) != 1 || got[0].Id != prodigy.Id {
		t.Errorf("a loan-listed player who now starts is withdrawn, got %+v", got)
	}

	bench := striker(301, 20)
	bench.Age = 19
	bench.ContractYearsRemaining = 3
	squad = append(loanSquad(), bench)
	if got := planLoanListingWithdrawals(squad, map[int64]bool{}, map[int64]bool{bench.Id: true}); len(got) != 0 {
		t.Errorf("a benched loan-listed player stays listed, got %+v", got)
	}
}

func TestLoanOutRoom(t *testing.T) {
	cases := []struct {
		listed, out int
		want        bool
	}{{0, 0, true}, {1, 1, true}, {2, 0, false}, {1, 2, false}, {0, 3, false}, {0, 2, true}}
	for _, c := range cases {
		if got := loanOutRoom(c.listed, c.out); got != c.want {
			t.Errorf("loanOutRoom(listed=%d, out=%d) = %v, want %v", c.listed, c.out, got, c.want)
		}
	}
}
