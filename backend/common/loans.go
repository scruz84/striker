package common

import (
	"context"
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
var ErrInvalidLoanFee = errors.New("loan fee must not be negative")
var ErrInvalidWageContributionPercent = errors.New("wage contribution percent must be between 0 and 100")
var ErrInvalidBuyOptionFee = errors.New("buy option fee must not be negative")
var ErrPlayerNotOnLoan = errors.New("player is not currently on loan")
var ErrPlayerNotOnLoanToTeam = errors.New("player is not out on loan to your team")
var ErrNoBuyOptionOnLoan = errors.New("this loan has no buy-option fee attached")

// MinLoanDurationSeasons/MaxLoanDurationSeasons bound how long a lender can offer a loan for.
const MinLoanDurationSeasons = 1
const MaxLoanDurationSeasons = 3

// MinWageContributionPercent/MaxWageContributionPercent bound how much of a loaned player's wage
// a lender can keep covering while they're out on loan - see CreateLoanListing.
const MinWageContributionPercent = 0
const MaxWageContributionPercent = 100

// LoanListing is an open offer to loan a player out - see GetActiveLoanListings. LoanFee,
// WageContributionPercent, and BuyOptionFee are all optional (zero = off) economic terms the
// lender can attach - see CreateLoanListing.
type LoanListing struct {
	ListingId       int64  `json:"listingId"`
	Player          Player `json:"player"`
	LendingTeamId   int64  `json:"lendingTeamId"`
	LendingTeamName string `json:"lendingTeamName"`
	DurationSeasons int    `json:"durationSeasons"`
	LoanFee         int64  `json:"loanFee"`
	// WageContributionPercent is how much of the player's wage the lender keeps covering for the
	// duration of the loan - see engine.deductWages.
	WageContributionPercent int `json:"wageContributionPercent"`
	// BuyOptionFee, when non-zero, is the fixed fee the borrower can pay at any time during the
	// loan to make the move permanent - see ExerciseLoanBuyOption. Zero means no buy option.
	BuyOptionFee int64     `json:"buyOptionFee"`
	ListedAt     time.Time `json:"listedAt"`
	// Caps is this listing's player's international caps count (see common.GetPlayerCapsBatch) -
	// lets the loan market's search offer a "minimum caps" filter without the frontend looking
	// each listed player up individually.
	Caps int `json:"caps"`
}

// PlayerLoan is an active loan - the player currently belongs (team_id) to ToTeamId, and is due
// back at FromTeamId once the current season reaches ReturnSeason - see
// engine.ReturnExpiredLoans. WageContributionPercent/BuyOptionFee are copied from the originating
// LoanListing at accept time (see AcceptLoan), since the listing row itself is deleted.
type PlayerLoan struct {
	LoanId                  int64 `json:"loanId"`
	PlayerId                int64 `json:"playerId"`
	FromTeamId              int64 `json:"fromTeamId"`
	ToTeamId                int64 `json:"toTeamId"`
	ReturnSeason            int   `json:"returnSeason"`
	WageContributionPercent int   `json:"wageContributionPercent"`
	BuyOptionFee            int64 `json:"buyOptionFee"`
	// ReleaseOnReturn is set when the lending club was reset after losing its owner (see
	// ResetReleasedTeam): the player becomes a free agent when the loan ends instead of going back.
	ReleaseOnReturn bool `json:"-"`
}

const InsertLoanListingStmtSql = "INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee) " +
	"VALUES ($1, $2, $3, $4, $5, $6)"
const DeleteLoanListingByPlayerAndTeamStmtSql = "DELETE FROM loan_listings WHERE player_id = $1 AND lending_team_id = $2"
const GetLoanListingForUpdateStmtSql = "SELECT player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee " +
	"FROM loan_listings WHERE listing_id = $1 " + ForUpdate
const DeleteLoanListingStmtSql = "DELETE FROM loan_listings WHERE listing_id = $1"

// GetActiveLoanListingsStmtSql returns each listing's player (playerColumnsSql) followed by the
// listing's own columns, so GetActiveLoanListings needs no per-listing player lookup.
const GetActiveLoanListingsStmtSql = "SELECT " + playerColumnsSql + ", ll.listing_id, ll.lending_team_id, t.name, ll.duration_seasons, " +
	"ll.loan_fee, ll.wage_contribution_percent, ll.buy_option_fee, ll.listed_at " +
	"FROM loan_listings ll JOIN teams t ON t.team_id = ll.lending_team_id JOIN players p ON p.player_id = ll.player_id " +
	"ORDER BY ll.listed_at DESC"
const IsPlayerLoanListedStmtSql = "SELECT 1 FROM loan_listings WHERE player_id = $1"
const IsPlayerOnActiveLoanStmtSql = "SELECT loan_id FROM player_loans WHERE player_id = $1"

const InsertPlayerLoanStmtSql = "INSERT INTO player_loans (player_id, from_team_id, to_team_id, return_season, wage_contribution_percent, buy_option_fee) " +
	"VALUES ($1, $2, $3, $4, $5, $6)"
const DeletePlayerLoanStmtSql = "DELETE FROM player_loans WHERE loan_id = $1"

// GetExpiredLoansStmtSql scopes by the BORROWING team's country (to_team_id) - a loan's
// return_season is denominated in the borrower's own season count (see AcceptLoan: "N of the
// borrower's own seasons"), so only that country's own rollover should ever return it (see
// engine.AdvanceSeasonsIfComplete / engine.ReturnExpiredLoans).
const GetExpiredLoansStmtSql = "SELECT pl.loan_id, pl.player_id, pl.from_team_id, pl.to_team_id, pl.return_season, pl.release_on_return " +
	"FROM player_loans pl JOIN teams t ON t.team_id = pl.to_team_id WHERE t.country = $1 AND pl.return_season <= $2"
const GetLoansToTeamStmtSql = "SELECT loan_id, player_id, from_team_id, to_team_id, return_season, wage_contribution_percent, buy_option_fee, release_on_return " +
	"FROM player_loans WHERE to_team_id = $1 ORDER BY loan_id"
const GetActiveLoanForPlayerForUpdateStmtSql = "SELECT loan_id, from_team_id, to_team_id, buy_option_fee FROM player_loans WHERE player_id = $1 " + ForUpdate

// GetTeamLoanWageContributionStmtSql sums, for every player teamId currently has out on loan with
// a wage_contribution_percent > 0, the slice of that player's wage teamId (the lender) keeps
// covering - see engine.deductWages, which charges this separately from the borrower's own
// (already-reduced) wage bill so the two sides add up to exactly the player's full wage.
const GetTeamLoanWageContributionStmtSql = "SELECT COALESCE(SUM(p.wage * pl.wage_contribution_percent / 100), 0) " +
	"FROM player_loans pl JOIN players p ON p.player_id = pl.player_id " +
	"WHERE pl.from_team_id = $1 AND pl.wage_contribution_percent > 0"

// GetTeamLoanWageContribution returns how much teamId owes this round covering part of the wage
// of every player it currently has out on loan under a wage-contribution split, scaled by its own
// division's wage factor (see DivisionWageFactor) like the rest of what it pays in wages.
func GetTeamLoanWageContribution(dbCtx *DatabaseContext, teamId int64) (int64, error) {
	var total int64
	if err := dbCtx.Db.QueryRow(GetTeamLoanWageContributionStmtSql, teamId).Scan(&total); err != nil {
		return 0, err
	}
	factor, err := GetTeamWageFactor(dbCtx, teamId)
	if err != nil {
		return 0, err
	}
	return ApplyWageFactor(total, factor), nil
}

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

// CreateLoanListing offers a player out on loan for durationSeasons, with three independent,
// all-optional economic terms (see openspec/specs/team-finances, "Loan money"): loanFee (a
// one-time fee the borrower pays when the loan is accepted - zero means free, the original
// behavior), wageContributionPercent (how much of the wage the lender keeps covering for the
// loan's duration - zero means the borrower pays it all, same as before), and buyOptionFee (a
// fixed fee the borrower can later pay to make the move permanent - zero means no buy option).
// Rejects a player that's already out on loan (ErrPlayerOnLoan - a borrowing team can't re-loan
// what isn't theirs) or already transfer-listed (ErrPlayerAlreadyListed - the two markets are
// mutually exclusive).
func CreateLoanListing(dbCtx *DatabaseContext, playerId int64, lendingTeamId int64, durationSeasons int,
	loanFee int64, wageContributionPercent int, buyOptionFee int64) error {
	if durationSeasons < MinLoanDurationSeasons || durationSeasons > MaxLoanDurationSeasons {
		return ErrInvalidLoanDuration
	}
	if loanFee < 0 {
		return ErrInvalidLoanFee
	}
	if wageContributionPercent < MinWageContributionPercent || wageContributionPercent > MaxWageContributionPercent {
		return ErrInvalidWageContributionPercent
	}
	if buyOptionFee < 0 {
		return ErrInvalidBuyOptionFee
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

	_, err = dbCtx.Db.Exec(InsertLoanListingStmtSql, playerId, lendingTeamId, durationSeasons, loanFee, wageContributionPercent, buyOptionFee)
	if err == nil {
		if lender, lerr := GetTeamById(lendingTeamId, dbCtx); lerr == nil {
			NotifyWatchers(dbCtx, playerId, WatchEventLoanListed, lender.TeamName, 0, lendingTeamId)
		}
	}
	if err != nil {
		if isUniqueViolation(err) {
			return ErrPlayerAlreadyLoanListed
		}
		return err
	}
	LoanListingsCounter.Add(context.Background(), 1)
	return nil
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

	listings := make([]LoanListing, 0)
	playerIds := make([]int64, 0)
	for rows.Next() {
		var l LoanListing
		player, err := scanPlayer(rows, &l.ListingId, &l.LendingTeamId, &l.LendingTeamName, &l.DurationSeasons,
			&l.LoanFee, &l.WageContributionPercent, &l.BuyOptionFee, &l.ListedAt)
		if err != nil {
			return nil, err
		}
		l.Player = *player
		listings = append(listings, l)
		playerIds = append(playerIds, player.Id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	caps, err := GetPlayerCapsBatch(dbCtx, playerIds)
	if err != nil {
		return nil, err
	}
	for i := range listings {
		listings[i].Caps = caps[listings[i].Player.Id]
	}
	return listings, nil
}

// AcceptLoan brings a listed player onto borrowingTeamId's squad for the listing's duration:
// moves the player, clears them from the lender's default lineup (mirrors ExecuteTransfer -
// they're no longer at that team, wherever they were slotted), charges the listing's loan fee (if
// any - like a mini transfer fee, paid once, upfront), records the loan (who it's owed back to
// and when, plus whatever wage-contribution/buy-option terms the listing carried), and drops the
// listing - all atomically. The listing row and (when a fee applies) the borrower's budget row
// are locked (SELECT ... FOR UPDATE) so two simultaneous accepts, or an accept racing a
// budget-changing operation, can't both succeed.
func AcceptLoan(dbCtx *DatabaseContext, listingId int64, borrowingTeamId int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}
	if !IsTeamSolvent(dbCtx, borrowingTeamId) {
		return ErrTeamInsolvent
	}
	// The loan's duration is "N of the borrower's own seasons" (see GetExpiredLoansStmtSql), so
	// returnSeason below is computed from the borrowing team's own country's current season.
	currentSeason, err := GetCurrentSeasonForTeam(dbCtx, borrowingTeamId)
	if err != nil {
		return err
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var playerId, lendingTeamId, loanFee, buyOptionFee int64
	var durationSeasons, wageContributionPercent int
	err = tx.QueryRow(GetLoanListingForUpdateStmtSql, listingId).Scan(&playerId, &lendingTeamId, &durationSeasons,
		&loanFee, &wageContributionPercent, &buyOptionFee)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLoanListingNotFound
	}
	if err != nil {
		return err
	}
	if lendingTeamId == borrowingTeamId {
		return ErrCannotAcceptOwnLoan
	}

	if loanFee > 0 {
		var borrowerBudget int64
		if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, borrowingTeamId).Scan(&borrowerBudget); err != nil {
			return err
		}
		if borrowerBudget < loanFee {
			return ErrInsufficientBudget
		}
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
	if _, err := tx.Exec(InsertPlayerLoanStmtSql, playerId, lendingTeamId, borrowingTeamId, returnSeason,
		wageContributionPercent, buyOptionFee); err != nil {
		return err
	}
	if err := RecordPlayerTransfer(tx, playerId, lendingTeamId, borrowingTeamId, loanFee, PLAYER_TRANSFER_LOAN); err != nil {
		return err
	}

	var flows LedgerFlows
	if loanFee > 0 {
		player, err := GetPlayerById(dbCtx, playerId)
		if err != nil {
			return err
		}
		lendingTeam, err := GetTeamById(lendingTeamId, dbCtx)
		if err != nil {
			return err
		}
		borrowingTeam, err := GetTeamById(borrowingTeamId, dbCtx)
		if err != nil {
			return err
		}
		playerName := player.FirstName + " " + player.LastName

		if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -loanFee, borrowingTeamId); err != nil {
			return err
		}
		if _, err := tx.Exec(UpdateTeamBudgetStmtSql, loanFee, lendingTeamId); err != nil {
			return err
		}
		if err := insertLedgerRow(tx, &flows, borrowingTeamId, -loanFee, TRANSACTION_LOAN_FEE,
			"Loan fee for "+playerName+" to "+lendingTeam.TeamName, playerId); err != nil {
			return err
		}
		if err := insertLedgerRow(tx, &flows, lendingTeamId, loanFee, TRANSACTION_LOAN_FEE,
			"Loan fee for "+playerName+" from "+borrowingTeam.TeamName, playerId); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	flows.Commit(context.Background())

	LoansAcceptedCounter.Add(context.Background(), 1)
	notifyLoanAccepted(dbCtx, playerId, lendingTeamId, borrowingTeamId)
	return nil
}

// ExerciseLoanBuyOption lets a borrowing team make a loaned-in player's move permanent by paying
// the buy-option fee attached to the loan (see CreateLoanListing's buyOptionFee) - the same money
// movement and bookkeeping as ExecuteTransfer, except the player already belongs (team_id) to
// the borrowing team, so there's nothing to move: just the fee, the club-history record, and
// dropping the now-settled loan row. The loan row and the borrower's budget row are locked
// (SELECT ... FOR UPDATE) for the duration.
func ExerciseLoanBuyOption(dbCtx *DatabaseContext, playerId int64, borrowingTeamId int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}
	if !IsTeamSolvent(dbCtx, borrowingTeamId) {
		return ErrTeamInsolvent
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var loanId, lendingTeamId, actualBorrowingTeamId, buyOptionFee int64
	err = tx.QueryRow(GetActiveLoanForPlayerForUpdateStmtSql, playerId).Scan(&loanId, &lendingTeamId, &actualBorrowingTeamId, &buyOptionFee)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPlayerNotOnLoan
	}
	if err != nil {
		return err
	}
	if actualBorrowingTeamId != borrowingTeamId {
		return ErrPlayerNotOnLoanToTeam
	}
	if buyOptionFee <= 0 {
		return ErrNoBuyOptionOnLoan
	}

	var borrowerBudget int64
	if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, borrowingTeamId).Scan(&borrowerBudget); err != nil {
		return err
	}
	if borrowerBudget < buyOptionFee {
		return ErrInsufficientBudget
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return err
	}
	lendingTeam, err := GetTeamById(lendingTeamId, dbCtx)
	if err != nil {
		return err
	}
	borrowingTeam, err := GetTeamById(borrowingTeamId, dbCtx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -buyOptionFee, borrowingTeamId); err != nil {
		return err
	}
	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, buyOptionFee, lendingTeamId); err != nil {
		return err
	}
	if err := RecordPlayerClubHistory(tx, playerId, borrowingTeamId); err != nil {
		return err
	}
	if err := RecordPlayerTransfer(tx, playerId, lendingTeamId, borrowingTeamId, buyOptionFee, PLAYER_TRANSFER_BUY_OPTION); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLoanStmtSql, loanId); err != nil {
		return err
	}

	playerName := player.FirstName + " " + player.LastName
	var flows LedgerFlows
	if err := insertLedgerRow(tx, &flows, borrowingTeamId, -buyOptionFee, TRANSACTION_TRANSFER_OUT,
		"Exercised buy option for "+playerName+" from "+lendingTeam.TeamName, playerId); err != nil {
		return err
	}
	if err := insertLedgerRow(tx, &flows, lendingTeamId, buyOptionFee, TRANSACTION_TRANSFER_IN,
		"Buy option exercised for "+playerName+" by "+borrowingTeam.TeamName, playerId); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	flows.Commit(context.Background())

	RecordTransferCompleted(context.Background(), buyOptionFee)

	_ = NotifyTeamOwner(dbCtx, lendingTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   borrowingTeam.TeamName + " exercised their buy option for " + playerName,
		Key:    "notifications.messages.loanBuyOptionExercised",
		Params: map[string]any{"player": playerName, "team": borrowingTeam.TeamName},
	})
	_ = NotifyTeamOwner(dbCtx, borrowingTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Signed " + playerName + " permanently from " + lendingTeam.TeamName,
		Key:    "notifications.messages.loanBuyOptionUsed",
		Params: map[string]any{"player": playerName, "team": lendingTeam.TeamName},
	})
	NotifyWatchers(dbCtx, playerId, WatchEventMoved, borrowingTeam.TeamName, 0, lendingTeamId, borrowingTeamId)

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
	NotifyWatchers(dbCtx, playerId, WatchEventMoved, borrowingTeam.TeamName, 0, lendingTeamId, borrowingTeamId)
}

