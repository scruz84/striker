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

// addBenchPlayers puts extra squad members (not on the pitch) into the match's HomePlayers, with
// the given value at every position.
func addBenchPlayers(match *common.MatchSimulation, skills map[int64]int) {
	for id, skill := range skills {
		p := benchPlayer(id, skill)
		p.FirstName, p.LastName = "P", "layer"
		(*match.HomePlayers)[id] = *p
	}
}

func TestSubstitutionSimulation_CapReachedByInjuriesSkipsTheScheduledChange(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {10}}
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 60}}
	match := newSubMatch(60, positions, subs)
	match.HomeSubstitutionsUsed = common.MaxScheduledSubstitutions // three injury replacements already

	SubstitutionSimulation{}.Apply(nil, match)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 1 || got[0] != 10 {
		t.Fatalf("expected no change once the cap is reached, got %v", got)
	}
	if match.HomeSubstitutionsUsed != common.MaxScheduledSubstitutions || len(*match.Events) != 0 {
		t.Fatalf("expected no extra substitution, used=%d events=%v", match.HomeSubstitutionsUsed, *match.Events)
	}
}

func TestSubstitutionSimulation_UnavailableIncomingPlayerIsReplacedByTheBestEligibleOne(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {10}}
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 65}}
	match := newSubMatch(65, positions, subs)
	addBenchPlayers(match, map[int64]int{91: 80, 92: 40})
	match.HomeUnavailablePlayers = map[int64]bool{99: true} // got injured since the lineup was saved

	SubstitutionSimulation{}.Apply(nil, match)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 1 || got[0] != 91 {
		t.Fatalf("expected the best eligible bench player 91 to come on, got %v", got)
	}
	if match.HomeSubstitutionsUsed != 1 || len(*match.Events) != 1 {
		t.Fatalf("expected one logged substitution, used=%d events=%v", match.HomeSubstitutionsUsed, *match.Events)
	}
	if got := (*match.Events)[0].TeamOwnerPlayers; got[0] != 10 || got[1] != 91 {
		t.Fatalf("expected the event to name 10 out and 91 in, got %v", got)
	}
	if !match.HomeSubstitutedOff[10] {
		t.Fatalf("expected the outgoing player to be marked substituted off")
	}
}

func TestSubstitutionSimulation_IncomingPlayerAlreadySubstitutedOffIsReplaced(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {10}}
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 70}}
	match := newSubMatch(70, positions, subs)
	addBenchPlayers(match, map[int64]int{91: 50})
	match.HomeSubstitutedOff = map[int64]bool{99: true}

	SubstitutionSimulation{}.Apply(nil, match)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 1 || got[0] != 91 {
		t.Fatalf("expected 91 on instead of the substituted-off 99, got %v", got)
	}
}

func TestSubstitutionSimulation_NoEligibleBenchPlayerSkipsWithoutCounting(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {10}}
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 65}}
	match := newSubMatch(65, positions, subs)
	match.HomeUnavailablePlayers = map[int64]bool{99: true} // and nobody else is on the bench

	SubstitutionSimulation{}.Apply(nil, match)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 1 || got[0] != 10 {
		t.Fatalf("expected lineup unchanged, got %v", got)
	}
	if match.HomeSubstitutionsUsed != 0 || len(*match.Events) != 0 {
		t.Fatalf("expected the skipped change not to count, used=%d events=%v", match.HomeSubstitutionsUsed, *match.Events)
	}
}

func TestSubstitutionSimulation_OutgoingPlayerAlreadyOffDoesNotCount(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {20}} // 10 was sent off earlier
	subs := []common.LineupSubstitution{{PlayerOutId: 10, PlayerInId: 99, Minute: 60}}
	match := newSubMatch(60, positions, subs)
	match.SentOff = map[int64]bool{10: true}

	SubstitutionSimulation{}.Apply(nil, match)

	if match.HomeSubstitutionsUsed != 0 || len(*match.Events) != 0 {
		t.Fatalf("expected a skipped change not to count, used=%d events=%v", match.HomeSubstitutionsUsed, *match.Events)
	}
}

func TestSubstitutionSimulation_ThreeScheduledChangesAfterAnEarlyInjuryOnlyFireTwo(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {10, 11, 12}}
	subs := []common.LineupSubstitution{
		{PlayerOutId: 10, PlayerInId: 91, Minute: 60},
		{PlayerOutId: 11, PlayerInId: 92, Minute: 60},
		{PlayerOutId: 12, PlayerInId: 93, Minute: 60},
	}
	match := newSubMatch(60, positions, subs)
	match.HomeSubstitutionsUsed = 1 // an injury replacement already happened

	SubstitutionSimulation{}.Apply(nil, match)

	if match.HomeSubstitutionsUsed != common.MaxScheduledSubstitutions {
		t.Fatalf("expected the team to stop at %d changes, got %d", common.MaxScheduledSubstitutions, match.HomeSubstitutionsUsed)
	}
	if len(*match.Events) != 2 {
		t.Fatalf("expected exactly two scheduled changes to fire, got %v", *match.Events)
	}
}
