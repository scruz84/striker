package common

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// team_id is coalesced to 0 - GetPlayerById is also used to look up a free agent (e.g. from
// SignFreeAgent), whose team_id is NULL, and Player.TeamId is a plain non-nullable int64.
// last_team_id is coalesced the same way, for the same reason (see Player.LastTeamId).
const GetPlayerByIdStmtSql = "SELECT player_id, COALESCE(team_id, 0), first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining, birth_country, " +
	"free_agent_seasons, COALESCE(last_team_id, 0) FROM players WHERE player_id = $1"

const InsertTransferListingStmtSql = "INSERT INTO transfer_listings (player_id, selling_team_id, asking_price) VALUES ($1, $2, $3)"

const GetActiveTransferListingsStmtSql = "SELECT tl.listing_id, tl.player_id, tl.selling_team_id, t.name, tl.asking_price, tl.listed_at " +
	"FROM transfer_listings tl JOIN teams t ON t.team_id = tl.selling_team_id ORDER BY tl.listed_at DESC"

const DeleteTransferListingByPlayerAndTeamStmtSql = "DELETE FROM transfer_listings WHERE player_id = $1 AND selling_team_id = $2"

const GetTransferListingForUpdateStmtSql = "SELECT player_id, selling_team_id, asking_price FROM transfer_listings WHERE listing_id = $1 " + ForUpdate

const DeleteTransferListingStmtSql = "DELETE FROM transfer_listings WHERE listing_id = $1"

const DeletePlayerLineupPositionsStmtSql = "DELETE FROM default_lineup_positions WHERE player_id = $1"

const UpdatePlayerTeamStmtSql = "UPDATE players SET team_id = $1 WHERE player_id = $2"

var ErrListingNotFound = errors.New("listing not found")
var ErrCannotBuyOwnPlayer = errors.New("cannot buy your own listed player")
var ErrInsufficientBudget = errors.New("insufficient budget")
var ErrPlayerAlreadyListed = errors.New("player is already listed for transfer")
var ErrPlayerNotListedByTeam = errors.New("player is not listed for transfer by your team")

// ErrTransferWindowClosed is returned by anything that moves a player between teams (listing,
// buying, loan-listing, accepting a loan) while common.IsTransferWindowOpen is false - see
// handlers.SetTransferWindowHandler for the admin toggle. Cancelling an existing listing is
// never blocked by it - getting your own player off the market isn't transfer activity.
var ErrTransferWindowClosed = errors.New("transfer window is closed")

// GetPlayerById fetches a single player by id, regardless of which team they belong to.
func GetPlayerById(dbCtx *DatabaseContext, playerId int64) (*Player, error) {
	var p Player
	var birthDay int64
	var avatar string
	var birthCountry sql.NullString
	err := dbCtx.Db.QueryRow(GetPlayerByIdStmtSql, playerId).Scan(&p.Id, &p.TeamId, &p.FirstName, &p.LastName, &birthDay,
		&p.GoalKeeper, &p.Defense, &p.Lateral, &p.Pass, &p.Midfield, &p.Dribbling, &p.Shot,
		&p.Header, &p.Quick, &p.Creative, &p.Technical, &p.Wage, &p.Age, &avatar, &p.ContractYearsRemaining, &birthCountry,
		&p.FreeAgentSeasons, &p.LastTeamId)
	if err != nil {
		return nil, err
	}
	p.BirthDay = time.UnixMilli(birthDay)
	p.Avatar = GetAvatarFromString(avatar)
	p.BirthCountry = birthCountry.String
	return &p, nil
}

const IsPlayerTransferListedStmtSql = "SELECT 1 FROM transfer_listings WHERE player_id = $1"

// IsPlayerTransferListed reports whether playerId already has an open transfer listing - used
// by the loan market (see common.CreateLoanListing) to keep the two markets mutually exclusive.
func IsPlayerTransferListed(dbCtx *DatabaseContext, playerId int64) (bool, error) {
	var exists int
	err := dbCtx.Db.QueryRow(IsPlayerTransferListedStmtSql, playerId).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// CreateTransferListing lists a player for transfer at a fixed asking price - there's no
// bidding/offer negotiation, any other team can buy it outright. Returns
// ErrPlayerAlreadyListed if the player already has an active listing (enforced by a unique
// constraint on transfer_listings.player_id) or is already loan-listed, and ErrPlayerOnLoan if
// the player is currently out on loan - a borrowing team doesn't get to sell what isn't theirs.
func CreateTransferListing(dbCtx *DatabaseContext, playerId int64, sellingTeamId int64, askingPrice int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return err
	}
	if onLoan {
		return ErrPlayerOnLoan
	}
	loanListed, err := IsPlayerLoanListed(dbCtx, playerId)
	if err != nil {
		return err
	}
	if loanListed {
		return ErrPlayerAlreadyLoanListed
	}

	_, err = dbCtx.Db.Exec(InsertTransferListingStmtSql, playerId, sellingTeamId, askingPrice)
	if err != nil && isUniqueViolation(err) {
		return ErrPlayerAlreadyListed
	}
	return err
}

