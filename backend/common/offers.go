package common

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
)

var ErrInvalidOfferAmount = errors.New("offer amount must be positive")
var ErrInvalidPersonalTerms = errors.New("personal terms must include a positive wage for 1 to " + strconv.Itoa(MaxContractYearsRemaining) + " seasons")
var ErrCannotOfferOnOwnPlayer = errors.New("cannot make an offer on your own player")
var ErrOfferAlreadyExists = errors.New("you already have a pending offer on this player")
var ErrOfferNotFound = errors.New("offer not found")
var ErrNotOfferSender = errors.New("you did not make this offer")
var ErrNotOfferRecipient = errors.New("this offer is not for one of your players")
var ErrOfferNoLongerValid = errors.New("the player no longer belongs to the team this offer was made to")
var ErrPlayerIsFreeAgent = errors.New("player is a free agent - sign them instead of making an offer")
var ErrOfferNegotiationWindow = errors.New("this offer can't be accepted yet - it must stay open for a while so other offers can still come in")
var ErrInvalidOfferNegotiationWindow = errors.New("offer negotiation window must be zero or greater")

const GetOfferNegotiationWindowMinutesStmtSql = "SELECT offer_negotiation_window_minutes FROM season"
const SetOfferNegotiationWindowMinutesStmtSql = "UPDATE season SET offer_negotiation_window_minutes = $1"

// GetOfferNegotiationWindowMinutes returns the admin-configured minimum time (in minutes) a
// transfer offer must sit pending before the receiving team is allowed to accept it (see
// AcceptOffer) - long enough that a seller who lists a player or otherwise attracts interest has
// a real chance to compare multiple offers (see GetOffersReceived, which already lists every
// pending offer on a player from every interested buyer) before locking one in, rather than the
// first offer received winning by default. Rejecting or withdrawing an offer is never gated by
// this - only accepting one is. See handlers.AdminSetOfferNegotiationWindowHandler.
func GetOfferNegotiationWindowMinutes(dbCtx *DatabaseContext) (int, error) {
	var minutes int
	err := dbCtx.Db.QueryRow(GetOfferNegotiationWindowMinutesStmtSql).Scan(&minutes)
	return minutes, err
}

// SetOfferNegotiationWindowMinutes updates the window. Zero is allowed (an offer becomes
// acceptable the instant it's made - the same way MaxFriendliesPerWeek's zero turns friendlies
// off entirely).
func SetOfferNegotiationWindowMinutes(dbCtx *DatabaseContext, minutes int) error {
	if minutes < 0 {
		return ErrInvalidOfferNegotiationWindow
	}
	_, err := dbCtx.Db.Exec(SetOfferNegotiationWindowMinutesStmtSql, minutes)
	return err
}

// offerAcceptableAt is the point in time an offer made at createdAt becomes acceptable, given the
// current negotiation window - pulled out as a pure function (same reasoning as
// canPromoteFromAcademyAt in academy.go) so the rule can be unit tested without a live database,
// and so the frontend can compute the same "available in Xh Ym" moment from an offer's own
// createdAt without a dedicated API field.
func offerAcceptableAt(createdAt time.Time, window time.Duration) time.Time {
	return createdAt.Add(window)
}

// canAcceptOfferAt reports whether an offer made at createdAt can be accepted as of now, given
// the current negotiation window.
func canAcceptOfferAt(createdAt time.Time, now time.Time, window time.Duration) bool {
	return !now.Before(offerAcceptableAt(createdAt, window))
}

// TransferOffer is a pending "scouting" offer - a buyer's proposed price for a specific player,
// plus the personal terms (wage/contract length) they're proposing to the player, made directly
// to whoever currently owns them, independent of whether that player is listed on the open
// market at all. See MakeTransferOffer/AcceptOffer/RejectOffer/WithdrawOffer.
type TransferOffer struct {
	OfferId      int64     `json:"offerId"`
	Player       Player    `json:"player"`
	FromTeamId   int64     `json:"fromTeamId"`
	FromTeamName string    `json:"fromTeamName"`
	ToTeamId     int64     `json:"toTeamId"`
	ToTeamName   string    `json:"toTeamName"`
	Amount       int64     `json:"amount"`
	OfferedWage  int       `json:"offeredWage"`
	OfferedYears int       `json:"offeredYears"`
	CreatedAt    time.Time `json:"createdAt"`
	// AcceptableAt is when this offer's negotiation window (see GetOfferNegotiationWindowMinutes)
	// elapses and the receiving team is allowed to accept it - computed server-side (see
	// offerAcceptableAt) rather than left for the frontend to derive from CreatedAt, so the
	// admin-configurable window's length has one source of truth instead of a duplicated
	// constant on each side.
	AcceptableAt time.Time `json:"acceptableAt"`
}

