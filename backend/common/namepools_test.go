package common

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/scruz84/striker/resources"
)

const (
	minPoolFirstNames = 300
	minPoolLastNames  = 500
)

var (
	namePoolFilePattern = regexp.MustCompile(`^init91_name_pool_([a-z]{2,3})\.sql$`)
	// One generated INSERT: target table, then the VALUES list up to its alias.
	namePoolInsertPattern = regexp.MustCompile(`(?s)INSERT INTO (names|lastnames)_([a-z]{2,3}) \([^)]*\)\s*SELECT v\.n, 1\.0 FROM \(VALUES (.*?)\) AS v\(n\)`)
	namePoolValuePattern  = regexp.MustCompile(`\('((?:[^']|'')*)'\)`)
)

// Every known country except Spain (whose ~29k-name dump predates the generator) has a generated
// name-pool migration with at least 300 distinct first names and 500 distinct surnames, all in
// Latin script - and no migration exists for a code that is not a known country.
func TestNamePoolFiles(t *testing.T) {
	files := map[string]string{} // code -> file name
	err := fs.WalkDir(resources.SQLFiles, "sql/namepools", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		m := namePoolFilePattern.FindStringSubmatch(d.Name())
		if m == nil {
			t.Errorf("%s: unexpected file in sql/namepools (want init91_name_pool_<code>.sql)", p)
			return nil
		}
		files[m[1]] = p
		return nil
	})
	if err != nil {
		t.Fatalf("walking sql/namepools: %s", err)
	}

	known := map[string]bool{}
	for _, country := range KnownCountries {
		known[country.Code] = true
		if country.Code == "es" {
			if _, ok := files["es"]; ok {
				t.Errorf("es keeps its existing name pool and must not have a generated file")
			}
			continue
		}
		if _, ok := files[country.Code]; !ok {
			t.Errorf("known country %q has no sql/namepools/init91_name_pool_%s.sql", country.Code, country.Code)
		}
	}
	for code := range files {
		if !known[code] {
			t.Errorf("sql/namepools has a file for %q, which is not a known country", code)
		}
	}

	for code, p := range files {
		raw, err := fs.ReadFile(resources.SQLFiles, p)
		if err != nil {
			t.Errorf("reading %s: %s", p, err)
			continue
		}
		pools := map[string]map[string]bool{"names": {}, "lastnames": {}}
		inserts := namePoolInsertPattern.FindAllStringSubmatch(string(raw), -1)
		if len(inserts) != 2 {
			t.Errorf("%s: %d name INSERTs, want 2 (first names and surnames)", p, len(inserts))
		}
		for _, insert := range inserts {
			if insert[2] != code {
				t.Errorf("%s: INSERT into a %s_%s table", p, insert[1], insert[2])
			}
			for _, value := range namePoolValuePattern.FindAllStringSubmatch(insert[3], -1) {
				name := strings.ReplaceAll(value[1], "''", "'")
				key := strings.ToLower(name)
				if pools[insert[1]][key] {
					t.Errorf("%s: duplicate %s name %q", p, insert[1], name)
				}
				pools[insert[1]][key] = true
				if len(name) > 40 {
					t.Errorf("%s: %q is longer than 40 characters", p, name)
				}
				for _, r := range name {
					if r == ' ' || r == '\'' || r == '-' || r == '.' {
						continue
					}
					if !unicode.Is(unicode.Latin, r) {
						t.Errorf("%s: %q has the non-Latin character %q", p, name, r)
						break
					}
				}
			}
		}
		if n := len(pools["names"]); n < minPoolFirstNames {
			t.Errorf("%s: %d distinct first names, want at least %d", p, n, minPoolFirstNames)
		}
		if n := len(pools["lastnames"]); n < minPoolLastNames {
			t.Errorf("%s: %d distinct surnames, want at least %d", p, n, minPoolLastNames)
		}
		if !strings.Contains(string(raw), "INSERT INTO name_pool_versions (country, version) VALUES ('"+code+"', 2)") {
			t.Errorf("%s: does not record pool version 2 for %q", p, code)
		}
	}
}

// The generated files sort after init.sql, which creates name_pool_versions and the name tables
// they fill.
func TestNamePoolFilesSortAfterTheBaseSchema(t *testing.T) {
	table, _ := schemaFileSortKey("init.sql")
	pool, _ := schemaFileSortKey(path.Join("namepools", "init91_name_pool_fj.sql"))
	if pool <= table {
		t.Errorf("name pool files sort at %d, not after the base schema at %d", pool, table)
	}
}
