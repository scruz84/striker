package common

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"math"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/scruz84/striker/resources"
	log "github.com/sirupsen/logrus"
)

// schemaMigrationLockKey is an arbitrary Postgres advisory lock key (see pg_advisory_lock),
// unique only in the sense that nothing else in this app uses it. EnsureSchema is called
// independently by both service.StartSvc and engine.StartEngine at startup - running --svc
// --engine together in one process (see cmd/striker.go) starts them as concurrent goroutines
// against the same database, and running them as separate containers started at the same time
// (see docker-compose.yml) races the same way across processes. Without this lock, two callers
// can both pass a `CREATE TABLE IF NOT EXISTS`'s existence check before either commits, then both
// attempt the literal CREATE - the loser gets a duplicate-key error on Postgres's internal
// pg_type catalog, since IF NOT EXISTS alone isn't safe against concurrent DDL (a documented
// Postgres limitation, not a bug in any individual migration file).
const schemaMigrationLockKey = 8817224

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
// earlier-numbered migration). The pattern match runs against the base filename alone, so a name
// carrying a subdirectory prefix still sorts by its number.
func schemaFileSortKey(name string) (int, string) {
	base := path.Base(name)
	if base == "init.sql" {
		return 0, name
	}
	if m := initFileNumberPattern.FindStringSubmatch(base); m != nil {
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
	ctx := context.Background()
	// A session-level advisory lock is tied to the connection that acquired it, not to dbCtx.Db
	// as a whole (a *sql.DB is a pool - a bare dbCtx.Db.Exec for the unlock could run on a
	// different underlying connection than the lock did) - so this dedicated connection carries
	// every statement EnsureSchema runs, lock through unlock.
	conn, err := dbCtx.Db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquiring a dedicated connection for schema migration: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", schemaMigrationLockKey); err != nil {
		return fmt.Errorf("acquiring schema migration lock: %w", err)
	}
	defer func() {
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", schemaMigrationLockKey); err != nil {
			log.Errorf("error releasing schema migration lock: %s", err)
		}
	}()

	var exists bool
	if err := conn.QueryRowContext(ctx, SchemaExistsStmtSql).Scan(&exists); err != nil {
		return fmt.Errorf("checking whether the schema already exists: %w", err)
	}

	names, err := collectSchemaFiles()
	if err != nil {
		return err
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
		return applyPendingMigrations(ctx, conn, names)
	}

	log.Info("schema not found - applying resources/sql/*.sql before seeding the game")
	for _, name := range names {
		if err := applySchemaFile(ctx, conn, name); err != nil {
			return err
		}
	}
	return nil
}

// schemaHistoryDir is the subdirectory of resources/sql/ where migrations get archived once
// they've been folded into init.sql. Files there are never applied automatically: several aren't
// idempotent (init21_contracts.sql's unguarded UPDATE re-randomizes every contract, init19/init22's
// name-pool INSERTs duplicate rows), and init.sql already contains everything they did.
const schemaHistoryDir = "history"

// collectSchemaFiles lists the embedded resources/sql/*.sql files, including any in subdirectories
// other than history/ (see schemaHistoryDir). Returned names are relative to sql/ itself (no
// leading "sql/"), matching applySchemaFile's own convention.
func collectSchemaFiles() ([]string, error) {
	var names []string
	err := fs.WalkDir(resources.SQLFiles, "sql", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == schemaHistoryDir {
				return fs.SkipDir
			}
			return nil
		}
		names = append(names, strings.TrimPrefix(p, "sql/"))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading embedded sql files: %w", err)
	}
	return names, nil
}

// applyPendingMigrations re-applies every numbered "initN_*.sql" migration (see
// initFileNumberPattern) against a database that already has a schema, so a new migration file
// added on a branch gets picked up automatically on the next startup instead of needing the
// manual psql step described in backend/README.md. Every one of these files must be idempotent
// (CREATE TABLE/ADD COLUMN/CREATE INDEX ... IF NOT EXISTS, UPDATE ... WHERE x IS NULL, INSERT ...
// WHERE NOT EXISTS), because they run again on every startup. Archived migrations under history/
// are never in names (see collectSchemaFiles).
//
// init.sql itself and the name-pool dumps (names_es_*.sql/lastnames_es_*.sql) are deliberately
// excluded here, unlike in the fresh-database path above: init.sql's base CREATE TABLE statements
// have no IF NOT EXISTS, and the dumps have no uniqueness constraint stopping a second INSERT
// from duplicating every row, so both are only ever safe to apply once, against a genuinely empty
// database.
func applyPendingMigrations(ctx context.Context, conn *sql.Conn, names []string) error {
	log.Info("schema already exists - checking for pending incremental migrations")
	for _, name := range names {
		if !initFileNumberPattern.MatchString(path.Base(name)) {
			continue
		}
		if err := applySchemaFile(ctx, conn, name); err != nil {
			return err
		}
	}
	return nil
}

func applySchemaFile(ctx context.Context, conn *sql.Conn, name string) error {
	contents, err := resources.SQLFiles.ReadFile("sql/" + name)
	if err != nil {
		return fmt.Errorf("reading %s: %w", name, err)
	}
	log.Infof("applying schema file %s", name)
	if _, err := conn.ExecContext(ctx, string(contents)); err != nil {
		return fmt.Errorf("applying %s: %w", name, err)
	}
	return nil
}
