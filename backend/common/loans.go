package common

import (
	"database/sql"
	"errors"
	"time"
)

var ErrPlayerAlreadyLoanListed = errors.New("player is already listed for loan")
var ErrLoanListingNotFound = errors.New("loan listing not found")
var ErrCannotAcceptOwnLoan = errors.New("cannot accept a loan for your own player")
var ErrPlayerOnLoan = errors.New("player is currently out on loan")
var ErrPlayerNotLoanListedByTeam = errors.New("player is not listed for loan by your team")
var ErrInvalidLoanDuration = errors.New("loan duration out of range")

// MinLoanDurationSeasons/MaxLoanDurationSeasons bound how long a lender can offer a loan for.
const MinLoanDurationSeasons = 1
const MaxLoanDurationSeasons = 3

// LoanListing is an open offer to loan a player out - see GetActiveLoanListings.
type LoanListing struct {
	ListingId       int64     `json:"listingId"`
	Player          Player    `json:"player"`
	LendingTeamId   int64     `json:"lendingTeamId"`
	LendingTeamName string    `json:"lendingTeamName"`
	DurationSeasons int       `json:"durationSeasons"`
	ListedAt        time.Time `json:"listedAt"`
}

// PlayerLoan is an active loan - the player currently belongs (team_id) to ToTeamId, and is due
// back at FromTeamId once the current season reaches ReturnSeason - see
// engine.ReturnExpiredLoans.
type PlayerLoan struct {
	LoanId       int64 `json:"loanId"`
	PlayerId     int64 `json:"playerId"`
	FromTeamId   int64 `json:"fromTeamId"`
	ToTeamId     int64 `json:"toTeamId"`
	ReturnSeason int   `json:"returnSeason"`
}

const InsertLoanListingStmtSql = "INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons) VALUES ($1, $2, $3)"
const DeleteLoanListingByPlayerAndTeamStmtSql = "DELETE FROM loan_listings WHERE player_id = $1 AND lending_team_id = $2"
const GetLoanListingForUpdateStmtSql = "SELECT player_id, lending_team_id, duration_seasons FROM loan_listings WHERE listing_id = $1 " + ForUpdate
const DeleteLoanListingStmtSql = "DELETE FROM loan_listings WHERE listing_id = $1"
const GetActiveLoanListingsStmtSql = "SELECT ll.listing_id, ll.player_id, ll.lending_team_id, t.name, ll.duration_seasons, ll.listed_at " +
	"FROM loan_listings ll JOIN teams t ON t.team_id = ll.lending_team_id ORDER BY ll.listed_at DESC"
const IsPlayerLoanListedStmtSql = "SELECT 1 FROM loan_listings WHERE player_id = $1"
const IsPlayerOnActiveLoanStmtSql = "SELECT loan_id FROM player_loans WHERE player_id = $1"

const InsertPlayerLoanStmtSql = "INSERT INTO player_loans (player_id, from_team_id, to_team_id, return_season) VALUES ($1, $2, $3, $4)"
const DeletePlayerLoanStmtSql = "DELETE FROM player_loans WHERE loan_id = $1"
const GetExpiredLoansStmtSql = "SELECT loan_id, player_id, from_team_id, to_team_id, return_season FROM player_loans WHERE return_season <= $1"

