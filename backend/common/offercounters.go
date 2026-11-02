package common

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
)

// One counter-offer round on a pending transfer offer (see openspec/specs/transfer-negotiation):
// the selling club answers with a higher fee, and the offer then waits for the buyer to accept it
// (AcceptCounterOffer), decline it (DeclineCounterOffer) or let it lapse (ExpireCounterOffers).
const (
	// CounterOfferLifetime is how long a counter waits for the buyer before it expires.
	CounterOfferLifetime = 72 * time.Hour
	// MaxCounterOfferMultiple caps a counter fee relative to the offered fee.
	MaxCounterOfferMultiple = 10
	// CpuCounterMinShare is the least share of its threshold an offer must reach for a CPU
	// Manager or Director to counter it instead of rejecting it; the counter is the threshold
	// rounded up to the next CpuCounterRounding.
	CpuCounterMinShare = 0.70
	CpuCounterRounding = 1000
)

var ErrInvalidCounterAmount = errors.New("a counter-offer must be above the offered fee and at most " + strconv.Itoa(MaxCounterOfferMultiple) + " times it")
var ErrOfferAlreadyCountered = errors.New("this offer has already been countered")
var ErrOfferCountered = errors.New("this offer has been countered - it now waits for the buyer's answer")
var ErrNoCounterOffer = errors.New("this offer has no counter-offer to answer")

const getOfferCounterStateStmtSql = "SELECT player_id, from_team_id, to_team_id, amount, counter_amount FROM transfer_offers WHERE offer_id = $1"

// counterOfferStmtSql is the whole guard in one statement: only the seller, only once, only a fee
// in range (the 10 is MaxCounterOfferMultiple) - two concurrent counters on one offer can't both land.
const counterOfferStmtSql = "UPDATE transfer_offers SET counter_amount = $3, countered_at = now() " +
	"WHERE offer_id = $1 AND to_team_id = $2 AND counter_amount IS NULL AND $3 > amount AND $3 <= amount * 10"

const declineCounterOfferStmtSql = "DELETE FROM transfer_offers WHERE offer_id = $1 AND from_team_id = $2 AND counter_amount IS NOT NULL " +
	"RETURNING player_id, to_team_id, amount, counter_amount"

// expireCounterOffersStmtSql hands each lapsed counter to exactly one caller, however many engine
// replicas run it at once, and leaves the buyer the usual rejection trace (a CPU buyer then waits
// out its cooldown before bidding for him again).
const expireCounterOffersStmtSql = "WITH gone AS (DELETE FROM transfer_offers WHERE countered_at IS NOT NULL AND countered_at < $1 " +
	"RETURNING offer_id, player_id, from_team_id, to_team_id, amount, counter_amount), " +
	"traced AS (INSERT INTO player_offer_rejections (player_id, from_team_id, amount) SELECT player_id, from_team_id, amount FROM gone) " +
	"SELECT offer_id, player_id, from_team_id, to_team_id, amount, counter_amount FROM gone"

// offerCounterState is a non-locking read of an offer's parties and counter, used to explain why a
// guarded statement touched no row.
type offerCounterState struct {
	playerId, fromTeamId, toTeamId, amount int64
	counterAmount                          sql.NullInt64
}

func getOfferCounterState(dbCtx *DatabaseContext, offerId int64) (*offerCounterState, error) {
	var s offerCounterState
	err := dbCtx.Db.QueryRow(getOfferCounterStateStmtSql, offerId).Scan(&s.playerId, &s.fromTeamId, &s.toTeamId, &s.amount, &s.counterAmount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// validCounterAmount is the fee rule of a counter: strictly above the offered fee and at most
// MaxCounterOfferMultiple times it.
func validCounterAmount(offered int64, counter int64) bool {
	return counter > offered && counter <= offered*MaxCounterOfferMultiple
}

// FormatFee renders a fee the way notification texts show it: "€1,234,000".
func FormatFee(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	digits := strconv.FormatInt(amount, 10)
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for i := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, digits[i])
	}
	if negative {
		return "-€" + string(out)
	}
	return "€" + string(out)
}

