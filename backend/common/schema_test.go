package common

import (
	"math/rand"
	"path"
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

// TestCollectSchemaFiles_FindsArchivedMigrations guards against the exact bug reported in
// production: resources/sql/history holds old numbered migrations (archived there once every
// currently-supported database had already applied them - see applyPendingMigrations's doc
// comment), and a plain fs.ReadDir("sql") only sees sql/'s immediate entries, silently skipping
// all of them. A database whose schema predated the archival then permanently missed whatever
// those files created (e.g. init34_friendly_lineup.sql's `friendly_lineup` table and
// init35_free_agent_premium.sql's `players.last_team_id` column), because collectSchemaFiles is
// the only thing that decides what applyPendingMigrations considers "pending".
func TestCollectSchemaFiles_FindsArchivedMigrations(t *testing.T) {
	names, err := collectSchemaFiles()
	if err != nil {
		t.Fatalf("collectSchemaFiles: %s", err)
	}
	want := []string{
		"history/init34_friendly_lineup.sql",
		"history/init35_free_agent_premium.sql",
	}
	for _, w := range want {
		found := false
		for _, n := range names {
			if n == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("collectSchemaFiles didn't find archived migration %q in %v", w, names)
		}
	}
}

// TestApplyPendingMigrations_RecognizesArchivedFiles guards the other half of the same bug: even
// once a file is found inside history/, it must still be recognized as a numbered migration by
// its base filename, not skipped because the subdirectory prefix stops initFileNumberPattern's
// anchored `^init` match.
func TestApplyPendingMigrations_RecognizesArchivedFiles(t *testing.T) {
	names := []string{
		"init.sql",
		"history/init34_friendly_lineup.sql",
		"history/init35_free_agent_premium.sql",
		"names_es_202304022215.sql",
	}
	var applied []string
	for _, name := range names {
		if initFileNumberPattern.MatchString(path.Base(name)) {
			applied = append(applied, name)
		}
	}
	want := []string{"history/init34_friendly_lineup.sql", "history/init35_free_agent_premium.sql"}
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
