package common

import "testing"

// national-teams "Call-up shortly before kickoff": "Pool full of strikers" and "Orders" - the fit
// pool is planned like a Director's competitive XI.
func TestPlanNationalCallUp_StrongestFormationCaptainAndOrders(t *testing.T) {
	pool := []*Player{keeperPlayer(1, 80, 80, 30)}
	striker := func(id int64) *Player {
		return &Player{Id: id, GoalKeeper: 5, Defense: 10, Lateral: 20, Pass: 70, Midfield: 30, Dribbling: 85, Shot: 90, Stamina: 80, Age: 25}
	}
	defender := func(id int64) *Player {
		return &Player{Id: id, GoalKeeper: 5, Defense: 75, Lateral: 40, Pass: 30, Midfield: 30, Dribbling: 10, Shot: 10, Stamina: 80, Age: 31}
	}
	midfielder := func(id int64) *Player {
		return &Player{Id: id, GoalKeeper: 5, Defense: 30, Lateral: 30, Pass: 50, Midfield: 50, Dribbling: 30, Shot: 30, Stamina: 80, Age: 27}
	}
	for id := int64(2); id <= 6; id++ {
		pool = append(pool, striker(id))
	}
	pool = append(pool, defender(7), defender(8))
	for id := int64(9); id <= 16; id++ {
		pool = append(pool, midfielder(id))
	}

	plan := planNationalCallUp(pool)
	_, balanced := SelectBestFormationAndLineup(pool, AiLevelDirector, nil)
	if len(plan.positions) != 11 {
		t.Fatalf("%d starters, want 11", len(plan.positions))
	}
	if plan.formation == FormationBalanced {
		t.Errorf("formation %d: five strong strikers should not be squeezed into a 4-4-2", plan.formation)
	}
	byId := map[int64]*Player{}
	for _, p := range pool {
		byId[p.Id] = p
	}
	if got, want := LineupStrength(&Lineup{Positions: plan.positions}, byId), lineupStrength(balanced, byId); got != want {
		t.Errorf("XI strength %d, want the strongest formation's %d", got, want)
	}
	if plan.roles.CaptainId == nil || byId[*plan.roles.CaptainId].Age != 31 {
		t.Errorf("captain %v, want one of the 31-year-old defenders", plan.roles.CaptainId)
	}
	want := AiConditionalOrders(AiLevelDirector)
	if len(plan.orders) != len(want) || plan.orders[0] != want[0] || plan.orders[1] != want[1] {
		t.Errorf("orders %+v, want the Director's %+v", plan.orders, want)
	}
}
