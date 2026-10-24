package common

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// MaxSimulationAttempts is how many simulation attempts a league match, friendly or cup tie gets
// before engine.SettleDefaultResults settles it 0-0 instead (see openspec match-scheduling,
// "Default result after repeated failures"). Every claim counts as one attempt, in the claiming
// statement itself (see GetSimulableMatches, GetStuckMatches, ...), except a failure caused by an
// unreachable database (see IsConnectionError, RecordSimulationFailure).
const MaxSimulationAttempts = 3

// SimulationRetryBackoff is the least time between two attempts at the same fixture, measured from
// the previous claim (last_simulation_attempt_at). It keeps a failing fixture from being retried
// every minute, and gives the default result room to land 30-50 minutes after the first attempt.
const SimulationRetryBackoff = 10 * time.Minute

// simulationErrorMaxLength caps the error text stored in last_simulation_error.
const simulationErrorMaxLength = 500

// simulationAttemptsLeftSql / simulationAttemptsUsedSql / simulationEligibleSql are the parts of the selectors' WHERE clauses
// that keep a fixture out of reach once it has used its attempts (and, for the latter, while it is
// still inside the retry backoff). They spell out MaxSimulationAttempts and SimulationRetryBackoff
// as literals (a Go const can't format an int into a string) -
// TestSimulationEligibleSql_MatchesConstants keeps the two in sync.
const simulationAttemptsLeftSql = "simulation_attempts < 3"
const simulationAttemptsUsedSql = "simulation_attempts >= 3"
const simulationEligibleSql = simulationAttemptsLeftSql + " and " +
	"(last_simulation_attempt_at is null or last_simulation_attempt_at <= now() - interval '10 minutes')"

// simulationClaimSetSql is the part of every claim's SET clause that counts the attempt.
const simulationClaimSetSql = "simulation_attempts = simulation_attempts + 1, last_simulation_attempt_at = now()"

// The three kinds of fixture a simulation can fail on, as used by RecordSimulationFailure and
// pending_post_match_effects.kind.
const (
	SimulationKindLeague   = "league"
	SimulationKindFriendly = "friendly"
	SimulationKindCup      = "cup"
)

// ErrAlreadyPlayed is returned by the Save*ResultTx functions when the fixture was already played
// by the time the result was saved - another simulation won, so the caller must stop without
// booking anything.
var ErrAlreadyPlayed = errors.New("fixture already played")

// IsConnectionError reports whether err means the database or its connection pool was unreachable,
// as opposed to a deterministic failure of the simulation or of a statement. Such a failure says
// nothing about the fixture, so it is not counted towards MaxSimulationAttempts: an outage must
// not default a whole round to 0-0.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, sql.ErrConnDone) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Class 08 is "connection exception"; 57P01 is admin_shutdown and 53300 is
		// too_many_connections.
		return strings.HasPrefix(pgErr.Code, "08") || pgErr.Code == "57P01" || pgErr.Code == "53300"
	}
	return false
}

const recordLeagueSimulationFailureStmtSql = "UPDATE matches SET in_simulation = false, last_simulation_error = $2, " +
	"simulation_attempts = CASE WHEN $3::boolean THEN GREATEST(simulation_attempts - 1, 0) ELSE simulation_attempts END " +
	"WHERE match_id = $1 AND played = false"

const recordFriendlySimulationFailureStmtSql = "UPDATE friendly_matches SET in_simulation = false, last_simulation_error = $2, " +
	"simulation_attempts = CASE WHEN $3::boolean THEN GREATEST(simulation_attempts - 1, 0) ELSE simulation_attempts END " +
	"WHERE match_id = $1 AND played = false"

const recordCupTieSimulationFailureStmtSql = "UPDATE cup_ties SET in_simulation = false, last_simulation_error = $2, " +
	"simulation_attempts = CASE WHEN $3::boolean THEN GREATEST(simulation_attempts - 1, 0) ELSE simulation_attempts END " +
	"WHERE tie_id = $1 AND played = false"

// RecordSimulationFailure releases a fixture's claim after a failed simulation (an error or a
// panic) and stores why, so the retry can come from the next per-minute tick once the backoff has
// passed instead of waiting for stuck recovery - and so the admin overview can show the reason.
// The attempt was already counted when the fixture was claimed; if cause is a connection error
// (see IsConnectionError) it is taken back. Only touches a fixture that is still unplayed, so a
// simulation that lost the race to another one never overwrites its winner. kind is one of the
// SimulationKind constants and id the match, friendly or tie id.
func RecordSimulationFailure(dbCtx *DatabaseContext, kind string, id int64, cause error) error {
	var stmt string
	switch kind {
	case SimulationKindLeague:
		stmt = recordLeagueSimulationFailureStmtSql
	case SimulationKindFriendly:
		stmt = recordFriendlySimulationFailureStmtSql
	case SimulationKindCup:
		stmt = recordCupTieSimulationFailureStmtSql
	default:
		return fmt.Errorf("unknown simulation kind %q", kind)
	}
	message := "unknown error"
	if cause != nil {
		message = cause.Error()
	}
	if runes := []rune(message); len(runes) > simulationErrorMaxLength {
		message = string(runes[:simulationErrorMaxLength])
	}
	_, err := dbCtx.Db.Exec(stmt, id, message, IsConnectionError(cause))
	return err
}

// claimCupTieAttemptStmtSql counts one more attempt at a cup tie, under the tie's simulation
// lock (see engine.simulateCupTieOnce). A cup tie has no in_simulation claim, so this is its
// claiming statement: the selector's eligibility filter is repeated here because the replica that
// waited for the lock may be acting on a selection made before the previous holder's failed
// attempt was recorded.
const claimCupTieAttemptStmtSql = "UPDATE cup_ties SET " + simulationClaimSetSql + " " +
	"WHERE tie_id = $1 AND played = false AND " + simulationEligibleSql

// ClaimCupTieAttempt counts an attempt at tieId and reports whether the caller may simulate it:
// false when the tie was played, has used its attempts, or failed too recently.
func ClaimCupTieAttempt(dbCtx *DatabaseContext, tieId int64) (bool, error) {
	result, err := dbCtx.Db.Exec(claimCupTieAttemptStmtSql, tieId)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
