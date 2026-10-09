package common

import (
	"time"
	_ "time/tzdata" // embed the IANA database so LoadLocation works even on a container image without one
)

// CountryTimezone maps a KnownCountries code to one representative IANA timezone - an
// approximation (a country spanning several real zones gets only one), good enough to judge
// whether it's daytime there without needing a per-team stored offset.
var CountryTimezone = map[string]string{
	"es":  "Europe/Madrid",
	"en":  "Europe/London",
	"fr":  "Europe/Paris",
	"de":  "Europe/Berlin",
	"it":  "Europe/Rome",
	"pt":  "Europe/Lisbon",
	"nl":  "Europe/Amsterdam",
	"se":  "Europe/Stockholm",
	"pl":  "Europe/Warsaw",
	"gr":  "Europe/Athens",
	"be":  "Europe/Brussels",
	"tr":  "Europe/Istanbul",
	"at":  "Europe/Vienna",
	"sco": "Europe/London",
	"wal": "Europe/London",
	"ie":  "Europe/Dublin",
	"ru":  "Europe/Moscow",
	"rs":  "Europe/Belgrade",
	"hr":  "Europe/Zagreb",
	"ba":  "Europe/Sarajevo",
	"si":  "Europe/Ljubljana",
	"mk":  "Europe/Skopje",
	"me":  "Europe/Podgorica",
	"xk":  "Europe/Belgrade",
	"br":  "America/Sao_Paulo",
	"ar":  "America/Argentina/Buenos_Aires",
	"mx":  "America/Mexico_City",
	"co":  "America/Bogota",
	"cl":  "America/Santiago",
	"pe":  "America/Lima",
	"uy":  "America/Montevideo",
	"us":  "America/New_York",
	"ca":  "America/Toronto",
	"ec":  "America/Guayaquil",
	"ve":  "America/Caracas",
	"py":  "America/Asuncion",
	"bo":  "America/La_Paz",
	"jp":  "Asia/Tokyo",
	"kr":  "Asia/Seoul",
	"cn":  "Asia/Shanghai",
	"in":  "Asia/Kolkata",
	"th":  "Asia/Bangkok",
	"vn":  "Asia/Ho_Chi_Minh",
	"id":  "Asia/Jakarta",
	"ph":  "Asia/Manila",
	"sa":  "Asia/Riyadh",
	"ir":  "Asia/Tehran",
	"qa":  "Asia/Qatar",
	"ae":  "Asia/Dubai",
	"uz":  "Asia/Tashkent",
	"ng":  "Africa/Lagos",
	"eg":  "Africa/Cairo",
	"gh":  "Africa/Accra",
	"ci":  "Africa/Abidjan",
	"cm":  "Africa/Douala",
	"sn":  "Africa/Dakar",
	"za":  "Africa/Johannesburg",
	"ma":  "Africa/Casablanca",
	"dz":  "Africa/Algiers",
	"ke":  "Africa/Nairobi",
	"tn":  "Africa/Tunis",
	"tz":  "Africa/Dar_es_Salaam",
	"ml":  "Africa/Bamako",
}

// LocalActivityStartHour/LocalActivityEndHour bound the local "awake hours" window (end
// exclusive) during which an AI-managed team may initiate a friendly challenge or run its
// proactive market-management pass - see engine.RunAiTeamDailyActivity. Chosen to cover a
// typical waking day without reaching into the small hours.
const (
	LocalActivityStartHour = 9
	LocalActivityEndHour   = 23
)

// countryLocations resolves every CountryTimezone entry's *time.Location exactly once at
// startup, rather than re-parsing the same handful of IANA zones over and over - countryLocation
// (called once per AI-managed team on every 5-minute scheduler tick, see
// engine.RunAiTeamDailyActivity) would otherwise call time.LoadLocation thousands of times a day
// for what are, across every team in the game, only as many distinct zones as CountryTimezone
// has entries. Populated once here and never mutated afterward, so concurrent reads need no
// locking.
var countryLocations = func() map[string]*time.Location {
	locations := make(map[string]*time.Location, len(CountryTimezone))
	for code, tz := range CountryTimezone {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			continue // countryLocation falls back to UTC for anything missing here
		}
		locations[code] = loc
	}
	return locations
}()

// countryLocation resolves code's representative *time.Location, falling back to UTC for an
// unmapped or unknown code - a missing timezone entry should never block AI activity outright.
func countryLocation(code string) *time.Location {
	if loc, ok := countryLocations[code]; ok {
		return loc
	}
	return time.UTC
}

// DefaultKickoffHour/DefaultKickoffMinute is the kickoff time a newly added country's leagues
// default to, expressed in that country's own local time (see LocalTimeToServerSchedule) - a
// plausible mid-afternoon matchday slot rather than a hardcoded server-time hour that would land
// oddly early or late depending on the country's timezone.
const (
	DefaultKickoffHour   = 14
	DefaultKickoffMinute = 0
)

// LocalTimeToServerSchedule converts an hour:minute expressed in countryCode's own local time
// (see CountryTimezone) to the equivalent hour:minute in the server process's own local time zone
// - the same time zone ComputeFixtureDates/NextScheduledOccurrence build fixture dates in (UTC on
// every deployment this project documents, see frontend/src/utils/scheduleTime.ts). Used by
// handlers.AdminAddCountryHandler so a country's matches kick off at the requested local time of
// day rather than whatever hour that number happens to be on the server's own clock.
//
// Anchored to today's date, like that frontend's own equivalent conversion, so it reflects
// whichever side of a daylight-saving transition is in effect right now rather than some fixed
// arbitrary date - the same practical tradeoff, not a perfect year-round mapping across a DST
// boundary.
func LocalTimeToServerSchedule(countryCode string, hour, minute int) (int, int) {
	loc := countryLocation(countryCode)
	now := time.Now().In(loc)
	local := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	server := local.Local()
	return server.Hour(), server.Minute()
}

// IsWithinLocalActivityWindow reports whether now falls within countryCode's local awake-hours
// window (see LocalActivityStartHour/LocalActivityEndHour) - the gate engine.RunAiTeamDailyActivity
// checks before letting an AI-managed team initiate a friendly challenge or run its proactive
// market pass, so CPU activity lands at a plausible local hour instead of any UTC instant at
// random.
func IsWithinLocalActivityWindow(now time.Time, countryCode string) bool {
	hour := now.In(countryLocation(countryCode)).Hour()
	return hour >= LocalActivityStartHour && hour < LocalActivityEndHour
}
