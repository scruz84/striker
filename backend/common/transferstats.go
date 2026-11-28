package common

import (
	"database/sql"
	"errors"
	"fmt"
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

// TopTransferFee is the largest fee any completed transfer (a bought listing, an accepted offer,
// an exercised buy option or a free agent premium - see topTransferFeeKinds) has ever paid for
// one player.
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

// getOffersInWindowsStmtSql only looks at the last day (the widest window it reports), so the
// transfer_offer_log_created_idx index bounds the work regardless of how long the log has grown.
const getOffersInWindowsStmtSql = "SELECT " +
	"count(*) FILTER (WHERE created_at > now() - interval '1 hour'), " +
	"count(*) FILTER (WHERE created_at > now() - interval '1 day') " +
	"FROM transfer_offer_log WHERE created_at > now() - interval '1 day'"

// getMostViewedListedPlayerStmtSql picks the currently-listed player with the highest
// view_count, ties broken by whoever's been listed longest (listed_at) so the answer is stable
// rather than arbitrary. A player never viewed (view_count = 0) is excluded - "most viewed"
// shouldn't report someone nobody has actually looked at.
const getMostViewedListedPlayerStmtSql = "SELECT p.player_id, p.view_count FROM transfer_listings tl " +
	"JOIN players p ON p.player_id = tl.player_id " +
	"WHERE p.view_count > 0 ORDER BY p.view_count DESC, tl.listed_at ASC LIMIT 1"

// topTransferFeeKinds are the player_transfers.kind values that count towards the biggest fee on
// record: every flow that pays a club for a player (a bought listing, an accepted offer, an
// exercised buy option, a free agent premium and the pre-player_transfers permanent moves).
// Loans (3) are a rental fee and releases (5) have no buyer, so neither counts. The list is
// spelled out in the query as literals so it matches the predicate of the partial index
// (player_transfers_fee_idx, init81_market_indexes.sql) - keep the two in sync.
var topTransferFeeKinds = fmt.Sprintf("%d, %d, %d, %d, %d", PLAYER_TRANSFER_LISTING, PLAYER_TRANSFER_OFFER,
	PLAYER_TRANSFER_FREE_AGENT, PLAYER_TRANSFER_BUY_OPTION, PLAYER_TRANSFER_PERMANENT)

// getTopTransferFeeStmtSql reads the biggest fee straight from player_transfers (see
// RecordPlayerTransfer), instead of re-pairing the two team_transactions rows of each transfer.
// The panel shows a player still in the game, so a retired player's transfer (kept since
// init122_retired_players.sql) gives way to the next biggest, as it did when it was deleted.
var getTopTransferFeeStmtSql = "SELECT pt.player_id, pt.fee, pt.to_team_id, bt.name, pt.from_team_id, st.name, pt.created_at " +
	"FROM player_transfers pt " +
	"JOIN teams bt ON bt.team_id = pt.to_team_id " +
	"JOIN teams st ON st.team_id = pt.from_team_id " +
	"WHERE pt.kind IN (" + topTransferFeeKinds + ") AND pt.fee > 0 " +
	"AND EXISTS (SELECT 1 FROM players p WHERE p.player_id = pt.player_id) " +
	"ORDER BY pt.fee DESC LIMIT 1"

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
	err = dbCtx.Db.QueryRow(getTopTransferFeeStmtSql).Scan(
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
