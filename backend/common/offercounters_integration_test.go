package common

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Counter-offers (openspec/specs/transfer-negotiation) against a real Postgres (needs
// STRIKER_TEST_DB_DSN). Every test seeds its own clubs, owners and player.

type counterRig struct {
	t                     *testing.T
	db                    *sql.DB
	dbCtx                 *DatabaseContext
	seller, buyer, player int64
	sellerUser, buyerUser int64
}

var counterRigSeq int

// newCounterRig seeds a human-owned seller and buyer (50,000,000 each), a player of the seller and
// an open transfer window.
func newCounterRig(t *testing.T) *counterRig {
	t.Helper()
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zc")
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatalf("seeding season: %s", err)
	}
	if _, err := db.Exec(`UPDATE season SET transfer_window_open = true`); err != nil {
		t.Fatalf("opening the window: %s", err)
	}
	counterRigSeq++
	tag := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + strconv.Itoa(counterRigSeq)
	r := &counterRig{t: t, db: db, dbCtx: dbCtx}
	r.seller = seedTransfersTeam(t, db, "Counter Seller "+tag, "zc")
	r.buyer = seedTransfersTeam(t, db, "Counter Buyer "+tag, "zc")
	r.player = seedTransfersPlayer(t, db, r.seller, "zc")
	r.sellerUser = r.own(r.seller, "seller-"+tag)
	r.buyerUser = r.own(r.buyer, "buyer-"+tag)
	return r
}

func (r *counterRig) own(teamId int64, tag string) int64 {
	r.t.Helper()
	var userId int64
	if err := r.db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ('Counter', 'dev', $1) RETURNING user_id`,
		tag+"@example.test").Scan(&userId); err != nil {
		r.t.Fatalf("seeding user: %s", err)
	}
	if _, err := r.db.Exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, userId, teamId); err != nil {
		r.t.Fatalf("setting owner: %s", err)
	}
	return userId
}

// offer opens a 800,000 offer (wage 1,500, 2 seasons) from the buyer, past the negotiation window.
func (r *counterRig) offer() int64 {
	r.t.Helper()
	var offerId int64
	if err := r.db.QueryRow(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount, offered_wage, offered_years, created_at)
		VALUES ($1, $2, $3, 800000, 1500, 2, now() - interval '3 days') RETURNING offer_id`, r.player, r.buyer, r.seller).Scan(&offerId); err != nil {
		r.t.Fatalf("seeding offer: %s", err)
	}
	return offerId
}

func (r *counterRig) int(query string, args ...any) int64 {
	r.t.Helper()
	var n int64
	if err := r.db.QueryRow(query, args...).Scan(&n); err != nil {
		r.t.Fatalf("%s: %s", query, err)
	}
	return n
}

func (r *counterRig) budget(teamId int64) int64 {
	return r.int(`SELECT budget FROM teams WHERE team_id = $1`, teamId)
}

func (r *counterRig) notifications(userId int64, key string) int64 {
	return r.int(`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND message_key = $2`, userId, key)
}

func (r *counterRig) offerExists(offerId int64) bool {
	return r.int(`SELECT COUNT(*) FROM transfer_offers WHERE offer_id = $1`, offerId) == 1
}

func TestCounterOffer_WaitsForBuyer(t *testing.T) {
	r := newCounterRig(t)
	offerId := r.offer()

	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1000000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}
	sent, err := GetOffersSent(r.dbCtx, r.buyer)
	if err != nil || len(sent) != 1 {
		t.Fatalf("GetOffersSent = %v (%v), want one offer", sent, err)
	}
	o := sent[0]
	if o.CounterAmount == nil || *o.CounterAmount != 1000000 || o.CounteredAt == nil || o.CounterExpiresAt == nil {
		t.Fatalf("sent offer counter = %v / %v / %v, want 1,000,000 with its times", o.CounterAmount, o.CounteredAt, o.CounterExpiresAt)
	}
	if got := o.CounterExpiresAt.Sub(*o.CounteredAt); got != CounterOfferLifetime {
		t.Errorf("counter lifetime = %s, want %s", got, CounterOfferLifetime)
	}
	if err := AcceptOffer(r.dbCtx, offerId, r.seller); !errors.Is(err, ErrOfferCountered) {
		t.Errorf("AcceptOffer on a countered offer = %v, want ErrOfferCountered", err)
	}
	if n := r.notifications(r.buyerUser, "notifications.messages.counterOffer"); n != 1 {
		t.Errorf("buyer counter notifications = %d, want 1", n)
	}
}

func TestCounterOffer_MustBeAboveTheOfferAndAtMostTenTimes(t *testing.T) {
	r := newCounterRig(t)
	offerId := r.offer()
	for _, amount := range []int64{800000, 500000, 8000001} {
		if err := CounterOffer(r.dbCtx, offerId, r.seller, amount); !errors.Is(err, ErrInvalidCounterAmount) {
			t.Errorf("CounterOffer(%d) = %v, want ErrInvalidCounterAmount", amount, err)
		}
	}
	if err := CounterOffer(r.dbCtx, offerId, r.buyer, 1000000); !errors.Is(err, ErrNotOfferRecipient) {
		t.Errorf("CounterOffer by the buyer = %v, want ErrNotOfferRecipient", err)
	}
	if n := r.int(`SELECT COUNT(*) FROM transfer_offers WHERE offer_id = $1 AND counter_amount IS NULL`, offerId); n != 1 {
		t.Errorf("offer was countered by an invalid request")
	}
}

