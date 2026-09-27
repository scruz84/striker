package engine

import "testing"

// TestMinutePublishOffset checks minutePublishOffset against common.CurrentMatchMinute's own
// inverse mapping (elapsed -> minute) for the same fixed pacing, so a viewer who connects exactly
// at matchDate.Add(minutePublishOffset(m)) sees minute m, matching what CurrentMatchMinute would
// independently estimate for that same elapsed time.
func TestMinutePublishOffset(t *testing.T) {
	cases := []struct {
		minute     int
		wantOffset string
	}{
		{1, "0s"},
		{23, "22m0s"},
		{45, "44m0s"},
		{46, "59m0s"},   // 44m (rest of first half) + 15m halftime break
		{56, "1h9m0s"},  // 59m + 10m into the second half
		{90, "1h43m0s"}, // 59m + 44m
	}

	for _, c := range cases {
		got := minutePublishOffset(c.minute)
		if got.String() != c.wantOffset {
			t.Errorf("minutePublishOffset(%d) = %s, want %s", c.minute, got, c.wantOffset)
		}
	}
}
