package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// oldWorld is a 1/4/16 x 10 country: 210 ranked teams, ids tier*10000 + group*100 + position.
func oldWorld() []RankedTeam {
	teams := make([]RankedTeam, 0, 210)
	for tier, groups := range map[int]int{1: 1, 2: 4, 3: 16} {
		for g := 1; g <= groups; g++ {
			for pos := 1; pos <= 10; pos++ {
				teams = append(teams, RankedTeam{
					TeamId:   int64(tier*10000 + g*100 + pos),
					Tier:     tier,
					Position: pos,
					Points:   60 - pos*5 + g,
				})
			}
		}
	}
	return teams
}

func countByTier(plan RestructurePlan) map[int]int {
	counts := make(map[int]int)
	for _, slot := range plan.Placement {
		counts[slot.Tier]++
	}
	return counts
}

// Spec scenario "From 1/4/16 to 1/2/4/8/16": tier 1 keeps its 10 teams, the 20 worst of tier 2
// drop to tier 3, 140 of tier 3 drop to tier 4 and 60 to tier 5; spec scenario "Top-up": 100 new
// CPU teams fill tier 5.
func TestPlanRestructure_OldWorldIntoTheDefaultPyramid(t *testing.T) {
	plan := PlanRestructure(oldWorld(), common.DefaultWorldTiers)
	counts := countByTier(plan)
	want := map[int]int{1: 10, 2: 20, 3: 40, 4: 80, 5: 60}
	for tier, n := range want {
		if counts[tier] != n {
			t.Errorf("tier %d holds %d existing teams, want %d", tier, counts[tier], n)
		}
	}
	missing := 0
	for slot, n := range plan.Missing {
		if slot.Tier != 5 {
			t.Errorf("group %+v is short of %d teams; only the bottom tier should be", slot, n)
		}
		missing += n
	}
	if missing != 100 {
		t.Errorf("%d CPU teams to create, want 100", missing)
	}
	// Tier 1 is unchanged; tier 2 keeps the best 20 of its 40 (positions 1-5 of each old group).
	for id, slot := range plan.Placement {
		oldTier := int(id / 10000)
		pos := int(id % 100)
		if oldTier == 1 && slot.Tier != 1 {
			t.Errorf("tier-1 team %d moved to tier %d", id, slot.Tier)
		}
		if oldTier == 2 && (pos <= 5) != (slot.Tier == 2) {
			t.Errorf("tier-2 team %d (position %d) placed in tier %d", id, pos, slot.Tier)
		}
	}
}

// Spec scenario "Snake order in tier 2": t1..t20 into 2 groups - group 1 gets t1, t4, t5, t8, ...
// and group 2 gets t2, t3, t6, t7, ...
func TestPlanRestructure_SnakeOrder(t *testing.T) {
	teams := make([]RankedTeam, 0, 30)
	for i := 1; i <= 10; i++ {
		teams = append(teams, RankedTeam{TeamId: int64(100 + i), Tier: 1, Position: i})
	}
	for i := 1; i <= 20; i++ {
		teams = append(teams, RankedTeam{TeamId: int64(i), Tier: 2, Position: i})
	}
	plan := PlanRestructure(teams, []common.TierConfig{
		{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10},
		{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 10},
	})
	for _, id := range []int64{1, 4, 5, 8, 9, 12} {
		if plan.Placement[id] != (GroupSlot{Tier: 2, Group: 1}) {
			t.Errorf("t%d in %+v, want tier 2 group 1", id, plan.Placement[id])
		}
	}
	for _, id := range []int64{2, 3, 6, 7, 10, 11} {
		if plan.Placement[id] != (GroupSlot{Tier: 2, Group: 2}) {
			t.Errorf("t%d in %+v, want tier 2 group 2", id, plan.Placement[id])
		}
	}
	if len(plan.Missing) != 0 {
		t.Errorf("no group should be short: %v", plan.Missing)
	}
}

func TestPlanRestructure_UnrankedTeamsSortLastInTheirTier(t *testing.T) {
	teams := []RankedTeam{
		{TeamId: 1, Tier: 1, Position: 0},
		{TeamId: 2, Tier: 1, Position: 2},
		{TeamId: 3, Tier: 1, Position: 1},
	}
	plan := PlanRestructure(teams, []common.TierConfig{
		{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 2},
		{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 2},
	})
	if plan.Placement[1].Tier != 2 || plan.Placement[3].Tier != 1 || plan.Placement[2].Tier != 1 {
		t.Errorf("the team with no standing should drop first: %+v", plan.Placement)
	}
}

// A country already in the target shape needs no restructure (see restructureIfNeeded): every
// team keeps its tier and the plan creates nobody.
func TestPlanRestructure_AlreadyInShape(t *testing.T) {
	teams := make([]RankedTeam, 0, 30)
	for i := 1; i <= 10; i++ {
		teams = append(teams, RankedTeam{TeamId: int64(100 + i), Tier: 1, Position: i})
	}
	for i := 1; i <= 20; i++ {
		teams = append(teams, RankedTeam{TeamId: int64(i), Tier: 2, Position: (i + 1) / 2})
	}
	target := []common.TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 10}, {Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 10}}
	plan := PlanRestructure(teams, target)
	for _, rt := range teams {
		if plan.Placement[rt.TeamId].Tier != rt.Tier {
			t.Errorf("team %d changed tier %d -> %d", rt.TeamId, rt.Tier, plan.Placement[rt.TeamId].Tier)
		}
	}
	if len(plan.Missing) != 0 {
		t.Errorf("nothing to create, got %v", plan.Missing)
	}
}
