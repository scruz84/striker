package common

import "testing"

func TestRegionForCountry(t *testing.T) {
	cases := map[string]string{
		"es": "europe",
		"en": "europe",
		"fr": "europe",
		"de": "europe",
		"it": "europe",
		"pt": "europe",
		"br": "americas",
		"jp": "asia",
		"ng": "africa",
	}
	for country, want := range cases {
		if got := RegionForCountry(country); got != want {
			t.Errorf("RegionForCountry(%q) = %q, want %q", country, got, want)
		}
	}
}

func TestRegionForCountryFallsBackToEurope(t *testing.T) {
	for _, code := range []string{"", "xx", "zz"} {
		if got := RegionForCountry(code); got != "europe" {
			t.Errorf("RegionForCountry(%q) = %q, want europe", code, got)
		}
	}
}

// Every KnownCountries entry must have a region, or it silently falls back to "europe" - which
// is fine for an actually-European country but would be a real bug for anything else added to
// KnownCountries without a matching countryRegions entry.
func TestEveryKnownCountryHasARegion(t *testing.T) {
	for _, country := range KnownCountries {
		if _, ok := countryRegions[country.Code]; !ok {
			t.Errorf("KnownCountries entry %q has no countryRegions mapping", country.Code)
		}
	}
}

// The region follows the confederation: UEFA is Europe, CONMEBOL/CONCACAF the Americas, AFC and
// OFC Asia (the Pacific nations join the Asian Cup), CAF Africa.
func TestRegionForCountry_NewMembersFollowConfederation(t *testing.T) {
	cases := map[string]string{
		"dk": "europe", "nir": "europe", "kz": "europe", "il": "europe", "ua": "europe",
		"jm": "americas", "cr": "americas", "pr": "americas", "sr": "americas", "cw": "americas",
		"au": "asia", "iq": "asia", "tw": "asia", "nz": "asia", "fj": "asia", "pf": "asia",
		"ao": "africa", "cd": "africa", "mu": "africa", "ss": "africa", "zw": "africa",
	}
	for code, want := range cases {
		if got := RegionForCountry(code); got != want {
			t.Errorf("RegionForCountry(%q) = %q, want %q", code, got, want)
		}
	}
}
