package common

import (
	"reflect"
	"testing"
)

type eventRow = struct {
	EventType       int
	TeamPlayersJson string
}

func TestTallySeasonAwardScores(t *testing.T) {
	events := []eventRow{
		{GOAL, "[1]"},
		{GOAL, "[1]"},
		{YELLOW_CARD, "[1]"},
		{GOAL, "[2]"},
		{RED_CARD, "[2]"},
		{INJURY, "[3]"}, // ignored - not a weighted event type
		{GOAL, ""},      // malformed - skipped
	}

	scores := tallySeasonAwardScores(events)

	want := map[int64]int{
		1: 3 + 3 - 1, // two goals, one yellow
		2: 3 - 3,     // one goal, one red
	}
	if !reflect.DeepEqual(scores, want) {
		t.Errorf("tallySeasonAwardScores() = %v, want %v", scores, want)
	}
}

func TestRankSeasonAwardScores(t *testing.T) {
	scores := map[int64]int{1: 5, 2: 5, 3: 0, 4: -1, 5: 8}
	ranked := rankSeasonAwardScores(scores)
	want := []int64{5, 1, 2} // 8, then tie at 5 broken by lower id; 0 and negative excluded
	if !reflect.DeepEqual(ranked, want) {
		t.Errorf("rankSeasonAwardScores() = %v, want %v", ranked, want)
	}
}

func TestRankSeasonAwardScoresNoQualifiers(t *testing.T) {
	scores := map[int64]int{1: 0, 2: -2}
	if ranked := rankSeasonAwardScores(scores); len(ranked) != 0 {
		t.Errorf("expected no qualifiers, got %v", ranked)
	}
}
