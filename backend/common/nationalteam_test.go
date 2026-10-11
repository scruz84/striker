package common

import (
	"math/rand"
	"testing"
)

// nationalTeamName is stored verbatim in teams.name and must never carry the old " National
// Team" suffix - every list localizes a national team's display name from its country code
// instead (see this function's own doc comment), so the stored value itself has to be the bare
// country name for that frontend lookup to actually match.
func TestNationalTeamName_HasNoSuffix(t *testing.T) {
	name := nationalTeamName("es")
	if name != "Spain" {
		t.Errorf("nationalTeamName(%q) = %q, want %q", "es", name, "Spain")
	}
}

func TestNationalTeamName_UnknownCountryFallsBackToUpperCode(t *testing.T) {
	name := nationalTeamName("zz")
	if name != "ZZ" {
		t.Errorf("nationalTeamName(%q) = %q, want %q", "zz", name, "ZZ")
	}
}

func TestPairNationalTeamCountries_EveryCountryAppearsExactlyOnce(t *testing.T) {
	countries := []string{"es", "en", "fr", "de", "br", "ar", "mx", "co"}
	r := rand.New(rand.NewSource(1))

	pairs := pairNationalTeamCountries(countries, r)

	seen := make(map[string]int)
	for _, pair := range pairs {
		seen[pair[0]]++
		seen[pair[1]]++
	}
	if len(seen) != len(countries) {
		t.Fatalf("expected all %d countries to appear, got %d distinct", len(countries), len(seen))
	}
	for _, country := range countries {
		if seen[country] != 1 {
			t.Errorf("country %q appeared %d times, want exactly 1", country, seen[country])
		}
	}
}

func TestPairNationalTeamCountries_PrefersSameRegion(t *testing.T) {
	// es/en are both "europe", br/ar are both "americas" (see regions.go) - with an even count in
	// each region, every pair should stay within its own region rather than crossing.
	countries := []string{"es", "en", "br", "ar"}
	r := rand.New(rand.NewSource(1))

	pairs := pairNationalTeamCountries(countries, r)

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	for _, pair := range pairs {
		if RegionForCountry(pair[0]) != RegionForCountry(pair[1]) {
			t.Errorf("pair %v crosses regions (%s vs %s), want same-region pairing", pair,
				RegionForCountry(pair[0]), RegionForCountry(pair[1]))
		}
	}
}

func TestPairNationalTeamCountries_OddTotalLeavesAtMostOneUnpaired(t *testing.T) {
	countries := []string{"es", "en", "fr", "br", "ar"}
	r := rand.New(rand.NewSource(2))

	pairs := pairNationalTeamCountries(countries, r)

	paired := 0
	for _, pair := range pairs {
		paired += 2
		if pair[0] == pair[1] {
			t.Fatalf("pair %v matches a country against itself", pair)
		}
	}
	if unpaired := len(countries) - paired; unpaired > 1 {
		t.Fatalf("expected at most 1 unpaired country, got %d", unpaired)
	}
}

func TestPairNationalTeamCountries_DeterministicGivenSameSeed(t *testing.T) {
	countries := []string{"es", "en", "fr", "de", "br", "ar", "jp", "kr", "ng", "eg"}

	first := pairNationalTeamCountries(countries, rand.New(rand.NewSource(42)))
	second := pairNationalTeamCountries(countries, rand.New(rand.NewSource(42)))

	if len(first) != len(second) {
		t.Fatalf("expected identical pairing counts for the same seed, got %d and %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("pairing %d differs between runs with the same seed: %v vs %v", i, first[i], second[i])
		}
	}
}

func TestPairNationalTeamCountries_EmptyInput(t *testing.T) {
	if pairs := pairNationalTeamCountries(nil, rand.New(rand.NewSource(1))); len(pairs) != 0 {
		t.Fatalf("expected no pairs for empty input, got %d", len(pairs))
	}
}

func TestPairNationalTeamCountries_SingleCountryGoesUnpaired(t *testing.T) {
	if pairs := pairNationalTeamCountries([]string{"es"}, rand.New(rand.NewSource(1))); len(pairs) != 0 {
		t.Fatalf("expected no pairs for a single country, got %d", len(pairs))
	}
}
