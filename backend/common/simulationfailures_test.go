package common

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"connection failure SQLSTATE 08006", &pgconn.PgError{Code: "08006"}, true},
		{"connection exception class 08001", &pgconn.PgError{Code: "08001"}, true},
		{"admin shutdown SQLSTATE 57P01", &pgconn.PgError{Code: "57P01"}, true},
		{"too many connections SQLSTATE 53300", &pgconn.PgError{Code: "53300"}, true},
		{"bad connection", driver.ErrBadConn, true},
		{"connection done", sql.ErrConnDone, true},
		{"pool wait deadline", context.DeadlineExceeded, true},
		{"wrapped bad connection", fmt.Errorf("saving: %w", driver.ErrBadConn), true},
		{"wrapped SQLSTATE", fmt.Errorf("saving: %w", &pgconn.PgError{Code: "57P01"}), true},
		{"connect error", &pgconn.ConnectError{}, true},
		{"unique violation SQLSTATE 23505", &pgconn.PgError{Code: "23505"}, false},
		{"foreign key violation SQLSTATE 23503", &pgconn.PgError{Code: "23503"}, false},
		{"plain error", errors.New("index out of range"), false},
		{"no rows", sql.ErrNoRows, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConnectionError(tt.err); got != tt.want {
				t.Errorf("IsConnectionError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// The SQL eligibility fragments spell the limits out as literals, because a Go const can't format
// an int into a string; this keeps them in step with the Go constants.
func TestSimulationEligibleSql_MatchesConstants(t *testing.T) {
	if want := fmt.Sprintf("simulation_attempts < %d", MaxSimulationAttempts); simulationAttemptsLeftSql != want {
		t.Errorf("simulationAttemptsLeftSql = %q, want %q", simulationAttemptsLeftSql, want)
	}
	if want := fmt.Sprintf("simulation_attempts >= %d", MaxSimulationAttempts); simulationAttemptsUsedSql != want {
		t.Errorf("simulationAttemptsUsedSql = %q, want %q", simulationAttemptsUsedSql, want)
	}
	if want := fmt.Sprintf("interval '%d minutes'", int(SimulationRetryBackoff.Minutes())); !strings.Contains(simulationEligibleSql, want) {
		t.Errorf("simulationEligibleSql = %q, want it to contain %q", simulationEligibleSql, want)
	}
}

func TestStandingsDeltaFor(t *testing.T) {
	tests := []struct {
		goalsFor, goalsAgainst int
		want                   StandingsDelta
	}{
		{2, 1, StandingsDelta{Points: 3, Wins: 1}},
		{0, 0, StandingsDelta{Points: 1, Draws: 1}},
		{1, 3, StandingsDelta{Losses: 1}},
	}
	for _, tt := range tests {
		if got := StandingsDeltaFor(tt.goalsFor, tt.goalsAgainst); got != tt.want {
			t.Errorf("StandingsDeltaFor(%d, %d) = %+v, want %+v", tt.goalsFor, tt.goalsAgainst, got, tt.want)
		}
	}
}
