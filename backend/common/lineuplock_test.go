package common

import (
	"testing"
	"time"
)

func TestIsLineupLocked_LocksWithinWindowAndAfterKickoff(t *testing.T) {
	cases := []struct {
		name       string
		fromNow    time.Duration
		wantLocked bool
	}{
		{"well before kickoff", LineupLockMinutes*time.Minute + time.Hour, false},
		{"just outside the window", LineupLockMinutes*time.Minute + time.Minute, false},
		{"just inside the window", LineupLockMinutes*time.Minute - time.Minute, true},
		{"at kickoff", 0, true},
		{"after kickoff", -time.Hour, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matchDate := time.Now().Add(c.fromNow)
			if got := IsLineupLocked(matchDate); got != c.wantLocked {
				t.Errorf("IsLineupLocked(now%+v) = %v, want %v", c.fromNow, got, c.wantLocked)
			}
		})
	}
}
