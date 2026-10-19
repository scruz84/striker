package common

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Limits on the offers CPU teams make for players they don't see on the transfer list. They exist
// to keep a human manager's inbox sane and the market from degenerating, not to ration signings:
// there is no per-season quota of any kind.
const (
	// AiMaxOpenOffersPerHumanPlayer is how many CPU offers may be open on one human-owned player
	// at once, always from different teams (the unique constraint on offers already stops a
	// second one from the same team).
	AiMaxOpenOffersPerHumanPlayer = 3
	// AiMaxOpenOffersPerTeam is how many offers one CPU team may have open at a time.
	AiMaxOpenOffersPerTeam = 4
	// AiOfferCooldown is how long a CPU team leaves a player alone after he rejected (or ignored)
	// its offer; when it comes back, it has to raise the bid - see AiRebidMultiple.
	AiOfferCooldown = 72 * time.Hour
	// AiOfferLifetime is how long a CPU offer stays open before its team withdraws it.
	AiOfferLifetime = 72 * time.Hour
	// AiRebidMultiple is the least a CPU team raises a bid by, relative to the one that was
	// rejected, when the cooldown is over.
	AiRebidMultiple = 1.1
	// offerRejectionRetention is how long a rejection is remembered at all.
	offerRejectionRetention = 30 * 24 * time.Hour
)

// OfferRejection is the trace of an offer that did not go through.
type OfferRejection struct {
	PlayerId   int64
	FromTeamId int64
	Amount     int64
	RejectedAt time.Time
}

const insertOfferRejectionStmtSql = "INSERT INTO player_offer_rejections (player_id, from_team_id, amount) VALUES ($1, $2, $3)"

// RecordOfferRejection logs an offer of amount from fromTeamId for playerId that was rejected or
// withdrawn without being accepted.
func RecordOfferRejection(ex sqlExecutor, playerId int64, fromTeamId int64, amount int64) error {
	_, err := ex.Exec(insertOfferRejectionStmtSql, playerId, fromTeamId, amount)
	return err
}

// getRecentOfferRejectionsStmtSql keeps only the latest rejection of each player for the team.
const getRecentOfferRejectionsStmtSql = "SELECT DISTINCT ON (player_id) player_id, from_team_id, amount, rejected_at " +
	"FROM player_offer_rejections WHERE from_team_id = $1 AND rejected_at > $2 ORDER BY player_id, rejected_at DESC"

// GetRecentOfferRejections returns, keyed by player id, the most recent rejection of each player
// that turned down one of fromTeamId's offers since the given time.
func GetRecentOfferRejections(dbCtx *DatabaseContext, fromTeamId int64, since time.Time) (map[int64]OfferRejection, error) {
	rows, err := dbCtx.Db.Query(getRecentOfferRejectionsStmtSql, fromTeamId, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rejections := make(map[int64]OfferRejection)
	for rows.Next() {
		var r OfferRejection
		if err := rows.Scan(&r.PlayerId, &r.FromTeamId, &r.Amount, &r.RejectedAt); err != nil {
			return nil, err
		}
		rejections[r.PlayerId] = r
	}
	return rejections, rows.Err()
}

// OfferRetryAllowed decides whether a CPU team may bid again for a player who rejected it: not
// during the cooldown, and afterwards only with a bid at least AiRebidMultiple above the rejected
// one. minBid is that least acceptable bid (0 when there is nothing to beat).
func OfferRetryAllowed(last *OfferRejection, now time.Time) (allowed bool, minBid int64) {
	if last == nil {
		return true, 0
	}
	if now.Sub(last.RejectedAt) < AiOfferCooldown {
		return false, 0
	}
	return true, int64(float64(last.Amount) * AiRebidMultiple)
}

const countOpenAiOffersOnPlayerStmtSql = "SELECT count(*) FROM transfer_offers o JOIN teams t ON t.team_id = o.from_team_id " +
	"WHERE o.player_id = $1 AND t.owner_id IS NULL"

// CountOpenAiOffersOnPlayer is how many offers from CPU-managed (unowned) teams are open on a
// player right now.
func CountOpenAiOffersOnPlayer(dbCtx *DatabaseContext, playerId int64) (int, error) {
	var count int
	err := dbCtx.Db.QueryRow(countOpenAiOffersOnPlayerStmtSql, playerId).Scan(&count)
	return count, err
}

const getTeamSentOfferPlayerIdsStmtSql = "SELECT player_id FROM transfer_offers WHERE from_team_id = $1"

// GetPlayersWithOfferFromTeam is the set of players fromTeamId currently has an open offer on.
func GetPlayersWithOfferFromTeam(dbCtx *DatabaseContext, fromTeamId int64) (map[int64]bool, error) {
	return queryPlayerIdSet(dbCtx, getTeamSentOfferPlayerIdsStmtSql, fromTeamId)
}

// expireStaleOffersStmtSql withdraws a team's offers older than a cutoff and remembers each as a
// rejection in the same statement, so the cooldown applies to an offer nobody answered too.
const expireStaleOffersStmtSql = "WITH gone AS (DELETE FROM transfer_offers WHERE from_team_id = $1 AND created_at < $2 " +
	"RETURNING player_id, from_team_id, amount) " +
	"INSERT INTO player_offer_rejections (player_id, from_team_id, amount) SELECT player_id, from_team_id, amount FROM gone"

const pruneOfferRejectionsStmtSql = "DELETE FROM player_offer_rejections WHERE rejected_at < $1"

// ExpireStaleOffers withdraws every offer fromTeamId made before olderThan and returns how many
// there were - a CPU team doesn't leave offers hanging forever on a human who never answers. Old
// rejection records are pruned along the way.
func ExpireStaleOffers(dbCtx *DatabaseContext, fromTeamId int64, olderThan time.Time) (int, error) {
	result, err := dbCtx.Db.Exec(expireStaleOffersStmtSql, fromTeamId, olderThan)
	if err != nil {
		return 0, err
	}
	withdrawn, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if withdrawn > 0 {
		TransferOffersResolvedCounter.Add(context.Background(), withdrawn, metric.WithAttributes(OfferOutcomeAttribute("withdrawn")))
	}
	if _, err := dbCtx.Db.Exec(pruneOfferRejectionsStmtSql, time.Now().Add(-offerRejectionRetention)); err != nil {
		return int(withdrawn), err
	}
	return int(withdrawn), nil
}