func TestCounterOffer_SecondCounterFails(t *testing.T) {
	r := newCounterRig(t)
	offerId := r.offer()
	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1000000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}
	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1200000); !errors.Is(err, ErrOfferAlreadyCountered) {
		t.Errorf("second CounterOffer = %v, want ErrOfferAlreadyCountered", err)
	}
	if got := r.int(`SELECT counter_amount FROM transfer_offers WHERE offer_id = $1`, offerId); got != 1000000 {
		t.Errorf("counter = %d, want the first one (1,000,000)", got)
	}
}

func TestAcceptCounterOffer_CompletesTheTransferAtTheCounterFee(t *testing.T) {
	r := newCounterRig(t)
	offerId := r.offer()
	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1000000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}
	if err := AcceptCounterOffer(r.dbCtx, offerId, r.seller); !errors.Is(err, ErrNotOfferSender) {
		t.Errorf("AcceptCounterOffer by the seller = %v, want ErrNotOfferSender", err)
	}
	sellerBefore, buyerBefore := r.budget(r.seller), r.budget(r.buyer)

	if err := AcceptCounterOffer(r.dbCtx, offerId, r.buyer); err != nil {
		t.Fatalf("AcceptCounterOffer: %s", err)
	}
	if got := r.budget(r.buyer); got != buyerBefore-1000000 {
		t.Errorf("buyer budget = %d, want %d", got, buyerBefore-1000000)
	}
	if got := r.budget(r.seller); got != sellerBefore+1000000 {
		t.Errorf("seller budget = %d, want %d", got, sellerBefore+1000000)
	}
	var teamId int64
	var wage, years int
	if err := r.db.QueryRow(`SELECT team_id, wage, contract_years_remaining FROM players WHERE player_id = $1`, r.player).Scan(&teamId, &wage, &years); err != nil {
		t.Fatal(err)
	}
	if teamId != r.buyer || wage != 1500 || years != 2 {
		t.Errorf("player = team %d wage %d years %d, want team %d on 1,500 for 2 seasons", teamId, wage, years, r.buyer)
	}
	if r.offerExists(offerId) {
		t.Error("the offer is still there")
	}
	if n := r.int(`SELECT COUNT(*) FROM player_transfers WHERE player_id = $1 AND fee = 1000000 AND kind = $2`, r.player, PLAYER_TRANSFER_OFFER); n != 1 {
		t.Errorf("transfer rows at the counter fee = %d, want 1", n)
	}
	if n := r.int(`SELECT COUNT(*) FROM team_transactions WHERE team_id = $1 AND amount = -1000000 AND reason = $2`, r.buyer, TRANSACTION_TRANSFER_OUT); n != 1 {
		t.Errorf("buyer ledger rows = %d, want 1", n)
	}
	if n := r.notifications(r.sellerUser, "notifications.messages.transferSold"); n != 1 {
		t.Errorf("seller sold notifications = %d, want 1", n)
	}
}

func TestAcceptCounterOffer_BuyerCannotAffordIt(t *testing.T) {
	r := newCounterRig(t)
	offerId := r.offer()
	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1000000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}
	if _, err := r.db.Exec(`UPDATE teams SET budget = 900000 WHERE team_id = $1`, r.buyer); err != nil {
		t.Fatal(err)
	}
	if err := AcceptCounterOffer(r.dbCtx, offerId, r.buyer); !errors.Is(err, ErrInsufficientBudget) {
		t.Fatalf("AcceptCounterOffer = %v, want ErrInsufficientBudget", err)
	}
	if got := r.budget(r.buyer); got != 900000 {
		t.Errorf("buyer budget = %d, want it untouched", got)
	}
	if got := r.int(`SELECT counter_amount FROM transfer_offers WHERE offer_id = $1`, offerId); got != 1000000 {
		t.Errorf("counter = %d, want it still open at 1,000,000", got)
	}
}

func TestDeclineCounterOffer_EndsTheOfferAndTellsTheSeller(t *testing.T) {
	r := newCounterRig(t)
	offerId := r.offer()
	if err := DeclineCounterOffer(r.dbCtx, offerId, r.buyer); !errors.Is(err, ErrNoCounterOffer) {
		t.Errorf("DeclineCounterOffer before a counter = %v, want ErrNoCounterOffer", err)
	}
	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1000000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}
	if err := DeclineCounterOffer(r.dbCtx, offerId, r.seller); !errors.Is(err, ErrNotOfferSender) {
		t.Errorf("DeclineCounterOffer by the seller = %v, want ErrNotOfferSender", err)
	}
	if err := DeclineCounterOffer(r.dbCtx, offerId, r.buyer); err != nil {
		t.Fatalf("DeclineCounterOffer: %s", err)
	}
	if r.offerExists(offerId) {
		t.Error("the offer is still there")
	}
	if n := r.notifications(r.sellerUser, "notifications.messages.counterDeclined"); n != 1 {
		t.Errorf("seller declined notifications = %d, want 1", n)
	}
	if err := DeclineCounterOffer(r.dbCtx, offerId, r.buyer); !errors.Is(err, ErrOfferNotFound) {
		t.Errorf("second DeclineCounterOffer = %v, want ErrOfferNotFound", err)
	}
}

