package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

// friendlySubsRig is a team with an owner and its default (league/cup) lineup, with the lineup
// routes behind a fake login. Needs STRIKER_TEST_DB_DSN (skipped when unset).
type friendlySubsRig struct {
	t       *testing.T
	dbCtx   *common.DatabaseContext
	db      *sql.DB
	teamId  int64
	lineup  *common.Lineup
	starter [3]int64 // outfield starters
	bench   [3]int64 // players not in the XI
	router  *gin.Engine
}

func newFriendlySubsRig(t *testing.T) *friendlySubsRig {
	t.Helper()
	dbCtx, db, teamId, lineup, _ := projectionTeam(t)
	var userId int64
	if err := db.QueryRow(`INSERT INTO users (name, provider, email) VALUES ('fs', 'dev', $1) RETURNING user_id`,
		"fs-"+strconv.FormatInt(teamId, 10)+"@example.test").Scan(&userId); err != nil {
		t.Fatalf("seeding user: %s", err)
	}
	if _, err := db.Exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, userId, teamId); err != nil {
		t.Fatalf("setting owner: %s", err)
	}
	r := &friendlySubsRig{t: t, dbCtx: dbCtx, db: db, teamId: teamId, lineup: lineup}

	started := map[int64]bool{}
	n := 0
	for _, p := range lineup.Positions {
		started[p.PlayerId] = true
		if p.Position != common.GOALKEEPER && n < 3 {
			r.starter[n] = p.PlayerId
			n++
		}
	}
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		t.Fatalf("GetPlayersFromTeamId: %s", err)
	}
	n = 0
	for _, p := range *players {
		if !started[p.Id] && n < 3 {
			r.bench[n] = p.Id
			n++
		}
	}
	if n < 3 {
		t.Fatalf("fixture needs 3 bench players, found %d", n)
	}

	gin.SetMode(gin.TestMode)
	r.router = gin.New()
	r.router.Use(func(c *gin.Context) {
		c.Set("JWT_PAYLOAD", ginjwt.MapClaims{common.UserId: strconv.FormatInt(userId, 10)})
	})
	r.router.GET("/lineup", GetLineupHandler(dbCtx))
	r.router.PUT("/lineup", SetLineupHandler(dbCtx))
	r.router.GET("/lineup/friendly", GetFriendlyDefaultLineupHandler(dbCtx))
	r.router.PUT("/lineup/friendly", SetFriendlyDefaultLineupHandler(dbCtx))
	r.router.GET("/friendly/:"+ID+"/lineup", GetFriendlyLineupHandler(dbCtx))
	return r
}

func (r *friendlySubsRig) put(path string, subs []common.LineupSubstitution) int {
	r.t.Helper()
	body, _ := json.Marshal(SetLineupRequest{Formation: r.lineup.Formation, Positions: r.lineup.Positions, Substitutions: subs})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.router.ServeHTTP(rec, req)
	return rec.Code
}

func (r *friendlySubsRig) get(path string) common.Lineup {
	r.t.Helper()
	rec := httptest.NewRecorder()
	r.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		r.t.Fatalf("GET %s = %d: %s", path, rec.Code, rec.Body.String())
	}
	var l common.Lineup
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		r.t.Fatalf("decoding %s: %s", path, err)
	}
	return l
}

func (r *friendlySubsRig) sub(out, in int64, minute int) common.LineupSubstitution {
	return common.LineupSubstitution{PlayerOutId: out, PlayerInId: in, Minute: minute}
}

func TestFriendlyDefaultLineup_SubstitutionsAreIndependentOfLeagueCup(t *testing.T) {
	r := newFriendlySubsRig(t)
	leagueSubs := []common.LineupSubstitution{r.sub(r.starter[0], r.bench[0], 55), r.sub(r.starter[1], r.bench[1], 65)}
	friendlySubs := []common.LineupSubstitution{r.sub(r.starter[2], r.bench[2], 60)}

	if code := r.put("/lineup", leagueSubs); code != http.StatusOK {
		t.Fatalf("league/cup save = %d", code)
	}
	if got := r.get("/lineup/friendly").Substitutions; len(got) != 0 {
		t.Fatalf("friendly set after the first league/cup save = %v, want empty", got)
	}

	// Friendly save with 1 substitution leaves the league/cup set unchanged.
	if code := r.put("/lineup/friendly", friendlySubs); code != http.StatusOK {
		t.Fatalf("friendly save = %d", code)
	}
	if got := r.get("/lineup/friendly").Substitutions; !reflect.DeepEqual(got, friendlySubs) {
		t.Fatalf("friendly set = %v, want %v", got, friendlySubs)
	}
	if got := r.get("/lineup").Substitutions; !reflect.DeepEqual(got, leagueSubs) {
		t.Fatalf("league/cup set = %v, want it untouched %v", got, leagueSubs)
	}

	// ...and a league/cup save leaves the friendly set unchanged.
	three := []common.LineupSubstitution{r.sub(r.starter[0], r.bench[0], 50), r.sub(r.starter[1], r.bench[1], 60), r.sub(r.starter[2], r.bench[2], 70)}
	if code := r.put("/lineup", three); code != http.StatusOK {
		t.Fatalf("league/cup save = %d", code)
	}
	if got := r.get("/lineup/friendly").Substitutions; !reflect.DeepEqual(got, friendlySubs) {
		t.Fatalf("friendly set after a league/cup save = %v, want %v", got, friendlySubs)
	}

	// An empty list clears only the friendly set.
	if code := r.put("/lineup/friendly", []common.LineupSubstitution{}); code != http.StatusOK {
		t.Fatalf("clearing save = %d", code)
	}
	if got := r.get("/lineup/friendly").Substitutions; len(got) != 0 {
		t.Fatalf("friendly set after clearing = %v, want empty", got)
	}
	if got := r.get("/lineup").Substitutions; !reflect.DeepEqual(got, three) {
		t.Fatalf("league/cup set after clearing the friendly one = %v, want %v", got, three)
	}
}

