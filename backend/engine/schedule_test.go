package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

func TestNextScheduledOccurrence_LandsOnSelectedWeekdayAtConfiguredTime(t *testing.T) {
	from := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC) // a Monday
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 14, Minute: 0}

	occurrence := NextScheduledOccurrence(from, schedule)

	if occurrence.Weekday() != time.Saturday {
		t.Fatalf("expected a Saturday, got %s", occurrence.Weekday())
	}
	if occurrence.Hour() != 14 || occurrence.Minute() != 0 {
		t.Fatalf("expected 14:00, got %02d:%02d", occurrence.Hour(), occurrence.Minute())
	}
	if occurrence.Before(from) {
		t.Fatalf("occurrence %v must not be before from %v", occurrence, from)
	}
}

func TestNextScheduledOccurrence_UsesTodayIfTimeHasNotPassedYet(t *testing.T) {
	saturdayMorning := time.Date(2026, 3, 7, 9, 0, 0, 0, time.UTC) // a Saturday, before 14:00
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 14, Minute: 0}

	occurrence := NextScheduledOccurrence(saturdayMorning, schedule)

	expected := time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC)
	if !occurrence.Equal(expected) {
		t.Fatalf("expected today at 14:00 (%v), got %v", expected, occurrence)
	}
}

func TestNextScheduledOccurrence_SkipsToNextWeekIfTimeAlreadyPassedToday(t *testing.T) {
	saturdayEvening := time.Date(2026, 3, 7, 20, 0, 0, 0, time.UTC) // a Saturday, after 14:00
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 14, Minute: 0}

	occurrence := NextScheduledOccurrence(saturdayEvening, schedule)

	expected := time.Date(2026, 3, 14, 14, 0, 0, 0, time.UTC) // the following Saturday
	if !occurrence.Equal(expected) {
		t.Fatalf("expected the following Saturday (%v), got %v", expected, occurrence)
	}
}

func TestComputeFixtureDates_SingleWeekdayGivesOneRoundPerWeek(t *testing.T) {
	from := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC) // a Monday
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdaySaturday, Hour: 14, Minute: 0}

	dates := ComputeFixtureDates(from, schedule, 4)

	for i, d := range dates {
		if d.Weekday() != time.Saturday {
			t.Fatalf("round %d: expected a Saturday, got %s", i, d.Weekday())
		}
		if i > 0 && dates[i].Sub(dates[i-1]) != 7*24*time.Hour {
			t.Fatalf("round %d: expected exactly 7 days after the previous round, got %v", i, dates[i].Sub(dates[i-1]))
		}
	}
}

func TestComputeFixtureDates_AllWeekdaysGivesOneRoundPerDay(t *testing.T) {
	from := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	schedule := common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}

	dates := ComputeFixtureDates(from, schedule, 5)

	for i := 1; i < len(dates); i++ {
		if dates[i].Sub(dates[i-1]) != 24*time.Hour {
			t.Fatalf("round %d: expected exactly 1 day after the previous round, got %v", i, dates[i].Sub(dates[i-1]))
		}
	}
}

func TestComputeFixtureDates_MultipleWeekdaysOnlyLandOnSelectedDays(t *testing.T) {
	from := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	schedule := common.MatchSchedule{WeekdaysMask: common.WeekdayTuesday | common.WeekdayFriday, Hour: 20, Minute: 0}

	dates := ComputeFixtureDates(from, schedule, 6)

	for i, d := range dates {
		if d.Weekday() != time.Tuesday && d.Weekday() != time.Friday {
			t.Fatalf("round %d: expected Tuesday or Friday, got %s", i, d.Weekday())
		}
		if i > 0 && !dates[i].After(dates[i-1]) {
			t.Fatalf("round %d: expected strictly increasing dates", i)
		}
	}
}

func TestComputeFixtureDates_ReturnsRequestedCount(t *testing.T) {
	schedule := common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}

	dates := ComputeFixtureDates(time.Now(), schedule, 0)
	if len(dates) != 0 {
		t.Fatalf("expected 0 dates, got %d", len(dates))
	}

	dates = ComputeFixtureDates(time.Now(), schedule, 3)
	if len(dates) != 3 {
		t.Fatalf("expected 3 dates, got %d", len(dates))
	}
}
