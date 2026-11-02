package common

import (
	"fmt"
	"time"
)

// LeagueTransfer is one row of a league's recent-transfers panel (league details page and the
// dashboard) - a completed move (bought listing, accepted offer, buy option, loan, free-agent
// signing that paid a premium) involving at least one team from the league, newest first. Same
// shape as TopTransferFee, which reports only the single biggest fee ever paid market-wide; this
// reports several, scoped to one league.
type LeagueTransfer struct {
	Player Player `json:"player"`
	// Amount is the fee paid (>= 0) - for a loan, the loan fee, zero when it was free.
	Amount int64 `json:"amount"`
	// Kind is one of the PLAYER_TRANSFER_* values - the panel marks PLAYER_TRANSFER_LOAN rows.
	Kind           int       `json:"kind"`
	BuyerTeamId    int64     `json:"buyerTeamId"`
	BuyerTeamName  string    `json:"buyerTeamName"`
	SellerTeamId   int64     `json:"sellerTeamId"`
	SellerTeamName string    `json:"sellerTeamName"`
	OccurredAt     time.Time `json:"occurredAt"`
}

// getRecentLeagueTransfersStmtSql reads the structured player_transfers log (see
// RecordPlayerTransfer), the same source as a player's transfer history. The inner joins on both
// teams drop releases (no destination) and free-agent signings that paid no premium (no origin
// club), neither of which is a move between clubs. A free-agent row only appears when a premium
// went to the previous club. $1 is the league, $2 the row limit.
var getRecentLeagueTransfersStmtSql = fmt.Sprintf("SELECT pt.player_id, pt.fee, pt.kind, pt.to_team_id, bt.name, pt.from_team_id, st.name, pt.created_at "+
	"FROM player_transfers pt "+
	"JOIN teams bt ON bt.team_id = pt.to_team_id "+
	"JOIN teams st ON st.team_id = pt.from_team_id "+
	"WHERE pt.kind IN (%d, %d, %d, %d, %d, %d) AND (bt.league_id = $1 OR st.league_id = $1) "+
	"ORDER BY pt.created_at DESC, pt.transfer_id DESC LIMIT $2",
	PLAYER_TRANSFER_LISTING, PLAYER_TRANSFER_OFFER, PLAYER_TRANSFER_FREE_AGENT, PLAYER_TRANSFER_LOAN,
	PLAYER_TRANSFER_BUY_OPTION, PLAYER_TRANSFER_PERMANENT)

// GetRecentLeagueTransfers returns leagueId's most recent completed moves, newest first, capped
// at limit - see LeagueTransfer.
func GetRecentLeagueTransfers(dbCtx *DatabaseContext, leagueId int64, limit int) ([]LeagueTransfer, error) {
	rows, err := dbCtx.Db.Query(getRecentLeagueTransfersStmtSql, leagueId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transfers := make([]LeagueTransfer, 0)
	for rows.Next() {
		var t LeagueTransfer
		var playerId int64
		if err := rows.Scan(&playerId, &t.Amount, &t.Kind, &t.BuyerTeamId, &t.BuyerTeamName, &t.SellerTeamId, &t.SellerTeamName, &t.OccurredAt); err != nil {
			return nil, err
		}
		player, err := GetPlayerById(dbCtx, playerId)
		if err != nil {
			continue
		}
		t.Player = *player
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}
