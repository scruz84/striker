package common

import (
	"fmt"
	"io/fs"
	"math"
	"regexp"
	"sort"
	"strconv"

	"github.com/scruz84/striker/resources"
	log "github.com/sirupsen/logrus"
)

// SchemaExistsStmtSql checks for the "teams" table - created by the very first schema file
// (resources/sql/init.sql) and never dropped - as a cheap proxy for "has the schema already
// been applied to this database".
const SchemaExistsStmtSql = "SELECT to_regclass('public.teams') IS NOT NULL"

// initFileNumberPattern pulls the migration number out of a "initN_description.sql" filename,
// so files can be ordered numerically (init2, init3, ... init10, init11) instead of
// lexicographically (init10, init11, ..., init2, init3 - see schemaFileSortKey).
var initFileNumberPattern = regexp.MustCompile(`^init(\d+)_`)

// schemaFileSortKey orders the base schema first, then numbered "initN_*.sql" migrations in
// true numeric order, then anything else (the names_es_*/lastnames_es_* data dumps)
// lexicographically after all of those - matching the dependency order these files are written
// to require (each migration only ever references tables/columns created by init.sql or an
// earlier-numbered migration).
func schemaFileSortKey(name string) (int, string) {
	if name == "init.sql" {
		return 0, name
	}
	if m := initFileNumberPattern.FindStringSubmatch(name); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return n, name
		}
	}
	return math.MaxInt32, name
}

// EnsureSchema applies every resources/sql/*.sql file, in dependency order (see
// schemaFileSortKey), if this database doesn't have the schema yet - so `--initialize` (see
// handlers.InitializeDatabase) is self-sufficient against a genuinely empty database, not just
// one that Postgres's own docker-entrypoint-initdb.d mechanism (or a manual psql run - see
// backend/README.md) already set up.
//
// If the schema already exists, it instead re-applies just the numbered "initN_*.sql" migrations
// (see applyPendingMigrations) - called on every --svc/--engine startup (see service.StartSvc/
// engine.StartEngine) so pulling a branch that adds a new migration file is picked up on the next
// restart, with no manual psql step required.
func EnsureSchema(dbCtx *DatabaseContext) error {
	var exists bool
	if err := dbCtx.Db.QueryRow(SchemaExistsStmtSql).Scan(&exists); err != nil {
		return fmt.Errorf("checking whether the schema already exists: %w", err)
	}

	entries, err := fs.ReadDir(resources.SQLFiles, "sql")
	if err != nil {
		return fmt.Errorf("reading embedded sql files: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Slice(names, func(i, j int) bool {
		ri, ni := schemaFileSortKey(names[i])
		rj, nj := schemaFileSortKey(names[j])
		if ri != rj {
			return ri < rj
		}
		return ni < nj
	})

	if exists {
		return applyPendingMigrations(dbCtx, names)
	}

	log.Info("schema not found - applying resources/sql/*.sql before seeding the game")
	for _, name := range names {
		if err := applySchemaFile(dbCtx, name); err != nil {
			return err
		}
	}
	return nil
}

// applyPendingMigrations re-applies every numbered "initN_*.sql" migration (see
// initFileNumberPattern) against a database that already has a schema, so a new migration file
// added on a branch gets picked up automatically on the next startup instead of needing the
// manual psql step described in backend/README.md. Every one of these files is written to be
// idempotent by convention (CREATE TABLE/ADD COLUMN/CREATE INDEX ... IF NOT EXISTS, UPDATE ...
// WHERE x IS NULL), so re-running an already-applied one is harmless.
//
// init.sql itself and the name-pool dumps (names_es_*.sql/lastnames_es_*.sql) are deliberately
// excluded here, unlike in the fresh-database path above: init.sql's base CREATE TABLE statements
// have no IF NOT EXISTS, and the dumps have no uniqueness constraint stopping a second INSERT
// from duplicating every row, so both are only ever safe to apply once, against a genuinely empty
// database.
func applyPendingMigrations(dbCtx *DatabaseContext, names []string) error {
	log.Info("schema already exists - checking for pending incremental migrations")
	for _, name := range names {
		if !initFileNumberPattern.MatchString(name) {
			continue
		}
		if err := applySchemaFile(dbCtx, name); err != nil {
			return err
		}
	}
	return nil
}

func applySchemaFile(dbCtx *DatabaseContext, name string) error {
	contents, err := resources.SQLFiles.ReadFile("sql/" + name)
	if err != nil {
		return fmt.Errorf("reading %s: %w", name, err)
	}
	log.Infof("applying schema file %s", name)
	if _, err := dbCtx.Db.Exec(string(contents)); err != nil {
		return fmt.Errorf("applying %s: %w", name, err)
	}
	return nil
}