// TransferOfferResult mirrors ContractRenewalOffer/FreeAgentSigningResult's shape - the player
// judges the personal terms (wage/years) on the spot, before the offer ever reaches the selling
// club, so a rejection here costs nothing and reports the player's real expectations immediately
// rather than leaving a doomed offer sitting pending for someone else to eventually decline.
type TransferOfferResult struct {
	Accepted           bool   `json:"accepted"`
	Reason             string `json:"reason,omitempty"`
	ExpectedWage       int    `json:"expectedWage"`
	MaxAcceptableYears int    `json:"maxAcceptableYears"`
	// Only meaningful when Accepted is true - the offer that was actually created.
	OfferId int64 `json:"offerId,omitempty"`
}

const InsertTransferOfferStmtSql = "INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount, offered_wage, offered_years) " +
	"VALUES ($1, $2, $3, $4, $5, $6) RETURNING offer_id"

const GetOffersReceivedStmtSql = "SELECT o.offer_id, o.player_id, o.from_team_id, tf.name, o.to_team_id, tt.name, o.amount, o.offered_wage, o.offered_years, o.created_at " +
	"FROM transfer_offers o JOIN teams tf ON tf.team_id = o.from_team_id JOIN teams tt ON tt.team_id = o.to_team_id " +
	"WHERE o.to_team_id = $1 ORDER BY o.created_at DESC"

const GetOffersSentStmtSql = "SELECT o.offer_id, o.player_id, o.from_team_id, tf.name, o.to_team_id, tt.name, o.amount, o.offered_wage, o.offered_years, o.created_at " +
	"FROM transfer_offers o JOIN teams tf ON tf.team_id = o.from_team_id JOIN teams tt ON tt.team_id = o.to_team_id " +
	"WHERE o.from_team_id = $1 ORDER BY o.created_at DESC"

const GetOfferOwnersStmtSql = "SELECT from_team_id, to_team_id FROM transfer_offers WHERE offer_id = $1"
const GetOfferForUpdateStmtSql = "SELECT player_id, from_team_id, to_team_id, amount, offered_wage, offered_years, created_at FROM transfer_offers WHERE offer_id = $1 " + ForUpdate
const DeleteTransferOfferStmtSql = "DELETE FROM transfer_offers WHERE offer_id = $1"
const DeleteOffersForPlayerStmtSql = "DELETE FROM transfer_offers WHERE player_id = $1"

// team_id is coalesced to 0 - a player can go from owned to a free agent (team_id NULL) while an
// offer on them sits pending (contract expiry, an early release, or AssignTeam reassigning their
// squad), and toTeamId is always a real (non-zero) team id, so the mismatch check below still
// correctly falls through to ErrOfferNoLongerValid instead of crashing on the NULL.
const GetPlayerTeamForUpdateStmtSql = "SELECT COALESCE(team_id, 0) FROM players WHERE player_id = $1 " + ForUpdate

// MakeTransferOffer proposes amount (a fee to the selling club) plus offeredWage/offeredYears (a
// contract to the player) for playerId, wherever they currently play - the offer goes to
// whoever owns them right now (their team_id at the time the offer is made). The player judges
// the personal terms immediately, exactly like MakeContractRenewalOffer's expected-wage/
// age-based-max-years check: if they wouldn't accept, no offer is created at all - there's no
// point spending a transfer fee negotiating with a club over terms the player will never sign.
// Rejects an offer on your own player, on a free agent (see SignFreeAgent - there's nobody to
// pay a fee to, and no team would ever see the offer to accept it), on a player currently out on
// loan (a borrowing team doesn't get to sell what isn't theirs - same rule as
// CreateTransferListing), a duplicate offer from the same team on the same player (enforced by a
// unique constraint), or anything while the transfer window is closed.
func MakeTransferOffer(dbCtx *DatabaseContext, playerId int64, fromTeamId int64, amount int64, offeredWage int, offeredYears int) (*TransferOfferResult, error) {
	if amount <= 0 {
		return nil, ErrInvalidOfferAmount
	}
	if offeredWage <= 0 || offeredYears < 1 || offeredYears > MaxContractYearsRemaining {
		return nil, ErrInvalidPersonalTerms
	}
	if !IsTransferWindowOpen(dbCtx) {
		return nil, ErrTransferWindowClosed
	}
	if !IsTeamSolvent(dbCtx, fromTeamId) {
		return nil, ErrTeamInsolvent
	}

	player, err := GetPlayerById(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	if player.TeamId == 0 {
		return nil, ErrPlayerIsFreeAgent
	}
	if player.TeamId == fromTeamId {
		return nil, ErrCannotOfferOnOwnPlayer
	}
	onLoan, err := IsPlayerOnActiveLoan(dbCtx, playerId)
	if err != nil {
		return nil, err
	}
	if onLoan {
		return nil, ErrPlayerOnLoan
	}

	expectedWage := PlayerWageFromSkills(player.GoalKeeper, player.Defense, player.Lateral, player.Pass,
		player.Midfield, player.Dribbling, player.Shot)
	maxYears := MaxAcceptableRenewalYears(player.Age)
	result := &TransferOfferResult{ExpectedWage: expectedWage, MaxAcceptableYears: maxYears}

	var reasons []string
	if offeredWage < expectedWage {
		reasons = append(reasons, "wage below what they expect")
	}
	if offeredYears > maxYears {
		reasons = append(reasons, "too many seasons for their age")
	}
	if len(reasons) > 0 {
		result.Reason = strings.Join(reasons, "; ")
		return result, nil
	}

	var offerId int64
	err = dbCtx.Db.QueryRow(InsertTransferOfferStmtSql, playerId, fromTeamId, player.TeamId, amount, offeredWage, offeredYears).Scan(&offerId)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrOfferAlreadyExists
		}
		return nil, err
	}
	if err := LogTransferOfferMade(dbCtx); err != nil {
		log.Errorf("error logging transfer offer activity: %s", err)
	}
	TransferOffersMadeCounter.Add(context.Background(), 1)

	result.Accepted = true
	result.OfferId = offerId
	return result, nil
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
			&r.offer.ToTeamId, &r.offer.ToTeamName, &r.offer.Amount, &r.offer.OfferedWage, &r.offer.OfferedYears, &r.offer.CreatedAt); err != nil {
			return nil, err
		}
		pending = append(pending, r)
	}
	rows.Close()

	windowMinutes, err := GetOfferNegotiationWindowMinutes(dbCtx)
	if err != nil {
		return nil, err
	}
	window := time.Duration(windowMinutes) * time.Minute

	offers := make([]TransferOffer, 0, len(pending))
	for _, r := range pending {
		player, err := GetPlayerById(dbCtx, r.playerId)
		if err != nil {
			continue
		}
		r.offer.Player = *player
		r.offer.AcceptableAt = offerAcceptableAt(r.offer.CreatedAt, window)
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
	if err != nil {
		return err
	}
	TransferOffersResolvedCounter.Add(context.Background(), 1, metric.WithAttributes(OfferOutcomeAttribute("withdrawn")))
	return nil
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
	if err != nil {
		return err
	}
	TransferOffersResolvedCounter.Add(context.Background(), 1, metric.WithAttributes(OfferOutcomeAttribute("rejected")))
	return nil
}

