package common

import "testing"

// friendlySquad is a keeper (id 1) and ten outfield regulars (ids 2-11) at skill 80, plus the
// reserves: a keeper (id 12, skill 60) and five outfielders (ids 13-17, skill 50), ids 16-17 young.
func friendlySquad() []*Player {
	squad := []*Player{keeperPlayer(1, 90, 80, 28)}
	for id := int64(2); id <= 11; id++ {
		squad = append(squad, fieldPlayer(id, 80, 80, 28))
	}
	squad = append(squad, keeperPlayer(12, 60, 80, 29))
	for id := int64(13); id <= 17; id++ {
		age := 27
		if id >= 16 {
			age = 20
		}
		squad = append(squad, fieldPlayer(id, 50, 80, age))
	}
	return squad
}

func startersOf(plan AiMatchPlan) map[int64]bool { return planIds(plan) }

func TestPlanFriendly_CaretakerStartsItsRegularsAndBringsOnThreeAtMinute60(t *testing.T) {
	squad := friendlySquad()

	plan := PlanAiMatchLineup(squad, nil, nil, AiLevelCaretaker, true, seeded())

	started := startersOf(plan)
	for id := int64(1); id <= 11; id++ {
		if !started[id] {
			t.Fatalf("expected regular %d to start, got %v", id, plan.Positions)
		}
	}
	if len(plan.Substitutions) != 3 {
		t.Fatalf("expected 3 substitutions, got %v", plan.Substitutions)
	}
	assertDistinctPlayers(t, plan.Substitutions)
	for _, s := range plan.Substitutions {
		if s.Minute != 60 || started[s.PlayerInId] {
			t.Fatalf("expected bench players on at minute 60, got %v", plan.Substitutions)
		}
		if s.PlayerInId == 12 {
			t.Fatalf("expected the reserve keeper not to replace an outfielder, got %v", plan.Substitutions)
		}
	}
}

func TestPlanFriendly_ManagerRestsItsFiveBestOutfieldRegulars(t *testing.T) {
	squad := friendlySquad()
	// ids 2-6 are clearly the five best outfield regulars
	for _, p := range squad {
		if p.Id >= 2 && p.Id <= 6 {
			p.Defense, p.Midfield, p.Shot, p.Lateral, p.Pass, p.Dribbling = 90, 90, 90, 90, 90, 90
		}
	}

	squad = append(squad, fieldPlayer(18, 50, 80, 27), fieldPlayer(19, 50, 80, 27), fieldPlayer(20, 50, 80, 27))

	plan := PlanAiMatchLineup(squad, nil, nil, AiLevelManager, true, seeded())

	started := startersOf(plan)
	for id := int64(2); id <= 6; id++ {
		if started[id] {
			t.Fatalf("expected rested regular %d not to start, got %v", id, plan.Positions)
		}
	}
	if len(plan.Positions) != 11 {
		t.Fatalf("expected a full XI, got %v", plan.Positions)
	}
	if len(plan.Substitutions) != 3 {
		t.Fatalf("expected 3 substitutions, got %v", plan.Substitutions)
	}
	for _, s := range plan.Substitutions {
		if started[s.PlayerInId] {
			t.Fatalf("expected substitutes who did not start, got %v", plan.Substitutions)
		}
		if s.PlayerInId >= 2 && s.PlayerInId <= 6 {
			t.Fatalf("expected the rested regulars not to come on either, got %v", plan.Substitutions)
		}
		if s.Minute != 60 {
			t.Fatalf("expected minute 60, got %v", plan.Substitutions)
		}
	}
}

func TestPlanFriendly_DirectorStartsReservesIncludingTheBackupKeeper(t *testing.T) {
	squad := friendlySquad()

	plan := PlanAiMatchLineup(squad, nil, nil, AiLevelDirector, true, seeded())

	started := startersOf(plan)
	if !started[12] || started[1] {
		t.Fatalf("expected the backup keeper (12) in goal instead of the regular keeper (1), got %v", plan.Positions)
	}
	for id := int64(13); id <= 17; id++ {
		if !started[id] {
			t.Fatalf("expected reserve %d to start, got %v", id, plan.Positions)
		}
	}
	if len(plan.Positions) != 11 {
		t.Fatalf("expected a full XI, got %v", plan.Positions)
	}
	if len(plan.Substitutions) != 3 {
		t.Fatalf("expected 3 substitutions, got %v", plan.Substitutions)
	}
	for _, s := range plan.Substitutions {
		if started[s.PlayerInId] || s.Minute != 60 {
			t.Fatalf("expected non-starters on at minute 60, got %v", plan.Substitutions)
		}
	}
}

