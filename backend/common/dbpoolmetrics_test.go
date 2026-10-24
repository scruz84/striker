package common

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/scruz84/striker/common/metrictest"
)

// stubDriver is a database/sql driver that never connects: enough for sql.DB.Stats() to report
// pool state without a Postgres.
type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("stub driver does not connect")
}

func init() { sql.Register("stub-dbpool", stubDriver{}) }

// TestRegisterDBPoolMetrics_ReportsInUseAndIdle: the pool gauge has one series per state, and the
// wait counters are exported; a second registration is a no-op.
func TestRegisterDBPoolMetrics_ReportsInUseAndIdle(t *testing.T) {
	metrictest.Install()
	db, err := sql.Open("stub-dbpool", "")
	if err != nil {
		t.Fatalf("sql.Open: %s", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := RegisterDBPoolMetrics(db); err != nil {
		t.Fatalf("RegisterDBPoolMetrics: %s", err)
	}
	if err := RegisterDBPoolMetrics(db); err != nil {
		t.Fatalf("a second RegisterDBPoolMetrics call must be a no-op, got %s", err)
	}

	for _, state := range []string{"in_use", "idle"} {
		if got := metrictest.Points(t, "striker.db.connections", attribute.String("state", state)); got != 1 {
			t.Errorf("striker.db.connections{state=%s} has %d series, want 1", state, got)
		}
	}
	collected := metrictest.Collect(t)
	for _, name := range []string{"striker.db.connection_waits", "striker.db.connection_wait_time"} {
		if _, ok := collected[name]; !ok {
			t.Errorf("%s was not exported", name)
		}
	}
}
