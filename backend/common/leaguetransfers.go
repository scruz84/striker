package common

import "time"

// LeagueTransfer is one row of a league's recent-transfers panel (league details page and the
// dashboard) - a completed transfer (a bought listing or an accepted offer) involving at least
// one team from the league, newest first. Same shape as TopTransferFee, which reports only the
// single biggest fee ever paid market-wide; this reports several, scoped to one league.
type LeagueTransfer struct {
	Player         Player    `json:"player"`
	Amount         int64     `json:"amount"`
	BuyerTeamId    int64     `json:"buyerTeamId"`
	BuyerTeamName  string    `json:"buyerTeamName"`
	SellerTeamId   int64     `json:"sellerTeamId"`
	SellerTeamName string    `json:"sellerTeamName"`
	OccurredAt     time.Time `json:"occurredAt"`
}

// getRecentLeagueTransfersStmtSql pairs up each completed transfer's two team_transactions rows
// (see ExecuteTransfer/AcceptOffer, which insert both atomically with the same created_at,
// player_id, and equal-and-opposite amounts) to reconstruct "who paid whom how much for which
// player", the same technique getTopTransferFeeStmtSql uses - filtered to transfers where the
// buyer or the seller (or both, for a transfer within the league) belongs to the given league.
const getRecentLeagueTransfersStmtSql = "SELECT o.player_id, -o.amount AS amount, o.team_id, bt.name, i.team_id, st.name, o.created_at " +
	"FROM team_transactions o " +
	"JOIN team_transactions i ON i.player_id = o.player_id AND i.created_at = o.created_at AND i.amount = -o.amount AND i.reason = $1 " +
	"JOIN teams bt ON bt.team_id = o.team_id " +
	"JOIN teams st ON st.team_id = i.team_id " +
	"WHERE o.reason = $2 AND o.player_id IS NOT NULL AND (bt.league_id = $3 OR st.league_id = $3) " +
	"ORDER BY o.created_at DESC LIMIT $4"

// GetRecentLeagueTransfers returns leagueId's most recent completed transfers, newest first,
// capped at limit - see LeagueTransfer.
func GetRecentLeagueTransfers(dbCtx *DatabaseContext, leagueId int64, limit int) ([]LeagueTransfer, error) {
	rows, err := dbCtx.Db.Query(getRecentLeagueTransfersStmtSql, TRANSACTION_TRANSFER_IN, TRANSACTION_TRANSFER_OUT, leagueId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transfers := make([]LeagueTransfer, 0)
	for rows.Next() {
		var t LeagueTransfer
		var playerId int64
		if err := rows.Scan(&playerId, &t.Amount, &t.BuyerTeamId, &t.BuyerTeamName, &t.SellerTeamId, &t.SellerTeamName, &t.OccurredAt); err != nil {
			return nil, err
		}
		player, err := GetPlayerById(dbCtx, playerId)
		if err != nil {
			continue
		}
		t.Player = *player
		transfers = append(transfers, t)
	}
	return transfers, nil
}