// Two accept calls on one counter (a double click, two tabs) move the money once.
func TestAcceptCounterOffer_ConcurrentAcceptsMoveTheMoneyOnce(t *testing.T) {
	r := newCounterRig(t)
	offerId := r.offer()
	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1000000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}
	buyerBefore := r.budget(r.buyer)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	start := make(chan struct{})
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = AcceptCounterOffer(r.dbCtx, offerId, r.buyer)
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrOfferNotFound):
		default:
			t.Errorf("unexpected error: %s", err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("%d accepts succeeded, want exactly 1 (%v)", succeeded, errs)
	}
	if got := r.budget(r.buyer); got != buyerBefore-1000000 {
		t.Errorf("buyer budget = %d, want %d (charged once)", got, buyerBefore-1000000)
	}
	if n := r.int(`SELECT COUNT(*) FROM team_transactions WHERE team_id = $1 AND reason = $2 AND amount = -1000000`, r.buyer, TRANSACTION_TRANSFER_OUT); n != 1 {
		t.Errorf("buyer ledger rows = %d, want 1", n)
	}
}

// An ignored counter expires once, even with two engine replicas running the expiry at the same
// instant: each club gets exactly one notification, and a fresh counter is left alone.
func TestExpireCounterOffers_OnceAcrossReplicas(t *testing.T) {
	r := newCounterRig(t)
	offerId := r.offer()
	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1000000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}
	if _, err := r.db.Exec(`UPDATE transfer_offers SET countered_at = now() - interval '73 hours' WHERE offer_id = $1`, offerId); err != nil {
		t.Fatal(err)
	}
	fresh := newCounterRig(t)
	freshOffer := fresh.offer()
	if err := CounterOffer(fresh.dbCtx, freshOffer, fresh.seller, 900000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}

	now := time.Now()
	var wg sync.WaitGroup
	counts := make([]int, 2)
	errs := make([]error, 2)
	start := make(chan struct{})
	for i := range counts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			counts[i], errs[i] = ExpireCounterOffers(r.dbCtx, now)
		}(i)
	}
	close(start)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatalf("ExpireCounterOffers: %s", err)
		}
	}

	if r.offerExists(offerId) {
		t.Error("the ignored counter is still there")
	}
	if !fresh.offerExists(freshOffer) {
		t.Error("a fresh counter expired")
	}
	if n := r.notifications(r.sellerUser, "notifications.messages.counterExpired"); n != 1 {
		t.Errorf("seller expiry notifications = %d, want 1", n)
	}
	if n := r.notifications(r.buyerUser, "notifications.messages.counterExpiredBuyer"); n != 1 {
		t.Errorf("buyer expiry notifications = %d, want 1", n)
	}
	if n := r.int(`SELECT COUNT(*) FROM player_offer_rejections WHERE player_id = $1 AND from_team_id = $2 AND amount = 800000`, r.player, r.buyer); n != 1 {
		t.Errorf("rejection traces = %d, want 1", n)
	}
}

// A CPU club (no owner) countering a human's offer tells the human, with both fees.
func TestCounterOffer_CpuCountersAHuman(t *testing.T) {
	r := newCounterRig(t)
	if _, err := r.db.Exec(`UPDATE teams SET owner_id = NULL, ai_level = $2 WHERE team_id = $1`, r.seller, AiLevelDirector); err != nil {
		t.Fatal(err)
	}
	offerId := r.offer()
	if err := CounterOffer(r.dbCtx, offerId, r.seller, 1235000); err != nil {
		t.Fatalf("CounterOffer: %s", err)
	}
	var notificationType int
	var params, link string
	if err := r.db.QueryRow(`SELECT notification_type, message_params::text, link FROM notifications
		WHERE user_id = $1 AND message_key = 'notifications.messages.counterOffer'`, r.buyerUser).Scan(&notificationType, &params, &link); err != nil {
		t.Fatalf("reading the counter notification: %s", err)
	}
	if notificationType != NOTIFICATION_COUNTER_OFFER || link != "/transfers?tab=offers" {
		t.Errorf("notification type %d link %q, want %d to the offers tab", notificationType, link, NOTIFICATION_COUNTER_OFFER)
	}
	for _, want := range []string{`"amount": "€800,000"`, `"counter": "€1,235,000"`} {
		if !strings.Contains(params, want) {
			t.Errorf("params %s lack %s", params, want)
		}
	}
}
