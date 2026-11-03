package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// openspec add-conditional-match-orders: "Order firing", "Substitution orders" and "Mentality
// change event" (match-simulation).

func attackingIfLosing(from int) common.LineupOrder {
	return common.LineupOrder{FromMinute: from, Condition: common.ORDER_CONDITION_LOSING, Action: common.ORDER_ACTION_MENTALITY, Mentality: common.MENTALITY_ATTACKING}
}

// orderMatch is a home side with a midfielder (10) and a bench player (99), playing orders.
func orderMatch(orders ...common.LineupOrder) *common.MatchSimulation {
	match := newSubMatch(0, map[int][]int64{common.GOALKEEPER: {1}, common.MIDFIELD: {10, 20}}, nil)
	(*match.HomePlayers)[99] = *namedPlayer(99)
	match.HomeOrders = orders
	return match
}

// playMinutes runs the substitution stage for minutes from..to; goals maps a minute to the score
// once that minute's scorer stage has run (the order stage of the next minute sees it).
func playMinutes(match *common.MatchSimulation, from, to int, goals map[int][2]int) {
	for minute := from; minute <= to; minute++ {
		match.Minute = minute
		SubstitutionSimulation{}.Apply(nil, match)
		if g, ok := goals[minute]; ok {
			match.HomeGoals, match.AwayGoals = g[0], g[1]
		}
	}
}

func mentalityEvents(match *common.MatchSimulation) []common.Event {
	out := make([]common.Event, 0)
	for _, e := range *match.Events {
		if e.EventType == common.MENTALITY_CHANGE {
			out = append(out, e)
		}
	}
	return out
}

func TestConditionalOrders_GoingBehindLate(t *testing.T) {
	match := orderMatch(attackingIfLosing(60))
	playMinutes(match, 1, 72, map[int][2]int{72: {0, 1}})
	if match.HomeMentality != common.MENTALITY_BALANCED {
		t.Fatalf("switched before going behind: %d", match.HomeMentality)
	}
	playMinutes(match, 73, 73, nil)
	if match.HomeMentality != common.MENTALITY_ATTACKING {
		t.Fatalf("minute 73: mentality %d, want Attacking", match.HomeMentality)
	}
	events := mentalityEvents(match)
	if len(events) != 1 || events[0].Minute != 73 || events[0].TeamOwner != 1 || events[0].Mentality == nil || *events[0].Mentality != common.MENTALITY_ATTACKING {
		t.Fatalf("events %+v, want one Attacking change for the home side at 73", events)
	}
	// Equalising doesn't undo a spent order.
	playMinutes(match, 74, 90, map[int][2]int{80: {1, 1}})
	if match.HomeMentality != common.MENTALITY_ATTACKING || len(mentalityEvents(match)) != 1 {
		t.Errorf("after equalising: mentality %d, %d events; want it to stay Attacking", match.HomeMentality, len(mentalityEvents(match)))
	}
}

func TestConditionalOrders_ConditionMetBeforeTheFromMinute(t *testing.T) {
	match := orderMatch(attackingIfLosing(60))
	playMinutes(match, 1, 59, map[int][2]int{20: {0, 1}})
	if match.HomeMentality != common.MENTALITY_BALANCED {
		t.Fatalf("fired before minute 60")
	}
	playMinutes(match, 60, 60, nil)
	if events := mentalityEvents(match); len(events) != 1 || events[0].Minute != 60 {
		t.Fatalf("events %+v, want the order to fire at 60", events)
	}
}

func TestConditionalOrders_ConditionNeverMet(t *testing.T) {
	match := orderMatch(attackingIfLosing(60))
	match.HomeMentality = common.MENTALITY_DEFENSIVE
	playMinutes(match, 1, 90, map[int][2]int{10: {1, 0}, 50: {2, 0}})
	if match.HomeMentality != common.MENTALITY_DEFENSIVE || len(mentalityEvents(match)) != 0 {
		t.Errorf("a winning side's 'if losing' order fired: mentality %d", match.HomeMentality)
	}
}

func TestConditionalOrders_TwoOrdersOnTheSameMinuteFireInSlotOrder(t *testing.T) {
	sub := common.LineupOrder{FromMinute: 70, Condition: common.ORDER_CONDITION_LOSING, Action: common.ORDER_ACTION_SUBSTITUTION, PlayerOutId: 10, PlayerInId: 99}
	match := orderMatch(attackingIfLosing(70), sub)
	playMinutes(match, 1, 70, map[int][2]int{30: {0, 1}})
	events := *match.Events
	if len(events) != 2 || events[0].EventType != common.MENTALITY_CHANGE || events[1].EventType != common.SUBSTITUTION || events[1].Minute != 70 {
		t.Fatalf("events %+v, want slot 1 (mentality) then slot 2 (substitution) at 70", events)
	}
}

