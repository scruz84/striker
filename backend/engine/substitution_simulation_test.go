package engine

import (
	"github.com/scruz84/striker/common"
	"testing"
)

func namedPlayer(id int64) *common.Player {
	return &common.Player{Id: id, FirstName: "P", LastName: "layer"}
}

func newSubMatch(minute int, positions map[int][]int64, subs []common.LineupSubstitution) *common.MatchSimulation {
	events := make([]common.Event, 0)
	players := map[int64]common.Player{}
	for _, ids := range positions {
		for _, id := range ids {
			players[id] = *namedPlayer(id)
		}
	}
	// The incoming player of every scheduled substitution must also be known (they're on the
	// bench, not yet in positions) - mirrors match.HomePlayers/AwayPlayers holding the whole
	// squad, not just the starters (see engine.playersToMapById).
	for _, sub := range subs {
		if _, ok := players[sub.PlayerInId]; !ok {
			players[sub.PlayerInId] = *namedPlayer(sub.PlayerInId)
		}
	}
	return &common.MatchSimulation{
		Minute:                minute,
		Home:                  1,
		Away:                  2,
		HomePlayersByPosition: &positions,
		HomePlayers:           &players,
		HomeSubstitutions:     subs,
		Events:                &events,
	}
}

func TestSubstitutionSimulation_SwapsPlayerAtScheduledMinute(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {10, 20}}
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 60}}
	match := newSubMatch(60, positions, subs)

	SubstitutionSimulation{}.Apply(nil, match)

	got := (*match.HomePlayersByPosition)[common.MIDFIELD]
	if len(got) != 2 || got[0] == 10 {
		t.Fatalf("expected player 10 replaced by 99 in midfield, got %v", got)
	}
	found := false
	for _, id := range got {
		if id == 99 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected incoming player 99 on the pitch, got %v", got)
	}
	if len(*match.Events) != 1 || (*match.Events)[0].EventType != common.SUBSTITUTION {
		t.Fatalf("expected exactly one substitution event, got %v", *match.Events)
	}
}

func TestSubstitutionSimulation_DoesNothingBeforeItsScheduledMinute(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {10}}
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 60}}
	match := newSubMatch(59, positions, subs)

	SubstitutionSimulation{}.Apply(nil, match)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 1 || got[0] != 10 {
		t.Fatalf("expected no change before minute 60, got %v", got)
	}
	if len(*match.Events) != 0 {
		t.Fatalf("expected no event before minute 60, got %v", *match.Events)
	}
}

func TestSubstitutionSimulation_SkipsWhenOutgoingPlayerAlreadyOffThePitch(t *testing.T) {
	// Player 10 was already sent off or injured earlier, so they're no longer in any position -
	// the scheduled substitution must be a harmless no-op, not a panic or a phantom entry.
	positions := map[int][]int64{common.MIDFIELD: {20}}
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 60}}
	match := newSubMatch(60, positions, subs)

	SubstitutionSimulation{}.Apply(nil, match)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 1 || got[0] != 20 {
		t.Fatalf("expected lineup unchanged when the outgoing player is already off the pitch, got %v", got)
	}
	if len(*match.Events) != 0 {
		t.Fatalf("expected no event when the substitution can't happen, got %v", *match.Events)
	}
}

func TestSubstitutionSimulation_SkipsWhenIncomingPlayerAlreadyOnPitch(t *testing.T) {
	// Player 99 (meant to come on) is somehow already fielded elsewhere (e.g. auto-backfilled
	// into another position pre-kickoff) - bringing them on again would duplicate them.
	positions := map[int][]int64{
		common.MIDFIELD: {10},
		common.STRIKER:  {99},
	}
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 60}}
	match := newSubMatch(60, positions, subs)

	SubstitutionSimulation{}.Apply(nil, match)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 1 || got[0] != 10 {
		t.Fatalf("expected midfield unchanged, got %v", got)
	}
	if got := (*match.HomePlayersByPosition)[common.STRIKER]; len(got) != 1 || got[0] != 99 {
		t.Fatalf("expected striker unchanged, got %v", got)
	}
	if len(*match.Events) != 0 {
		t.Fatalf("expected no event when the incoming player is already on the pitch, got %v", *match.Events)
	}
}

func TestSubstitutionSimulation_AppliesMultipleScheduledSubstitutionsInTheSameMinute(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {10, 20}}
	subs := []common.LineupSubstitution{
		{PlayerOutId: 10, PlayerInId: 91, Minute: 46},
		{PlayerOutId: 20, PlayerInId: 92, Minute: 46},
	}
	match := newSubMatch(46, positions, subs)

	SubstitutionSimulation{}.Apply(nil, match)

	got := (*match.HomePlayersByPosition)[common.MIDFIELD]
	if len(got) != 2 {
		t.Fatalf("expected midfield to still have exactly 2 players, got %v", got)
	}
	for _, id := range got {
		if id == 10 || id == 20 {
			t.Fatalf("expected both original starters replaced, got %v", got)
		}
	}
	if len(*match.Events) != 2 {
		t.Fatalf("expected two substitution events, got %v", *match.Events)
	}
}
