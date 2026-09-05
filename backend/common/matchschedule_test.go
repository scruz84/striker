package common

import "testing"

func TestValidateMatchSchedule_RejectsNoWeekdaysSelected(t *testing.T) {
	err := ValidateMatchSchedule(MatchSchedule{WeekdaysMask: 0, Hour: 16, Minute: 0})
	if err != ErrInvalidMatchSchedule {
		t.Fatalf("expected ErrInvalidMatchSchedule for an empty weekday mask, got %v", err)
	}
}

func TestValidateMatchSchedule_RejectsOutOfRangeMaskBits(t *testing.T) {
	// Bits beyond Saturday (1<<6 = 64) aren't real weekdays; a mask of only those bits should be
	// treated the same as "nothing selected".
	err := ValidateMatchSchedule(MatchSchedule{WeekdaysMask: 1 << 7, Hour: 16, Minute: 0})
	if err != ErrInvalidMatchSchedule {
		t.Fatalf("expected ErrInvalidMatchSchedule for a mask with no valid weekday bits, got %v", err)
	}
}

func TestValidateMatchSchedule_RejectsOutOfRangeTime(t *testing.T) {
	cases := []MatchSchedule{
		{WeekdaysMask: WeekdaySaturday, Hour: -1, Minute: 0},
		{WeekdaysMask: WeekdaySaturday, Hour: 24, Minute: 0},
		{WeekdaysMask: WeekdaySaturday, Hour: 14, Minute: -1},
		{WeekdaysMask: WeekdaySaturday, Hour: 14, Minute: 60},
	}
	for _, s := range cases {
		if err := ValidateMatchSchedule(s); err != ErrInvalidMatchSchedule {
			t.Fatalf("expected ErrInvalidMatchSchedule for %+v, got %v", s, err)
		}
	}
}

func TestValidateMatchSchedule_AcceptsValidSchedules(t *testing.T) {
	cases := []MatchSchedule{
		{WeekdaysMask: WeekdaySaturday, Hour: 14, Minute: 0},
		{WeekdaysMask: AllWeekdaysMask, Hour: 16, Minute: 0},
		{WeekdaysMask: WeekdayTuesday | WeekdayFriday, Hour: 20, Minute: 30},
		{WeekdaysMask: WeekdaySunday, Hour: 0, Minute: 0},
		{WeekdaysMask: WeekdaySunday, Hour: 23, Minute: 59},
	}
	for _, s := range cases {
		if err := ValidateMatchSchedule(s); err != nil {
			t.Fatalf("expected %+v to be valid, got %v", s, err)
		}
	}
}

func TestSetMatchSchedule_RejectsInvalidScheduleWithoutTouchingDatabase(t *testing.T) {
	// Validation runs before any database access, so this is safe to exercise with a nil
	// DatabaseContext.
	if err := SetMatchSchedule(nil, MatchSchedule{WeekdaysMask: 0, Hour: 16, Minute: 0}); err != ErrInvalidMatchSchedule {
		t.Fatalf("expected ErrInvalidMatchSchedule, got %v", err)
	}
}
