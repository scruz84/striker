package common

import (
	"context"
	"database/sql"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
)

// The engine may run as several replicas (see docker-compose.yml), and every replica fires the
// same cron triggers at the same instant. Any job that reads state and then generates data from
// it (fixtures, cup rounds, national-team fixtures, simulations, call-ups) must therefore be
// guarded so only one replica performs a given piece of work - see openspec/specs/engine-replicas.
//
// Older locks key the single-bigint advisory lock space directly with an entity id (leagueId,
// cupId, teamId) or a fixed constant. Those keys stay as they are, so a replica running the
// previous build during a rolling deploy still contends on the same keys. New locks use
// NamespacedLockKey, which puts the namespace in the top 16 bits so they can never collide with a
// bare entity id or with hashtext(), whose values fit in 32 bits.
const (
	LockNamespaceCupTieSimulation   int64 = 1
	LockNamespaceWorldCupAdvance    int64 = 2
	LockNamespaceNationalTeamCallUp int64 = 3
	// LockNamespaceStandings serializes everything that changes one league's classification rows:
	// a result save (an increment, see SaveLeagueMatchResultTx) and the one-off rebuild of drifted
	// rows (engine.RepairStandingsOnce). Keyed by league id.
	LockNamespaceStandings int64 = 4
)

// NamespacedLockKey builds an advisory lock key for id inside namespace (see the constants above).
func NamespacedLockKey(namespace int64, id int64) int64 {
	return namespace<<48 | (id & (1<<48 - 1))
}

// TryWithSessionAdvisoryLock runs fn while holding the session-level advisory lock key, taken
// without waiting on a dedicated connection (a *sql.DB is a pool, so the lock and unlock must run
// on the same connection). It returns false without running fn when another session already holds
// the lock: that replica is doing the work, and the next tick picks up whatever it leaves behind.
// fn must re-read the state it acts on, since another replica may have finished the same work
// just before the lock was taken.
func TryWithSessionAdvisoryLock(dbCtx *DatabaseContext, key int64, fn func() error) (bool, error) {
	return withSessionAdvisoryLock(dbCtx, key, false, fn)
}

// WithSessionAdvisoryLock is TryWithSessionAdvisoryLock's blocking form: it waits for the lock
// instead of skipping, for work that must not be lost when another session holds it.
func WithSessionAdvisoryLock(dbCtx *DatabaseContext, key int64, fn func() error) error {
	_, err := withSessionAdvisoryLock(dbCtx, key, true, fn)
	return err
}

func withSessionAdvisoryLock(dbCtx *DatabaseContext, key int64, wait bool, fn func() error) (bool, error) {
	ctx := context.Background()
	conn, err := dbCtx.Db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	if wait {
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
			return false, err
		}
	} else {
		var acquired bool
		if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
			return false, err
		}
		if !acquired {
			return false, nil
		}
	}
	defer func() {
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", key); err != nil {
			log.Errorf("error releasing advisory lock %d: %s", key, err)
		}
	}()
	return true, fn()
}

// ClaimEngineJobSlotStmtSql records slot as jobName's last run only if it is newer than the one
// already stored. Under concurrent calls for the same slot, the losers wait on the winner's row
// lock and then see the WHERE fail, so exactly one call returns a row.
const ClaimEngineJobSlotStmtSql = "INSERT INTO engine_job_runs (job_name, last_slot) VALUES ($1, $2) " +
	"ON CONFLICT (job_name) DO UPDATE SET last_slot = EXCLUDED.last_slot " +
	"WHERE engine_job_runs.last_slot < EXCLUDED.last_slot RETURNING job_name"

// ClaimEngineJobSlot reports whether this replica won jobName's run for the period containing now:
// now truncated to period (a whole number of minutes, hours or days, aligned to UTC), so every
// replica firing the same cron trigger computes the same slot. Only the winner should run the job.
// A failed claim (database error) returns false: skipping one period is safer than running twice.
func ClaimEngineJobSlot(dbCtx *DatabaseContext, jobName string, now time.Time, period time.Duration) bool {
	slot := now.UTC().Truncate(period)
	var claimed string
	err := dbCtx.Db.QueryRow(ClaimEngineJobSlotStmtSql, jobName, slot).Scan(&claimed)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		log.Errorf("error claiming %s's run for slot %s: %s", jobName, slot.Format(time.RFC3339), err)
		return false
	}
	return true
}
