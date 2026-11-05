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
				if err := ValidateRoles(plan.Positions, plan.Roles, &squad); err != nil {
					t.Fatalf("squad %d level %d friendly=%v: roles invalid: %s", i, level, friendly, err)
				}
			}
		}
	}
	if checked < 150 || withSubstitutions < 300 {
		t.Fatalf("expected the random squads to exercise the planner: %d squads checked, %d plans with substitutions", checked, withSubstitutions)
	}
}

// CPU mentality (openspec/specs/cpu-team-management "CPU mentality").
func TestPickAiMentality(t *testing.T) {
	cases := []struct {
		name               string
		level, own, opp    int
		isHome, isFriendly bool
		want               int
	}{
		{"manager favourite", AiLevelManager, 1150, 1000, true, false, MENTALITY_ATTACKING},
		{"manager slight favourite", AiLevelManager, 1140, 1000, true, false, MENTALITY_BALANCED},
		{"manager underdog", AiLevelManager, 870, 1000, false, false, MENTALITY_DEFENSIVE},
		{"director away underdog", AiLevelDirector, 950, 1000, false, false, MENTALITY_DEFENSIVE},
		{"director home even", AiLevelDirector, 1000, 1000, true, false, MENTALITY_BALANCED},
		{"director home slight favourite", AiLevelDirector, 1020, 1000, true, false, MENTALITY_BALANCED},
		{"director home favourite", AiLevelDirector, 1030, 1000, true, false, MENTALITY_ATTACKING},
		{"director away favourite", AiLevelDirector, 1150, 1000, false, false, MENTALITY_BALANCED},
		{"caretaker", AiLevelCaretaker, 2000, 1000, true, false, MENTALITY_BALANCED},
		{"friendly", AiLevelDirector, 2000, 1000, true, true, MENTALITY_BALANCED},
		{"unknown strength", AiLevelDirector, 1000, 0, true, false, MENTALITY_BALANCED},
	}
	for _, c := range cases {
		if got := PickAiMentality(c.level, c.own, c.opp, c.isHome, c.isFriendly); got != c.want {
			t.Errorf("%s: PickAiMentality = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestValidateMentality(t *testing.T) {
	for _, m := range []int{-1, 0, 1} {
		if err := ValidateMentality(m); err != nil {
			t.Errorf("ValidateMentality(%d) = %v", m, err)
		}
	}
	for _, m := range []int{-2, 2} {
		if err := ValidateMentality(m); err == nil {
			t.Errorf("ValidateMentality(%d) accepted", m)
		}
	}
}

// CPU conditional orders by AI level (openspec/specs/cpu-team-management "CPU conditional orders").
func TestAiConditionalOrders(t *testing.T) {
	if got := AiConditionalOrders(AiLevelCaretaker); len(got) != 0 {
		t.Errorf("Caretaker orders = %+v, want none", got)
	}
	manager := AiConditionalOrders(AiLevelManager)
	if len(manager) != 1 || manager[0] != (LineupOrder{FromMinute: 70, Condition: ORDER_CONDITION_LOSING, Action: ORDER_ACTION_MENTALITY, Mentality: MENTALITY_ATTACKING}) {
		t.Errorf("Manager orders = %+v", manager)
	}
	director := AiConditionalOrders(AiLevelDirector)
	if len(director) != 2 ||
		director[0] != (LineupOrder{FromMinute: 65, Condition: ORDER_CONDITION_LOSING, Action: ORDER_ACTION_MENTALITY, Mentality: MENTALITY_ATTACKING}) ||
		director[1] != (LineupOrder{FromMinute: 75, Condition: ORDER_CONDITION_WINNING, Action: ORDER_ACTION_MENTALITY, Mentality: MENTALITY_DEFENSIVE}) {
		t.Errorf("Director orders = %+v", director)
	}
	for _, o := range director {
		if err := ValidateOrders(nil, []LineupOrder{o}, &[]*Player{}, nil); err != nil {
			t.Errorf("a CPU order fails validation: %s", err)
		}
	}
}

func TestPlanAiMatchLineup_OrdersOnlyInCompetitiveMatches(t *testing.T) {
	squad := make([]*Player, 0, 18)
	for i := int64(1); i <= 18; i++ {
		squad = append(squad, fieldPlayer(i, 50, 80, 27))
	}
	squad[0].GoalKeeper, squad[1].GoalKeeper = 70, 65
	competitive := PlanAiMatchLineup(squad, nil, nil, AiLevelDirector, false, rand.New(rand.NewSource(1)))
	if len(competitive.Orders) != 2 {
		t.Errorf("Director league plan has %d orders, want 2", len(competitive.Orders))
	}
	friendly := PlanAiMatchLineup(squad, nil, nil, AiLevelDirector, true, rand.New(rand.NewSource(1)))
	if len(friendly.Orders) != 0 {
		t.Errorf("Director friendly plan has %d orders, want none", len(friendly.Orders))
	}
}

// CPU captain (openspec/specs/cpu-team-management "CPU captain").
func TestPlanAiMatchLineup_VeteranCaptain(t *testing.T) {
	for _, friendly := range []bool{false, true} {
		squad := aiSquad(40, 90)
		// The keeper is the oldest of all, but never captain; among the outfield starters 33 beats
		// 31 and 24.
		squad[0].Age = 38
		squad[3].Age, squad[5].Age, squad[7].Age = 24, 31, 33
		plan := PlanAiMatchLineup(squad, nil, nil, AiLevelManager, friendly, seeded())
		if plan.Roles.CaptainId == nil || *plan.Roles.CaptainId != squad[7].Id {
			t.Errorf("friendly=%v: captain %v, want the 33-year-old %d", friendly, plan.Roles.CaptainId, squad[7].Id)
		}
		// Takers: automatic.
		if plan.Roles.PenaltyTakerId != nil || plan.Roles.FreeKickTakerId != nil {
			t.Errorf("friendly=%v: CPU named set-piece takers %+v, want automatic", friendly, plan.Roles)
		}
	}
}

func TestAiCaptain_TieGoesToLowestId(t *testing.T) {
	squad := []*Player{keeperPlayer(1, 80, 90, 40), fieldPlayer(9, 50, 90, 30), fieldPlayer(4, 50, 90, 30), fieldPlayer(2, 50, 90, 22)}
	positions := []LineupPosition{{PlayerId: 1, Position: GOALKEEPER}, {PlayerId: 9, Position: DEFENSE}, {PlayerId: 4, Position: MIDFIELD}, {PlayerId: 2, Position: STRIKER}}
	if got := AiCaptain(squad, positions, nil); got == nil || *got != 4 {
		t.Errorf("captain %v, want 4 (two 30-year-olds, lowest id)", got)
	}
	if got := AiCaptain(squad, positions[:1], nil); got != nil {
		t.Errorf("captain %d with only a keeper, want none", *got)
	}
}

// Lineup roles validation (openspec/specs/match-simulation "Lineup roles").
func TestValidateRoles(t *testing.T) {
	squad := aiSquad(40, 90) // 1 keeper, 2-11 outfield starters, 12 bench keeper, 13-18 bench outfielders
	positions := []LineupPosition{{PlayerId: 1, Position: GOALKEEPER}}
	for id := int64(2); id <= 11; id++ {
		positions = append(positions, LineupPosition{PlayerId: id, Position: DEFENSE})
	}
	ptr := func(v int64) *int64 { return &v }
	cases := []struct {
		name  string
		roles LineupRoles
		ok    bool
	}{
		{"automatic", LineupRoles{}, true},
		{"roles saved", LineupRoles{CaptainId: ptr(2), FreeKickTakerId: ptr(10)}, true},
		{"bench taker", LineupRoles{PenaltyTakerId: ptr(13)}, true},
		{"goalkeeper captain", LineupRoles{CaptainId: ptr(1)}, true},
		{"goalkeeper as taker", LineupRoles{PenaltyTakerId: ptr(1)}, false},
		{"bench goalkeeper as taker", LineupRoles{FreeKickTakerId: ptr(12)}, false},
		{"not in the squad", LineupRoles{CaptainId: ptr(99)}, false},
	}
	for _, c := range cases {
		err := ValidateRoles(positions, c.roles, &squad)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

// cpu-team-management "CPU captain": "Veteran due off" - the oldest outfield starter is passed over
// when the plan takes him off, and taken back when every outfielder is due off.
func TestAiCaptain_SkipsScheduledOutgoingPlayer(t *testing.T) {
	squad := []*Player{keeperPlayer(1, 80, 90, 40), fieldPlayer(2, 50, 90, 24), fieldPlayer(3, 50, 90, 31), fieldPlayer(4, 50, 90, 33)}
	positions := []LineupPosition{{PlayerId: 1, Position: GOALKEEPER}, {PlayerId: 2, Position: DEFENSE}, {PlayerId: 3, Position: MIDFIELD}, {PlayerId: 4, Position: STRIKER}}
	if got := AiCaptain(squad, positions, nil); got == nil || *got != 4 {
		t.Errorf("captain %v, want the 33-year-old 4", got)
	}
	off := []LineupSubstitution{{PlayerOutId: 4, PlayerInId: 9, Minute: 70}}
	if got := AiCaptain(squad, positions, off); got == nil || *got != 3 {
		t.Errorf("captain %v with the 33-year-old due off, want the 31-year-old 3", got)
	}
	allOff := []LineupSubstitution{{PlayerOutId: 2, PlayerInId: 7}, {PlayerOutId: 3, PlayerInId: 8}, {PlayerOutId: 4, PlayerInId: 9}}
	if got := AiCaptain(squad, positions, allOff); got == nil || *got != 4 {
		t.Errorf("captain %v with every outfielder due off, want the oldest, 4", got)
	}
}

// A Director's plan never hands the armband to a starter it schedules off.
func TestPlanAiMatchLineup_CaptainStaysOnThePitch(t *testing.T) {
	for _, level := range []int{AiLevelCaretaker, AiLevelManager, AiLevelDirector} {
		for _, friendly := range []bool{false, true} {
			// Tired starters, a strong bench and the oldest outfielders the most tired, so every
			// level plans changes that would take its veterans off.
			squad := aiSquad(66, 30)
			for _, p := range squad {
				p.Age = 20 + int(p.Id)
				if p.Id >= 2 && p.Id <= 11 {
					p.Stamina = 40 - int(p.Id)*3
				}
			}
			plan := PlanAiMatchLineup(squad, nil, nil, level, friendly, rand.New(rand.NewSource(1)))
			if len(plan.Substitutions) == 0 {
				t.Fatalf("level %d friendly=%v: no substitutions planned, the test squad must force some", level, friendly)
			}
			if plan.Roles.CaptainId == nil {
				t.Fatalf("level %d friendly=%v: no captain", level, friendly)
			}
			for _, s := range plan.Substitutions {
				if s.PlayerOutId == *plan.Roles.CaptainId {
					t.Errorf("level %d friendly=%v: captain %d is scheduled off at %d", level, friendly, s.PlayerOutId, s.Minute)
				}
			}
		}
	}
}

// cpu-team-management "Director cup orders".
func TestAiCupDrawOrder(t *testing.T) {
	fav := AiCupDrawOrder(AiLevelDirector, 1050, 1000)
	if fav == nil || fav.FromMinute != 75 || fav.Condition != ORDER_CONDITION_DRAWING || fav.Action != ORDER_ACTION_MENTALITY || fav.Mentality != MENTALITY_ATTACKING {
		t.Errorf("favourite: %+v, want from 75 if drawing go Attacking", fav)
	}
	if even := AiCupDrawOrder(AiLevelDirector, 1000, 1000); even == nil || even.Mentality != MENTALITY_ATTACKING {
		t.Errorf("even sides: %+v, want Attacking", even)
	}
	if dog := AiCupDrawOrder(AiLevelDirector, 950, 1000); dog == nil || dog.Mentality != MENTALITY_DEFENSIVE {
		t.Errorf("underdog: %+v, want Defensive", dog)
	}
	if o := AiCupDrawOrder(AiLevelManager, 1050, 1000); o != nil {
		t.Errorf("Manager: %+v, want none", o)
	}
	if o := AiCupDrawOrder(AiLevelDirector, 0, 1000); o != nil {
		t.Errorf("unknown strength: %+v, want none", o)
	}
}
