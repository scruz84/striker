package common

import (
	"math/rand"
	"testing"
)

// greedyLineupForTest is the slot-by-slot pick SelectBestLineupForFormation used before it became
// an optimal assignment: each slot, in fill order, takes the best remaining player.
func greedyLineupForTest(players []*Player, formation int) map[int][]int64 {
	lineup := make(map[int][]int64)
	used := map[int64]bool{}
	for _, position := range formationSlotOrder(formation) {
		best, bestValue := int64(-1), -1
		for _, p := range players {
			if v := PositionValue(position, *p); !used[p.Id] && v > bestValue {
				best, bestValue = p.Id, v
			}
		}
		if best >= 0 {
			lineup[position] = append(lineup[position], best)
			used[best] = true
		}
	}
	return lineup
}

func totalLineupValue(lineup map[int][]int64, players []*Player) int {
	byId := map[int64]*Player{}
	for _, p := range players {
		byId[p.Id] = p
	}
	total := 0
	for position, ids := range lineup {
		for _, id := range ids {
			total += PositionValue(position, *byId[id])
		}
	}
	return total
}

func positionOf(lineup map[int][]int64, id int64) int {
	for position, ids := range lineup {
		for _, other := range ids {
			if other == id {
				return position
			}
		}
	}
	return -1
}

// openspec cpu-team-management "Best XI by optimal assignment": a squad whose only flank players
// are a wide midfielder (Lateral, Midfield and Pass 75) and a full-back (Lateral and Defense 60),
// with a keeper, two centre-backs and weak reserves. The greedy fill reached the back line first and
// put the wide midfielder at full-back (58), leaving wide midfield to a reserve; the optimal XI plays
// the full-back at full-back (57) and the wide midfielder in wide midfield (60).
func TestSelectBestLineupForFormation_OptimalAssignmentPlaysSpecialistsInTheirSlots(t *testing.T) {
	wideMid := &Player{Id: 1, GoalKeeper: 25, Defense: 25, Lateral: 75, Pass: 75, Midfield: 75, Dribbling: 25, Shot: 25}
	fullBack := &Player{Id: 2, GoalKeeper: 10, Defense: 60, Lateral: 60, Pass: 40, Midfield: 40, Dribbling: 40, Shot: 40}
	keeper := &Player{Id: 3, GoalKeeper: 70, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10}
	players := []*Player{wideMid, fullBack, keeper}
	for id := int64(4); id <= 5; id++ {
		players = append(players, &Player{Id: id, GoalKeeper: 10, Defense: 70, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10})
	}
	for id := int64(6); id <= 14; id++ {
		players = append(players, &Player{Id: id, GoalKeeper: 10, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10})
	}

	greedy := greedyLineupForTest(players, FormationBalanced)
	if positionOf(greedy, 1) != LATERAL_DEFENSE_LEFT {
		t.Fatalf("test premise: the greedy fill should put the wide midfielder at left-back, got %v", greedy)
	}

	lineup := SelectBestLineupForFormation(players, FormationBalanced)
	wideAt, backAt := positionOf(lineup, 1), positionOf(lineup, 2)
	if wideAt != LATERAL_MIDFIELD_LEFT && wideAt != LATERAL_MIDFIELD_RIGHT {
		t.Errorf("the wide midfielder should play wide midfield, plays %d", wideAt)
	}
	if backAt != LATERAL_DEFENSE_LEFT && backAt != LATERAL_DEFENSE_RIGHT {
		t.Errorf("the full-back should play full-back, plays %d", backAt)
	}
	if got := PositionValue(wideAt, *wideMid) + PositionValue(backAt, *fullBack); got != 117 {
		t.Errorf("the pair should be worth 117, got %d", got)
	}
	if got, was := totalLineupValue(lineup, players), totalLineupValue(greedy, players); got <= was {
		t.Errorf("the optimal XI should be worth more than the greedy one: %d vs %d", got, was)
	}
}

// The optimal XI is never worth less than the greedy one, fills every slot, picks nobody twice
// and does not depend on the order the squad comes in.
func TestSelectBestLineupForFormation_NeverWorseThanGreedyAndDeterministic(t *testing.T) {
	rng := rand.New(rand.NewSource(9))
	for trial := 0; trial < 300; trial++ {
		players := make([]*Player, 0, 18)
		r := func() int { return 30 + rng.Intn(31) }
		for id := int64(1); id <= 18; id++ {
			players = append(players, &Player{Id: id, GoalKeeper: r(), Defense: r(), Lateral: r(), Pass: r(), Midfield: r(), Dribbling: r(), Shot: r()})
		}
		formation := AllFormations[trial%len(AllFormations)]
		lineup := SelectBestLineupForFormation(players, formation)
		seen := map[int64]bool{}
		count := 0
		for position, ids := range lineup {
			if len(ids) != FormationPositionCounts(formation)[position] {
				t.Fatalf("formation %d position %d has %d players, want %d", formation, position, len(ids), FormationPositionCounts(formation)[position])
			}
			for _, id := range ids {
				if seen[id] {
					t.Fatalf("player %d picked twice", id)
				}
				seen[id] = true
				count++
			}
		}
		if count != 11 {
			t.Fatalf("picked %d players, want 11", count)
		}
		if got, greedy := totalLineupValue(lineup, players), totalLineupValue(greedyLineupForTest(players, formation), players); got < greedy {
			t.Fatalf("optimal XI worth %d, less than the greedy %d", got, greedy)
		}
		reversed := make([]*Player, len(players))
		for i, p := range players {
			reversed[len(players)-1-i] = p
		}
		again := SelectBestLineupForFormation(reversed, formation)
		for position, ids := range lineup {
			for i, id := range ids {
				if again[position][i] != id {
					t.Fatalf("the XI depends on squad order: %v vs %v", lineup, again)
				}
			}
		}
	}
}
