package common

import "errors"

// Weekday bit flags for MatchSchedule.WeekdaysMask, matching Go's time.Weekday numbering
// (Sunday = 0 ... Saturday = 6) so bit i corresponds to time.Weekday(i).
const (
	WeekdaySunday    = 1 << 0
	WeekdayMonday    = 1 << 1
	WeekdayTuesday   = 1 << 2
	WeekdayWednesday = 1 << 3
	WeekdayThursday  = 1 << 4
	WeekdayFriday    = 1 << 5
	WeekdaySaturday  = 1 << 6
	AllWeekdaysMask  = WeekdaySunday | WeekdayMonday | WeekdayTuesday | WeekdayWednesday |
		WeekdayThursday | WeekdayFriday | WeekdaySaturday
)

// Defaults match the previous fixed cadence closely enough to not surprise anyone who never
// touches this setting: a match round every day, in the evening.
const (
	DefaultMatchScheduleWeekdaysMask = AllWeekdaysMask
	DefaultMatchScheduleHour         = 16
	DefaultMatchScheduleMinute       = 0
)

// MatchSchedule is the admin-configured recurring schedule used to date newly generated league
// fixtures - e.g. "Saturdays at 14:00" (WeekdaysMask = WeekdaySaturday, Hour = 14) or "every day
// at 16:00" (WeekdaysMask = AllWeekdaysMask, Hour = 16). See engine.ComputeFixtureDates for how
// it turns into actual match dates.
type MatchSchedule struct {
	WeekdaysMask int `json:"weekdaysMask"`
	Hour         int `json:"hour"`
	Minute       int `json:"minute"`
}

var ErrInvalidMatchSchedule = errors.New("match schedule must select at least one weekday and a time between 00:00 and 23:59")

// ValidateMatchSchedule checks a schedule is usable before it's persisted - at least one weekday
// selected, and hour/minute in range. Pure so it's testable without a database.
func ValidateMatchSchedule(s MatchSchedule) error {
	if s.WeekdaysMask&AllWeekdaysMask == 0 {
		return ErrInvalidMatchSchedule
	}
	if s.Hour < 0 || s.Hour > 23 || s.Minute < 0 || s.Minute > 59 {
		return ErrInvalidMatchSchedule
	}
	return nil
}

const GetMatchScheduleStmtSql = "SELECT match_schedule_weekdays, match_schedule_minute_of_day FROM season"
const SetMatchScheduleStmtSql = "UPDATE season SET match_schedule_weekdays = $1, match_schedule_minute_of_day = $2"

// GetMatchSchedule returns the current admin-configured schedule, falling back to the default
// (every day at 16:00) on a lookup error - fixture generation must never be blocked by a
// settings-row hiccup.
func GetMatchSchedule(dbCtx *DatabaseContext) MatchSchedule {
	var mask, minuteOfDay int
	if err := dbCtx.Db.QueryRow(GetMatchScheduleStmtSql).Scan(&mask, &minuteOfDay); err != nil {
		return MatchSchedule{
			WeekdaysMask: DefaultMatchScheduleWeekdaysMask,
			Hour:         DefaultMatchScheduleHour,
			Minute:       DefaultMatchScheduleMinute,
		}
	}
	return MatchSchedule{WeekdaysMask: mask, Hour: minuteOfDay / 60, Minute: minuteOfDay % 60}
}

// SetMatchSchedule persists a new schedule. It only affects fixtures generated after this call
// (season rollover, or a fresh world init) - matches already dated keep their existing date.
func SetMatchSchedule(dbCtx *DatabaseContext, s MatchSchedule) error {
	if err := ValidateMatchSchedule(s); err != nil {
		return err
	}
	minuteOfDay := s.Hour*60 + s.Minute
	_, err := dbCtx.Db.Exec(SetMatchScheduleStmtSql, s.WeekdaysMask, minuteOfDay)
	return err
}
