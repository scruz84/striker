package common

import (
	"testing"
	"time"
)

func TestNextAllowedFriendlyDate_ReturnsUnchangedWhenTodayIsAllowed(t *testing.T) {
	// A Wednesday (2024-01-03).
	candidate := time.Date(2024, 1, 3, 18, 5, 0, 0, time.UTC)
	got := nextAllowedFriendlyDate(candidate, AllWeekdaysMask)
	if !got.Equal(candidate) {
		t.Fatalf("expected candidate to be returned unchanged when every day is allowed, got %v", got)
	}
}

func TestNextAllowedFriendlyDate_SkipsForwardToNextAllowedWeekday(t *testing.T) {
	// A Wednesday (2024-01-03), only weekends (Sat/Sun) allowed - should land on Saturday
	// 2024-01-06, keeping the original time-of-day.
	candidate := time.Date(2024, 1, 3, 18, 5, 0, 0, time.UTC)
	got := nextAllowedFriendlyDate(candidate, WeekdaySaturday|WeekdaySunday)
	want := time.Date(2024, 1, 6, 18, 5, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestNextAllowedFriendlyDate_WrapsAroundToNextWeek(t *testing.T) {
	// A Saturday (2024-01-06), only Friday allowed - the nearest Friday is 6 days later.
	candidate := time.Date(2024, 1, 6, 9, 0, 0, 0, time.UTC)
	got := nextAllowedFriendlyDate(candidate, WeekdayFriday)
	want := time.Date(2024, 1, 12, 9, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestSetFriendlyScheduleWeekdays_RejectsEmptyMaskWithoutTouchingDatabase(t *testing.T) {
	if err := SetFriendlyScheduleWeekdays(nil, 0); err != ErrInvalidFriendlySchedule {
		t.Fatalf("expected ErrInvalidFriendlySchedule, got %v", err)
	}
}

func TestStartOfDay_ReturnsMidnightUtcOnTsOwnDay(t *testing.T) {
	got := StartOfDay(time.Date(2024, 1, 3, 18, 5, 30, 0, time.UTC))
	want := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestStartOfDay_ConvertsToUtcFirst(t *testing.T) {
	loc := time.FixedZone("UTC-5", -5*60*60)
	// 2024-01-03 23:00 UTC-5 is 2024-01-04 04:00 UTC - StartOfDay should land on the UTC day.
	got := StartOfDay(time.Date(2024, 1, 3, 23, 0, 0, 0, loc))
	want := time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestIsFriendlyKickoffSlot(t *testing.T) {
	cases := []struct {
		kickoff time.Time
		want    bool
	}{
		{time.Date(2026, 11, 13, 18, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 11, 13, 18, 15, 0, 0, time.UTC), true},
		{time.Date(2026, 11, 13, 18, 30, 0, 0, time.UTC), true},
		{time.Date(2026, 11, 13, 18, 45, 0, 0, time.UTC), true},
		{time.Date(2026, 11, 13, 18, 5, 0, 0, time.UTC), false},
		{time.Date(2026, 11, 13, 18, 50, 0, 0, time.UTC), false},
		{time.Date(2026, 11, 13, 18, 15, 30, 0, time.UTC), false},
		// 20:45 in Kathmandu (UTC+5:45) is 15:00 UTC - still a slot.
		{time.Date(2026, 11, 13, 20, 45, 0, 0, time.FixedZone("NPT", 5*3600+45*60)), true},
	}
	for _, c := range cases {
		if got := IsFriendlyKickoffSlot(c.kickoff); got != c.want {
			t.Errorf("IsFriendlyKickoffSlot(%v) = %v, want %v", c.kickoff, got, c.want)
		}
	}
}

func TestNextFriendlyKickoffSlot(t *testing.T) {
	cases := []struct {
		from, want time.Time
	}{
		{time.Date(2026, 11, 13, 18, 0, 0, 0, time.UTC), time.Date(2026, 11, 13, 18, 0, 0, 0, time.UTC)},
		{time.Date(2026, 11, 13, 18, 0, 1, 0, time.UTC), time.Date(2026, 11, 13, 18, 15, 0, 0, time.UTC)},
		{time.Date(2026, 11, 13, 18, 37, 0, 0, time.UTC), time.Date(2026, 11, 13, 18, 45, 0, 0, time.UTC)},
		{time.Date(2026, 11, 13, 23, 50, 0, 0, time.UTC), time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		if got := nextFriendlyKickoffSlot(c.from); !got.Equal(c.want) {
			t.Errorf("nextFriendlyKickoffSlot(%v) = %v, want %v", c.from, got, c.want)
		}
	}
}
