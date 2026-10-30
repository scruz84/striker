package engine

import (
	"github.com/scruz84/striker/common"
	"testing"
)

// injuryMatch builds a home-side match around a squad and the given pitch positions, the way
// runMatchSimulation would have left it just before an injury is handled.
func injuryMatch(positions map[int][]int64, players map[int64]common.Player) (*common.MatchSimulation, teamSide) {
	events := make([]common.Event, 0)
	match := &common.MatchSimulation{
		Minute:                30,
		Home:                  1,
		Away:                  2,
		Events:                &events,
		SentOff:               map[int64]bool{},
		HomePlayersByPosition: &positions,
		HomePlayers:           &players,
	}
	return match, sideOf(match, true)
}

func TestBestEligibleBenchPlayer_PicksBestEligibleBenchPlayer(t *testing.T) {
	match, side := injuryMatch(map[int][]int64{common.MIDFIELD: {20}}, map[int64]common.Player{
		20: *benchPlayer(20, 50),
		91: *benchPlayer(91, 80),
		92: *benchPlayer(92, 40),
	})

	replacement, found := bestEligibleBenchPlayer(match, side, common.MIDFIELD, 999)

	if !found || replacement != 91 {
		t.Fatalf("expected the stronger bench player 91, got %d (found=%v)", replacement, found)
	}
}

func TestBestEligibleBenchPlayer_ExcludesPlayersCurrentlyOnPitch(t *testing.T) {
	match, side := injuryMatch(map[int][]int64{
		common.MIDFIELD: {20},
		common.STRIKER:  {91}, // fielded elsewhere - not a free bench player
	}, map[int64]common.Player{
		20: *benchPlayer(20, 50),
		91: *benchPlayer(91, 80),
		92: *benchPlayer(92, 40),
	})

	replacement, found := bestEligibleBenchPlayer(match, side, common.MIDFIELD, 999)

	if !found || replacement != 92 {
		t.Fatalf("expected the only free bench player 92, got %d (found=%v)", replacement, found)
	}
}

func TestBestEligibleBenchPlayer_ExcludesUnavailableSentOffInjuredAndSubstitutedOff(t *testing.T) {
	match, side := injuryMatch(map[int][]int64{common.MIDFIELD: {20}}, map[int64]common.Player{
		20: *benchPlayer(20, 50),
		91: *benchPlayer(91, 90), // pre-existing suspension/injury
		92: *benchPlayer(92, 80), // sent off earlier this match
		93: *benchPlayer(93, 70), // injured earlier this match
		95: *benchPlayer(95, 60), // substituted off earlier this match
		94: *benchPlayer(94, 30), // only genuinely eligible bench player
	})
	match.SentOff = map[int64]bool{92: true}
	match.PendingUnavailabilities = []common.PendingUnavailability{
		{PlayerId: 93, Reason: common.UNAVAILABLE_INJURY, Matches: 2},
	}
	match.HomeUnavailablePlayers = map[int64]bool{91: true}
	match.HomeSubstitutedOff = map[int64]bool{95: true}
	side = sideOf(match, true)

	replacement, found := bestEligibleBenchPlayer(match, side, common.MIDFIELD, 999)

	if !found || replacement != 94 {
		t.Fatalf("expected the only eligible bench player 94, got %d (found=%v)", replacement, found)
	}
}

func TestBestEligibleBenchPlayer_NoEligiblePlayers(t *testing.T) {
	match, side := injuryMatch(map[int][]int64{common.MIDFIELD: {20}}, map[int64]common.Player{20: *benchPlayer(20, 50)})

	_, found := bestEligibleBenchPlayer(match, side, common.MIDFIELD, 999)

	if found {
		t.Fatalf("expected no eligible replacement when the whole squad is already on the pitch")
	}
}

func TestBestEligibleBenchPlayer_TiesBreakOnLowestId(t *testing.T) {
	match, side := injuryMatch(map[int][]int64{common.MIDFIELD: {}}, map[int64]common.Player{
		93: *benchPlayer(93, 60),
		91: *benchPlayer(91, 60),
		92: *benchPlayer(92, 60),
	})

	for i := 0; i < 30; i++ {
		if replacement, _ := bestEligibleBenchPlayer(match, side, common.MIDFIELD); replacement != 91 {
			t.Fatalf("expected equal-value ties to resolve to the lowest id (91), got %d", replacement)
		}
	}
}