func TestFriendlyDefaultLineup_SubstitutionValidation(t *testing.T) {
	r := newFriendlySubsRig(t)
	stored := []common.LineupSubstitution{r.sub(r.starter[0], r.bench[0], 60)}
	if code := r.put("/lineup/friendly", stored); code != http.StatusOK {
		t.Fatalf("friendly save = %d", code)
	}

	// A suspended (uninjured) bench player may come on in a friendly, though not in a league match.
	if _, err := r.db.Exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 2)`,
		r.bench[1], common.UNAVAILABLE_SUSPENSION); err != nil {
		t.Fatalf("suspending: %s", err)
	}
	suspended := []common.LineupSubstitution{r.sub(r.starter[1], r.bench[1], 70)}
	if code := r.put("/lineup/friendly", suspended); code != http.StatusOK {
		t.Fatalf("suspended incoming player in a friendly = %d, want 200", code)
	}
	if code := r.put("/lineup", suspended); code != http.StatusBadRequest {
		t.Fatalf("suspended incoming player in a league save = %d, want 400", code)
	}

	// An injured incoming player and a fourth substitution are rejected and store nothing.
	if _, err := r.db.Exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 3)`,
		r.bench[2], common.UNAVAILABLE_INJURY); err != nil {
		t.Fatalf("injuring: %s", err)
	}
	if code := r.put("/lineup/friendly", []common.LineupSubstitution{r.sub(r.starter[2], r.bench[2], 80)}); code != http.StatusBadRequest {
		t.Fatalf("injured incoming player = %d, want 400", code)
	}
	four := []common.LineupSubstitution{
		r.sub(r.starter[0], r.bench[0], 50), r.sub(r.starter[1], r.bench[1], 60),
		r.sub(r.starter[2], r.bench[0], 70), r.sub(r.starter[0], r.bench[1], 80),
	}
	if code := r.put("/lineup/friendly", four); code != http.StatusBadRequest {
		t.Fatalf("four substitutions = %d, want 400", code)
	}
	if got := r.get("/lineup/friendly").Substitutions; !reflect.DeepEqual(got, suspended) {
		t.Fatalf("friendly set after rejected saves = %v, want the last accepted %v", got, suspended)
	}
}

// A friendly without an override opens on the friendly default's own substitutions, not the
// league/cup ones ("Opening a friendly without an override").
func TestGetFriendlyLineup_FallbackCarriesTheFriendlySubstitutions(t *testing.T) {
	r := newFriendlySubsRig(t)
	leagueSubs := []common.LineupSubstitution{r.sub(r.starter[0], r.bench[0], 55), r.sub(r.starter[1], r.bench[1], 65)}
	friendlySubs := []common.LineupSubstitution{r.sub(r.starter[2], r.bench[2], 60)}
	if code := r.put("/lineup", leagueSubs); code != http.StatusOK {
		t.Fatalf("league/cup save = %d", code)
	}
	if code := r.put("/lineup/friendly", friendlySubs); code != http.StatusOK {
		t.Fatalf("friendly save = %d", code)
	}
	var otherTeam, friendlyId int64
	if err := r.db.QueryRow(`INSERT INTO teams (name, country, league_id) SELECT name || ' B', country, league_id FROM teams WHERE team_id = $1 RETURNING team_id`, r.teamId).Scan(&otherTeam); err != nil {
		t.Fatalf("seeding opponent: %s", err)
	}
	if err := r.db.QueryRow(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '3 days') RETURNING match_id`,
		r.teamId, otherTeam).Scan(&friendlyId); err != nil {
		t.Fatalf("seeding friendly: %s", err)
	}

	got := r.get("/friendly/" + strconv.FormatInt(friendlyId, 10) + "/lineup")

	if !reflect.DeepEqual(got.Substitutions, friendlySubs) {
		t.Fatalf("per-friendly fallback substitutions = %v, want the friendly default's %v", got.Substitutions, friendlySubs)
	}
}