// CancelTransferListing removes a player's listing, but only if it belongs to sellingTeamId.
func CancelTransferListing(dbCtx *DatabaseContext, playerId int64, sellingTeamId int64) error {
	result, err := dbCtx.Db.Exec(DeleteTransferListingByPlayerAndTeamStmtSql, playerId, sellingTeamId)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPlayerNotListedByTeam
	}
	return nil
}

// GetActiveTransferListings returns every open listing with full player details, newest
// first.
func GetActiveTransferListings(dbCtx *DatabaseContext) ([]TransferListing, error) {
	rows, err := dbCtx.Db.Query(GetActiveTransferListingsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type row struct {
		listing  TransferListing
		playerId int64
	}
	pending := make([]row, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.listing.ListingId, &r.playerId, &r.listing.SellingTeamId,
			&r.listing.SellingTeamName, &r.listing.AskingPrice, &r.listing.ListedAt); err != nil {
			return nil, err
		}
		pending = append(pending, r)
	}
	rows.Close()

	listings := make([]TransferListing, 0, len(pending))
	for _, r := range pending {
		player, err := GetPlayerById(dbCtx, r.playerId)
		if err != nil {
			continue
		}
		r.listing.Player = *player
		listings = append(listings, r.listing)
	}
	return listings, nil
}

// ExecuteTransfer buys a listed player at its asking price: moves the player to the buyer,
// debits/credits both teams' budgets, logs both sides of the ledger, drops the listing, and
// clears the player from the seller's default lineup (they no longer belong to that team) -
// all atomically in one DB transaction. The listing row and the buyer's budget row are locked
// (SELECT ... FOR UPDATE) for the duration, so two simultaneous buys of the same listing, or a
// buy racing a budget-changing operation, can't both succeed.
func ExecuteTransfer(dbCtx *DatabaseContext, listingId int64, buyerTeamId int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var playerId, sellingTeamId, askingPrice int64
	err = tx.QueryRow(GetTransferListingForUpdateStmtSql, listingId).Scan(&playerId, &sellingTeamId, &askingPrice)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrListingNotFound
	}
	if err != nil {
		return err
	}
	if sellingTeamId == buyerTeamId {
		return ErrCannotBuyOwnPlayer
	}

	var buyerBudget int64
	if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, buyerTeamId).Scan(&buyerBudget); err != nil {
		return err
	}
	if buyerBudget < askingPrice {
		return ErrInsufficientBudget
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return err
	}
	buyerTeam, err := GetTeamById(buyerTeamId, dbCtx)
	if err != nil {
		return err
	}
	sellerTeam, err := GetTeamById(sellingTeamId, dbCtx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -askingPrice, buyerTeamId); err != nil {
		return err
	}
	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, askingPrice, sellingTeamId); err != nil {
		return err
	}
	if _, err := tx.Exec(UpdatePlayerTeamStmtSql, buyerTeamId, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteTransferListingStmtSql, listingId); err != nil {
		return err
	}

	playerName := player.FirstName + " " + player.LastName
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, buyerTeamId, -askingPrice, TRANSACTION_TRANSFER_OUT,
		"Bought "+playerName+" from "+sellerTeam.TeamName, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, sellingTeamId, askingPrice, TRANSACTION_TRANSFER_IN,
		"Sold "+playerName+" to "+buyerTeam.TeamName, playerId); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	TransfersCompletedCounter.Add(context.Background(), 1)
	TransferVolumeCounter.Add(context.Background(), askingPrice)

	// Best-effort, after the money has actually moved - a notification failure must never
	// undo a completed transfer.
	_ = NotifyTeamOwner(dbCtx, sellingTeamId, NOTIFICATION_TRANSFER, "Sold "+playerName+" to "+buyerTeam.TeamName)
	_ = NotifyTeamOwner(dbCtx, buyerTeamId, NOTIFICATION_TRANSFER, "Bought "+playerName+" from "+sellerTeam.TeamName)

	return nil
}

func isUniqueViolation(err error) bool {
	// pgx/lib/pq both surface this in the error text; avoids an extra driver-specific import
	// just to type-assert a SQLSTATE code.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate key")
}
