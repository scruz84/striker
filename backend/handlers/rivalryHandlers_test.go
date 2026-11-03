package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
)

// The rivals endpoint and the fixtures' rivalry level (openspec add-emergent-rivalries "Rivalry
// visibility"). Needs STRIKER_TEST_DB_DSN.
func TestRivalryHandlers_TeamRivalsAndFixtureLevels(t *testing.T) {
	dbCtx, db, team, _, _ := projectionTeam(t)
	newTeam := func(name string) int64 {
		var id int64
		if err := db.QueryRow(`INSERT INTO teams (name, country, league_id) SELECT $2, country, league_id FROM teams WHERE team_id = $1 RETURNING team_id`, team, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	clasico, rival, minor := newTeam("RV Clasico"), newTeam("RV Rival"), newTeam("RV Minor")
	for other, score := range map[int64]float64{clasico: 22, rival: 9, minor: 5} {
		if _, err := db.Exec(`INSERT INTO team_rivalries (team_low_id, team_high_id, country, score) SELECT LEAST($1::bigint, $2::bigint), GREATEST($1::bigint, $2::bigint), country, $3 FROM teams WHERE team_id = $1`,
			team, other, score); err != nil {
			t.Fatal(err)
		}
	}
	// A played match against the Rival, stored as played with no level, and an upcoming one.
	var played, upcoming int64
	if err := db.QueryRow(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played, rivalry_level)
		SELECT league_id, 1, 1, now() - interval '7 days', $1, $2, true, 0 FROM teams WHERE team_id = $1 RETURNING match_id`, team, rival).Scan(&played); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		SELECT league_id, 1, 2, now() + interval '7 days', $2, $1 FROM teams WHERE team_id = $1 RETURNING match_id`, team, clasico).Scan(&upcoming); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/team/:"+ID+"/rivalries", TeamRivalriesHandler(dbCtx))
	router.GET("/team/:"+ID+"/matches", TeamMatchesHandler(dbCtx))
	id := strconv.FormatInt(team, 10)

	rec := serve(router, http.MethodGet, "/team/"+id+"/rivalries")
	var rivals []common.TeamRivalry
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &rivals) != nil {
		t.Fatalf("GET rivalries = %d: %s", rec.Code, rec.Body.String())
	}
	if len(rivals) != 2 || rivals[0].TeamId != clasico || rivals[0].Level != common.RIVALRY_CLASICO || rivals[1].TeamId != rival || rivals[1].Level != common.RIVALRY_RIVAL {
		t.Errorf("rivals = %+v, want the Clásico at 22 then the Rival at 9", rivals)
	}

	rec = serve(router, http.MethodGet, "/team/"+id+"/matches")
	var matches []common.TeamMatchSummary
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &matches) != nil {
		t.Fatalf("GET matches = %d: %s", rec.Code, rec.Body.String())
	}
	levels := map[int64]int{}
	for _, m := range matches {
		levels[m.MatchId] = m.Rivalry
	}
	if levels[played] != common.RIVALRY_NONE {
		t.Errorf("played match shows rivalry %d, want its stored none", levels[played])
	}
	if levels[upcoming] != common.RIVALRY_CLASICO {
		t.Errorf("upcoming match shows rivalry %d, want the current Clásico", levels[upcoming])
	}
}