// offerNames returns the player's and both clubs' names for a notification; empty strings when a
// lookup fails (the notification still goes out, just less descriptive).
func offerNames(dbCtx *DatabaseContext, playerId, fromTeamId, toTeamId int64) (player, buyer, seller string) {
	if p, err := GetPlayerById(dbCtx, playerId); err == nil {
		player = p.FirstName + " " + p.LastName
	}
	if t, err := GetTeamById(fromTeamId, dbCtx); err == nil {
		buyer = t.TeamName
	}
	if t, err := GetTeamById(toTeamId, dbCtx); err == nil {
		seller = t.TeamName
	}
	return player, buyer, seller
}

// CounterOffer lets the club an offer was made to answer it once with a higher fee, while the
// transfer window is open. The offer then can't be accepted at its original fee (see AcceptOffer):
// it waits for the buyer, who is told (NOTIFICATION_COUNTER_OFFER).
func CounterOffer(dbCtx *DatabaseContext, offerId int64, toTeamId int64, amount int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}
	state, err := getOfferCounterState(dbCtx, offerId)
	if err != nil {
		return err
	}
	if state.toTeamId != toTeamId {
		return ErrNotOfferRecipient
	}
	if state.counterAmount.Valid {
		return ErrOfferAlreadyCountered
	}
	if !validCounterAmount(state.amount, amount) {
		return ErrInvalidCounterAmount
	}

	result, err := dbCtx.Db.Exec(counterOfferStmtSql, offerId, toTeamId, amount)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		// Someone got there between the read above and the update: it was countered (or resolved).
		if _, err := getOfferCounterState(dbCtx, offerId); err != nil {
			return err
		}
		return ErrOfferAlreadyCountered
	}

	player, _, seller := offerNames(dbCtx, state.playerId, state.fromTeamId, state.toTeamId)
	_ = NotifyTeamOwnerWithLink(dbCtx, state.fromTeamId, NOTIFICATION_COUNTER_OFFER, NotificationMessage{
		Text: seller + " countered your " + FormatFee(state.amount) + " offer for " + player + " with " + FormatFee(amount),
		Key:  "notifications.messages.counterOffer",
		Params: map[string]any{"player": player, "playerId": state.playerId, "team": seller,
			"amount": FormatFee(state.amount), "counter": FormatFee(amount)},
	}, "/transfers?tab=offers")
	return nil
}

