package engine

import (
	"time"

	"github.com/scruz84/striker/common"
)

// NextScheduledOccurrence returns the earliest date/time at or after from that falls on one of
// schedule's selected weekdays at its configured hour:minute. If from itself is already on a
// matching weekday and that time hasn't passed yet, from's own day is returned - so generating
// fixtures at, say, 09:00 on a scheduled Saturday still lands the first match later that same
// Saturday instead of skipping a full week ahead.
func NextScheduledOccurrence(from time.Time, schedule common.MatchSchedule) time.Time {
	dayStart := time.Date(from.Year(), from.Month(), from.Day(), schedule.Hour, schedule.Minute, 0, 0, from.Location())
	for offset := 0; offset < 8; offset++ {
		candidate := dayStart.AddDate(0, 0, offset)
		bit := 1 << int(candidate.Weekday())
		if schedule.WeekdaysMask&bit != 0 && !candidate.Before(from) {
			return candidate
		}
	}
	// Unreachable as long as schedule passed ValidateMatchSchedule (at least one weekday set) -
	// every weekday recurs within 7 days - but fall back to from rather than a zero time.
	return from
}

// ComputeFixtureDates returns roundCount dates, one per fixture round, following schedule's
// selected weekdays starting from `from`. A single selected weekday gives one round per week (a
// classic weekly league); every weekday selected gives one round per day.
func ComputeFixtureDates(from time.Time, schedule common.MatchSchedule, roundCount int) []time.Time {
	dates := make([]time.Time, roundCount)
	next := from
	for i := 0; i < roundCount; i++ {
		occurrence := NextScheduledOccurrence(next, schedule)
		dates[i] = occurrence
		next = occurrence.Add(time.Minute) // search for the next round must start strictly after this one
	}
	return dates
}
