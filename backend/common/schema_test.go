package common

import (
	"math/rand"
	"sort"
	"strings"
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

// TestCollectSchemaFiles_SkipsHistory guards against re-applying archived migrations on every
// startup: resources/sql/history/ holds files that aren't idempotent (init21_contracts.sql's
// unguarded UPDATE re-randomizes every player's contract; init19/init22 duplicate name-pool rows),
// and init.sql already contains everything they did.
func TestCollectSchemaFiles_SkipsHistory(t *testing.T) {
	names, err := collectSchemaFiles()
	if err != nil {
		t.Fatalf("collectSchemaFiles: %s", err)
	}
	if len(names) == 0 {
		t.Fatal("collectSchemaFiles returned no files")
	}
	hasInit := false
	for _, n := range names {
		if strings.HasPrefix(n, schemaHistoryDir+"/") {
			t.Errorf("collectSchemaFiles returned archived file %q", n)
		}
		if n == "init.sql" {
			hasInit = true
		}
	}
	if !hasInit {
		t.Errorf("collectSchemaFiles didn't find init.sql in %v", names)
	}
}

func TestSchemaFileSortKey_NumericNotLexical(t *testing.T) {
	names := []string{
		"names_es_202304022215.sql",
		"init10_stadium.sql",
		"history/init2_league_tier.sql",
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

	// history/init2_league_tier.sql sorts right where a bare init2_league_tier.sql would (by
	// number, base filename only) - this is the "archived under history/" case above's sort-order
	// half: numeric ordering must not be defeated by the added subdirectory prefix.
	want := []string{
		"init.sql",
		"history/init2_league_tier.sql",
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