// AcceptCounterOffer lets the buyer take the seller's counter: the same atomic transfer as an
// accepted offer (see executeOfferTx), at the counter fee and with the personal terms the player
// already agreed to. The negotiation window doesn't apply - the seller already answered - but the
// transfer window, the buyer's solvency and budget and the player still being the seller's do;
// any of those failing leaves the counter open. Two concurrent calls move the money once: the
// second waits on the offer's row lock and then finds it gone.
func AcceptCounterOffer(dbCtx *DatabaseContext, offerId int64, fromTeamId int64) error {
	if !IsTransferWindowOpen(dbCtx) {
		return ErrTransferWindowClosed
	}
	if !IsTeamSolvent(dbCtx, fromTeamId) {
		return ErrTeamInsolvent
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	offer, err := lockOffer(tx, offerId)
	if err != nil {
		return err
	}
	if offer.fromTeamId != fromTeamId {
		return ErrNotOfferSender
	}
	if !offer.counterAmount.Valid {
		return ErrNoCounterOffer
	}

	done, err := executeOfferTx(dbCtx, tx, offer, offer.counterAmount.Int64)
	if err != nil {
		return err
	}
	return finishOfferTransfer(dbCtx, tx, done, "counter_accepted")
}

// DeclineCounterOffer lets the buyer turn the seller's counter down, which ends the offer; the
// seller is told. Allowed whatever the state of the transfer window, like withdrawing an offer.
func DeclineCounterOffer(dbCtx *DatabaseContext, offerId int64, fromTeamId int64) error {
	var playerId, toTeamId, amount, counter int64
	err := dbCtx.Db.QueryRow(declineCounterOfferStmtSql, offerId, fromTeamId).Scan(&playerId, &toTeamId, &amount, &counter)
	if errors.Is(err, sql.ErrNoRows) {
		state, err := getOfferCounterState(dbCtx, offerId)
		if err != nil {
			return err
		}
		if state.fromTeamId != fromTeamId {
			return ErrNotOfferSender
		}
		return ErrNoCounterOffer
	}
	if err != nil {
		return err
	}
	TransferOffersResolvedCounter.Add(context.Background(), 1, metric.WithAttributes(OfferOutcomeAttribute("counter_declined")))

	player, buyer, _ := offerNames(dbCtx, playerId, fromTeamId, toTeamId)
	_ = NotifyTeamOwner(dbCtx, toTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
		Text:   buyer + " declined your " + FormatFee(counter) + " counter-offer for " + player,
		Key:    "notifications.messages.counterDeclined",
		Params: map[string]any{"player": player, "team": buyer, "counter": FormatFee(counter)},
	})
	return nil
}

// ExpireCounterOffers removes every counter the buyer left unanswered for CounterOfferLifetime and
// tells both clubs; it returns how many lapsed. Safe to run on several engine replicas at once:
// the single DELETE ... RETURNING hands each lapsed offer to exactly one of them.
func ExpireCounterOffers(dbCtx *DatabaseContext, now time.Time) (int, error) {
	rows, err := dbCtx.Db.Query(expireCounterOffersStmtSql, now.Add(-CounterOfferLifetime))
	if err != nil {
		return 0, err
	}
	type lapsed struct{ offerId, playerId, fromTeamId, toTeamId, amount, counter int64 }
	var gone []lapsed
	for rows.Next() {
		var l lapsed
		if err := rows.Scan(&l.offerId, &l.playerId, &l.fromTeamId, &l.toTeamId, &l.amount, &l.counter); err != nil {
			rows.Close()
			return 0, err
		}
		gone = append(gone, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(gone) > 0 {
		TransferOffersResolvedCounter.Add(context.Background(), int64(len(gone)), metric.WithAttributes(OfferOutcomeAttribute("counter_expired")))
	}

	for _, l := range gone {
		player, buyer, seller := offerNames(dbCtx, l.playerId, l.fromTeamId, l.toTeamId)
		if err := NotifyTeamOwner(dbCtx, l.toTeamId, NOTIFICATION_TRANSFER, NotificationMessage{
			Text:   buyer + " did not answer your " + FormatFee(l.counter) + " counter-offer for " + player + " - the offer has expired",
			Key:    "notifications.messages.counterExpired",
			Params: map[string]any{"player": player, "team": buyer, "counter": FormatFee(l.counter)},
		}); err != nil {
			log.Errorf("error notifying the seller of expired counter-offer %d: %s", l.offerId, err)
		}
		if err := NotifyTeamOwnerWithLink(dbCtx, l.fromTeamId, NOTIFICATION_OFFER_DECLINED, NotificationMessage{
			Text:   "Your offer for " + player + " expired: you did not answer " + seller + "'s " + FormatFee(l.counter) + " counter-offer",
			Key:    "notifications.messages.counterExpiredBuyer",
			Params: map[string]any{"player": player, "team": seller, "counter": FormatFee(l.counter)},
		}, "/player/"+strconv.FormatInt(l.playerId, 10)+"?renegotiate=1"); err != nil {
			log.Errorf("error notifying the buyer of expired counter-offer %d: %s", l.offerId, err)
		}
	}
	return len(gone), nil
}