// IsPlayerOnActiveLoan reports whether playerId is currently out on loan - used to stop a
// borrowing team from transfer-listing or re-loaning a player that isn't really theirs to sell.
func IsPlayerOnActiveLoan(dbCtx *DatabaseContext, playerId int64) (bool, error) {
	var loanId int64
	err := dbCtx.Db.QueryRow(IsPlayerOnActiveLoanStmtSql, playerId).Scan(&loanId)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// IsPlayerLoanListed reports whether playerId already has an open loan listing.
func IsPlayerLoanListed(dbCtx *DatabaseContext, playerId int64) (bool, error) {
	var exists int
	err := dbCtx.Db.QueryRow(IsPlayerLoanListedStmtSql, playerId).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// CreateLoanListing offers a player out on loan for durationSeasons. Rejects a player that's
// already out on loan (ErrPlayerOnLoan - a borrowing team can't re-loan what isn't theirs) or
// already transfer-listed (ErrPlayerAlreadyListed - the two markets are mutually exclusive).
func CreateLoanListing(dbCtx *DatabaseContext, playerId int64, lendingTeamId int64, durationSeasons int) error {
	if durationSeasons < MinLoanDurationSeasons || durationSeasons > MaxLoanDurationSeasons {
		return ErrInvalidLoanDuration
	}
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
	transferListed, err := IsPlayerTransferListed(dbCtx, playerId)
	if err != nil {
		return err
	}
	if transferListed {
		return ErrPlayerAlreadyListed
	}

	_, err = dbCtx.Db.Exec(InsertLoanListingStmtSql, playerId, lendingTeamId, durationSeasons)
	if err != nil && isUniqueViolation(err) {
		return ErrPlayerAlreadyLoanListed
	}
	return err
}

// CancelLoanListing removes a player's loan listing, but only if it belongs to lendingTeamId.
func CancelLoanListing(dbCtx *DatabaseContext, playerId int64, lendingTeamId int64) error {
	result, err := dbCtx.Db.Exec(DeleteLoanListingByPlayerAndTeamStmtSql, playerId, lendingTeamId)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPlayerNotLoanListedByTeam
	}
	return nil
}

// GetActiveLoanListings returns every open loan offer with full player details, newest first.
func GetActiveLoanListings(dbCtx *DatabaseContext) ([]LoanListing, error) {
	rows, err := dbCtx.Db.Query(GetActiveLoanListingsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type row struct {
		listing  LoanListing
		playerId int64
	}
	pending := make([]row, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.listing.ListingId, &r.playerId, &r.listing.LendingTeamId,
			&r.listing.LendingTeamName, &r.listing.DurationSeasons, &r.listing.ListedAt); err != nil {
			return nil, err
		}
		pending = append(pending, r)
	}
	rows.Close()

	listings := make([]LoanListing, 0, len(pending))
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

// AcceptLoan brings a listed player onto borrowingTeamId's squad for the listing's duration:
// moves the player, clears them from the lender's default lineup (mirrors ExecuteTransfer -
// they're no longer at that team, wherever they were slotted), records the loan (who it's
// owed back to and when), and drops the listing - all atomically. The listing row is locked
// (SELECT ... FOR UPDATE) so two simultaneous accepts of the same listing can't both succeed.
func AcceptLoan(dbCtx *DatabaseContext, listingId int64, borrowingTeamId int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}
	if !IsTeamSolvent(dbCtx, borrowingTeamId) {
		return ErrTeamInsolvent
	}
	currentSeason := GetCurrentSeason(dbCtx)

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var playerId, lendingTeamId int64
	var durationSeasons int
	err = tx.QueryRow(GetLoanListingForUpdateStmtSql, listingId).Scan(&playerId, &lendingTeamId, &durationSeasons)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLoanListingNotFound
	}
	if err != nil {
		return err
	}
	if lendingTeamId == borrowingTeamId {
		return ErrCannotAcceptOwnLoan
	}

	if err := movePlayerToTeam(tx, borrowingTeamId, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteLoanListingStmtSql, listingId); err != nil {
		return err
	}
	returnSeason := currentSeason + durationSeasons
	if _, err := tx.Exec(InsertPlayerLoanStmtSql, playerId, lendingTeamId, borrowingTeamId, returnSeason); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	notifyLoanAccepted(dbCtx, playerId, lendingTeamId, borrowingTeamId)
	return nil
}

func notifyLoanAccepted(dbCtx *DatabaseContext, playerId int64, lendingTeamId int64, borrowingTeamId int64) {
	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return
	}
	playerName := player.FirstName + " " + player.LastName
	lendingTeam, errLending := GetTeamById(lendingTeamId, dbCtx)
	borrowingTeam, errBorrowing := GetTeamById(borrowingTeamId, dbCtx)
	if errLending != nil || errBorrowing != nil {
		return
	}
	_ = NotifyTeamOwner(dbCtx, borrowingTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Loaned in " + playerName + " from " + lendingTeam.TeamName,
		Key:    "notifications.messages.loanedIn",
		Params: map[string]any{"player": playerName, "team": lendingTeam.TeamName},
	})
	_ = NotifyTeamOwner(dbCtx, lendingTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Loaned out " + playerName + " to " + borrowingTeam.TeamName,
		Key:    "notifications.messages.loanedOut",
		Params: map[string]any{"player": playerName, "team": borrowingTeam.TeamName},
	})
}

// GetExpiredLoans lists every active loan whose return_season has arrived (<= season).
func GetExpiredLoans(dbCtx *DatabaseContext, season int) ([]PlayerLoan, error) {
	rows, err := dbCtx.Db.Query(GetExpiredLoansStmtSql, season)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loans := make([]PlayerLoan, 0)
	for rows.Next() {
		var loan PlayerLoan
		if err := rows.Scan(&loan.LoanId, &loan.PlayerId, &loan.FromTeamId, &loan.ToTeamId, &loan.ReturnSeason); err != nil {
			return nil, err
		}
		loans = append(loans, loan)
	}
	return loans, nil
}

// ReturnLoan sends a loaned player back to its original owner: moves the player, clears them
// from the borrowing team's default lineup (same reasoning as AcceptLoan - they're leaving),
// and removes the loan record.
func ReturnLoan(dbCtx *DatabaseContext, loan PlayerLoan) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := movePlayerToTeam(tx, loan.FromTeamId, loan.PlayerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, loan.PlayerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLoanStmtSql, loan.LoanId); err != nil {
		return err
	}

	return tx.Commit()
}
