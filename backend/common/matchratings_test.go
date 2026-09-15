package common

import "testing"

func lineupOf(playerIds ...int64) *Lineup {
	positions := make([]LineupPosition, len(playerIds))
	for i, id := range playerIds {
		positions[i] = LineupPosition{PlayerId: id, Position: i}
	}
	return &Lineup{Positions: positions}
}

func TestComputeMatchRatings_NilWithoutLineupsOrEvents(t *testing.T) {
	match := &WeekMatch{MatchId: 1, Events: &[]Event{}}
	if ComputeMatchRatings(match) != nil {
		t.Fatal("expected nil without both lineups")
	}

	match = &WeekMatch{MatchId: 1, HomeLineup: lineupOf(1), AwayLineup: lineupOf(2)}
	if ComputeMatchRatings(match) != nil {
		t.Fatal("expected nil without events")
	}
}

func TestComputeMatchRatings_GoalsAndCardsMoveRatingInTheExpectedDirection(t *testing.T) {
	match := &WeekMatch{
		MatchId:    1,
		HomeGoals:  2,
		AwayGoals:  0,
		HomeLineup: lineupOf(1, 2),
		AwayLineup: lineupOf(3, 4),
		Events: &[]Event{
			{EventType: GOAL, TeamOwnerPlayers: []int64{1}},
			{EventType: YELLOW_CARD, TeamOwnerPlayers: []int64{3}},
			{EventType: RED_CARD, TeamOwnerPlayers: []int64{4}},
		},
	}

	ratings := ComputeMatchRatings(match)
	if ratings == nil {
		t.Fatal("expected ratings, got nil")
	}
	r := *ratings

	if r[1] <= r[2] {
		t.Fatalf("scorer (player 1) should rate higher than a teammate who didn't feature in any event: %v vs %v", r[1], r[2])
	}
	if r[3] >= matchRatingBase {
		t.Fatalf("yellow-carded player should rate below base %v, got %v", matchRatingBase, r[3])
	}
	if r[4] >= r[3] {
		t.Fatalf("red-carded player (4) should rate lower than yellow-carded player (3): %v vs %v", r[4], r[3])
	}
	// Home won 2-0 - home players should trend higher than the away loser's teammate who had no
	// events at all (player 2 vs a hypothetical away equivalent isn't directly comparable here
	// since both away players had card events, but the losing side's baseline should still sit
	// below the winning side's untouched player).
	if r[2] <= matchRatingBase {
		t.Fatalf("winning team's uninvolved player should rate above base %v, got %v", matchRatingBase, r[2])
	}
}

func TestComputeMatchRatings_ClampedToValidRange(t *testing.T) {
	events := make([]Event, 0, 10)
	for i := 0; i < 10; i++ {
		events = append(events, Event{EventType: RED_CARD, TeamOwnerPlayers: []int64{1}})
	}
	match := &WeekMatch{
		MatchId:    1,
		HomeLineup: lineupOf(1),
		AwayLineup: lineupOf(2),
		Events:     &events,
	}

	ratings := ComputeMatchRatings(match)
	r := *ratings
	if r[1] < matchRatingMin || r[1] > matchRatingMax {
		t.Fatalf("rating %v out of [%v, %v] range", r[1], matchRatingMin, matchRatingMax)
	}
}

func TestComputeMatchRatings_DeterministicAcrossCalls(t *testing.T) {
	match := &WeekMatch{
		MatchId:    42,
		HomeGoals:  1,
		AwayGoals:  1,
		HomeLineup: lineupOf(10, 11),
		AwayLineup: lineupOf(20, 21),
		Events: &[]Event{
			{EventType: GOAL, TeamOwnerPlayers: []int64{10}},
			{EventType: GOAL, TeamOwnerPlayers: []int64{20}},
		},
	}

	first := *ComputeMatchRatings(match)
	second := *ComputeMatchRatings(match)
	for playerId, rating := range first {
		if second[playerId] != rating {
			t.Fatalf("rating for player %d changed between calls: %v vs %v", playerId, rating, second[playerId])
		}
	}
}

func TestComputeMatchRatings_OnlyRatesPlayersInALineup(t *testing.T) {
	match := &WeekMatch{
		MatchId:    1,
		HomeLineup: lineupOf(1),
		AwayLineup: lineupOf(2),
		Events: &[]Event{
			// Player 99 never appears in either lineup (e.g. bad data) - must not create a
			// spurious rating entry for them.
			{EventType: GOAL, TeamOwnerPlayers: []int64{99}},
		},
	}

	ratings := *ComputeMatchRatings(match)
	if _, ok := ratings[99]; ok {
		t.Fatal("expected no rating entry for a player not in either lineup")
	}
	if len(ratings) != 2 {
		t.Fatalf("expected exactly 2 rated players, got %d", len(ratings))
	}
}
