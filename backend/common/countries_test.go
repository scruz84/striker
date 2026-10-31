package common

import (
	"strings"
	"testing"
)

// Every KnownCountries entry needs a name-pool table pair, a place-name pool, and team/stadium
// naming patterns - missing any of these doesn't fail outright (GetRandomName/RandomTeamName/
// RandomStadiumName all fall back to another country's pool), but silently generates a country
// using someone else's names/club style, which is exactly the kind of drift this guards against.
// minPlaceNames is the smallest place-name pool a country may have, so RandomTeamName and
// RandomStadiumName produce varied names across an 18-team league.
const minPlaceNames = 12

func TestEveryKnownCountryHasNamingData(t *testing.T) {
	for _, country := range KnownCountries {
		if _, ok := nameTableSuffixByCountry[country.Code]; !ok {
			t.Errorf("KnownCountries entry %q has no nameTableSuffixByCountry mapping", country.Code)
		}
		places, ok := placeNamesByCountry[country.Code]
		if !ok {
			t.Errorf("KnownCountries entry %q has no placeNamesByCountry mapping", country.Code)
		} else if len(places) < minPlaceNames {
			t.Errorf("KnownCountries entry %q has %d place names, want at least %d", country.Code, len(places), minPlaceNames)
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

// The catalog is the FIFA membership: 211 associations, split by confederation. Codes are ISO
// 3166-1 alpha-2 except the home nations (en, sco, wal, nir) and Kosovo (xk).
func TestKnownCountriesAreFIFAMembership(t *testing.T) {
	confederations := map[string]string{
		"uefa": "es en fr de it pt nl se pl gr be tr at sco wal ie ru rs hr ba si mk me xk " +
			"al ad am az by bg cy cz dk ee fo fi ge gi hu is il kz lv li lt lu mt md nir no ro sm sk ch ua",
		"conmebol": "br ar co cl pe uy ec ve py bo",
		"concacaf": "mx us ca ai ag aw bs bb bz bm vg ky cr cu cw dm do sv gd gt gy ht hn jm ms ni pa pr " +
			"kn lc vc sr tt tc vi",
		"afc": "jp kr cn in th vn id ph sa ir qa ae uz af au bh bd bt bn kh tw gu hk iq jo kw kg la lb mo " +
			"my mv mn mm np kp om pk ps sg lk sy tj tl tm ye",
		"caf": "ng eg gh ci cm sn za ma dz ke tn tz ml ao bj bw bf bi cv cf td km cg cd dj gq er sz et ga " +
			"gm gn gw ls lr ly mg mw mr mu mz na ne rw st sc sl so ss sd tg ug zm zw",
		"ofc": "as ck fj nc nz pg ws sb pf to vu",
	}
	wantCounts := map[string]int{"uefa": 55, "conmebol": 10, "concacaf": 35, "afc": 46, "caf": 54, "ofc": 11}

	confederationOf := map[string]string{}
	for name, codes := range confederations {
		for _, code := range strings.Fields(codes) {
			if prev, dup := confederationOf[code]; dup {
				t.Errorf("code %q listed in both %s and %s", code, prev, name)
			}
			confederationOf[code] = name
		}
	}

	if len(KnownCountries) != 211 {
		t.Errorf("len(KnownCountries) = %d, want 211", len(KnownCountries))
	}
	codes := map[string]bool{}
	names := map[string]bool{}
	counts := map[string]int{}
	for _, country := range KnownCountries {
		if codes[country.Code] {
			t.Errorf("duplicate country code %q", country.Code)
		}
		if names[country.Name] {
			t.Errorf("duplicate country name %q", country.Name)
		}
		codes[country.Code] = true
		names[country.Name] = true
		conf, ok := confederationOf[country.Code]
		if !ok {
			t.Errorf("country %q (%s) belongs to no confederation in this test", country.Code, country.Name)
			continue
		}
		counts[conf]++
	}
	for conf, want := range wantCounts {
		if counts[conf] != want {
			t.Errorf("%s has %d known countries, want %d", conf, counts[conf], want)
		}
	}
}

// The four home nations are separate countries, and the FIFA members that are territories keep
// the codes the flag set and the locale files use.
func TestKnownCountriesNamedExamples(t *testing.T) {
	want := map[string]string{
		"en": "England", "sco": "Scotland", "wal": "Wales", "nir": "Northern Ireland",
		"fo": "Faroe Islands", "gi": "Gibraltar", "hk": "Hong Kong", "mo": "Macau", "tw": "Chinese Taipei",
		"ps": "Palestine", "pr": "Puerto Rico", "cw": "Curaçao", "aw": "Aruba", "pf": "Tahiti", "nc": "New Caledonia",
	}
	got := map[string]string{}
	for _, country := range KnownCountries {
		got[country.Code] = country.Name
	}
	for code, name := range want {
		if got[code] != name {
			t.Errorf("KnownCountries[%q] = %q, want %q", code, got[code], name)
		}
	}
}

// nameTableSuffix is concatenated into SQL, so anything but 2-3 lowercase letters must panic.
func TestNameTableSuffixRejectsUnsafeCodes(t *testing.T) {
	for _, code := range []string{"es", "sco", "nir"} {
		if got := nameTableSuffix(code); got != code {
			t.Errorf("nameTableSuffix(%q) = %q", code, got)
		}
	}
	for _, code := range []string{"", "e", "ES", "es;", "es_1", "toolong", "e s", "é"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("nameTableSuffix(%q) did not panic", code)
				}
			}()
			nameTableSuffix(code)
		}()
	}
}

// Every known country has a derived name-table suffix, and nothing else does.
func TestNameTableSuffixesCoverKnownCountries(t *testing.T) {
	if len(nameTableSuffixByCountry) != len(KnownCountries) {
		t.Errorf("%d suffixes for %d known countries", len(nameTableSuffixByCountry), len(KnownCountries))
	}
}

// RandomTeamName/RandomStadiumName fall back to the English pools for a code without data, so a
// country that quietly borrowed English naming would still "work". Every country's own data is
// used when its names can only come from its own place-name pool.
func TestRandomNamesUseEachCountrysOwnPlaceNames(t *testing.T) {
	for _, country := range KnownCountries {
		places := placeNamesByCountry[country.Code]
		for i := 0; i < 20; i++ {
			for _, name := range []string{RandomTeamName(country.Code), RandomStadiumName(country.Code)} {
				found := false
				for _, place := range places {
					if strings.Contains(name, place) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("name %q for %q uses no place from its own pool", name, country.Code)
				}
			}
		}
	}
}
