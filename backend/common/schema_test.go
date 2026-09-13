package common

import (
	"math/rand"
	"sort"
	"testing"
)

// TestApplyPendingMigrations_SkipsNonNumberedFiles guards the exclusion list in
// applyPendingMigrations's doc comment: on a database that already has a schema, only numbered
// "initN_*.sql" migrations may be re-applied - never init.sql (non-idempotent CREATE TABLE) or
// the name-pool dumps (plain INSERTs with no uniqueness constraint), both of which are only safe
// to run once against a genuinely empty database.
func TestApplyPendingMigrations_SkipsNonNumberedFiles(t *testing.T) {
	names := []string{
		"init.sql",
		"init2_league_tier.sql",
		"init30_player_birth_country.sql",
		"names_es_202304022215.sql",
		"lastnames_es_202304022215.sql",
	}
	var applied []string
	for _, name := range names {
		if initFileNumberPattern.MatchString(name) {
			applied = append(applied, name)
		}
	}
	want := []string{"init2_league_tier.sql", "init30_player_birth_country.sql"}
	if len(applied) != len(want) {
		t.Fatalf("got %v, want %v", applied, want)
	}
	for i := range want {
		if applied[i] != want[i] {
			t.Fatalf("got %v, want %v", applied, want)
		}
	}
}

func TestSchemaFileSortKey_NumericNotLexical(t *testing.T) {
	names := []string{
		"names_es_202304022215.sql",
		"init10_stadium.sql",
		"init2_league_tier.sql",
		"init.sql",
		"init9_admin.sql",
		"lastnames_es_202304022215.sql",
		"init29_player_position_minutes.sql",
	}
	// Shuffle so the sort itself is doing the work, not incidental input order.
	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })

	sort.Slice(names, func(i, j int) bool {
		ri, ni := schemaFileSortKey(names[i])
		rj, nj := schemaFileSortKey(names[j])
		if ri != rj {
			return ri < rj
		}
		return ni < nj
	})

	want := []string{
		"init.sql",
		"init2_league_tier.sql",
		"init9_admin.sql",
		"init10_stadium.sql",
		"init29_player_position_minutes.sql",
		"lastnames_es_202304022215.sql",
		"names_es_202304022215.sql",
	}
	if len(names) != len(want) {
		t.Fatalf("unexpected length: got %v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("position %d: got %q, want %q (full order: %v)", i, names[i], want[i], names)
		}
	}
}
