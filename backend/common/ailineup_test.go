package common

import (
	"math"
	"math/rand"
	"testing"
)

// fieldPlayer is an outfielder: every core skill at skill except goalkeeping, so BestCorePosition
// never calls him a keeper.
func fieldPlayer(id int64, skill int, stamina int, age int) *Player {
	return &Player{
		Id: id, GoalKeeper: 10, Defense: skill, Lateral: skill, Pass: skill,
		Midfield: skill, Dribbling: skill, Shot: skill, Stamina: stamina, Age: age,
	}
}

// keeperPlayer is a goalkeeper: goalkeeping at skill, everything else low.
func keeperPlayer(id int64, skill int, stamina int, age int) *Player {
	return &Player{
		Id: id, GoalKeeper: skill, Defense: 10, Lateral: 10, Pass: 10,
		Midfield: 10, Dribbling: 10, Shot: 10, Stamina: stamina, Age: age,
	}
}

// aiSquad is a keeper (id 1) and ten outfield starters (ids 2-11) at skill 70, plus a bench of a
// keeper (id 12) and six outfielders (ids 13-18) at benchSkill. Everyone has the given stamina.
func aiSquad(benchSkill int, stamina int) []*Player {
	squad := []*Player{keeperPlayer(1, 90, stamina, 28)}
	for id := int64(2); id <= 11; id++ {
		squad = append(squad, fieldPlayer(id, 70, stamina, 28))
	}
	squad = append(squad, keeperPlayer(12, 60, stamina, 28))
	for id := int64(13); id <= 18; id++ {
		squad = append(squad, fieldPlayer(id, benchSkill, stamina, 28))
	}
	return squad
}

func setStamina(squad []*Player, stamina int, ids ...int64) {
	for _, p := range squad {
		for _, id := range ids {
			if p.Id == id {
				p.Stamina = stamina
			}
		}
	}
}

func planIds(plan AiMatchPlan) map[int64]bool {
	ids := make(map[int64]bool, len(plan.Positions))
	for _, p := range plan.Positions {
		ids[p.PlayerId] = true
	}
	return ids
}

func seeded() *rand.Rand { return rand.New(rand.NewSource(1)) }

func TestPlanAiMatchLineup_InjuredStarterManagerPicksFromAvailablePlayers(t *testing.T) {
	squad := aiSquad(60, 80)
	squad[5].Defense, squad[5].Midfield, squad[5].Shot, squad[5].Lateral, squad[5].Pass, squad[5].Dribbling = 95, 95, 95, 95, 95, 95 // id 6: the star
	injured := map[int64]bool{6: true}

	plan := PlanAiMatchLineup(squad, injured, injured, AiLevelManager, false, seeded())

	if planIds(plan)[6] {
		t.Fatalf("expected a Manager's XI to leave the injured star out, got %v", plan.Positions)
	}
	if len(plan.Positions) != 11 {
		t.Fatalf("expected a full XI, got %v", plan.Positions)
	}
	for _, s := range plan.Substitutions {
		if s.PlayerInId == 6 || s.PlayerOutId == 6 {
			t.Fatalf("expected the injured star in no substitution, got %v", plan.Substitutions)
		}
	}
}

func TestPlanAiMatchLineup_InjuredStarterCaretakerStillNamesHim(t *testing.T) {
	squad := aiSquad(60, 80)
	squad[5].Defense, squad[5].Midfield, squad[5].Shot, squad[5].Lateral, squad[5].Pass, squad[5].Dribbling = 95, 95, 95, 95, 95, 95 // id 6: the star
	injured := map[int64]bool{6: true}

	plan := PlanAiMatchLineup(squad, injured, injured, AiLevelCaretaker, false, seeded())

	if !planIds(plan)[6] {
		t.Fatalf("expected a Caretaker's XI to still name the injured star for the backfill, got %v", plan.Positions)
	}
	for _, s := range plan.Substitutions {
		if injured[s.PlayerInId] {
			t.Fatalf("expected no unavailable player scheduled to come on, got %v", plan.Substitutions)
		}
	}
}

