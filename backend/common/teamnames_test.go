package common

import (
	"strings"
	"testing"
)

func TestRandomTeamName_UsesCountrySpecificPlaceName(t *testing.T) {
	for _, country := range []string{"es", "en", "de", "fr"} {
		name := RandomTeamName(country)
		if name == "" {
			t.Fatalf("expected a non-empty name for country %q", country)
		}
		found := false
		for _, place := range placeNamesByCountry[country] {
			if strings.Contains(name, place) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected name %q to contain one of %v", name, placeNamesByCountry[country])
		}
	}
}

func TestRandomTeamName_UnknownCountryFallsBackToEnglish(t *testing.T) {
	name := RandomTeamName("zz")
	if name == "" {
		t.Fatal("expected a non-empty fallback name")
	}
}
