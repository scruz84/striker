package common

import (
	"testing"
	"time"
)

// Every KnownCountries entry needs a CountryTimezone mapping that actually loads - missing one
// doesn't fail outright (countryLocation falls back to UTC), but silently judges that country's
// AI activity window by the wrong clock, defeating the point of the local-hour gate.
func TestEveryKnownCountryHasTimezone(t *testing.T) {
	for _, country := range KnownCountries {
		tz, ok := CountryTimezone[country.Code]
		if !ok {
			t.Errorf("KnownCountries entry %q has no CountryTimezone mapping", country.Code)
			continue
		}
		if _, err := time.LoadLocation(tz); err != nil {
			t.Errorf("CountryTimezone[%q] = %q does not load: %s", country.Code, tz, err)
		}
	}
}

func TestIsWithinLocalActivityWindow(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("failed to load Europe/Madrid: %s", err)
	}

	cases := []struct {
		name string
		hour int
		want bool
	}{
		{"before window", 8, false},
		{"window start", 9, true},
		{"mid window", 15, true},
		{"last hour of window", 22, true},
		{"window end (exclusive)", 23, false},
		{"after window", 2, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			now := time.Date(2026, 6, 15, c.hour, 30, 0, 0, loc)
			if got := IsWithinLocalActivityWindow(now, "es"); got != c.want {
				t.Errorf("IsWithinLocalActivityWindow at %dh local = %v, want %v", c.hour, got, c.want)
			}
		})
	}
}

// Two countries far enough apart in UTC offset should disagree about whether the same instant is
// inside their respective local activity windows - the whole point of gating per-country.
func TestIsWithinLocalActivityWindow_DiffersByCountry(t *testing.T) {
	// 03:00 UTC: night in Spain (Europe/Madrid, UTC+1/+2), afternoon in Japan (Asia/Tokyo, UTC+9).
	now := time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC)
	if IsWithinLocalActivityWindow(now, "es") {
		t.Error("expected 03:00 UTC to be outside Spain's local activity window")
	}
	if !IsWithinLocalActivityWindow(now, "jp") {
		t.Error("expected 03:00 UTC to be inside Japan's local activity window")
	}
}

func TestIsWithinLocalActivityWindow_UnknownCountryFallsBackToUTC(t *testing.T) {
	now := time.Date(2026, 6, 15, 15, 0, 0, 0, time.UTC)
	if !IsWithinLocalActivityWindow(now, "zz") {
		t.Error("expected an unmapped country code to fall back to UTC and judge 15:00 UTC as within window")
	}
}

// LocalTimeToServerSchedule's whole point is that converting a country-local hour:minute to the
// server's own time zone and back lands on the same hour:minute it started from - whatever that
// server time zone actually is wherever this test runs (see its own doc comment: it always
// builds off time.Now(), it can't be handed a fixed instant to assert an exact expected hour
// against without also hardcoding the test's assumed server time zone).
func TestLocalTimeToServerSchedule_RoundTripsBackToOriginalLocalTime(t *testing.T) {
	for _, country := range []string{"es", "jp", "us", "ae"} {
		wantHour, wantMinute := 14, 0
		serverHour, serverMinute := LocalTimeToServerSchedule(country, wantHour, wantMinute)

		now := time.Now()
		server := time.Date(now.Year(), now.Month(), now.Day(), serverHour, serverMinute, 0, 0, time.Local)
		gotLocal := server.In(countryLocation(country))

		if gotLocal.Hour() != wantHour || gotLocal.Minute() != wantMinute {
			t.Errorf("LocalTimeToServerSchedule(%q, %d, %d) = (%d, %d), converting back to local gave %02d:%02d, want %02d:%02d",
				country, wantHour, wantMinute, serverHour, serverMinute, gotLocal.Hour(), gotLocal.Minute(), wantHour, wantMinute)
		}
	}
}

// NextLocalSaturdayKickoff should always land on a Saturday, at the requested local hour:minute,
// and strictly in the future - the three properties ScheduleNationalTeamFriendlies relies on.
func TestNextLocalSaturdayKickoff_LandsOnSaturdayAtTheRequestedLocalTime(t *testing.T) {
	for _, country := range []string{"es", "jp", "us", "ae", "zz"} {
		kickoff := NextLocalSaturdayKickoff(country, 21, 0)
		local := kickoff.In(countryLocation(country))

		if local.Weekday() != time.Saturday {
			t.Errorf("NextLocalSaturdayKickoff(%q) = %v, want a Saturday (got %s)", country, kickoff, local.Weekday())
		}
		if local.Hour() != 21 || local.Minute() != 0 {
			t.Errorf("NextLocalSaturdayKickoff(%q) local time = %02d:%02d, want 21:00", country, local.Hour(), local.Minute())
		}
		if !kickoff.After(time.Now()) {
			t.Errorf("NextLocalSaturdayKickoff(%q) = %v, want a moment strictly after now", country, kickoff)
		}
	}
}

// Two countries far enough apart in UTC offset should get genuinely different kickoff instants
// for "the same" Saturday 21:00 local - the whole point of computing it per country rather than
// sharing one instant across every fixture in a weekly round.
func TestNextLocalSaturdayKickoff_DiffersByCountry(t *testing.T) {
	spain := NextLocalSaturdayKickoff("es", 21, 0)
	japan := NextLocalSaturdayKickoff("jp", 21, 0)
	if spain.Equal(japan) {
		t.Errorf("expected Spain and Japan's Saturday 21:00 local kickoffs to be different instants, both were %v", spain)
	}
}

