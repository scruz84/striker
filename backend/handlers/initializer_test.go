package handlers

import "testing"

func TestRandomSeededCountry_AlwaysReturnsASeededCountry(t *testing.T) {
	seeded := make(map[string]bool, len(seededCountries))
	for _, c := range seededCountries {
		seeded[c] = true
	}

	// Random, so run it enough times to have a real chance of catching a bug in the bound check.
	for i := 0; i < 50; i++ {
		country := randomSeededCountry()
		if !seeded[country] {
			t.Fatalf("randomSeededCountry() returned %q, which isn't in seededCountries %v", country, seededCountries)
		}
	}
}
