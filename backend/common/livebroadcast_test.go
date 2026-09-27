package common

import (
	"testing"
	"time"
)

func TestCurrentMatchMinute(t *testing.T) {
	cases := []struct {
		name    string
		elapsed time.Duration
		want    int
	}{
		{"kickoff", 0, 1},
		{"first minute in progress", 30 * time.Second, 1},
		{"midway through first half", 22 * time.Minute, 23},
		{"last minute of first half", 44*time.Minute + 59*time.Second, 45},
		{"during halftime break", 45*time.Minute + 5*time.Minute, 45},
		{"right as second half kicks off", 45*time.Minute + liveMatchHalftimeDelay, 46},
		{"midway through second half", 45*time.Minute + liveMatchHalftimeDelay + 10*time.Minute, 56},
		{"final minute", 45*time.Minute + liveMatchHalftimeDelay + 44*time.Minute + 59*time.Second, 90},
		{"well past full time", 45*time.Minute + liveMatchHalftimeDelay + 45*time.Minute, 90},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CurrentMatchMinute(time.Now().Add(-c.elapsed))
			if got != c.want {
				t.Fatalf("expected minute %d, got %d", c.want, got)
			}
		})
	}
}
