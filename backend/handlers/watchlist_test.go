package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

// The watchlist endpoints act on the caller's own club only, and no public payload reveals who
// follows a player (openspec add-player-watchlist "Watchlist privacy"). Needs STRIKER_TEST_DB_DSN.

func watchlistRouter(dbCtx *common.DatabaseContext, userId int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("JWT_PAYLOAD", ginjwt.MapClaims{common.UserId: strconv.FormatInt(userId, 10)})
	})
	router.GET("/watchlist", GetWatchlistHandler(dbCtx))
	router.PUT("/watchlist/:"+ID, WatchPlayerHandler(dbCtx))
	router.DELETE("/watchlist/:"+ID, UnwatchPlayerHandler(dbCtx))
	router.GET("/team/:"+ID+"/", Team(dbCtx))
	router.GET("/player/:"+ID+"/", PlayerDetailsHandler(dbCtx))
	router.GET("/transfers", BrowseTransfersHandler(dbCtx))
	return router
}

func serve(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestWatchlistHandlers_OwnClubOnlyAndPrivate(t *testing.T) {
	dbCtx, db, sellerTeam, _, _ := projectionTeam(t)
	dbCtx.ObjectCache = &loansTestCache{values: map[string]string{}} // the team page caches names
	if _, err := db.Exec(`UPDATE season SET transfer_window_open = true`); err != nil {
		t.Fatal(err)
	}
	clubFor := func(name string) (teamId int64, userId int64) {
		t.Helper()
		if err := db.QueryRow(`INSERT INTO teams (name, country, league_id, budget) SELECT $2, country, league_id, 0 FROM teams WHERE team_id = $1 RETURNING team_id`,
			sellerTeam, name).Scan(&teamId); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ($1, 'dev', $2) RETURNING user_id`,
			name, name+strconv.FormatInt(teamId, 10)+"@example.test").Scan(&userId); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, userId, teamId); err != nil {
			t.Fatal(err)
		}
		return teamId, userId
	}
	_, aliceUser := clubFor("WL Alice")
	_, bobUser := clubFor("WL Bob")
	var sellerUser int64
	if err := db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ('WL Seller', 'dev', $1) RETURNING user_id`,
		"wl-seller"+strconv.FormatInt(sellerTeam, 10)+"@example.test").Scan(&sellerUser); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, sellerUser, sellerTeam); err != nil {
		t.Fatal(err)
	}
	players, err := common.GetPlayersFromTeamId(dbCtx, sellerTeam)
	if err != nil || len(*players) < 2 {
		t.Fatalf("seller squad: %v", err)
	}
	target := (*players)[1].Id
	path := "/watchlist/" + strconv.FormatInt(target, 10)
	alice, bob, seller := watchlistRouter(dbCtx, aliceUser), watchlistRouter(dbCtx, bobUser), watchlistRouter(dbCtx, sellerUser)

	if rec := serve(alice, http.MethodPut, path); rec.Code != http.StatusOK {
		t.Fatalf("Alice follows = %d: %s", rec.Code, rec.Body.String())
	}
	list := func(router *gin.Engine) []common.WatchlistEntry {
		t.Helper()
		rec := serve(router, http.MethodGet, "/watchlist")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /watchlist = %d", rec.Code)
		}
		var entries []common.WatchlistEntry
		if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
			t.Fatal(err)
		}
		return entries
	}
	if got := list(alice); len(got) != 1 || got[0].Player.Id != target {
		t.Errorf("Alice's watchlist = %+v", got)
	}
	if got := list(bob); len(got) != 0 {
		t.Errorf("Bob sees %d followed players, want none", len(got))
	}
	// Bob unfollowing never touches Alice's list.
	if rec := serve(bob, http.MethodDelete, path); rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
	if got := list(alice); len(got) != 1 {
		t.Errorf("Bob's unfollow changed Alice's list: %+v", got)
	}
	// The seller can't follow his own player.
	if rec := serve(seller, http.MethodPut, path); rec.Code != http.StatusBadRequest {
		t.Errorf("seller following his own player = %d, want 400", rec.Code)
	}
	if rec := serve(alice, http.MethodPut, "/watchlist/999999999"); rec.Code != http.StatusNotFound {
		t.Errorf("following an unknown player = %d, want 404", rec.Code)
	}

	// Selling club: nothing the seller can see says the player is followed.
	if err := common.CreateTransferListing(dbCtx, target, sellerTeam, 250000); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"/team/" + strconv.FormatInt(sellerTeam, 10) + "/", "/player/" + strconv.FormatInt(target, 10) + "/", "/transfers"} {
		rec := serve(seller, http.MethodGet, p)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d", p, rec.Code)
		}
		body := strings.ToLower(rec.Body.String())
		for _, word := range []string{"watch", "follow"} {
			if strings.Contains(body, word) {
				t.Errorf("GET %s mentions %q", p, word)
			}
		}
	}
	rows, err := db.Query(`SELECT message, COALESCE(message_key, '') FROM notifications WHERE user_id = $1`, sellerUser)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var msg, key string
		if err := rows.Scan(&msg, &key); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(msg+key), "watch") {
			t.Errorf("the seller got a watchlist notification: %s", msg)
		}
	}
}
