package engine

import (
	"testing"

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
	got := pickSurplusPlayerToList(squad, none(), none())
	if got == nil {
		t.Fatal("expected a surplus defender to be listed from a starting-size squad")
	}
	if want := squad[7]; got.Id != want.Id {
		t.Errorf("listed player %d, want the weakest defender %d", got.Id, want.Id)
	}
}

func TestPickSurplusPlayerToList_NoSurplusNoListing(t *testing.T) {
	squad := squadOf(3, 4, 4, 4) // every position exactly min+1
	if got := pickSurplusPlayerToList(squad, none(), none()); got != nil {
		t.Errorf("listed player %d, want nobody without a surplus beyond surplusDepthMargin", got.Id)
	}
}

func TestPickSurplusPlayerToList_RespectsMinSquadSizeAfterSale(t *testing.T) {
	squad := squadOf(2, 6, 4, 4) // 16 players: selling one would drop below minSquadSizeAfterSale
	if got := pickSurplusPlayerToList(squad, none(), none()); got != nil {
		t.Errorf("listed player %d, want nobody when the sale would leave fewer than %d", got.Id, minSquadSizeAfterSale)
	}
}

func TestPickSurplusPlayerToList_SkipsLoanListedPlayer(t *testing.T) {
	squad := squadOf(2, 6, 5, 5)
	weakestDefender := squad[7]
	got := pickSurplusPlayerToList(squad, none(), map[int64]bool{weakestDefender.Id: true})
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
		got := pickSurplusPlayerToList(squad, listed, none())
		if got == nil {
			t.Fatalf("round %d: expected another defender to be listed", i)
		}
		if listed[got.Id] {
			t.Fatalf("round %d: re-picked already-listed player %d", i, got.Id)
		}
		listed[got.Id] = true
	}
	// Defense now has 5 unlisted (min+2) but 17 unlisted overall: one more is still allowed...
	if got := pickSurplusPlayerToList(squad, listed, none()); got == nil {
		t.Fatal("expected a third listing while defense still has min+2 unlisted")
	} else {
		listed[got.Id] = true
	}
	// ...but then defense is at min+1 and the squad at 16, so it stops.
	if got := pickSurplusPlayerToList(squad, listed, none()); got != nil {
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
	got := pickSurplusPlayerToList(owned, none(), none())
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
