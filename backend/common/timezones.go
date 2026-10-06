package common

import (
	"time"
	_ "time/tzdata" // embed the IANA database so LoadLocation works even on a container image without one
)

// CountryTimezone maps a KnownCountries code to one representative IANA timezone - an
// approximation (a country spanning several real zones gets only one), good enough to judge
// whether it's daytime there without needing a per-team stored offset.
var CountryTimezone = map[string]string{
	"es": "Europe/Madrid",
	"en": "Europe/London",
	"fr": "Europe/Paris",
	"de": "Europe/Berlin",
	"it": "Europe/Rome",
	"pt": "Europe/Lisbon",
	"nl": "Europe/Amsterdam",
	"se": "Europe/Stockholm",
	"pl": "Europe/Warsaw",
	"gr": "Europe/Athens",
	"br": "America/Sao_Paulo",
	"ar": "America/Argentina/Buenos_Aires",
	"mx": "America/Mexico_City",
	"co": "America/Bogota",
	"cl": "America/Santiago",
	"pe": "America/Lima",
	"uy": "America/Montevideo",
	"us": "America/New_York",
	"ca": "America/Toronto",
	"ec": "America/Guayaquil",
	"jp": "Asia/Tokyo",
	"kr": "Asia/Seoul",
	"cn": "Asia/Shanghai",
	"in": "Asia/Kolkata",
	"th": "Asia/Bangkok",
	"vn": "Asia/Ho_Chi_Minh",
	"id": "Asia/Jakarta",
	"ph": "Asia/Manila",
	"sa": "Asia/Riyadh",
	"ir": "Asia/Tehran",
	"ng": "Africa/Lagos",
	"eg": "Africa/Cairo",
	"gh": "Africa/Accra",
	"ci": "Africa/Abidjan",
	"cm": "Africa/Douala",
	"sn": "Africa/Dakar",
	"za": "Africa/Johannesburg",
	"ma": "Africa/Casablanca",
	"dz": "Africa/Algiers",
	"ke": "Africa/Nairobi",
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

// IsWithinLocalActivityWindow reports whether now falls within countryCode's local awake-hours
// window (see LocalActivityStartHour/LocalActivityEndHour) - the gate engine.RunAiTeamDailyActivity
// checks before letting an AI-managed team initiate a friendly challenge or run its proactive
// market pass, so CPU activity lands at a plausible local hour instead of any UTC instant at
// random.
func IsWithinLocalActivityWindow(now time.Time, countryCode string) bool {
	hour := now.In(countryLocation(countryCode)).Hour()
	return hour >= LocalActivityStartHour && hour < LocalActivityEndHour
}
