package common

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// team_id is coalesced to 0 - GetPlayerById is also used to look up a free agent (e.g. from
// SignFreeAgent), whose team_id is NULL, and Player.TeamId is a plain non-nullable int64.
// last_team_id is coalesced the same way, for the same reason (see Player.LastTeamId).
//
// The columns are qualified with the alias p so the same list serves GetPlayerById and the listing
// queries that join players in (GetActiveTransferListings, GetActiveLoanListings) - scanPlayer
// reads them back in this order.
const playerColumnsSql = "p.player_id, COALESCE(p.team_id, 0), p.first_name, p.last_name, p.birth_day, p.goal_keeper, p.defense, p.\"lateral\", p.pass, p.midfield, " +
	"p.dribbling, p.shot, p.\"header\", p.quick, p.creative, p.technical, p.wage, p.age, p.avatar, p.contract_years_remaining, p.birth_country, " +
	"p.free_agent_seasons, COALESCE(p.last_team_id, 0), COALESCE(p.shirt_number, 0), p.stamina, p.form, " +
	"p.release_clause, p.release_clause_set_at"

const GetPlayerByIdStmtSql = "SELECT " + playerColumnsSql + " FROM players p WHERE p.player_id = $1"

const InsertTransferListingStmtSql = "INSERT INTO transfer_listings (player_id, selling_team_id, asking_price) VALUES ($1, $2, $3)"

// GetActiveTransferListingsStmtSql returns each listing's player (playerColumnsSql) followed by
// the listing's own columns, so GetActiveTransferListings needs no per-listing player lookup.
const GetActiveTransferListingsStmtSql = "SELECT " + playerColumnsSql + ", tl.listing_id, tl.selling_team_id, t.name, tl.asking_price, tl.listed_at " +
	"FROM transfer_listings tl JOIN teams t ON t.team_id = tl.selling_team_id JOIN players p ON p.player_id = tl.player_id " +
	"ORDER BY tl.listed_at DESC"

const DeleteTransferListingByPlayerAndTeamStmtSql = "DELETE FROM transfer_listings WHERE player_id = $1 AND selling_team_id = $2"

const GetTransferListingForUpdateStmtSql = "SELECT player_id, selling_team_id, asking_price FROM transfer_listings WHERE listing_id = $1 " + ForUpdate

const DeleteTransferListingStmtSql = "DELETE FROM transfer_listings WHERE listing_id = $1"

const DeletePlayerLineupPositionsStmtSql = "DELETE FROM default_lineup_positions WHERE player_id = $1"

// clearPlayerMarketTx takes playerId off the market of teamId, the club he is leaving: every
// pending offer on him and his transfer and loan listings go, inside the caller's transaction, so
// no listing or offer outlives his move (a later buyer would otherwise pay his old club).
func clearPlayerMarketTx(tx *sql.Tx, playerId, teamId int64) error {
	if _, err := tx.Exec(DeleteOffersForPlayerStmtSql, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteTransferListingByPlayerAndTeamStmtSql, playerId, teamId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteLoanListingByPlayerAndTeamStmtSql, playerId, teamId); err != nil {
		return err
	}
	return nil
}

const UpdatePlayerTeamStmtSql = "UPDATE players SET team_id = $1, shirt_number = $2 WHERE player_id = $3"

// movePlayerToTeam moves playerId onto teamId's roster, assigning them teamId's lowest free
// shirt number in the same statement (see nextFreeShirtNumber) - used by every path that hands a
// player straight to a new team (a transfer, a loan taking effect or ending, an accepted
// transfer offer), as opposed to SignFreeAgent/PromoteFromAcademy, which fold the same idea into
// their own more specific UPDATE statements.
func movePlayerToTeam(tx *sql.Tx, teamId int64, playerId int64) error {
	shirtNumber, err := nextFreeShirtNumber(tx, teamId)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(UpdatePlayerTeamStmtSql, teamId, shirtNumber, playerId); err != nil {
		return err
	}
	// A club that signs a player it follows stops following him (openspec player-watchlist
	// "Signed by the follower").
	_, err = tx.Exec(DeletePlayerFromWatchlistStmtSql, teamId, playerId)
	return err
}

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

