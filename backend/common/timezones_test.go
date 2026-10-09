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

func TestLocalTimeToServerSchedule_UnknownCountryFallsBackToUTC(t *testing.T) {
	gotHour, gotMinute := LocalTimeToServerSchedule("zz", 14, 0)
	wantHour, wantMinute := LocalTimeToServerSchedule("__utc_reference__", 14, 0)
	if gotHour != wantHour || gotMinute != wantMinute {
		t.Errorf("expected an unmapped country code to fall back to UTC same as any other unmapped code, got (%d,%d) vs (%d,%d)",
			gotHour, gotMinute, wantHour, wantMinute)
	}
}
