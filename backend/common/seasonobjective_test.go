package common

import "testing"

// cpu-team-management "CPU season objective".
func TestSeasonObjectiveFor(t *testing.T) {
	// A 16-team league, points best to worst: 60, 58, 55, ... down the table.
	table := func(points ...int) []int { return points }
	league := table(60, 58, 55, 50, 47, 45, 44, 40, 38, 35, 33, 30, 28, 25, 22, 20)
	cases := []struct {
		name string
		s    SeasonSituation
		want string
	}{
		{"early season", SeasonSituation{Points: league, Index: 14, Played: 9, Remaining: 21, PromotionSlots: 2, RelegationSlots: 2}, SeasonObjectiveOpen},
		{"just above the drop", SeasonSituation{Points: table(40, 35, 30, 26, 24, 22, 20, 18, 15), Index: 6, Played: 12, Remaining: 18, PromotionSlots: 2, RelegationSlots: 2}, SeasonObjectiveRelegationFight},
		{"in the zone", SeasonSituation{Points: league, Index: 15, Played: 20, Remaining: 10, PromotionSlots: 2, RelegationSlots: 2}, SeasonObjectiveRelegationFight},
		{"chasing promotion", SeasonSituation{Points: table(45, 42, 41, 40, 30, 20, 10), Index: 3, Played: 20, Remaining: 10, PromotionSlots: 2, RelegationSlots: 2}, SeasonObjectivePromotionPush},
		{"dead rubber", SeasonSituation{Points: table(62, 60, 50, 45, 40, 35, 30, 28), Index: 3, Played: 27, Remaining: 3, PromotionSlots: 2, RelegationSlots: 2}, SeasonObjectiveNothingAtStake},
		{"top division title race", SeasonSituation{Points: table(50, 48, 30, 20), Index: 1, Played: 20, Remaining: 10, PromotionSlots: 1, RelegationSlots: 1}, SeasonObjectivePromotionPush},
		{"mid-table with time left", SeasonSituation{Points: league, Index: 7, Played: 15, Remaining: 15, PromotionSlots: 2, RelegationSlots: 2}, SeasonObjectiveOpen},
		{"bottom tier: last place has nothing at stake", SeasonSituation{Points: league, Index: 15, Played: 20, Remaining: 10, PromotionSlots: 2}, SeasonObjectiveNothingAtStake},
		{"club not in the table", SeasonSituation{Points: league, Index: -1, Played: 20, Remaining: 10}, SeasonObjectiveOpen},
	}
	for _, c := range cases {
		if got := SeasonObjectiveFor(c.s); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// cpu-transfer-market "Market urgency".
func TestAdjustProfileForObjective(t *testing.T) {
	manager := AiTransferProfileFor(AiLevelManager)
	fight := AdjustProfileForObjective(manager, SeasonObjectiveRelegationFight)
	if fight.MinScore != 3 || fight.BudgetReserve != 0.25 {
		t.Errorf("Manager fighting relegation: min score %.1f reserve %.2f, want 3 and 0.25", fight.MinScore, fight.BudgetReserve)
	}
	if manager.MinScore != 5 {
		t.Errorf("the caller's profile changed: %+v", manager)
	}
	if push := AdjustProfileForObjective(manager, SeasonObjectivePromotionPush); push.MinScore != 5 {
		t.Errorf("Manager in a promotion push: min score %.1f, want 5", push.MinScore)
	}
	director := AiTransferProfileFor(AiLevelDirector)
	if push := AdjustProfileForObjective(director, SeasonObjectivePromotionPush); push.MinScore != 2 {
		t.Errorf("Director in a promotion push: min score %.1f, want 2", push.MinScore)
	}
	if fight := AdjustProfileForObjective(director, SeasonObjectiveRelegationFight); fight.MinScore != 1 {
		t.Errorf("Director fighting relegation: min score %.1f, want 1 (3 - 2)", fight.MinScore)
	}
	caretaker := AiTransferProfileFor(AiLevelCaretaker)
	if got := AdjustProfileForObjective(caretaker, SeasonObjectiveRelegationFight); got != caretaker {
		t.Errorf("Caretaker profile changed: %+v", got)
	}
}

// cpu-team-management "Youth minutes when nothing is at stake".
func TestGiveYouthStarts(t *testing.T) {
	newSquad := func() []*Player {
		squad := aiSquad(30, 90) // starters 2-11 at 70, bench 13-18 at 30
		squad[1].Defense, squad[2].Defense = 60, 62
		return squad
	}
	plan := func(squad []*Player, youth int) AiMatchPlan {
		return PlanAiMatchLineupWithOptions(squad, nil, nil, AiLevelDirector, false, nil, AiPlanOptions{YouthStarts: youth})
	}
	starts := func(p AiMatchPlan, id int64) bool {
		for _, pos := range p.Positions {
			if pos.PlayerId == id {
				return true
			}
		}
		return false
	}

	// Two prospects get a start: 19- and 20-year-olds worth about 85-90% of the weakest starters.
	squad := newSquad()
	base := plan(squad, 0)
	weakest := map[int64]bool{}
	for _, pos := range base.Positions {
		weakest[pos.PlayerId] = true
	}
	squad[13].Age, squad[13].Defense, squad[13].Lateral, squad[13].Midfield, squad[13].Pass, squad[13].Shot, squad[13].Dribbling = 19, 60, 60, 60, 60, 60, 60
	squad[14].Age, squad[14].Defense, squad[14].Lateral, squad[14].Midfield, squad[14].Pass, squad[14].Shot, squad[14].Dribbling = 20, 62, 62, 62, 62, 62, 62
	withYouth := plan(squad, 2)
	if !starts(withYouth, squad[13].Id) || !starts(withYouth, squad[14].Id) {
		t.Fatalf("prospects %d and %d do not start: %+v", squad[13].Id, squad[14].Id, withYouth.Positions)
	}
	if len(withYouth.Positions) != 11 {
		t.Errorf("%d starters, want 11", len(withYouth.Positions))
	}
	if err := ValidateLineup(withYouth.Formation, withYouth.Positions, &squad, nil, true); err != nil {
		t.Errorf("the youth XI fails validation: %s", err)
	}
	if err := ValidateSubstitutions(withYouth.Positions, withYouth.Substitutions, &squad, nil); err != nil {
		t.Errorf("its substitutions fail validation: %s", err)
	}

	// Prospect too raw: a 20-year-old worth well under 80% of every starter stays on the bench.
	raw := newSquad()
	raw[13].Age = 20
	if p := plan(raw, 2); starts(p, raw[13].Id) {
		t.Errorf("the raw prospect %d starts", raw[13].Id)
	}

}
