// Package resources embeds the SQL schema/seed-data files under resources/sql so the binary can
// apply them itself (see common.EnsureSchema) instead of depending on them being run separately -
// either by Postgres's own docker-entrypoint-initdb.d (fresh volumes only) or by hand via psql.
package resources

import "embed"

//go:embed sql
var SQLFiles embed.FS
