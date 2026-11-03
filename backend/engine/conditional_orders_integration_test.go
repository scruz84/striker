package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// End to end (openspec add-conditional-match-orders; needs STRIKER_TEST_DB_DSN): a CPU Director's
// kickoff snapshot stores its orders, a human's saved orders are kept, a fired mentality order is
// stored as a MENTALITY_CHANGE event with its mentality, and nobody's orders show in the played
// match's lineups.
func TestSimulateMatch_OrdersFireAndAreStoredWithTheSnapshot(t *testing.T) {
	e := newSimEnv(t, "ko1")
	e.exec(`UPDATE teams SET ai_level = $2, owner_id = NULL WHERE team_id = $1`, e.home, common.AiLevelDirector)
	matchId := e.leagueMatch(0)

	// The away side kicks off Defensive and goes Attacking from minute 1 while drawing - which it
	// is at the start of minute 1, so the order fires straight away.
	lineups, err := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatal(err)
	}
	away := (*lineups)[e.away]
	humanOrders := []common.LineupOrder{{FromMinute: 1, Condition: common.ORDER_CONDITION_DRAWING, Action: common.ORDER_ACTION_MENTALITY, Mentality: common.MENTALITY_ATTACKING}}
	if err := common.SetLineupForMatch(e.dbCtx, matchId, e.away, away.Formation, common.MENTALITY_DEFENSIVE, away.Positions, nil, humanOrders); err != nil {
		t.Fatal(err)
	}

	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}

	cpuOrders, err := common.GetMatchLineupOrders(e.dbCtx, matchId, e.home)
	if err != nil || len(cpuOrders) != 2 {
		t.Fatalf("Director snapshot orders = %+v (%v), want its 2", cpuOrders, err)
	}
	if kept, err := common.GetMatchLineupOrders(e.dbCtx, matchId, e.away); err != nil || len(kept) != 1 || kept[0] != humanOrders[0] {
		t.Errorf("human orders after the match = %+v (%v), want them kept", kept, err)
	}
	events := *common.GetMatchEvents(e.dbCtx, matchId)
	var change *common.Event
	for i := range events {
		if events[i].EventType == common.MENTALITY_CHANGE && events[i].TeamOwner == e.away {
			change = &events[i]
		}
	}
	if change == nil || change.Minute != 1 || change.Mentality == nil || *change.Mentality != common.MENTALITY_ATTACKING {
		t.Fatalf("stored away MENTALITY_CHANGE = %+v, want Attacking at minute 1", change)
	}
	for _, ev := range events {
		if ev.EventType != common.MENTALITY_CHANGE && ev.Mentality != nil {
			t.Errorf("a %d event carries a mentality", ev.EventType)
		}
	}

	stored, err := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatal(err)
	}
	if len((*stored)[e.home].Orders) != 0 || len((*stored)[e.away].Orders) != 0 {
		t.Errorf("the played match's lineups carry orders: %+v / %+v", (*stored)[e.home].Orders, (*stored)[e.away].Orders)
	}
}
