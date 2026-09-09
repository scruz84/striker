package common

import (
	"database/sql"
	"errors"
	"time"
)

var ErrInvalidOfferAmount = errors.New("offer amount must be positive")
var ErrCannotOfferOnOwnPlayer = errors.New("cannot make an offer on your own player")
var ErrOfferAlreadyExists = errors.New("you already have a pending offer on this player")
var ErrOfferNotFound = errors.New("offer not found")
var ErrNotOfferSender = errors.New("you did not make this offer")
var ErrNotOfferRecipient = errors.New("this offer is not for one of your players")
var ErrOfferNoLongerValid = errors.New("the player no longer belongs to the team this offer was made to")
var ErrPlayerIsFreeAgent = errors.New("player is a free agent - sign them instead of making an offer")

// TransferOffer is a pending "scouting" offer - a buyer's proposed price for a specific player,
// made directly to whoever currently owns them, independent of whether that player is listed
// on the open market at all. See MakeTransferOffer/AcceptOffer/RejectOffer/WithdrawOffer.
type TransferOffer struct {
	OfferId      int64     `json:"offerId"`
	Player       Player    `json:"player"`
	FromTeamId   int64     `json:"fromTeamId"`
	FromTeamName string    `json:"fromTeamName"`
	ToTeamId     int64     `json:"toTeamId"`
	ToTeamName   string    `json:"toTeamName"`
	Amount       int64     `json:"amount"`
	CreatedAt    time.Time `json:"createdAt"`
}

const InsertTransferOfferStmtSql = "INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount) VALUES ($1, $2, $3, $4)"

const GetOffersReceivedStmtSql = "SELECT o.offer_id, o.player_id, o.from_team_id, tf.name, o.to_team_id, tt.name, o.amount, o.created_at " +
	"FROM transfer_offers o JOIN teams tf ON tf.team_id = o.from_team_id JOIN teams tt ON tt.team_id = o.to_team_id " +
	"WHERE o.to_team_id = $1 ORDER BY o.created_at DESC"

const GetOffersSentStmtSql = "SELECT o.offer_id, o.player_id, o.from_team_id, tf.name, o.to_team_id, tt.name, o.amount, o.created_at " +
	"FROM transfer_offers o JOIN teams tf ON tf.team_id = o.from_team_id JOIN teams tt ON tt.team_id = o.to_team_id " +
	"WHERE o.from_team_id = $1 ORDER BY o.created_at DESC"

const GetOfferOwnersStmtSql = "SELECT from_team_id, to_team_id FROM transfer_offers WHERE offer_id = $1"
const GetOfferForUpdateStmtSql = "SELECT player_id, from_team_id, to_team_id, amount FROM transfer_offers WHERE offer_id = $1 " + ForUpdate
const DeleteTransferOfferStmtSql = "DELETE FROM transfer_offers WHERE offer_id = $1"
const DeleteOffersForPlayerStmtSql = "DELETE FROM transfer_offers WHERE player_id = $1"

// team_id is coalesced to 0 - a player can go from owned to a free agent (team_id NULL) while an
// offer on them sits pending (contract expiry, an early release, or AssignTeam reassigning their
// squad), and toTeamId is always a real (non-zero) team id, so the mismatch check below still
// correctly falls through to ErrOfferNoLongerValid instead of crashing on the NULL.
const GetPlayerTeamForUpdateStmtSql = "SELECT COALESCE(team_id, 0) FROM players WHERE player_id = $1 " + ForUpdate

// MakeTransferOffer proposes amount for playerId, wherever they currently play - the offer goes
// to whoever owns them right now (their team_id at the time the offer is made). Rejects an
// offer on your own player, on a free agent (see SignFreeAgent - there's nobody to pay, and no
// team would ever see the offer to accept it), on a player currently out on loan (a borrowing
// team doesn't get to sell what isn't theirs - same rule as CreateTransferListing), a duplicate
// offer from the same team on the same player (enforced by a unique constraint), or anything
// while the transfer window is closed.
func MakeTransferOffer(dbCtx *DatabaseContext, playerId int64, fromTeamId int64, amount int64) error {
	if amount <= 0 {
		return ErrInvalidOfferAmount
	}
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return err
	}
	if player.TeamId == 0 {
		return ErrPlayerIsFreeAgent
	}
	if player.TeamId == fromTeamId {
		return ErrCannotOfferOnOwnPlayer
	}
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return err
	}
	if onLoan {
		return ErrPlayerOnLoan
	}

	_, err = dbCtx.Db.Exec(InsertTransferOfferStmtSql, playerId, fromTeamId, player.TeamId, amount)
	if err != nil && isUniqueViolation(err) {
		return ErrOfferAlreadyExists
	}
	return err
}

// scanTransferOffers reads every row from a GetOffersReceivedStmtSql/GetOffersSentStmtSql query
// and hydrates each with its full Player - same two-pass pattern (collect rows, then fetch
// players) as GetActiveTransferListings/GetActiveLoanListings.
func scanTransferOffers(dbCtx *DatabaseContext, rows *sql.Rows) ([]TransferOffer, error) {
	defer rows.Close()

	type row struct {
		offer    TransferOffer
		playerId int64
	}
	pending := make([]row, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.offer.OfferId, &r.playerId, &r.offer.FromTeamId, &r.offer.FromTeamName,
			&r.offer.ToTeamId, &r.offer.ToTeamName, &r.offer.Amount, &r.offer.CreatedAt); err != nil {
			return nil, err
		}
		pending = append(pending, r)
	}
	rows.Close()

	offers := make([]TransferOffer, 0, len(pending))
	for _, r := range pending {
		player, err := GetPlayerById(dbCtx, r.playerId)
		if err != nil {
			continue
		}
		r.offer.Player = *player
		offers = append(offers, r.offer)
	}
	return offers, nil
}