func TestPlanAiMatchLineup_PositionsAreDeterministic(t *testing.T) {
	squad := aiSquad(60, 80)
	first := PlanAiMatchLineup(squad, nil, nil, AiLevelDirector, false, seeded())
	for i := 0; i < 20; i++ {
		again := PlanAiMatchLineup(squad, nil, nil, AiLevelDirector, false, seeded())
		if again.Formation != first.Formation || len(again.Positions) != len(first.Positions) {
			t.Fatalf("expected the same plan for the same seed")
		}
		for j := range again.Positions {
			if again.Positions[j] != first.Positions[j] {
				t.Fatalf("expected the same position order, got %v vs %v", again.Positions, first.Positions)
			}
		}
	}
}

func TestAiSubstitutionGain_MatchesTheSpecFormula(t *testing.T) {
	starter := aiStarter{player: fieldPlayer(2, 70, 40, 28), position: MIDFIELD}
	bench := fieldPlayer(13, 66, 90, 25)
	minute := 70

	got := aiSubstitutionGain(starter, bench, minute)

	starterFactor := 1 - 0.30*(float64(minute+90)/180)*(99-40)/99
	benchFactor := 1 - 0.30*(float64(90-minute)/180)*(99-90)/99
	want := float64(90-minute) * (float64(PositionValue(MIDFIELD, *bench))*benchFactor - float64(PositionValue(MIDFIELD, *starter.player))*starterFactor)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("expected gain %v, got %v", want, got)
	}
}

func TestPlanCompetitiveSubstitutions_CaretakerChangesItsMostTiredStarterAtMinute70(t *testing.T) {
	squad := aiSquad(60, 80)
	setStamina(squad, 20, 5)
	setStamina(squad, 35, 7)
	plan := PlanAiMatchLineup(squad, nil, nil, AiLevelCaretaker, false, seeded())

	if len(plan.Substitutions) != 1 {
		t.Fatalf("expected exactly 1 substitution, got %v", plan.Substitutions)
	}
	sub := plan.Substitutions[0]
	if sub.PlayerOutId != 5 || sub.Minute != 70 {
		t.Fatalf("expected the lowest-Stamina starter (5) off at minute 70, got %+v", sub)
	}
	if sub.PlayerInId < 13 || sub.PlayerInId > 18 {
		t.Fatalf("expected an outfield bench player on, got %+v", sub)
	}
}

func TestPlanCompetitiveSubstitutions_CaretakerDoesNothingWhenNobodyIsTiredEnough(t *testing.T) {
	squad := aiSquad(60, 80)
	setStamina(squad, 50, 5) // exactly 50 is not "below 50"

	if plan := PlanAiMatchLineup(squad, nil, nil, AiLevelCaretaker, false, seeded()); len(plan.Substitutions) != 0 {
		t.Fatalf("expected none (Stamina 50 is not below 50), got %v", plan.Substitutions)
	}
}

func TestPlanCompetitiveSubstitutions_CaretakerNeedsABenchPlayerWorthEightyPercent(t *testing.T) {
	squad := aiSquad(50, 80) // 50 < 80% of 70 (56)
	setStamina(squad, 20, 5)

	if plan := PlanAiMatchLineup(squad, nil, nil, AiLevelCaretaker, false, seeded()); len(plan.Substitutions) != 0 {
		t.Fatalf("expected none when the bench is worth under 80%% of the tired starter, got %v", plan.Substitutions)
	}
}

func TestPlanCompetitiveSubstitutions_ManagerMakesTheTopTwoAtMinute70(t *testing.T) {
	squad := aiSquad(66, 99)
	setStamina(squad, 5, 4, 6, 8) // three tired starters; bench fresher and nearly as good
	plan := PlanAiMatchLineup(squad, nil, nil, AiLevelManager, false, seeded())

	if len(plan.Substitutions) != 2 {
		t.Fatalf("expected a Manager to stop at 2 changes (one stays free), got %v", plan.Substitutions)
	}
	assertDistinctPlayers(t, plan.Substitutions)
	for _, s := range plan.Substitutions {
		if s.Minute != 70 {
			t.Fatalf("expected a Manager's changes at minute 70, got %v", plan.Substitutions)
		}
	}
}

