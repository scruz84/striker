package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "Director cup orders": only a Director's cup tie gains the drawing order,
// after its usual two orders.
func TestSetAiKickoffTactics_CupDrawOrder(t *testing.T) {
	cases := []struct {
		name      string
		level     int
		kind      kickoffKind
		own, opp  int
		wantOrder bool
		mentality int
	}{
		{"favourite level late", common.AiLevelDirector, kickoffCup, 1050, 1000, true, common.MENTALITY_ATTACKING},
		{"underdog level late", common.AiLevelDirector, kickoffCup, 950, 1000, true, common.MENTALITY_DEFENSIVE},
		{"league match", common.AiLevelDirector, kickoffLeague, 1050, 1000, false, 0},
		{"manager cup tie", common.AiLevelManager, kickoffCup, 1050, 1000, false, 0},
	}
	for _, c := range cases {
		lineup := &common.Lineup{Orders: common.AiConditionalOrders(c.level)}
		before := len(lineup.Orders)
		setAiKickoffTactics(c.level, c.kind, lineup, c.own, c.opp, true)
		if !c.wantOrder {
			if len(lineup.Orders) != before {
				t.Errorf("%s: orders %+v, want the level's usual %d", c.name, lineup.Orders, before)
			}
			continue
		}
		if len(lineup.Orders) != before+1 {
			t.Fatalf("%s: %d orders, want %d", c.name, len(lineup.Orders), before+1)
		}
		last := lineup.Orders[len(lineup.Orders)-1]
		if last.FromMinute != 75 || last.Condition != common.ORDER_CONDITION_DRAWING || last.Mentality != c.mentality {
			t.Errorf("%s: drawing order %+v, want from 75 drawing mentality %d", c.name, last, c.mentality)
		}
		if err := common.ValidateOrders(nil, lineup.Orders, &[]*common.Player{}, nil); err != nil {
			t.Errorf("%s: orders fail validation: %s", c.name, err)
		}
	}
}

// The drawing order needs a free slot: a lineup already at MaxConditionalOrders keeps its orders.
func TestSetAiKickoffTactics_NoRoomForTheDrawingOrder(t *testing.T) {
	full := make([]common.LineupOrder, common.MaxConditionalOrders)
	lineup := &common.Lineup{Orders: full}
	setAiKickoffTactics(common.AiLevelDirector, kickoffCup, lineup, 1050, 1000, true)
	if len(lineup.Orders) != common.MaxConditionalOrders {
		t.Errorf("%d orders, want %d", len(lineup.Orders), common.MaxConditionalOrders)
	}
}
