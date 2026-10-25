package common

import (
	"os"
	"path/filepath"
	"testing"
)

// The client shows each country's flag from frontend/public/flags/<code>.svg (see
// CountryFlag.tsx), so a country added to KnownCountries without its file would render with no
// flag. Skipped when the frontend tree is absent (a backend-only checkout).
func TestKnownCountriesHaveFlagFiles(t *testing.T) {
	dir := filepath.Join("..", "..", "frontend", "public", "flags")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no frontend flags directory: %s", err)
	}
	for _, country := range KnownCountries {
		if _, err := os.Stat(filepath.Join(dir, country.Code+".svg")); err != nil {
			t.Errorf("KnownCountries entry %q has no flag file: %s", country.Code, err)
		}
	}
}
