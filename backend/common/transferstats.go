package common

import (
	"database/sql"
	"errors"
	"time"
)

// TransferMarketStats is the transfer market page's activity panel (issue #72): how busy the
// market currently is (listed players, recent offers), which listed player is drawing the most
// attention, and the biggest transfer fee ever paid. MostViewedPlayer/TopTransferFee are nil
// when there's nothing to report yet (no listed player has been viewed, or no transfer has ever
// completed).
type TransferMarketStats struct {
	ListedPlayersCount int               `json:"listedPlayersCount"`
	OffersLastHour     int               `json:"offersLastHour"`
	OffersLastDay      int               `json:"offersLastDay"`
	MostViewedPlayer   *MostViewedPlayer `json:"mostViewedPlayer,omitempty"`
	TopTransferFee     *TopTransferFee   `json:"topTransferFee,omitempty"`
}

// MostViewedPlayer is the currently-listed player whose profile (see
// handlers.PlayerDetailsHandler, which increments players.view_count on every fetch) has been
// viewed the most.
type MostViewedPlayer struct {
	Player    Player `json:"player"`
	ViewCount int64  `json:"viewCount"`
}

// TopTransferFee is the largest fee any completed transfer (a bought listing or an accepted
// offer - both log a matched pair of team_transactions rows, see ExecuteTransfer/AcceptOffer)
// has ever paid for one player.
type TopTransferFee struct {
	Player         Player    `json:"player"`
	Amount         int64     `json:"amount"`
	BuyerTeamId    int64     `json:"buyerTeamId"`
	BuyerTeamName  string    `json:"buyerTeamName"`
	SellerTeamId   int64     `json:"sellerTeamId"`
	SellerTeamName string    `json:"sellerTeamName"`
	OccurredAt     time.Time `json:"occurredAt"`
}

const getListedPlayersCountStmtSql = "SELECT count(*) FROM transfer_listings"

const getOffersInWindowsStmtSql = "SELECT " +
	"count(*) FILTER (WHERE created_at > now() - interval '1 hour'), " +
	"count(*) FILTER (WHERE created_at > now() - interval '1 day') " +
	"FROM transfer_offer_log"

// getMostViewedListedPlayerStmtSql picks the currently-listed player with the highest
// view_count, ties broken by whoever's been listed longest (listed_at) so the answer is stable
// rather than arbitrary. A player never viewed (view_count = 0) is excluded - "most viewed"
// shouldn't report someone nobody has actually looked at.
const getMostViewedListedPlayerStmtSql = "SELECT p.player_id, p.view_count FROM transfer_listings tl " +
	"JOIN players p ON p.player_id = tl.player_id " +
	"WHERE p.view_count > 0 ORDER BY p.view_count DESC, tl.listed_at ASC LIMIT 1"

// getTopTransferFeeStmtSql pairs up each completed transfer's two team_transactions rows (see
// ExecuteTransfer/AcceptOffer, which insert both atomically with the same created_at, player_id,
// and equal-and-opposite amounts) to reconstruct "who paid whom how much for which player",
// without needing a dedicated transfer-history table.
const getTopTransferFeeStmtSql = "SELECT o.player_id, -o.amount AS amount, o.team_id, bt.name, i.team_id, st.name, o.created_at " +
	"FROM team_transactions o " +
	"JOIN team_transactions i ON i.player_id = o.player_id AND i.created_at = o.created_at AND i.amount = -o.amount AND i.reason = $1 " +
	"JOIN teams bt ON bt.team_id = o.team_id " +
	"JOIN teams st ON st.team_id = i.team_id " +
	"WHERE o.reason = $2 AND o.player_id IS NOT NULL " +
	"ORDER BY amount DESC LIMIT 1"

// GetTransferMarketStats computes every figure on the transfer market's activity panel in one
// call - see TransferMarketStats.
func GetTransferMarketStats(dbCtx *DatabaseContext) (*TransferMarketStats, error) {
	stats := &TransferMarketStats{}

	if err := dbCtx.Db.QueryRow(getListedPlayersCountStmtSql).Scan(&stats.ListedPlayersCount); err != nil {
		return nil, err
	}

	if err := dbCtx.Db.QueryRow(getOffersInWindowsStmtSql).Scan(&stats.OffersLastHour, &stats.OffersLastDay); err != nil {
		return nil, err
	}

	var mostViewedPlayerId int64
	var viewCount int64
	err := dbCtx.Db.QueryRow(getMostViewedListedPlayerStmtSql).Scan(&mostViewedPlayerId, &viewCount)
	switch {
	case err == nil:
		if player, err := GetPlayerById(dbCtx, mostViewedPlayerId); err == nil {
			stats.MostViewedPlayer = &MostViewedPlayer{Player: *player, ViewCount: viewCount}
		}
	case !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}

	var topFee TopTransferFee
	var topFeePlayerId int64
	err = dbCtx.Db.QueryRow(getTopTransferFeeStmtSql, TRANSACTION_TRANSFER_IN, TRANSACTION_TRANSFER_OUT).Scan(
		&topFeePlayerId, &topFee.Amount, &topFee.BuyerTeamId, &topFee.BuyerTeamName,
		&topFee.SellerTeamId, &topFee.SellerTeamName, &topFee.OccurredAt)
	switch {
	case err == nil:
		if player, err := GetPlayerById(dbCtx, topFeePlayerId); err == nil {
			topFee.Player = *player
			stats.TopTransferFee = &topFee
		}
	case !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}

	return stats, nil
}

// IncrementPlayerViewCount records one more view of playerId's profile - see
// handlers.PlayerDetailsHandler and MostViewedPlayer.
func IncrementPlayerViewCount(dbCtx *DatabaseContext, playerId int64) error {
	_, err := dbCtx.Db.Exec("UPDATE players SET view_count = view_count + 1 WHERE player_id = $1", playerId)
	return err
}

// LogTransferOfferMade records that an offer was made, for the transfer market stats panel's
// "offers in the last hour/day" figures (see GetTransferMarketStats) - a permanent log,
// deliberately separate from transfer_offers itself, which is deleted the moment an offer
// resolves (see MakeTransferOffer).
func LogTransferOfferMade(dbCtx *DatabaseContext) error {
	_, err := dbCtx.Db.Exec("INSERT INTO transfer_offer_log DEFAULT VALUES")
	return err
}