func TestConditionalOrders_SwitchToDefensiveAndAlreadyAttacking(t *testing.T) {
	protect := common.LineupOrder{FromMinute: 75, Condition: common.ORDER_CONDITION_WINNING, Action: common.ORDER_ACTION_MENTALITY, Mentality: common.MENTALITY_DEFENSIVE}
	match := orderMatch(protect)
	playMinutes(match, 1, 75, map[int][2]int{40: {1, 0}})
	if events := mentalityEvents(match); len(events) != 1 || events[0].Minute != 75 || *events[0].Mentality != common.MENTALITY_DEFENSIVE {
		t.Fatalf("events %+v, want a Defensive change at 75", events)
	}

	already := orderMatch(attackingIfLosing(60))
	already.HomeMentality = common.MENTALITY_ATTACKING
	playMinutes(already, 1, 90, map[int][2]int{10: {0, 1}})
	if len(*already.Events) != 0 || already.HomeMentality != common.MENTALITY_ATTACKING {
		t.Errorf("an Attacking side's 'go Attacking' order logged %+v", *already.Events)
	}
	if !already.HomeOrdersFired[0] {
		t.Error("the order wasn't spent")
	}
}

func TestConditionalOrders_AwaySideReadsTheScoreFromItsOwnPointOfView(t *testing.T) {
	match := newSubMatch(0, map[int][]int64{common.MIDFIELD: {10}}, nil)
	awayPositions := map[int][]int64{common.MIDFIELD: {30}}
	awayPlayers := map[int64]common.Player{30: *namedPlayer(30)}
	match.AwayPlayersByPosition, match.AwayPlayers = &awayPositions, &awayPlayers
	match.AwayOrders = []common.LineupOrder{attackingIfLosing(60)}
	playMinutes(match, 1, 60, map[int][2]int{20: {1, 0}})
	if match.AwayMentality != common.MENTALITY_ATTACKING || match.HomeMentality != common.MENTALITY_BALANCED {
		t.Errorf("away losing 1-0: away %d home %d, want the away side Attacking", match.AwayMentality, match.HomeMentality)
	}
	if e := mentalityEvents(match); len(e) != 1 || e[0].TeamOwner != 2 {
		t.Errorf("events %+v, want one change for the away side", e)
	}
}

func TestConditionalOrders_SubstitutionOrderCapAlreadyReached(t *testing.T) {
	sub := common.LineupOrder{FromMinute: 60, Condition: common.ORDER_CONDITION_DRAWING, Action: common.ORDER_ACTION_SUBSTITUTION, PlayerOutId: 10, PlayerInId: 99}
	match := orderMatch(sub)
	match.HomeSubstitutionsUsed = common.MaxScheduledSubstitutions
	playMinutes(match, 60, 90, nil)
	if len(*match.Events) != 0 || !match.HomeOrdersFired[0] || match.HomeSubstitutionsUsed != common.MaxScheduledSubstitutions {
		t.Errorf("events %+v fired %v used %d: want the order spent with no substitution", *match.Events, match.HomeOrdersFired, match.HomeSubstitutionsUsed)
	}
}

func TestConditionalOrders_SubstitutionOrderOutgoingPlayerSentOff(t *testing.T) {
	sub := common.LineupOrder{FromMinute: 60, Condition: common.ORDER_CONDITION_DRAWING, Action: common.ORDER_ACTION_SUBSTITUTION, PlayerOutId: 10, PlayerInId: 99}
	match := orderMatch(sub)
	removePlayerFromPosition(match.HomePlayersByPosition, common.MIDFIELD, 10) // sent off earlier
	match.SentOff = map[int64]bool{10: true}
	playMinutes(match, 60, 90, nil)
	if len(*match.Events) != 0 || !match.HomeOrdersFired[0] || match.HomeSubstitutionsUsed != 0 {
		t.Errorf("events %+v used %d: want the order spent without a substitution", *match.Events, match.HomeSubstitutionsUsed)
	}
}

// Injury replacements, scheduled substitutions and substitution orders share one cap of 3.
func TestConditionalOrders_ShareTheSubstitutionCap(t *testing.T) {
	positions := map[int][]int64{common.GOALKEEPER: {1}, common.MIDFIELD: {10, 20, 30, 40}}
	scheduled := []common.LineupSubstitution{{PlayerOutId: 20, PlayerInId: 98, Minute: 50}}
	match := newSubMatch(0, positions, scheduled)
	for _, id := range []int64{97, 99} {
		(*match.HomePlayers)[id] = *namedPlayer(id)
	}
	match.HomeSubstitutionsUsed = 1 // an injury replacement earlier
	match.HomeOrders = []common.LineupOrder{
		{FromMinute: 60, Condition: common.ORDER_CONDITION_DRAWING, Action: common.ORDER_ACTION_SUBSTITUTION, PlayerOutId: 30, PlayerInId: 99},
		{FromMinute: 60, Condition: common.ORDER_CONDITION_DRAWING, Action: common.ORDER_ACTION_SUBSTITUTION, PlayerOutId: 40, PlayerInId: 97},
	}
	playMinutes(match, 1, 90, nil)
	subs := 0
	for _, e := range *match.Events {
		if e.EventType == common.SUBSTITUTION {
			subs++
		}
	}
	if subs != 2 || match.HomeSubstitutionsUsed != 3 {
		t.Errorf("%d substitutions, %d used: want the scheduled one and the first order (3 with the injury), not the second", subs, match.HomeSubstitutionsUsed)
	}
	if !match.HomeOrdersFired[1] {
		t.Error("the second order wasn't spent")
	}
}

