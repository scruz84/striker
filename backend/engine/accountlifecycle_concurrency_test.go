package engine

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// Several --engine replicas fire the account lifecycle triggers at the same instant. Against a
// real Postgres (STRIKER_TEST_DB_DSN; skipped when unset): one release per idle owner, one warning
// per stage and one reset per club (no duplicated squad, one ledger row).

func lifecycleInt(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %s: %s", query, err)
	}
	return n
}

func TestAccountLifecycle_ConcurrentReplicasReleaseWarnAndResetOnce(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)

	if _, err := db.Exec(`INSERT INTO countries (name) VALUES ('jm') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var leagueId int64
	if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('jm', 1, 'I', 1) RETURNING league_id`).Scan(&leagueId); err != nil {
		t.Fatal(err)
	}

	newUser := func(daysIdle int, withClub bool) (userId, teamId int64) {
		email := fmt.Sprintf("lifecycle-%d@example.com", time.Now().UnixNano())
		if err := db.QueryRow(`INSERT INTO users (name, provider, email, country, last_seen_at) VALUES ('Idle Person', 'google', $1, 'jm', now() - make_interval(days => $2)) RETURNING user_id`,
			email, daysIdle).Scan(&userId); err != nil {
			t.Fatal(err)
		}
		if !withClub {
			return userId, 0
		}
		if _, err := db.Exec(`INSERT INTO teams (name, country, league_id, ai_level) VALUES ('Free Club', 'jm', $1, 1)`, leagueId); err != nil {
			t.Fatal(err)
		}
		if _, err := common.AssignTeam(dbCtx, userId, "jm"); err != nil {
			t.Fatalf("AssignTeam: %s", err)
		}
		if err := db.QueryRow(`SELECT team_id FROM teams WHERE owner_id = $1`, userId).Scan(&teamId); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE teams SET budget = 1000 WHERE team_id = $1`, teamId); err != nil {
			t.Fatal(err)
		}
		return userId, teamId
	}

	idleA, clubA := newUser(61, true)
	idleB, clubB := newUser(75, true)
	warned, clubW := newUser(48, true)
	active, clubActive := newUser(2, true)
	longIdle, _ := newUser(366, false)

	// Three replicas run the daily pass at the same instant.
	runConcurrently(replicas, func() { RunAccountLifecycle(dbCtx, time.Now()) })

	for _, club := range []int64{clubA, clubB} {
		if n := lifecycleInt(t, db, `SELECT count(*) FROM teams WHERE team_id = $1 AND owner_id IS NULL AND ai_level = 0 AND reset_pending_since IS NOT NULL`, club); n != 1 {
			t.Errorf("club %d should be released and queued exactly once, found %d", club, n)
		}
	}
	for _, club := range []int64{clubW, clubActive} {
		if n := lifecycleInt(t, db, `SELECT count(*) FROM teams WHERE team_id = $1 AND owner_id IS NOT NULL`, club); n != 1 {
			t.Errorf("club %d must keep its owner", club)
		}
	}
	if n := lifecycleInt(t, db, `SELECT count(*) FROM notifications WHERE user_id = $1 AND message_key = 'notifications.account.releaseWarning'`, warned); n != 1 {
		t.Errorf("an owner idle 48 days gets exactly one release warning, got %d", n)
	}
	if n := lifecycleInt(t, db, `SELECT count(*) FROM notifications WHERE user_id = $1`, active); n != 0 {
		t.Errorf("an active owner gets no warning, got %d", n)
	}
	if n := lifecycleInt(t, db, `SELECT count(*) FROM users WHERE user_id = $1`, longIdle); n != 0 {
		t.Errorf("an account idle more than %d days must be erased", common.AccountErasureDays)
	}
	if name := lifecycleInt(t, db, `SELECT count(*) FROM users WHERE user_id = $1 AND released_team_name IS NOT NULL`, idleA); name != 1 {
		t.Errorf("the released owner must be told which club was released")
	}

	// A re-run neither warns nor releases again.
	RunAccountLifecycle(dbCtx, time.Now())
	if n := lifecycleInt(t, db, `SELECT count(*) FROM notifications WHERE user_id = $1`, warned); n != 1 {
		t.Errorf("a re-run must not repeat the warning, got %d notifications", n)
	}

	// Three replicas run the reset pass at the same instant.
	runConcurrently(replicas, func() { ResetReleasedTeams(dbCtx) })
	for _, club := range []int64{clubA, clubB} {
		if n := lifecycleInt(t, db, `SELECT count(*) FROM players WHERE team_id = $1 AND in_academy = false`, club); n != int64(common.SquadSize) {
			t.Errorf("club %d has %d senior players after concurrent resets, want exactly %d", club, n, common.SquadSize)
		}
		if n := lifecycleInt(t, db, `SELECT count(*) FROM team_transactions WHERE team_id = $1 AND reason = $2`, club, common.TRANSACTION_TEAM_RESET); n != 1 {
			t.Errorf("club %d has %d reset ledger rows, want 1", club, n)
		}
		if n := lifecycleInt(t, db, `SELECT count(*) FROM teams WHERE team_id = $1 AND reset_pending_since IS NULL AND owner_id IS NULL AND ai_level >= 1`, club); n != 1 {
			t.Errorf("club %d should be a plain CPU club after its reset", club)
		}
	}
	_ = idleB
}