func TestPlanCompetitiveSubstitutions_DirectorMakesThreeAndTheLastIsNotBefore75(t *testing.T) {
	squad := aiSquad(66, 99)
	setStamina(squad, 5, 4, 6, 8, 9)
	plan := PlanAiMatchLineup(squad, nil, nil, AiLevelDirector, false, seeded())

	if len(plan.Substitutions) != 3 {
		t.Fatalf("expected a Director to make 3 changes, got %v", plan.Substitutions)
	}
	assertDistinctPlayers(t, plan.Substitutions)
	for _, s := range plan.Substitutions {
		if s.Minute < 55 || s.Minute > 85 || s.Minute%5 != 0 {
			t.Fatalf("expected minutes 55-85 in steps of 5, got %v", plan.Substitutions)
		}
	}
	if last := plan.Substitutions[2]; last.Minute < 75 {
		t.Fatalf("expected the third change at minute 75 or later, got %+v", last)
	}
}

func TestPlanCompetitiveSubstitutions_NothingWorthChangingSchedulesNone(t *testing.T) {
	for _, level := range []int{AiLevelCaretaker, AiLevelManager, AiLevelDirector} {
		squad := aiSquad(40, 99) // fresh starters, much weaker bench
		if plan := PlanAiMatchLineup(squad, nil, nil, level, false, seeded()); len(plan.Substitutions) != 0 {
			t.Fatalf("level %d: expected no substitutions, got %v", level, plan.Substitutions)
		}
	}
}

func assertDistinctPlayers(t *testing.T, subs []LineupSubstitution) {
	t.Helper()
	seen := map[int64]bool{}
	for _, s := range subs {
		if seen[s.PlayerOutId] || seen[s.PlayerInId] {
			t.Fatalf("expected no player in two pairs, got %v", subs)
		}
		seen[s.PlayerOutId], seen[s.PlayerInId] = true, true
	}
}

func TestPlanAiMatchLineup_EveryPlanPassesTheHumanValidation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	checked, withSubstitutions := 0, 0
	for i := 0; i < 200; i++ {
		squad := make([]*Player, 0, 24)
		size := 14 + rng.Intn(11)
		squad = append(squad, keeperPlayer(1, 40+rng.Intn(50), rng.Intn(100), 18+rng.Intn(20)))
		if rng.Intn(2) == 0 {
			squad = append(squad, keeperPlayer(2, 20+rng.Intn(50), rng.Intn(100), 18+rng.Intn(20)))
		}
		for id := int64(len(squad)) + 1; len(squad) < size; id++ {
			squad = append(squad, fieldPlayer(id, 20+rng.Intn(70), rng.Intn(100), 17+rng.Intn(22)))
		}
		injured := map[int64]bool{}
		suspended := map[int64]bool{}
		for _, p := range squad {
			switch rng.Intn(12) {
			case 0:
				injured[p.Id] = true
			case 1:
				suspended[p.Id] = true
			}
		}
		leagueUnavailable := map[int64]bool{}
		for id := range injured {
			leagueUnavailable[id] = true
		}
		for id := range suspended {
			leagueUnavailable[id] = true
		}
		if len(squad)-len(leagueUnavailable) < 12 {
			continue // too thin to field a full XI and a bench; covered by the thin-squad test
		}

		checked++
		for _, level := range []int{AiLevelCaretaker, AiLevelManager, AiLevelDirector} {
			for _, friendly := range []bool{false, true} {
				plan := PlanAiMatchLineup(squad, leagueUnavailable, injured, level, friendly, rand.New(rand.NewSource(int64(i))))

				unavailable := leagueUnavailable
				if friendly {
					unavailable = injured
				}
				if len(plan.Substitutions) > 0 {
					withSubstitutions++
				}
				enforce := level >= AiLevelManager
				if err := ValidateLineup(plan.Formation, plan.Positions, &squad, unavailable, enforce); err != nil {
					t.Fatalf("squad %d level %d friendly=%v: lineup invalid: %s", i, level, friendly, err)
				}
				if err := ValidateSubstitutions(plan.Positions, plan.Substitutions, &squad, unavailable); err != nil {
					t.Fatalf("squad %d level %d friendly=%v: substitutions invalid: %s", i, level, friendly, err)
				}
			}
		}
	}
	if checked < 150 || withSubstitutions < 300 {
		t.Fatalf("expected the random squads to exercise the planner: %d squads checked, %d plans with substitutions", checked, withSubstitutions)
	}
}