func TestBringOnInjuryReplacement_AddsReplacementAndCountsAsASubstitution(t *testing.T) {
	// Mirrors Apply()'s real ordering: the injured player (10) has already been removed from
	// their position group by the time this runs, leaving it short one.
	match, side := injuryMatch(map[int][]int64{common.MIDFIELD: {}}, map[int64]common.Player{
		10: *benchPlayer(10, 50), // just injured, off the pitch
		91: *benchPlayer(91, 80), // eligible bench player
	})

	bringOnInjuryReplacement(match, side, common.MIDFIELD, 10)

	got := (*match.HomePlayersByPosition)[common.MIDFIELD]
	if len(got) != 1 || got[0] != 91 {
		t.Fatalf("expected replacement 91 on the pitch at midfield, got %v", got)
	}
	if match.HomeSubstitutionsUsed != 1 {
		t.Fatalf("expected substitutions used to be incremented to 1, got %d", match.HomeSubstitutionsUsed)
	}
	if len(*match.Events) != 1 || (*match.Events)[0].EventType != common.SUBSTITUTION {
		t.Fatalf("expected exactly one substitution event, got %v", *match.Events)
	}
}

func TestBringOnInjuryReplacement_DoesNothingWithNoSubstitutionsLeft(t *testing.T) {
	match, side := injuryMatch(map[int][]int64{common.MIDFIELD: {}}, map[int64]common.Player{
		10: *benchPlayer(10, 50), // just injured, off the pitch
		91: *benchPlayer(91, 80), // would be eligible, but no subs remain
	})
	match.HomeSubstitutionsUsed = common.MaxScheduledSubstitutions

	bringOnInjuryReplacement(match, side, common.MIDFIELD, 10)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 0 {
		t.Fatalf("expected no replacement fielded once substitutions are exhausted, got %v", got)
	}
	if len(*match.Events) != 0 {
		t.Fatalf("expected no event once substitutions are exhausted, got %v", *match.Events)
	}
}

func TestBringOnInjuryReplacement_DoesNothingWithNoEligibleBenchPlayer(t *testing.T) {
	// just injured, no one else in the squad
	match, side := injuryMatch(map[int][]int64{common.MIDFIELD: {}}, map[int64]common.Player{10: *benchPlayer(10, 50)})

	bringOnInjuryReplacement(match, side, common.MIDFIELD, 10)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 0 {
		t.Fatalf("expected no replacement when the squad has nobody eligible, got %v", got)
	}
	if match.HomeSubstitutionsUsed != 0 {
		t.Fatalf("expected substitutions used unchanged when nobody comes on, got %d", match.HomeSubstitutionsUsed)
	}
	if len(*match.Events) != 0 {
		t.Fatalf("expected no event when nobody comes on, got %v", *match.Events)
	}
}

func TestBringOnInjuryReplacement_BestOptionAlreadyCameOff(t *testing.T) {
	// Player 91, the best midfielder on the bench, came on at minute 60 and was substituted off
	// again; player 92 is the next best. The injury at minute 80 must bring on 92, not 91.
	match, side := injuryMatch(map[int][]int64{common.MIDFIELD: {}}, map[int64]common.Player{
		10: *benchPlayer(10, 50), // just injured
		91: *benchPlayer(91, 90), // best option, already substituted off
		92: *benchPlayer(92, 60),
		93: *benchPlayer(93, 30),
	})
	match.Minute = 80
	match.HomeSubstitutionsUsed = 1
	match.HomeSubstitutedOff = map[int64]bool{91: true}
	side = sideOf(match, true)

	bringOnInjuryReplacement(match, side, common.MIDFIELD, 10)

	if got := (*match.HomePlayersByPosition)[common.MIDFIELD]; len(got) != 1 || got[0] != 92 {
		t.Fatalf("expected the next best eligible player 92, got %v", got)
	}
	if !match.HomeSubstitutedOff[10] {
		t.Fatalf("expected the injured player to be marked substituted off")
	}
}