// scanPlayer reads a Player from the playerColumnsSql columns (in that order) at the start of
// the row. extra are scanned after them, for queries that select more columns past the player's.
func scanPlayer(row rowScanner, extra ...any) (*Player, error) {
	var p Player
	var birthDay int64
	var avatar string
	var birthCountry sql.NullString
	var clause sql.NullInt64
	var clauseSetAt sql.NullTime
	dest := append([]any{&p.Id, &p.TeamId, &p.FirstName, &p.LastName, &birthDay,
		&p.GoalKeeper, &p.Defense, &p.Lateral, &p.Pass, &p.Midfield, &p.Dribbling, &p.Shot,
		&p.Header, &p.Quick, &p.Creative, &p.Technical, &p.Wage, &p.Age, &avatar, &p.ContractYearsRemaining, &birthCountry,
		&p.FreeAgentSeasons, &p.LastTeamId, &p.ShirtNumber, &p.Stamina, &p.Form, &clause, &clauseSetAt}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	p.setReleaseClause(clause, clauseSetAt)
	p.BirthDay = time.UnixMilli(birthDay)
	p.Avatar = GetAvatarFromString(avatar)
	p.BirthCountry = birthCountry.String
	return &p, nil
}

// GetPlayerById fetches a single player by id, regardless of which team they belong to.
func GetPlayerById(dbCtx *DatabaseContext, playerId int64) (*Player, error) {
	return scanPlayer(dbCtx.Db.QueryRow(GetPlayerByIdStmtSql, playerId))
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

const GetTransferListedPlayerIdsForTeamStmtSql = "SELECT player_id FROM transfer_listings WHERE selling_team_id = $1"

// GetTransferListedPlayerIdsForTeam returns the set of teamId's own players currently listed for
// transfer, keyed by player_id - one query to check an entire squad at once (see
// GetTeamDetailsById's per-player skill redaction) instead of the N+1 that IsPlayerTransferListed
// would need called once per player.
func GetTransferListedPlayerIdsForTeam(dbCtx *DatabaseContext, teamId int64) (map[int64]bool, error) {
	rows, err := dbCtx.Db.Query(GetTransferListedPlayerIdsForTeamStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	listed := make(map[int64]bool)
	for rows.Next() {
		var playerId int64
		if err := rows.Scan(&playerId); err != nil {
			return nil, err
		}
		listed[playerId] = true
	}
	return listed, rows.Err()
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
	if signed, err := IsPlayerPreContracted(dbCtx.Db, playerId); err != nil {
		return err
	} else if signed {
		return ErrPlayerPreContracted
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
	if err != nil {
		return err
	}
	if seller, err := GetTeamById(sellingTeamId, dbCtx); err == nil {
		NotifyWatchers(dbCtx, playerId, WatchEventTransferListed, seller.TeamName, askingPrice, sellingTeamId)
	}
	return nil
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

	listings := make([]TransferListing, 0)
	playerIds := make([]int64, 0)
	for rows.Next() {
		var l TransferListing
		player, err := scanPlayer(rows, &l.ListingId, &l.SellingTeamId, &l.SellingTeamName, &l.AskingPrice, &l.ListedAt)
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
	if !IsTeamSolvent(dbCtx, buyerTeamId) {
		return ErrTeamInsolvent
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
	// A listing whose player has since left the seller (released, contract expired) is stale: it
	// goes, and nobody pays the old club for him.
	var ownerTeamId int64
	if err := tx.QueryRow(GetPlayerTeamForUpdateStmtSql, playerId).Scan(&ownerTeamId); err != nil {
		return err
	}
	if ownerTeamId != sellingTeamId {
		if _, err := tx.Exec(DeleteTransferListingStmtSql, listingId); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrListingNotFound
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
	// Judged from the seller's squad before the move; a domestic sale of a key player feeds the
	// pair's rivalry once the transfer has committed (see openspec rivalries).
	keyPlayer := isKeyPlayerOf(dbCtx, sellingTeamId, playerId)

	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -askingPrice, buyerTeamId); err != nil {
		return err
	}
	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, askingPrice, sellingTeamId); err != nil {
		return err
	}
	if err := movePlayerToTeam(tx, buyerTeamId, playerId); err != nil {
		return err
	}
	if err := RecordPlayerClubHistory(tx, playerId, buyerTeamId); err != nil {
		return err
	}
	if err := RecordPlayerTransfer(tx, playerId, sellingTeamId, buyerTeamId, askingPrice, PLAYER_TRANSFER_LISTING); err != nil {
		return err
	}
	if _, err := tx.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return err
	}
	if _, err := tx.Exec(DeleteTransferListingStmtSql, listingId); err != nil {
		return err
	}
	if err := clearPlayerMarketTx(tx, playerId, sellingTeamId); err != nil {
		return err
	}
	// Sold: every pending pre-contract offer on him is off (openspec/specs/pre-contracts).
	preContractBuyers, err := cancelPendingPreContracts(tx, playerId)
	if err != nil {
		return err
	}

	playerName := player.FirstName + " " + player.LastName
	var flows LedgerFlows
	if err := insertLedgerRow(tx, &flows, buyerTeamId, -askingPrice, TRANSACTION_TRANSFER_OUT,
		"Bought "+playerName+" from "+sellerTeam.TeamName, playerId); err != nil {
		return err
	}
	if err := insertLedgerRow(tx, &flows, sellingTeamId, askingPrice, TRANSACTION_TRANSFER_IN,
		"Sold "+playerName+" to "+buyerTeam.TeamName, playerId); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	flows.Commit(context.Background())
	notifyPreContractsCancelled(dbCtx, playerId, preContractBuyers)

	// Counted alongside negotiated offers (see MakeTransferOffer) toward the transfer market
	// stats panel's "offers made" figure, so an AI-managed team buying a listing outright -
	// the only way it ever acquires a listed player - shows up there too, not just human
	// negotiated offers.
	if err := LogTransferOfferMade(dbCtx); err != nil {
		log.Errorf("error logging transfer activity: %s", err)
	}

	RecordTransferCompleted(context.Background(), askingPrice)

	// Best-effort, after the money has actually moved - a notification failure must never
	// undo a completed transfer.
	_ = NotifyTeamOwner(dbCtx, sellingTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Sold " + playerName + " to " + buyerTeam.TeamName,
		Key:    "notifications.messages.transferSold",
		Params: map[string]any{"player": playerName, "team": buyerTeam.TeamName},
	})
	_ = NotifyTeamOwner(dbCtx, buyerTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Bought " + playerName + " from " + sellerTeam.TeamName,
		Key:    "notifications.messages.transferBought",
		Params: map[string]any{"player": playerName, "team": sellerTeam.TeamName},
	})
	NotifyWatchers(dbCtx, playerId, WatchEventMoved, buyerTeam.TeamName, 0, sellingTeamId, buyerTeamId)
	if keyPlayer {
		AddRivalryEvent(dbCtx, sellingTeamId, buyerTeamId, RivalryKeyPlayerSalePoints)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	// pgx/lib/pq both surface this in the error text; avoids an extra driver-specific import
	// just to type-assert a SQLSTATE code.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate key")
}

const GetTeamLoanedInPlayerIdsStmtSql = "SELECT player_id FROM player_loans WHERE to_team_id = $1"
const GetTeamTransferListedPlayerIdsStmtSql = "SELECT player_id FROM transfer_listings WHERE selling_team_id = $1"
const GetTeamLoanListedPlayerIdsStmtSql = "SELECT player_id FROM loan_listings WHERE lending_team_id = $1"
const GetTeamLoanedOutPlayerIdsStmtSql = "SELECT player_id FROM player_loans WHERE from_team_id = $1"
const GetTeamTransferListingsStmtSql = "SELECT player_id, asking_price, listed_at FROM transfer_listings WHERE selling_team_id = $1"

// TeamMarketCommitments is which of a team's current squad players are already tied up on the
// market one way or another - see GetTeamMarketCommitments. Each map is keyed by player id.
type TeamMarketCommitments struct {
	// LoanedIn are players on loan to this team from another club - not this team's to sell.
	LoanedIn map[int64]bool
	// TransferListed are players this team has already put on the transfer list.
	TransferListed map[int64]bool
	// LoanListed are players this team has offered out on loan.
	LoanListed map[int64]bool
	// LoanedOut are players this team owns that are currently out on loan to another club.
	LoanedOut map[int64]bool
}

// GetTeamMarketCommitments fetches TeamMarketCommitments for teamId in three small queries -
// used by AI-managed teams (see engine.listSurplusPlayer) to judge which squad players they
// actually own and can still list, instead of picking one blind and swallowing the error.
func GetTeamMarketCommitments(dbCtx *DatabaseContext, teamId int64) (*TeamMarketCommitments, error) {
	loanedIn, err := queryPlayerIdSet(dbCtx, GetTeamLoanedInPlayerIdsStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	listed, err := queryPlayerIdSet(dbCtx, GetTeamTransferListedPlayerIdsStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	loanListed, err := queryPlayerIdSet(dbCtx, GetTeamLoanListedPlayerIdsStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	loanedOut, err := queryPlayerIdSet(dbCtx, GetTeamLoanedOutPlayerIdsStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	return &TeamMarketCommitments{LoanedIn: loanedIn, TransferListed: listed, LoanListed: loanListed, LoanedOut: loanedOut}, nil
}

// TeamTransferListing is one of a team's own open transfer listings: what it asks and since when.
type TeamTransferListing struct {
	PlayerId    int64
	AskingPrice int64
	ListedAt    time.Time
}

// GetTeamTransferListings returns teamId's open transfer listings keyed by player id - lets a CPU
// team see how long a surplus player has gone unsold (see engine.manageLoanListings).
func GetTeamTransferListings(dbCtx *DatabaseContext, teamId int64) (map[int64]TeamTransferListing, error) {
	rows, err := dbCtx.Db.Query(GetTeamTransferListingsStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	listings := make(map[int64]TeamTransferListing)
	for rows.Next() {
		var l TeamTransferListing
		if err := rows.Scan(&l.PlayerId, &l.AskingPrice, &l.ListedAt); err != nil {
			return nil, err
		}
		listings[l.PlayerId] = l
	}
	return listings, rows.Err()
}

func queryPlayerIdSet(dbCtx *DatabaseContext, query string, teamId int64) (map[int64]bool, error) {
	rows, err := dbCtx.Db.Query(query, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}