func TestPlanFriendly_DirectorPrefersPlayersAged21OrUnderOffTheBench(t *testing.T) {
	squad := friendlySquad()
	// two more young reserves who would not start (the XI has 6 reserve slots filled already)
	squad = append(squad, fieldPlayer(18, 45, 80, 19), fieldPlayer(19, 45, 80, 18), fieldPlayer(20, 45, 80, 17), fieldPlayer(21, 45, 80, 30))

	plan := PlanAiMatchLineup(squad, nil, nil, AiLevelDirector, true, seeded())

	young := map[int64]bool{}
	for _, p := range squad {
		if p.Age <= 21 {
			young[p.Id] = true
		}
	}
	started := startersOf(plan)
	youngOnBench := 0
	for id := range young {
		if !started[id] {
			youngOnBench++
		}
	}
	want := youngOnBench
	if want > 3 {
		want = 3
	}
	got := 0
	for _, s := range plan.Substitutions {
		if young[s.PlayerInId] {
			got++
		}
	}
	if got != want {
		t.Fatalf("expected %d of the substitutes to be aged 21 or under, got %d: %v", want, got, plan.Substitutions)
	}
}

func TestPlanFriendly_DirectorFillsGapsWithTheLowestValuedNonKeyRegular(t *testing.T) {
	squad := friendlySquad()
	// regular 11 is barely better than a reserve, so losing him costs no strength: not a key player
	for _, p := range squad {
		if p.Id == 11 {
			p.Defense, p.Midfield, p.Shot, p.Lateral, p.Pass, p.Dribbling = 55, 55, 55, 55, 55, 55
		}
	}

	plan := PlanAiMatchLineup(squad, nil, nil, AiLevelDirector, true, seeded())

	if !startersOf(plan)[11] {
		t.Fatalf("expected the non-key regular 11 to fill a gap in the reserve XI, got %v", plan.Positions)
	}
}

func TestPlanFriendly_ThinSquadFieldsElevenAndAtMostTwoSubstitutes(t *testing.T) {
	for _, level := range []int{AiLevelCaretaker, AiLevelManager, AiLevelDirector} {
		squad := friendlySquad()[:13]
		plan := PlanAiMatchLineup(squad, nil, nil, level, true, seeded())

		if len(plan.Positions) != 11 {
			t.Fatalf("level %d: expected 11 starters from 13 fit players, got %v", level, plan.Positions)
		}
		if len(plan.Substitutions) > 2 {
			t.Fatalf("level %d: expected at most 2 substitutions, got %v", level, plan.Substitutions)
		}
		if err := ValidateLineup(plan.Formation, plan.Positions, &squad, nil, true); err != nil {
			t.Fatalf("level %d: %s", level, err)
		}
		if err := ValidateSubstitutions(plan.Positions, plan.Substitutions, &squad, nil); err != nil {
			t.Fatalf("level %d: %s", level, err)
		}
	}
}

func TestPlanFriendly_ThirteenFitPlayersOutOfALargerSquad(t *testing.T) {
	squad := friendlySquad()
	unavailable := map[int64]bool{14: true, 15: true, 16: true, 17: true} // injured: only 13 remain
	for _, level := range []int{AiLevelManager, AiLevelDirector} {
		plan := PlanAiMatchLineup(squad, unavailable, unavailable, level, true, seeded())

		if len(plan.Positions) != 11 || len(plan.Substitutions) > 2 {
			t.Fatalf("level %d: expected 11 starters and at most 2 substitutions, got %v / %v", level, plan.Positions, plan.Substitutions)
		}
		for _, p := range plan.Positions {
			if unavailable[p.PlayerId] {
				t.Fatalf("level %d: injured player %d in the XI", level, p.PlayerId)
			}
		}
	}
}

func TestPlanFriendly_SuspendedPlayersAreEligible(t *testing.T) {
	squad := friendlySquad()
	suspended := map[int64]bool{2: true, 3: true} // barred from league matches only
	injured := map[int64]bool{}

	plan := PlanAiMatchLineup(squad, suspended, injured, AiLevelCaretaker, true, seeded())
	// A Caretaker's regulars ignore availability altogether; the suspended pair may start.
	// For a Manager the suspended pair is not a regular, but may still be rested-from-the-rest.
	if err := ValidateLineup(plan.Formation, plan.Positions, &squad, injured, true); err != nil {
		t.Fatalf("expected suspended players to be allowed in a friendly: %s", err)
	}

	manager := PlanAiMatchLineup(squad, suspended, injured, AiLevelManager, true, seeded())
	if err := ValidateLineup(manager.Formation, manager.Positions, &squad, injured, true); err != nil {
		t.Fatalf("expected a valid Manager friendly lineup: %s", err)
	}
	if err := ValidateSubstitutions(manager.Positions, manager.Substitutions, &squad, injured); err != nil {
		t.Fatalf("expected valid Manager friendly substitutions: %s", err)
	}
	suspendedPlays := false
	for _, p := range manager.Positions {
		if suspended[p.PlayerId] {
			suspendedPlays = true
		}
	}
	for _, s := range manager.Substitutions {
		if suspended[s.PlayerInId] {
			suspendedPlays = true
		}
	}
	if !suspendedPlays {
		t.Fatalf("expected a suspended player to be eligible and used by a Manager in a friendly, got %v / %v", manager.Positions, manager.Substitutions)
	}
}
