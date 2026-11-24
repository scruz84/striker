package common

import (
	"testing"
	"time"
)

// TestPruneLedgerRows covers "Old ledger rows": team-level rows older than the cutoff go, a
// player's transfer row of the same age and recent rows stay, and the walk crosses several pages
// and stops at the first page that reaches the cutoff. It starts after the ledger's current last
// id, so other tests' rows in the shared database are never looked at.
func TestPruneLedgerRows(t *testing.T) {
	w := newArchiveWorld(t)
	_, _, team, _ := w.country(1)
	player := w.scan(`INSERT INTO players (first_name, last_name, team_id) VALUES ('Led', 'Ger', $1) RETURNING player_id`, team)
	start := w.scan(`SELECT COALESCE(max(transaction_id), 0) FROM team_transactions`)
	now := time.Now()
	row := func(age time.Duration, playerId any) int64 {
		return w.scan(`INSERT INTO team_transactions (team_id, amount, reason, description, created_at, player_id) VALUES ($1, -10, $2, 'test', $3, $4) RETURNING transaction_id`,
			team, TRANSACTION_WAGES, now.Add(-age), playerId)
	}
	var old []int64
	for i := 0; i < 5; i++ {
		old = append(old, row(200*24*time.Hour, nil))
	}
	transfer := row(200*24*time.Hour, player)
	// Pages of 2: [old old] [old old] [old transfer] [recent recent] [beyond]. The walk stops after
	// the page holding recent rows, so beyond, an old row out of id order, waits for a later pass.
	recent := row(10*24*time.Hour, nil)
	row(9*24*time.Hour, nil)
	beyond := row(200*24*time.Hour, nil)

	deleted, err := pruneLedgerRowsAfter(w.dbCtx, start, now.Add(-LedgerRetention), 2)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != int64(len(old)) {
		t.Errorf("deleted %d rows, want %d", deleted, len(old))
	}
	exists := func(id int64) bool {
		return w.scan(`SELECT count(*) FROM team_transactions WHERE transaction_id = $1`, id) == 1
	}
	for _, id := range old {
		if exists(id) {
			t.Errorf("old team-level row %d kept", id)
		}
	}
	for name, id := range map[string]int64{"player transfer row": transfer, "recent row": recent, "row after the stop": beyond} {
		if !exists(id) {
			t.Errorf("%s %d deleted", name, id)
		}
	}
}

// TestPruneOfferLog covers "Old offer log rows": rows older than the cutoff go, recent ones stay.
func TestPruneOfferLog(t *testing.T) {
	w := newArchiveWorld(t)
	now := time.Now()
	old := w.scan(`INSERT INTO transfer_offer_log (created_at) VALUES ($1) RETURNING event_id`, now.Add(-40*24*time.Hour))
	recent := w.scan(`INSERT INTO transfer_offer_log (created_at) VALUES ($1) RETURNING event_id`, now.Add(-time.Hour))
	deleted, err := PruneOldOfferLog(w.dbCtx, now.Add(-OfferLogRetention))
	if err != nil {
		t.Fatal(err)
	}
	if deleted < 1 {
		t.Errorf("deleted %d rows, want at least ours", deleted)
	}
	if w.scan(`SELECT count(*) FROM transfer_offer_log WHERE event_id = $1`, old) != 0 {
		t.Error("a 40-day-old offer log row was kept")
	}
	if w.scan(`SELECT count(*) FROM transfer_offer_log WHERE event_id = $1`, recent) != 1 {
		t.Error("a recent offer log row was deleted")
	}
}
