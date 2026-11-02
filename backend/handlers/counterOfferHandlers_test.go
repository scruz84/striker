package handlers

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

// The counter-offer endpoints (openspec/specs/transfer-negotiation): only the seller can counter,
// only the buyer can accept or decline. Runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped
// when unset).
type counterHandlerRig struct {
	t                     *testing.T
	db                    *sql.DB
	dbCtx                 *common.DatabaseContext
	seller, buyer         int64
	sellerUser, buyerUser int64
	player                int64
	router                *gin.Engine
	as                    int64 // the user the next request is made as
}

func newCounterHandlerRig(t *testing.T) *counterHandlerRig {
	t.Helper()
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	dbCtx := &common.DatabaseContext{Db: db, ObjectCache: &loansTestCache{values: map[string]string{}}}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.Exec(`INSERT INTO countries (name, current_season) VALUES ('tl', 1) ON CONFLICT (name) DO NOTHING`)
	must(err)
	_, err = db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`)
	must(err)
	_, err = db.Exec(`UPDATE season SET transfer_window_open = true`)
	must(err)

	tag := strconv.FormatInt(time.Now().UnixNano(), 10)
	r := &counterHandlerRig{t: t, db: db, dbCtx: dbCtx}
	seed := func(name string) (int64, int64) {
		var leagueId, teamId, userId int64
		must(db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ('tl', 1, '1A', 1) RETURNING league_id`).Scan(&leagueId))
		must(db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ($1, 'dev', $2) RETURNING user_id`, name, name+"-"+tag+"@example.test").Scan(&userId))
		must(db.QueryRow(`INSERT INTO teams (name, country, league_id, budget, owner_id) VALUES ($1, 'tl', $2, 50000000, $3) RETURNING team_id`,
			name+" "+tag, leagueId, userId).Scan(&teamId))
		return teamId, userId
	}
	r.seller, r.sellerUser = seed("Counter Handler Seller")
	r.buyer, r.buyerUser = seed("Counter Handler Buyer")
	must(db.QueryRow(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield,
		dribbling, shot, "header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country, shirt_number)
		VALUES ($1, 'Test', 'Player', 0, 40, 40, 40, 58, 58, 40, 40, false, false, false, false, '', 1000, 26, 3, 'tl', 1)
		RETURNING player_id`, r.seller).Scan(&r.player))

	gin.SetMode(gin.TestMode)
	r.router = gin.New()
	r.router.Use(func(c *gin.Context) {
		c.Set("JWT_PAYLOAD", ginjwt.MapClaims{common.UserId: strconv.FormatInt(r.as, 10)})
	})
	r.router.POST("/offers/:"+ID+"/counter", CounterOfferHandler(dbCtx))
	r.router.POST("/offers/:"+ID+"/accept-counter", AcceptCounterOfferHandler(dbCtx))
	r.router.POST("/offers/:"+ID+"/decline-counter", DeclineCounterOfferHandler(dbCtx))
	return r
}

func (r *counterHandlerRig) offer() int64 {
	r.t.Helper()
	var offerId int64
	if err := r.db.QueryRow(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount, offered_wage, offered_years)
		VALUES ($1, $2, $3, 800000, 1500, 2) RETURNING offer_id`, r.player, r.buyer, r.seller).Scan(&offerId); err != nil {
		r.t.Fatalf("seeding offer: %s", err)
	}
	return offerId
}

func (r *counterHandlerRig) post(as int64, path string, body string) int {
	r.t.Helper()
	r.as = as
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.router.ServeHTTP(rec, req)
	return rec.Code
}

func TestCounterOfferHandlers_OnlyTheSellerCounters(t *testing.T) {
	r := newCounterHandlerRig(t)
	offerId := r.offer()
	path := "/offers/" + strconv.FormatInt(offerId, 10) + "/counter"

	if code := r.post(r.buyerUser, path, `{"amount": 1000000}`); code != http.StatusForbidden {
		t.Errorf("buyer counter = %d, want 403", code)
	}
	if code := r.post(r.sellerUser, path, `{"amount": 800000}`); code != http.StatusBadRequest {
		t.Errorf("counter not above the offer = %d, want 400", code)
	}
	if code := r.post(r.sellerUser, path, `{"amount": 1000000}`); code != http.StatusOK {
		t.Errorf("seller counter = %d, want 200", code)
	}
	if code := r.post(r.sellerUser, path, `{"amount": 1100000}`); code != http.StatusConflict {
		t.Errorf("second counter = %d, want 409", code)
	}
}

func TestCounterOfferHandlers_OnlyTheBuyerAnswers(t *testing.T) {
	r := newCounterHandlerRig(t)
	offerId := r.offer()
	base := "/offers/" + strconv.FormatInt(offerId, 10)
	if code := r.post(r.sellerUser, base+"/counter", `{"amount": 1000000}`); code != http.StatusOK {
		t.Fatalf("seller counter = %d, want 200", code)
	}

	if code := r.post(r.sellerUser, base+"/accept-counter", ``); code != http.StatusForbidden {
		t.Errorf("seller accept-counter = %d, want 403", code)
	}
	if code := r.post(r.sellerUser, base+"/decline-counter", ``); code != http.StatusForbidden {
		t.Errorf("seller decline-counter = %d, want 403", code)
	}
	if code := r.post(r.buyerUser, base+"/accept-counter", ``); code != http.StatusOK {
		t.Errorf("buyer accept-counter = %d, want 200", code)
	}
	if code := r.post(r.buyerUser, base+"/decline-counter", ``); code != http.StatusNotFound {
		t.Errorf("decline after the transfer = %d, want 404", code)
	}

	second := r.offerFromSellerToBuyer()
	base = "/offers/" + strconv.FormatInt(second, 10)
	if code := r.post(r.sellerUser, base+"/decline-counter", ``); code != http.StatusConflict {
		t.Errorf("decline without a counter = %d, want 409", code)
	}
	if code := r.post(r.sellerUser, base+"/counter", `{"amount": 1000000}`); code != http.StatusForbidden {
		t.Errorf("counter by the buyer of the new offer = %d, want 403", code)
	}
	if code := r.post(r.buyerUser, base+"/counter", `{"amount": 1000000}`); code != http.StatusOK {
		t.Errorf("counter by the new owner = %d, want 200", code)
	}
	if code := r.post(r.sellerUser, base+"/decline-counter", ``); code != http.StatusOK {
		t.Errorf("decline-counter by its buyer = %d, want 200", code)
	}
}

// offerFromSellerToBuyer opens an offer from the original seller for the player, who now belongs
// to the buyer.
func (r *counterHandlerRig) offerFromSellerToBuyer() int64 {
	r.t.Helper()
	var offerId int64
	if err := r.db.QueryRow(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount, offered_wage, offered_years)
		VALUES ($1, $2, $3, 800000, 1500, 2) RETURNING offer_id`, r.player, r.seller, r.buyer).Scan(&offerId); err != nil {
		r.t.Fatalf("seeding offer: %s", err)
	}
	return offerId
}
