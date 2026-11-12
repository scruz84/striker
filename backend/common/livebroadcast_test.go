package common

import (
	"testing"
	"time"
)

func TestCurrentMatchMinute(t *testing.T) {
	cases := []struct {
		name        string
		elapsed     time.Duration
		first       int
		second      int
		wantMinute  int
		wantAddedOn int
	}{
		{"kickoff", 0, 0, 0, 1, 0},
		{"first minute in progress", 30 * time.Second, 0, 0, 1, 0},
		{"midway through first half", 22 * time.Minute, 0, 0, 23, 0},
		{"last minute of first half", 44*time.Minute + 59*time.Second, 0, 0, 45, 0},
		{"during halftime break", 45*time.Minute + 5*time.Minute, 0, 0, 45, 0},
		{"right as second half kicks off", 45*time.Minute + liveMatchHalftimeDelay - time.Minute, 0, 0, 46, 0},
		{"midway through second half", 45*time.Minute + liveMatchHalftimeDelay + 9*time.Minute, 0, 0, 56, 0},
		{"last minute before 90", 45*time.Minute + liveMatchHalftimeDelay + 42*time.Minute + 59*time.Second, 0, 0, 89, 0},
		{"final minute", 45*time.Minute + liveMatchHalftimeDelay + 43*time.Minute, 0, 0, 90, 0},
		{"well past full time", 45*time.Minute + liveMatchHalftimeDelay + 45*time.Minute, 0, 0, 90, 0},
		// "Estimate in first-half added time" (division-live-scores): 46 minutes in, 3 added minutes.
		{"first-half added time", 46 * time.Minute, 3, 4, 45, 2},
		{"last first-half added minute", 47*time.Minute + 59*time.Second, 3, 4, 45, 3},
		{"break after added time", 50 * time.Minute, 3, 4, 45, 3},
		{"second half unchanged by added time", 59 * time.Minute, 3, 4, 46, 0},
		{"second-half added time", 105 * time.Minute, 3, 4, 90, 2},
		// "Estimate past the last added minute" (division-live-scores): 115 minutes in, 4 added minutes.
		{"past the last added minute", 115 * time.Minute, 3, 4, 90, 4},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			minute, added := CurrentMatchMinute(time.Now().Add(-c.elapsed), c.first, c.second)
			if minute != c.wantMinute || added != c.wantAddedOn {
				t.Fatalf("expected %d+%d, got %d+%d", c.wantMinute, c.wantAddedOn, minute, added)
			}
		})
	}
}
