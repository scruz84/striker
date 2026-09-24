package engine

import (
	"github.com/scruz84/striker/common"
	"testing"
)

func TestPickInjuryReplacement_PicksBestEligibleBenchPlayer(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {20}}
	players := map[int64]common.Player{
		20: *benchPlayer(20, 50),
		91: *benchPlayer(91, 80),
		92: *benchPlayer(92, 40),
	}
	match := &common.MatchSimulation{SentOff: map[int64]bool{}}

	replacement, found := pickInjuryReplacement(match, &positions, &players, map[int64]bool{}, common.MIDFIELD, 999)

	if !found || replacement != 91 {
		t.Fatalf("expected the stronger bench player 91, got %d (found=%v)", replacement, found)
	}
}

func TestPickInjuryReplacement_ExcludesPlayersCurrentlyOnPitch(t *testing.T) {
	positions := map[int][]int64{
		common.MIDFIELD: {20},
		common.STRIKER:  {91}, // fielded elsewhere - not a free bench player
	}
	players := map[int64]common.Player{
		20: *benchPlayer(20, 50),
		91: *benchPlayer(91, 80),
		92: *benchPlayer(92, 40),
	}
	match := &common.MatchSimulation{SentOff: map[int64]bool{}}

	replacement, found := pickInjuryReplacement(match, &positions, &players, map[int64]bool{}, common.MIDFIELD, 999)

	if !found || replacement != 92 {
		t.Fatalf("expected the only free bench player 92, got %d (found=%v)", replacement, found)
	}
}

func TestPickInjuryReplacement_ExcludesUnavailableSentOffAndAlreadyInjured(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {20}}
	players := map[int64]common.Player{
		20: *benchPlayer(20, 50),
		91: *benchPlayer(91, 90), // pre-existing suspension/injury
		92: *benchPlayer(92, 80), // sent off earlier this match
		93: *benchPlayer(93, 70), // injured earlier this match
		94: *benchPlayer(94, 30), // only genuinely eligible bench player
	}
	match := &common.MatchSimulation{
		SentOff: map[int64]bool{92: true},
		PendingUnavailabilities: []common.PendingUnavailability{
			{PlayerId: 93, Reason: common.UNAVAILABLE_INJURY, Matches: 2},
		},
	}
	unavailable := map[int64]bool{91: true}

	replacement, found := pickInjuryReplacement(match, &positions, &players, unavailable, common.MIDFIELD, 999)

	if !found || replacement != 94 {
		t.Fatalf("expected the only eligible bench player 94, got %d (found=%v)", replacement, found)
	}
}

func TestPickInjuryReplacement_NoEligiblePlayers(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {20}}
	players := map[int64]common.Player{20: *benchPlayer(20, 50)}
	match := &common.MatchSimulation{SentOff: map[int64]bool{}}

	_, found := pickInjuryReplacement(match, &positions, &players, map[int64]bool{}, common.MIDFIELD, 999)

	if found {
		t.Fatalf("expected no eligible replacement when the whole squad is already on the pitch")
	}
}

func TestBringOnInjuryReplacement_AddsReplacementAndCountsAsASubstitution(t *testing.T) {
	// Mirrors Apply()'s real ordering: the injured player (10) has already been removed from
	// their position group by the time this runs, leaving it short one.
	positions := map[int][]int64{common.MIDFIELD: {}}
	players := map[int64]common.Player{
		10: *benchPlayer(10, 50), // just injured, off the pitch
		91: *benchPlayer(91, 80), // eligible bench player
	}
	events := make([]common.Event, 0)
	match := &common.MatchSimulation{
		Minute:  30,
		Home:    1,
		Away:    2,
		Events:  &events,
		SentOff: map[int64]bool{},
	}
	used := 0

	bringOnInjuryReplacement(match, &positions, &players, map[int64]bool{}, &used, common.MIDFIELD, 10, match.Home, match.Away)

	got := positions[common.MIDFIELD]
	if len(got) != 1 || got[0] != 91 {
		t.Fatalf("expected replacement 91 on the pitch at midfield, got %v", got)
	}
	if used != 1 {
		t.Fatalf("expected substitutions used to be incremented to 1, got %d", used)
	}
	if len(*match.Events) != 1 || (*match.Events)[0].EventType != common.SUBSTITUTION {
		t.Fatalf("expected exactly one substitution event, got %v", *match.Events)
	}
}

func TestBringOnInjuryReplacement_DoesNothingWithNoSubstitutionsLeft(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {}}
	players := map[int64]common.Player{
		10: *benchPlayer(10, 50), // just injured, off the pitch
		91: *benchPlayer(91, 80), // would be eligible, but no subs remain
	}
	events := make([]common.Event, 0)
	match := &common.MatchSimulation{Minute: 30, Home: 1, Away: 2, Events: &events, SentOff: map[int64]bool{}}
	used := common.MaxScheduledSubstitutions

	bringOnInjuryReplacement(match, &positions, &players, map[int64]bool{}, &used, common.MIDFIELD, 10, match.Home, match.Away)

	if got := positions[common.MIDFIELD]; len(got) != 0 {
		t.Fatalf("expected no replacement fielded once substitutions are exhausted, got %v", got)
	}
	if len(*match.Events) != 0 {
		t.Fatalf("expected no event once substitutions are exhausted, got %v", *match.Events)
	}
}

func TestBringOnInjuryReplacement_DoesNothingWithNoEligibleBenchPlayer(t *testing.T) {
	positions := map[int][]int64{common.MIDFIELD: {}}
	players := map[int64]common.Player{10: *benchPlayer(10, 50)} // just injured, no one else in the squad
	events := make([]common.Event, 0)
	match := &common.MatchSimulation{Minute: 30, Home: 1, Away: 2, Events: &events, SentOff: map[int64]bool{}}
	used := 0

	bringOnInjuryReplacement(match, &positions, &players, map[int64]bool{}, &used, common.MIDFIELD, 10, match.Home, match.Away)

	if got := positions[common.MIDFIELD]; len(got) != 0 {
		t.Fatalf("expected no replacement when the squad has nobody eligible, got %v", got)
	}
	if used != 0 {
		t.Fatalf("expected substitutions used unchanged when nobody comes on, got %d", used)
	}
	if len(*match.Events) != 0 {
		t.Fatalf("expected no event when nobody comes on, got %v", *match.Events)
	}
}