func TestLocalTimeToServerSchedule_UnknownCountryFallsBackToUTC(t *testing.T) {
	gotHour, gotMinute := LocalTimeToServerSchedule("zz", 14, 0)
	wantHour, wantMinute := LocalTimeToServerSchedule("__utc_reference__", 14, 0)
	if gotHour != wantHour || gotMinute != wantMinute {
		t.Errorf("expected an unmapped country code to fall back to UTC same as any other unmapped code, got (%d,%d) vs (%d,%d)",
			gotHour, gotMinute, wantHour, wantMinute)
	}
}

// NextLocalSaturdayKickoffAfter anchors on earliest rather than now: a Saturday whose 21:00 local
// is still ahead of earliest is used, one already past rolls to the next week, and an earliest on
// a Sunday lands six days later. Covers UTC-11 (American Samoa), UTC+0 (São Tomé) and UTC+13
// (Samoa), where "Saturday" starts at very different UTC instants.
func TestNextLocalSaturdayKickoffAfter(t *testing.T) {
	cases := []struct {
		country  string
		earliest time.Time
		want     time.Time
	}{
		// Saturday 31 Oct 2026, 20:46 UTC = 20:46 in São Tomé: tonight's 21:00 is still ahead.
		{"st", time.Date(2026, 10, 31, 20, 46, 0, 0, time.UTC), time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)},
		// Same Saturday, 21:30 UTC: tonight's kickoff passed, so next Saturday.
		{"st", time.Date(2026, 10, 31, 21, 30, 0, 0, time.UTC), time.Date(2026, 11, 7, 21, 0, 0, 0, time.UTC)},
		// Sunday 1 Nov: six days later.
		{"st", time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC), time.Date(2026, 11, 7, 21, 0, 0, 0, time.UTC)},
		// Saturday 07:00 UTC is Friday 20:00 in Pago Pago (UTC-11): its Saturday 21:00 is 08:00 UTC Sunday.
		{"as", time.Date(2026, 10, 31, 7, 0, 0, 0, time.UTC), time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)},
		// Saturday 07:00 UTC is Saturday 20:00 in Apia (UTC+13): its 21:00 is 08:00 UTC the same day.
		{"ws", time.Date(2026, 10, 31, 7, 0, 0, 0, time.UTC), time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)},
		// Saturday 09:00 UTC is already Saturday 22:00 in Apia: next week.
		{"ws", time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC), time.Date(2026, 11, 7, 8, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got := NextLocalSaturdayKickoffAfter(c.country, 21, 0, c.earliest)
		if !got.Equal(c.want) {
			t.Errorf("NextLocalSaturdayKickoffAfter(%q, %v) = %v, want %v", c.country, c.earliest, got.UTC(), c.want)
		}
	}
}

func TestNextLocalKickoffAfter(t *testing.T) {
	cases := []struct {
		name         string
		country      string
		hour, minute int
		earliest     time.Time
		want         time.Time
	}{
		// 13 Nov 2026, Madrid is UTC+1: 17:00 local is 16:00 UTC.
		{"later today", "es", 17, 0, time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC), time.Date(2026, 11, 13, 16, 0, 0, 0, time.UTC)},
		{"exactly at kickoff", "es", 17, 0, time.Date(2026, 11, 13, 16, 0, 0, 0, time.UTC), time.Date(2026, 11, 13, 16, 0, 0, 0, time.UTC)},
		{"already passed today", "es", 17, 0, time.Date(2026, 11, 13, 16, 1, 0, 0, time.UTC), time.Date(2026, 11, 14, 16, 0, 0, 0, time.UTC)},
		// Summer time: Madrid is UTC+2, so 20:45 local is 18:45 UTC.
		{"daylight saving", "es", 20, 45, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), time.Date(2026, 7, 1, 18, 45, 0, 0, time.UTC)},
		// The night before the October change (UTC+2) to a kickoff after it (UTC+1).
		{"across the DST change", "es", 17, 0, time.Date(2026, 10, 24, 20, 0, 0, 0, time.UTC), time.Date(2026, 10, 25, 16, 0, 0, 0, time.UTC)},
		// 23:00 UTC is already the next day in Apia (UTC+13), 12:00 local: its 17:00 is 04:00 UTC.
		{"local date ahead of UTC", "ws", 17, 0, time.Date(2026, 11, 13, 23, 0, 0, 0, time.UTC), time.Date(2026, 11, 14, 4, 0, 0, 0, time.UTC)},
		{"unknown country uses UTC", "zz", 20, 45, time.Date(2026, 11, 13, 21, 0, 0, 0, time.UTC), time.Date(2026, 11, 14, 20, 45, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got := NextLocalKickoffAfter(c.country, c.hour, c.minute, c.earliest)
		if !got.Equal(c.want) {
			t.Errorf("%s: NextLocalKickoffAfter(%q, %02d:%02d, %v) = %v, want %v", c.name, c.country, c.hour, c.minute, c.earliest, got.UTC(), c.want)
		}
	}
}
