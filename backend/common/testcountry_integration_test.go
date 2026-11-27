package common

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"
)

// unusedTestCountryCode returns a made-up four-letter country code (countries.name is varchar(4))
// starting with prefix that no country in the test database has yet. The database keeps every
// earlier run's countries, and with only 26^3 codes per prefix a blind random draw soon picks one
// of them, failing the test's INSERT on the unique name.
func unusedTestCountryCode(t *testing.T, db *sql.DB, prefix byte) string {
	t.Helper()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for attempt := 0; ; attempt++ {
		code := []byte{prefix, 0, 0, 0}
		for i := 1; i < 4; i++ {
			code[i] = byte('a' + r.Intn(26))
		}
		var taken bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM countries WHERE name = $1)`, string(code)).Scan(&taken); err != nil {
			t.Fatalf("checking country code %s: %s", code, err)
		}
		if !taken || attempt == 1000 {
			return string(code)
		}
	}
}
