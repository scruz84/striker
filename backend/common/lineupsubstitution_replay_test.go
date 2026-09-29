package common

import "testing"

func TestCopyLineup_NilIsNil(t *testing.T) {
	if got := CopyLineup(nil); got != nil {
		t.Fatalf("expected CopyLineup(nil) to return nil, got %v", got)
	}
}

func TestCopyLineup_MutatingTheCopyLeavesTheOriginalAlone(t *testing.T) {
	original := &Lineup{
		TeamId:    1,
		Positions: []LineupPosition{{PlayerId: 10, Position: DEFENSE}},
	}

	cp := CopyLineup(original)
	cp.Positions[0].PlayerId = 99

	if original.Positions[0].PlayerId != 10 {
		t.Fatalf("expected the original lineup's position to stay 10, got %v", original.Positions[0].PlayerId)
	}
	if cp.Positions[0].PlayerId != 99 {
		t.Fatalf("expected the copy's position to be 99, got %v", cp.Positions[0].PlayerId)
	}
}

func substitutionEvent(teamOwner int64, outId int64, inId int64) Event {
	return Event{
		EventType:        SUBSTITUTION,
		TeamOwner:        teamOwner,
		TeamOwnerPlayers: []int64{outId, inId},
	}
}

func TestApplySubstitutionToLineup_SwapsTheOutgoingPlayerForTheIncomingOne(t *testing.T) {
	lineup := &Lineup{
		TeamId: 1,
		Positions: []LineupPosition{
			{PlayerId: 10, Position: DEFENSE},
			{PlayerId: 20, Position: MIDFIELD},
		},
	}

	ApplySubstitutionToLineup(lineup, substitutionEvent(1, 20, 21))

	if lineup.Positions[0].PlayerId != 10 {
		t.Fatalf("expected the untouched slot to stay 10, got %v", lineup.Positions[0].PlayerId)
	}
	if lineup.Positions[1].PlayerId != 21 {
		t.Fatalf("expected the substituted slot to become 21, got %v", lineup.Positions[1].PlayerId)
	}
}

func TestApplySubstitutionToLineup_IgnoresTheOtherTeamsLineup(t *testing.T) {
	awayLineup := &Lineup{
		TeamId:    2,
		Positions: []LineupPosition{{PlayerId: 20, Position: MIDFIELD}},
	}

	// A home-team substitution event should never touch the away team's lineup, even if (as
	// here) their player IDs happen not to collide - TeamId is what decides applicability.
	ApplySubstitutionToLineup(awayLineup, substitutionEvent(1, 20, 21))

	if awayLineup.Positions[0].PlayerId != 20 {
		t.Fatalf("expected the other team's lineup to stay untouched, got %v", awayLineup.Positions[0].PlayerId)
	}
}

func TestApplySubstitutionToLineup_IgnoresNonSubstitutionEvents(t *testing.T) {
	lineup := &Lineup{TeamId: 1, Positions: []LineupPosition{{PlayerId: 10, Position: DEFENSE}}}

	ApplySubstitutionToLineup(lineup, Event{EventType: GOAL, TeamOwner: 1, TeamOwnerPlayers: []int64{10, 99}})

	if lineup.Positions[0].PlayerId != 10 {
		t.Fatalf("expected a non-substitution event to leave the lineup untouched, got %v", lineup.Positions[0].PlayerId)
	}
}

func TestApplySubstitutionToLineup_IgnoresAnUnknownOutgoingPlayer(t *testing.T) {
	lineup := &Lineup{TeamId: 1, Positions: []LineupPosition{{PlayerId: 10, Position: DEFENSE}}}

	// The outgoing player isn't in this lineup at all (e.g. already substituted, or data drift) -
	// should be a no-op, not a panic or a spurious add.
	ApplySubstitutionToLineup(lineup, substitutionEvent(1, 999, 21))

	if len(lineup.Positions) != 1 || lineup.Positions[0].PlayerId != 10 {
		t.Fatalf("expected the lineup to be untouched, got %+v", lineup.Positions)
	}
}

func TestApplySubstitutionToLineup_NilLineupDoesNotPanic(t *testing.T) {
	ApplySubstitutionToLineup(nil, substitutionEvent(1, 10, 21))
}

// A chain of substitutions replayed in order should each apply against whatever the previous one
// left behind - the same way WeekMatchById replays a whole match's event history in one pass.
func TestApplySubstitutionToLineup_ChainedSubstitutionsApplyInOrder(t *testing.T) {
	lineup := &Lineup{TeamId: 1, Positions: []LineupPosition{{PlayerId: 10, Position: STRIKER}}}

	ApplySubstitutionToLineup(lineup, substitutionEvent(1, 10, 11))
	ApplySubstitutionToLineup(lineup, substitutionEvent(1, 11, 12))

	if lineup.Positions[0].PlayerId != 12 {
		t.Fatalf("expected the final substitution's incoming player (12), got %v", lineup.Positions[0].PlayerId)
	}
}
