package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// TestMinutePublishOffset checks minutePublishOffset against common.CurrentMatchMinute's own
// inverse mapping (elapsed -> minute) for the same pacing, so a viewer who connects exactly at
// matchDate.Add(minutePublishOffset(m, a)) sees minute m+a, matching what CurrentMatchMinute would
// independently estimate for that same elapsed time.
func TestMinutePublishOffset(t *testing.T) {
	cases := []struct {
		minute, added int
		wantOffset    string
	}{
		{1, 0, "0s"},
		{23, 0, "22m0s"},
		{45, 0, "44m0s"},
		{45, 1, "45m0s"},
		{45, 5, "49m0s"},
		{46, 0, "59m0s"},   // 44m (rest of first half) + 15m, whatever the first half's added time
		{56, 0, "1h9m0s"},  // 59m + 10m into the second half
		{90, 0, "1h43m0s"}, // 59m + 44m
		{90, 1, "1h44m0s"},
		{90, 5, "1h48m0s"},
	}

	for _, c := range cases {
		got := minutePublishOffset(c.minute, c.added)
		if got.String() != c.wantOffset {
			t.Errorf("minutePublishOffset(%d+%d) = %s, want %s", c.minute, c.added, got, c.wantOffset)
		}
		minute, added := common.CurrentMatchMinute(time.Now().Add(-got-time.Second), 5, 5)
		if minute != c.minute || added != c.added {
			t.Errorf("CurrentMatchMinute at %s = %d+%d, want %d+%d", got, minute, added, c.minute, c.added)
		}
	}
}

// "Publish schedule with added time" and "Shortest break" (match-scheduling "Live replay of a
// simulated result").
func TestMinutePublishOffset_ScheduleWithAddedTime(t *testing.T) {
	kickoff := time.Date(2026, 11, 14, 14, 0, 0, 0, time.UTC)
	at := func(minute, added int) string { return kickoff.Add(minutePublishOffset(minute, added)).Format("15:04") }
	for _, c := range []struct {
		minute, added int
		want          string
	}{{45, 0, "14:44"}, {45, 1, "14:45"}, {45, 3, "14:47"}, {46, 0, "14:59"}, {90, 0, "15:43"}, {90, 1, "15:44"}, {90, 5, "15:48"}} {
		if got := at(c.minute, c.added); got != c.want {
			t.Errorf("%d+%d published at %s, want %s", c.minute, c.added, got, c.want)
		}
	}
	// With the maximum 5 first-half added minutes the break lasts 10 minutes, and 46 still opens at 59.
	if gap := minutePublishOffset(46, 0) - minutePublishOffset(45, MaxAddedFirstHalf); gap != 10*time.Minute {
		t.Errorf("shortest break %s, want 10m", gap)
	}
	if gap := minutePublishOffset(46, 0) - minutePublishOffset(45, MinAddedMinutes); gap != 14*time.Minute {
		t.Errorf("longest break %s, want 14m", gap)
	}
	if gap := 14 * time.Minute; gap >= common.LiveMatchStaleTimeout {
		t.Errorf("the longest break (%s) must stay under LiveMatchStaleTimeout (%s)", gap, common.LiveMatchStaleTimeout)
	}
}

// TestLongestBroadcastFitsLiveWindow (stoppage-time "Longest broadcast fits the live-broadcast
// window"): the longest possible replay must publish its final whistle at least 5 minutes before
// common.LiveBroadcastWindow closes, or a time-gated view (head-to-head, top scorers, the frontend's
// mightStillBeLive...) could reveal a result still on air.
func TestLongestBroadcastFitsLiveWindow(t *testing.T) {
	if got := MaxBroadcastDuration(); got != 108*time.Minute {
		t.Errorf("MaxBroadcastDuration() = %s, want 1h48m (90+5)", got)
	}
	if margin := common.LiveBroadcastWindow - MaxBroadcastDuration(); margin < 5*time.Minute {
		t.Fatalf("the longest replay ends %s before the %s live-broadcast window closes, want at least 5m", margin, common.LiveBroadcastWindow)
	}
}