// GetOffersReceived lists every pending offer made to teamId (offers on its own players).
func GetOffersReceived(dbCtx *DatabaseContext, teamId int64) ([]TransferOffer, error) {
	rows, err := dbCtx.Db.Query(GetOffersReceivedStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	return scanTransferOffers(dbCtx, rows)
}

// GetOffersSent lists every pending offer teamId has made on other teams' players.
func GetOffersSent(dbCtx *DatabaseContext, teamId int64) ([]TransferOffer, error) {
	rows, err := dbCtx.Db.Query(GetOffersSentStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	return scanTransferOffers(dbCtx, rows)
}

// WithdrawOffer lets the team that made an offer cancel it - allowed regardless of the transfer
// window's state, same as cancelling a listing.
func WithdrawOffer(dbCtx *DatabaseContext, offerId int64, fromTeamId int64) error {
	var actualFromTeamId, actualToTeamId int64
	err := dbCtx.Db.QueryRow(GetOfferOwnersStmtSql, offerId).Scan(&actualFromTeamId, &actualToTeamId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOfferNotFound
	}
	if err != nil {
		return err
	}
	if actualFromTeamId != fromTeamId {
		return ErrNotOfferSender
	}
	_, err = dbCtx.Db.Exec(DeleteTransferOfferStmtSql, offerId)
	return err
}

// RejectOffer lets the team an offer was made to decline it - also allowed regardless of the
// transfer window's state.
func RejectOffer(dbCtx *DatabaseContext, offerId int64, toTeamId int64) error {
	var actualFromTeamId, actualToTeamId int64
	err := dbCtx.Db.QueryRow(GetOfferOwnersStmtSql, offerId).Scan(&actualFromTeamId, &actualToTeamId)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOfferNotFound
	}
	if err != nil {
		return err
	}
	if actualToTeamId != toTeamId {
		return ErrNotOfferRecipient
	}
	_, err = dbCtx.Db.Exec(DeleteTransferOfferStmtSql, offerId)
	return err
}

// AcceptOffer executes the transfer at the offer's amount: moves the player, debits/credits both
// budgets, logs both sides of the ledger, clears the seller's default lineup, and drops every
// other pending offer on the same player (they're all stale once it's sold) plus any open
// listing for it - all atomically. Re-checks that the player still belongs to toTeamId (they
// might have been sold or loaned out since the offer was made) and that the buyer can still
// afford it, since an offer can sit pending for a while before it's acted on.
func AcceptOffer(dbCtx *DatabaseContext, offerId int64, toTeamId int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var playerId, fromTeamId, actualToTeamId, amount int64
	err = tx.QueryRow(GetOfferForUpdateStmtSql, offerId).Scan(&playerId, &fromTeamId, &actualToTeamId, &amount)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOfferNotFound
	}
	if err != nil {
		return err
	}
	if actualToTeamId != toTeamId {
		return ErrNotOfferRecipient
	}

	var currentOwnerTeamId int64
	if err := tx.QueryRow(GetPlayerTeamForUpdateStmtSql, playerId).Scan(&currentOwnerTeamId); err != nil {
		return err
	}
	if currentOwnerTeamId != toTeamId {
		return ErrOfferNoLongerValid
	}

	var buyerBudget int64
	if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, fromTeamId).Scan(&buyerBudget); err != nil {
		return err
	}
	if buyerBudget < amount {
		return ErrInsufficientBudget
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return err
	}
	buyerTeam, err := GetTeamById(fromTeamId, dbCtx)
	if err != nil {
		return err
	}
	sellerTeam, err := GetTeamById(toTeamId, dbCtx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -amount, fromTeamId); err != nil {
		return err
	}
	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, amount, toTeamId); err != nil {
		return err
	}
	if _, err := tx.Exec(UpdatePlayerTeamStmtSql, fromTeamId, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteOffersForPlayerStmtSql, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteTransferListingByPlayerAndTeamStmtSql, playerId, toTeamId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteLoanListingByPlayerAndTeamStmtSql, playerId, toTeamId); err != nil {
		return err
	}

	playerName := player.FirstName + " " + player.LastName
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, fromTeamId, -amount, TRANSACTION_TRANSFER_OUT,
		"Bought "+playerName+" from "+sellerTeam.TeamName, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, toTeamId, amount, TRANSACTION_TRANSFER_IN,
		"Sold "+playerName+" to "+buyerTeam.TeamName, playerId); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	_ = NotifyTeamOwner(dbCtx, toTeamId, NOTIFICATION_TRANSFER, "Sold "+playerName+" to "+buyerTeam.TeamName)
	_ = NotifyTeamOwner(dbCtx, fromTeamId, NOTIFICATION_TRANSFER, "Bought "+playerName+" from "+sellerTeam.TeamName)

	return nil
}
