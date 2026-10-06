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
