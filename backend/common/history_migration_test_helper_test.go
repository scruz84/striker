package common

import (
	"context"
	"database/sql"
	"testing"
)

// applyHistoryMigration applies one archived migration from resources/sql/history/ - the backfill
// tests of migrations folded into init.sql run them explicitly, since EnsureSchema never does.
func applyHistoryMigration(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("acquiring a connection for %s: %s", name, err)
	}
	defer conn.Close()
	if err := applySchemaFile(ctx, conn, "history/"+name); err != nil {
		t.Fatalf("applying %s: %s", name, err)
	}
}
