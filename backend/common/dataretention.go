package common

import "time"

// Fixed retention for append-only logs that nothing reads past a short window
// (openspec/specs/match-data-retention "Old ledger rows" and "Old offer log rows"). The daily
// engine.InstantArchiveJob deletes the rows older than these, whatever the instant archive setting.

// LedgerRetention is how long a team-level ledger row (wages, gate, sponsorship, prize money...,
// player_id NULL) is kept. The finance page only ever reads a team's latest RecentTransactionsLimit
// (50) rows, which every club writes within a few weeks of matches, so older ones are never read.
// Rows about a player (transfers, releases) stay: they are that player's history on his page.
const LedgerRetention = 180 * 24 * time.Hour

// OfferLogRetention is how long a transfer_offer_log row is kept. Only the last day is ever counted
// (getOffersInWindowsStmtSql).
const OfferLogRetention = 30 * 24 * time.Hour

// RetentionBatch is how many rows one retention statement looks at, so each statement's
// transaction stays short.
const RetentionBatch = 5000

// PruneLedgerPageStmtSql walks team_transactions in primary-key order from $1 (exclusive), one page
// of $2 rows, and deletes the page's team-level rows created before $3. It returns the page's last
// id, how many rows it deleted, and whether the whole page is older than $3: ids grow with
// created_at, so a page reaching $3 means everything after it is recent. Walking the primary key
// needs no extra index, and only the player-tied rows kept are walked again each day.
const PruneLedgerPageStmtSql = "WITH page AS (" +
	"SELECT transaction_id, player_id, created_at FROM team_transactions WHERE transaction_id > $1 ORDER BY transaction_id LIMIT $2), " +
	"del AS (DELETE FROM team_transactions t USING page p WHERE t.transaction_id = p.transaction_id " +
	"AND p.player_id IS NULL AND p.created_at < $3 RETURNING 1) " +
	"SELECT COALESCE((SELECT max(transaction_id) FROM page), 0), (SELECT count(*) FROM del), " +
	"COALESCE((SELECT bool_and(created_at < $3) FROM page), false)"

// PruneOfferLogStmtSql deletes up to $2 offer log rows created before $1, oldest first through
// transfer_offer_log_created_idx.
const PruneOfferLogStmtSql = "DELETE FROM transfer_offer_log WHERE event_id IN (" +
	"SELECT event_id FROM transfer_offer_log WHERE created_at < $1 ORDER BY created_at LIMIT $2)"

// PruneOldLedgerRows deletes every team-level ledger row created before cutoff, page by page, and
// returns how many it deleted, even when it stops on an error.
func PruneOldLedgerRows(dbCtx *DatabaseContext, cutoff time.Time) (int64, error) {
	return pruneLedgerRowsAfter(dbCtx, 0, cutoff, RetentionBatch)
}

// pruneLedgerRowsAfter is PruneOldLedgerRows starting after transaction id `after`, with pages of
// `batch` rows.
func pruneLedgerRowsAfter(dbCtx *DatabaseContext, after int64, cutoff time.Time, batch int) (int64, error) {
	var total int64
	for {
		var last, deleted int64
		var allOld bool
		if err := dbCtx.Db.QueryRow(PruneLedgerPageStmtSql, after, batch, cutoff).Scan(&last, &deleted, &allOld); err != nil {
			return total, err
		}
		total += deleted
		if last == 0 || !allOld {
			return total, nil
		}
		after = last
	}
}

// PruneOldOfferLog deletes every transfer_offer_log row created before cutoff, batch by batch, and
// returns how many it deleted, even when it stops on an error.
func PruneOldOfferLog(dbCtx *DatabaseContext, cutoff time.Time) (int64, error) {
	var total int64
	for {
		res, err := dbCtx.Db.Exec(PruneOfferLogStmtSql, cutoff, RetentionBatch)
		if err != nil {
			return total, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}
		total += n
		if n == 0 {
			return total, nil
		}
	}
}
