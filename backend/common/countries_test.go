package common

import "testing"

// Every KnownCountries entry needs a name-pool table pair, a place-name pool, and team/stadium
// naming patterns - missing any of these doesn't fail outright (GetRandomName/RandomTeamName/
// RandomStadiumName all fall back to another country's pool), but silently generates a country
// using someone else's names/club style, which is exactly the kind of drift this guards against.
func TestEveryKnownCountryHasNamingData(t *testing.T) {
	for _, country := range KnownCountries {
		if _, ok := nameTableSuffixByCountry[country.Code]; !ok {
			t.Errorf("KnownCountries entry %q has no nameTableSuffixByCountry mapping", country.Code)
		}
		if _, ok := placeNamesByCountry[country.Code]; !ok {
			t.Errorf("KnownCountries entry %q has no placeNamesByCountry mapping", country.Code)
		}
		if _, ok := teamNamePatternsByCountry[country.Code]; !ok {
			t.Errorf("KnownCountries entry %q has no teamNamePatternsByCountry mapping", country.Code)
		}
		if _, ok := stadiumNamePatternsByCountry[country.Code]; !ok {
			t.Errorf("KnownCountries entry %q has no stadiumNamePatternsByCountry mapping", country.Code)
		}
	}
}

// IsKnownCountry should agree with KnownCountries for every entry - a basic sanity check on the
// lookup itself.
func TestIsKnownCountry(t *testing.T) {
	for _, country := range KnownCountries {
		if !IsKnownCountry(country.Code) {
			t.Errorf("IsKnownCountry(%q) = false, want true", country.Code)
		}
	}
	if IsKnownCountry("zz") {
		t.Error("IsKnownCountry(\"zz\") = true, want false")
	}
}

func TestSeededCountries_FallsBackToDefaultsWhenUnset(t *testing.T) {
	t.Setenv("SEEDED_COUNTRIES", "")

	got := SeededCountries()

	if len(got) != len(DefaultSeededCountries) {
		t.Fatalf("SeededCountries() = %v, want %v", got, DefaultSeededCountries)
	}
	for i, code := range DefaultSeededCountries {
		if got[i] != code {
			t.Fatalf("SeededCountries() = %v, want %v", got, DefaultSeededCountries)
		}
	}
}

func TestSeededCountries_UsesConfiguredKnownCountries(t *testing.T) {
	t.Setenv("SEEDED_COUNTRIES", " it, PT ,es")

	got := SeededCountries()

	want := []string{"it", "pt", "es"}
	if len(got) != len(want) {
		t.Fatalf("SeededCountries() = %v, want %v", got, want)
	}
	for i, code := range want {
		if got[i] != code {
			t.Fatalf("SeededCountries() = %v, want %v", got, want)
		}
	}
}

func TestSeededCountries_SkipsUnknownCodesButKeepsKnownOnes(t *testing.T) {
	t.Setenv("SEEDED_COUNTRIES", "es,zz,en")

	got := SeededCountries()

	want := []string{"es", "en"}
	if len(got) != len(want) {
		t.Fatalf("SeededCountries() = %v, want %v", got, want)
	}
	for i, code := range want {
		if got[i] != code {
			t.Fatalf("SeededCountries() = %v, want %v", got, want)
		}
	}
}

func TestSeededCountries_FallsBackToDefaultsWhenNothingConfiguredIsKnown(t *testing.T) {
	t.Setenv("SEEDED_COUNTRIES", "zz, yy")

	got := SeededCountries()

	if len(got) != len(DefaultSeededCountries) {
		t.Fatalf("SeededCountries() = %v, want fallback to defaults %v", got, DefaultSeededCountries)
	}
}
