package handlers

import (
	"testing"

	"github.com/scruz84/striker/common"
)

func TestRandomSeededCountry_AlwaysReturnsASeededCountry(t *testing.T) {
	configured := common.SeededCountries()
	seeded := make(map[string]bool, len(configured))
	for _, c := range configured {
		seeded[c] = true
	}

	// Random, so run it enough times to have a real chance of catching a bug in the bound check.
	for i := 0; i < 50; i++ {
		country := randomSeededCountry()
		if !seeded[country] {
			t.Fatalf("randomSeededCountry() returned %q, which isn't in common.SeededCountries() %v", country, configured)
		}
	}
}