// AcceptOffer executes the transfer at the offer's amount: moves the player, debits/credits both
// budgets, logs both sides of the ledger, clears the seller's default lineup, and drops every
// other pending offer on the same player (they're all stale once it's sold) plus any open
// listing for it - all atomically. Re-checks that the player still belongs to toTeamId (they
// might have been sold or loaned out since the offer was made) and that the buyer can still
// afford it, since an offer can sit pending for a while before it's acted on. Also refuses an
// offer still inside its admin-configured negotiation window (see ErrOfferNegotiationWindow,
// GetOfferNegotiationWindowMinutes) - rejecting or withdrawing an offer is never gated this way,
// only accepting one, so a seller comparing offers never risks losing the ones they'd rather
// keep open.
func AcceptOffer(dbCtx *DatabaseContext, offerId int64, toTeamId int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}

	windowMinutes, err := GetOfferNegotiationWindowMinutes(dbCtx)
	if err != nil {
		return err
	}
	window := time.Duration(windowMinutes) * time.Minute

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var playerId, fromTeamId, actualToTeamId, amount int64
	var offeredWage, offeredYears int
	var createdAt time.Time
	err = tx.QueryRow(GetOfferForUpdateStmtSql, offerId).Scan(&playerId, &fromTeamId, &actualToTeamId, &amount, &offeredWage, &offeredYears, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOfferNotFound
	}
	if err != nil {
		return err
	}
	if actualToTeamId != toTeamId {
		return ErrNotOfferRecipient
	}
	if !canAcceptOfferAt(createdAt, time.Now(), window) {
		return ErrOfferNegotiationWindow
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
	if err := movePlayerToTeam(tx, fromTeamId, playerId); err != nil {
		return err
	}
	if err := RecordPlayerClubHistory(tx, playerId, fromTeamId); err != nil {
		return err
	}
	// Applies the personal terms the player already agreed to when this offer was made (see
	// MakeTransferOffer) - not re-judged against their expectations here, the same way an
	// accepted contract renewal offer's terms aren't re-litigated later.
	if _, err := tx.Exec(ApplyRenewalOfferStmtSql, offeredYears, offeredWage, playerId); err != nil {
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

	RecordTransferCompleted(context.Background(), amount)
	TransferOffersResolvedCounter.Add(context.Background(), 1, metric.WithAttributes(OfferOutcomeAttribute("accepted")))

	_ = NotifyTeamOwner(dbCtx, toTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Sold " + playerName + " to " + buyerTeam.TeamName,
		Key:    "notifications.messages.transferSold",
		Params: map[string]any{"player": playerName, "team": buyerTeam.TeamName},
	})
	_ = NotifyTeamOwner(dbCtx, fromTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   "Bought " + playerName + " from " + sellerTeam.TeamName,
		Key:    "notifications.messages.transferBought",
		Params: map[string]any{"player": playerName, "team": sellerTeam.TeamName},
	})

	return nil
}
