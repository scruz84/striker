package common

import (
	"math"
	"testing"
)

// observability "CPU level gauges": "Two Directors" and "Owned club".
func TestFoldAiLevelStats(t *testing.T) {
	stats := foldAiLevelStats([]aiLevelRow{
		// Two Director rows as GROUP BY would never split them, to check they add up.
		{level: AiLevelDirector, teams: 1, budget: 300000, points: 10, matches: 5},
		{level: AiLevelDirector, teams: 1, budget: -250000, insolvent: 1, points: 4, matches: 5},
		// An owned club that still carries a Director level counts as human.
		{level: AiLevelDirector, owned: true, teams: 1, budget: 1000, points: 3, matches: 1},
		// An unowned club with no level is not managed at all.
		{level: 0, teams: 5, budget: 999},
	})
	if len(stats) != 4 || stats[0].Level != AiLevelBucketHuman || stats[3].Level != AiLevelBucketDirector {
		t.Fatalf("buckets %+v, want human, caretaker, manager, director", stats)
	}
	director := stats[3]
	if director.Teams != 2 || director.AverageBudget != 25000 || director.Insolvent != 1 || math.Abs(director.PointsPerMatch-1.4) > 1e-9 {
		t.Errorf("director %+v, want 2 clubs, 25,000 average, 1 insolvent, 1.4 points per match", director)
	}
	if human := stats[0]; human.Teams != 1 || human.PointsPerMatch != 3 {
		t.Errorf("human %+v, want the owned club", human)
	}
	if caretaker := stats[1]; caretaker.Teams != 0 || caretaker.AverageBudget != 0 || caretaker.PointsPerMatch != 0 {
		t.Errorf("empty caretaker bucket %+v, want zeros", caretaker)
	}
}
