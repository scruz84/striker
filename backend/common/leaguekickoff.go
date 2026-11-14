package common

import (
	"errors"
	"fmt"
	"time"
)

// A country's league kickoff (see openspec/specs/match-scheduling "Country league kickoff
// setting") is the local minute of day its tier 1 kicks off; every lower tier kicks off
// LeagueKickoffTierStagger later, so tier t plays at kickoff + 15 min x (t - 1) on the country's
// own clock. Stored in countries.league_kickoff_minute (init110_league_pyramid_playoffs.sql).
const (
	LeagueKickoffTierStagger   = 15 * time.Minute
	DefaultLeagueKickoffMinute = DefaultKickoffHour*60 + DefaultKickoffMinute
	minutesPerDay              = 24 * 60
)

var (
	ErrInvalidLeagueKickoff = errors.New("league kickoff must be a quarter-hour between 00:00 and 23:45")
	ErrLeagueKickoffTooLate = errors.New("choose an earlier time")
)

// TierKickoffMinute is the local minute of day tier kicks off in a country whose tier 1 kicks off
// at kickoffMinute.
func TierKickoffMinute(kickoffMinute, tier int) int {
	if tier < 1 {
		tier = 1
	}
	return kickoffMinute + int(LeagueKickoffTierStagger/time.Minute)*(tier-1)
}

// ValidateCountryLeagueKickoff checks kickoffMinute is a quarter-hour and that the deepest of
// tierCount tiers still kicks off before midnight. Pure.
func ValidateCountryLeagueKickoff(kickoffMinute, tierCount int) error {
	if kickoffMinute < 0 || kickoffMinute >= minutesPerDay || kickoffMinute%15 != 0 {
		return ErrInvalidLeagueKickoff
	}
	if last := TierKickoffMinute(kickoffMinute, tierCount); last >= minutesPerDay {
		return fmt.Errorf("tier %d would kick off at %02d:%02d the next day; %w", tierCount, (last/60)%24, last%60, ErrLeagueKickoffTooLate)
	}
	return nil
}

const getCountryLeagueKickoffStmtSql = "SELECT league_kickoff_minute FROM countries WHERE name = $1"
const setCountryLeagueKickoffStmtSql = "UPDATE countries SET league_kickoff_minute = $2 WHERE name = $1"

// GetCountryLeagueKickoff returns country's tier-1 local kickoff minute, the default 14:00 on a
// lookup error - fixture generation must never be blocked by a settings hiccup.
func GetCountryLeagueKickoff(dbCtx *DatabaseContext, country string) int {
	var minute int
	if err := dbCtx.Db.QueryRow(getCountryLeagueKickoffStmtSql, country).Scan(&minute); err != nil {
		return DefaultLeagueKickoffMinute
	}
	return minute
}

// SetCountryLeagueKickoff validates and stores country's tier-1 local kickoff minute. tierCount is
// the deepest tier it must leave room for (the country's or the saved structure's, whichever is
// deeper).
func SetCountryLeagueKickoff(dbCtx *DatabaseContext, country string, kickoffMinute, tierCount int) error {
	if err := ValidateCountryLeagueKickoff(kickoffMinute, tierCount); err != nil {
		return err
	}
	_, err := dbCtx.Db.Exec(setCountryLeagueKickoffStmtSql, country, kickoffMinute)
	return err
}

// LocalKickoffOnDayAfter returns the instant at minuteOfDay on the local calendar day after
// day's own date in loc, moved one day further for as long as it is earlier than earliest. Built
// with time.Date in loc, so the local clock time holds across daylight-saving changes. Pure.
func LocalKickoffOnDayAfter(loc *time.Location, day time.Time, minuteOfDay int, earliest time.Time) time.Time {
	d := day.In(loc)
	hour, minute := minuteOfDay/60, minuteOfDay%60
	offset := 1
	candidate := time.Date(d.Year(), d.Month(), d.Day()+offset, hour, minute, 0, 0, loc)
	for candidate.Before(earliest) {
		offset++
		candidate = time.Date(d.Year(), d.Month(), d.Day()+offset, hour, minute, 0, 0, loc)
	}
	return candidate
}