// GetExpiredLoans lists every active loan, owed back by a borrower in country, whose
// return_season has arrived (<= season).
func GetExpiredLoans(dbCtx *DatabaseContext, country string, season int) ([]PlayerLoan, error) {
	rows, err := dbCtx.Db.Query(GetExpiredLoansStmtSql, country, season)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loans := make([]PlayerLoan, 0)
	for rows.Next() {
		var loan PlayerLoan
		if err := rows.Scan(&loan.LoanId, &loan.PlayerId, &loan.FromTeamId, &loan.ToTeamId, &loan.ReturnSeason, &loan.ReleaseOnReturn); err != nil {
			return nil, err
		}
		loans = append(loans, loan)
	}
	return loans, nil
}

// ReturnLoan sends a loaned player back to its original owner: moves the player, clears them
// from the borrowing team's default lineup (same reasoning as AcceptLoan - they're leaving),
// and removes the loan record. A loan flagged release_on_return (its lender was reset after
// losing its owner) frees the player instead - see returnLoanInTx.
func ReturnLoan(dbCtx *DatabaseContext, loan PlayerLoan) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := returnLoanInTx(tx, loan); err != nil {
		return err
	}

	return tx.Commit()
}

// GetLoansToTeam lists every player currently on loan to teamId with the terms the loan carries
// (return season, wage contribution, buy-option fee) - lets a CPU team decide on a buy option.
func GetLoansToTeam(dbCtx *DatabaseContext, teamId int64) ([]PlayerLoan, error) {
	rows, err := dbCtx.Db.Query(GetLoansToTeamStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loans := make([]PlayerLoan, 0)
	for rows.Next() {
		var loan PlayerLoan
		if err := rows.Scan(&loan.LoanId, &loan.PlayerId, &loan.FromTeamId, &loan.ToTeamId, &loan.ReturnSeason,
			&loan.WageContributionPercent, &loan.BuyOptionFee, &loan.ReleaseOnReturn); err != nil {
			return nil, err
		}
		loans = append(loans, loan)
	}
	return loans, rows.Err()
}

// LoanedOutPlayer is a player a club has out on loan, with where he is and on what terms - see
// GetLoanedOutPlayers. The borrowing club's wage share is the complement of WageContributionPercent
// (the part the lender keeps paying); BuyOptionFee is zero when the loan carries no buy option.
type LoanedOutPlayer struct {
	Player                  Player `json:"player"`
	BorrowingTeamId         int64  `json:"borrowingTeamId"`
	BorrowingTeamName       string `json:"borrowingTeamName"`
	BorrowingDivisionId     int64  `json:"borrowingDivisionId"`
	BorrowingDivisionName   string `json:"borrowingDivisionName"`
	ReturnSeason            int    `json:"returnSeason"`
	WageContributionPercent int    `json:"wageContributionPercent"`
	BuyOptionFee            int64  `json:"buyOptionFee"`
	GoalsForBorrower        int    `json:"goalsForBorrower"`
}

// LoanedInPlayer marks a player of a club's squad who is only there on loan, so the squad list
// can flag him without a per-player status request - see TeamDetails.LoanedIn.
type LoanedInPlayer struct {
	PlayerId       int64  `json:"playerId"`
	ParentTeamId   int64  `json:"parentTeamId"`
	ParentTeamName string `json:"parentTeamName"`
	ReturnSeason   int    `json:"returnSeason"`
}

// GetLoanedOutPlayersStmtSql returns each loaned player's columns (playerColumnsSql) followed by
// the loan's terms and the borrowing club and its division. A release-on-return loan (the lender
// was reset, see ResetReleasedTeam) never shows: the player is not coming back to that club.
const GetLoanedOutPlayersStmtSql = "SELECT " + playerColumnsSql + ", pl.to_team_id, t.name, t.league_id, l.name, pl.return_season, " +
	"pl.wage_contribution_percent, pl.buy_option_fee " +
	"FROM player_loans pl JOIN players p ON p.player_id = pl.player_id " +
	"JOIN teams t ON t.team_id = pl.to_team_id JOIN leagues l ON l.league_id = t.league_id " +
	"WHERE pl.from_team_id = $1 AND NOT pl.release_on_return ORDER BY pl.return_season, p.last_name, p.player_id"

// GetLoanedInPlayersStmtSql lists the players on loan to a club with their parent club.
const GetLoanedInPlayersStmtSql = "SELECT pl.player_id, pl.from_team_id, t.name, pl.return_season " +
	"FROM player_loans pl JOIN teams t ON t.team_id = pl.from_team_id WHERE pl.to_team_id = $1 ORDER BY pl.loan_id"

// GetLoanedOutPlayers lists the players teamId has out on loan, soonest return first, with the
// goals each has scored for his current (borrowing) club. Never nil.
func GetLoanedOutPlayers(dbCtx *DatabaseContext, teamId int64) ([]LoanedOutPlayer, error) {
	rows, err := dbCtx.Db.Query(GetLoanedOutPlayersStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loaned := make([]LoanedOutPlayer, 0)
	for rows.Next() {
		var l LoanedOutPlayer
		player, err := scanPlayer(rows, &l.BorrowingTeamId, &l.BorrowingTeamName, &l.BorrowingDivisionId, &l.BorrowingDivisionName,
			&l.ReturnSeason, &l.WageContributionPercent, &l.BuyOptionFee)
		if err != nil {
			return nil, err
		}
		l.Player = *player
		loaned = append(loaned, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	// A club has at most a handful of players out (a CPU club is capped by AiMaxLoanedOut), so one
	// small read per player is fine.
	for i := range loaned {
		stats, err := GetPlayerCareerStats(dbCtx, loaned[i].Player.Id, loaned[i].BorrowingTeamId)
		if err != nil {
			return nil, err
		}
		loaned[i].GoalsForBorrower = stats.GoalsForTeam
	}
	return loaned, nil
}

// GetLoanedInPlayers lists the players currently on loan to teamId. Never nil.
func GetLoanedInPlayers(dbCtx *DatabaseContext, teamId int64) ([]LoanedInPlayer, error) {
	rows, err := dbCtx.Db.Query(GetLoanedInPlayersStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loaned := make([]LoanedInPlayer, 0)
	for rows.Next() {
		var l LoanedInPlayer
		if err := rows.Scan(&l.PlayerId, &l.ParentTeamId, &l.ParentTeamName, &l.ReturnSeason); err != nil {
			return nil, err
		}
		loaned = append(loaned, l)
	}
	return loaned, rows.Err()
}