func TestOrderConditionHolds(t *testing.T) {
	cases := []struct {
		condition, gf, ga int
		want              bool
	}{
		{common.ORDER_CONDITION_LOSING, 0, 1, true}, {common.ORDER_CONDITION_LOSING, 1, 1, false},
		{common.ORDER_CONDITION_LOSING_BY_2, 0, 1, false}, {common.ORDER_CONDITION_LOSING_BY_2, 1, 3, true},
		{common.ORDER_CONDITION_DRAWING, 2, 2, true}, {common.ORDER_CONDITION_DRAWING, 2, 1, false},
		{common.ORDER_CONDITION_WINNING, 1, 0, true}, {common.ORDER_CONDITION_WINNING, 0, 0, false},
		{common.ORDER_CONDITION_WINNING_BY_2, 2, 1, false}, {common.ORDER_CONDITION_WINNING_BY_2, 3, 1, true},
	}
	for _, c := range cases {
		if got := common.OrderConditionHolds(c.condition, c.gf, c.ga); got != c.want {
			t.Errorf("condition %d at %d-%d = %v, want %v", c.condition, c.gf, c.ga, got, c.want)
		}
	}
}

// Golden check (task 2.3): a side without orders gets exactly what the scheduled substitutions
// alone gave before orders existed - same pitch, same counter, same events, minute by minute.
func TestConditionalOrders_NoOrdersPlaysAsBefore(t *testing.T) {
	positions := func() map[int][]int64 { return map[int][]int64{common.GOALKEEPER: {1}, common.MIDFIELD: {10, 20, 30}} }
	scheduled := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 98, Minute: 55}, {PlayerOutId: 20, PlayerInId: 97, Minute: 70}}
	withStage := newSubMatch(0, positions(), scheduled)
	reference := newSubMatch(0, positions(), scheduled)
	for minute := 1; minute <= 90; minute++ {
		withStage.Minute, reference.Minute = minute, minute
		SubstitutionSimulation{}.Apply(nil, withStage)
		applyScheduledSubstitutions(reference, sideOf(reference, true))
		applyScheduledSubstitutions(reference, sideOf(reference, false))
		if minute == 30 {
			withStage.HomeGoals, reference.HomeGoals = 1, 1
		}
	}
	if len(*withStage.Events) != len(*reference.Events) || withStage.HomeSubstitutionsUsed != reference.HomeSubstitutionsUsed {
		t.Fatalf("events %+v / used %d, want %+v / %d", *withStage.Events, withStage.HomeSubstitutionsUsed, *reference.Events, reference.HomeSubstitutionsUsed)
	}
	for i := range *withStage.Events {
		a, b := (*withStage.Events)[i], (*reference.Events)[i]
		if a.EventType != b.EventType || a.Minute != b.Minute || a.TeamOwnerPlayers[0] != b.TeamOwnerPlayers[0] || a.TeamOwnerPlayers[1] != b.TeamOwnerPlayers[1] {
			t.Errorf("event %d: %+v, want %+v", i, a, b)
		}
	}
	if withStage.HomeMentality != common.MENTALITY_BALANCED {
		t.Errorf("mentality changed without orders")
	}
}

// CPU conditional orders (openspec/specs/cpu-team-management "CPU conditional orders").
func TestCpuConditionalOrders_DirectorProtectingALead(t *testing.T) {
	match := orderMatch(common.AiConditionalOrders(common.AiLevelDirector)...)
	playMinutes(match, 1, 74, map[int][2]int{30: {1, 0}})
	if match.HomeMentality != common.MENTALITY_BALANCED {
		t.Fatalf("switched before minute 75")
	}
	playMinutes(match, 75, 75, nil)
	if match.HomeMentality != common.MENTALITY_DEFENSIVE {
		t.Errorf("minute 75 leading 1-0: mentality %d, want Defensive", match.HomeMentality)
	}
}

func TestCpuConditionalOrders_ManagerChasingTheGame(t *testing.T) {
	match := orderMatch(common.AiConditionalOrders(common.AiLevelManager)...)
	playMinutes(match, 1, 79, map[int][2]int{79: {0, 1}})
	if match.HomeMentality != common.MENTALITY_BALANCED {
		t.Fatalf("switched while drawing")
	}
	playMinutes(match, 80, 80, nil)
	if e := mentalityEvents(match); match.HomeMentality != common.MENTALITY_ATTACKING || len(e) != 1 || e[0].Minute != 80 {
		t.Errorf("losing 0-1 at the start of 80: mentality %d events %+v, want Attacking at 80", match.HomeMentality, e)
	}
}
