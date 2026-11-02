package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

type previewRig struct {
	t          *testing.T
	db         *sql.DB
	dbCtx      *common.DatabaseContext
	leagueId   int64
	ownerTeam  int64
	opponent   int64
	router     func(userId int64) *gin.Engine
	ownerUser  int64
	outsider   int64
	opponentXI []common.LineupPosition
}

// newPreviewRig seeds an owner club (default lineup saved), an opponent club that already played
// a finished league match with a stored kickoff lineup, and an outsider with his own club. Needs
// STRIKER_TEST_DB_DSN (skipped when unset).
func newPreviewRig(t *testing.T) *previewRig {
	t.Helper()
	dbCtx, db, ownerTeam, _, _ := projectionTeam(t)
	r := &previewRig{t: t, db: db, dbCtx: dbCtx, ownerTeam: ownerTeam}
	if err := db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, ownerTeam).Scan(&r.leagueId); err != nil {
		t.Fatal(err)
	}
	var country string
	if err := db.QueryRow(`SELECT country FROM teams WHERE team_id = $1`, ownerTeam).Scan(&country); err != nil {
		t.Fatal(err)
	}

	seedClub := func(name string) int64 {
		var id int64
		if err := db.QueryRow(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, name, country, r.leagueId).Scan(&id); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 14; i++ {
			keeper := 10
			if i == 0 {
				keeper = 70
			}
			if _, err := db.Exec(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield, dribbling, shot,
				header, quick, creative, technical, avatar, age, shirt_number, stamina, form)
				VALUES ($1, 'O', $2, 0, $3, 55, 55, 55, 55, 55, 55, false, false, false, false, '{}', 25, $4, 80, 50)`,
				id, fmt.Sprintf("%s%d", name, i), keeper, i+1); err != nil {
				t.Fatal(err)
			}
		}
		return id
	}
	r.opponent = seedClub(country + " Opp")
	outsiderTeam := seedClub(country + " Out")

	seedUser := func(label string, teamId int64) int64 {
		var id int64
		if err := db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ($1, 'dev', $2) RETURNING user_id`,
			label, label+"-"+strconv.FormatInt(teamId, 10)+"@example.test").Scan(&id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, id, teamId); err != nil {
			t.Fatal(err)
		}
		return id
	}
	r.ownerUser = seedUser("pvowner", ownerTeam)
	r.outsider = seedUser("pvout", outsiderTeam)

	best, err := common.BestLineupForTeam(dbCtx, r.opponent)
	if err != nil {
		t.Fatal(err)
	}
	r.opponentXI = best.Positions
	past := r.insertLeagueMatch(-48*time.Hour, true, r.opponent, outsiderTeam)
	if err := common.SetLineupForMatch(dbCtx, past, r.opponent, best.Formation, best.Positions, nil); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	r.router = func(userId int64) *gin.Engine {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("JWT_PAYLOAD", ginjwt.MapClaims{common.UserId: strconv.FormatInt(userId, 10)})
		})
		router.GET("/preview/:"+KindParam+"/:"+ID, OpponentPreviewHandler(dbCtx))
		return router
	}
	return r
}

func (r *previewRig) insertLeagueMatch(offset time.Duration, played bool, home int64, away int64) int64 {
	r.t.Helper()
	var id int64
	if err := r.db.QueryRow(`INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id, home_goals, away_goals)
		VALUES ($1, 1, $2, $3, $4, $5, 1, 0) RETURNING match_id`, r.leagueId, played, time.Now().Add(offset), home, away).Scan(&id); err != nil {
		r.t.Fatal(err)
	}
	return id
}

func (r *previewRig) get(userId int64, kind string, id int64) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.router(userId).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/preview/"+kind+"/"+strconv.FormatInt(id, 10), nil))
	return rec
}

func TestOpponentPreviewHandler_ParticipantGetsTheFullPreview(t *testing.T) {
	r := newPreviewRig(t)
	upcoming := r.insertLeagueMatch(24*time.Hour, false, r.ownerTeam, r.opponent)

	rec := r.get(r.ownerUser, common.MatchKindLeague, upcoming)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Opponent struct {
			Id   int64 `json:"id"`
			Form []any `json:"form"`
		} `json:"opponent"`
		Likely     *struct{ Players []map[string]any } `json:"likely"`
		Projection *struct {
			Own struct{ Midfield float64 } `json:"own"`
		} `json:"projection"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Opponent.Id != r.opponent || len(body.Opponent.Form) != 1 {
		t.Fatalf("opponent = %+v", body.Opponent)
	}
	if body.Likely == nil || len(body.Likely.Players) != len(r.opponentXI) {
		t.Fatalf("likely = %+v", body.Likely)
	}
	for _, p := range body.Likely.Players {
		for _, forbidden := range []string{"goalKeeper", "defense", "midfield", "shot", "lateral"} {
			if _, leaked := p[forbidden]; leaked {
				t.Fatalf("the likely XI leaks the %q skill: %v", forbidden, p)
			}
		}
	}
	if body.Projection == nil || body.Projection.Own.Midfield <= 0 {
		t.Fatalf("projection = %+v", body.Projection)
	}
}

func TestOpponentPreviewHandler_NonParticipantIsRefused(t *testing.T) {
	r := newPreviewRig(t)
	upcoming := r.insertLeagueMatch(24*time.Hour, false, r.ownerTeam, r.opponent)

	if rec := r.get(r.outsider, common.MatchKindLeague, upcoming); rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", rec.Code, rec.Body.String())
	}
}

func TestOpponentPreviewHandler_LockedFixtureIsRefused(t *testing.T) {
	r := newPreviewRig(t)
	locked := r.insertLeagueMatch(10*time.Minute, false, r.ownerTeam, r.opponent)

	if rec := r.get(r.ownerUser, common.MatchKindLeague, locked); rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestOpponentPreviewHandler_UnknownKindIsABadRequest(t *testing.T) {
	r := newPreviewRig(t)
	upcoming := r.insertLeagueMatch(24*time.Hour, false, r.ownerTeam, r.opponent)

	if rec := r.get(r.ownerUser, "tournament", upcoming); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestOpponentPreviewHandler_ProjectionIncludesTheHomeAdvantage(t *testing.T) {
	r := newPreviewRig(t)
	atHome := r.insertLeagueMatch(24*time.Hour, false, r.ownerTeam, r.opponent)
	away := r.insertLeagueMatch(25*time.Hour, false, r.opponent, r.ownerTeam)

	midfield := func(id int64) (own float64, opponent float64) {
		rec := r.get(r.ownerUser, common.MatchKindLeague, id)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		var body struct {
			Projection struct {
				Own      struct{ Midfield float64 } `json:"own"`
				Opponent struct{ Midfield float64 } `json:"opponent"`
			} `json:"projection"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Projection.Own.Midfield, body.Projection.Opponent.Midfield
	}
	ownHome, opponentVisiting := midfield(atHome)
	ownAway, opponentHosting := midfield(away)

	if ownHome <= ownAway || opponentHosting <= opponentVisiting {
		t.Fatalf("the home side must rate higher: own %v at home vs %v away, opponent %v hosting vs %v visiting", ownHome, ownAway, opponentHosting, opponentVisiting)
	}
}
