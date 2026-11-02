package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/scruz84/striker/common"
)

// The lineup endpoints accept and return a mentality (openspec add-team-mentality); a payload
// without it saves Balanced, and anything outside -1..1 is rejected. Needs STRIKER_TEST_DB_DSN.

func (r *friendlySubsRig) putRaw(path string, mentality *int) int {
	r.t.Helper()
	payload := map[string]any{"formation": r.lineup.Formation, "positions": r.lineup.Positions, "substitutions": []common.LineupSubstitution{}}
	if mentality != nil {
		payload["mentality"] = *mentality
	}
	body, _ := json.Marshal(payload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.router.ServeHTTP(rec, req)
	return rec.Code
}

func intPtr(v int) *int { return &v }

func TestLineupHandlers_Mentality(t *testing.T) {
	r := newFriendlySubsRig(t)
	var leagueId, opponent int64
	if err := r.db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, r.teamId).Scan(&leagueId); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow(`INSERT INTO teams (name, country, league_id, budget) SELECT 'Mentality Opponent', country, league_id, 0 FROM teams WHERE team_id = $1 RETURNING team_id`,
		r.teamId).Scan(&opponent); err != nil {
		t.Fatal(err)
	}
	var matchId int64
	if err := r.db.QueryRow(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, now() + interval '3 days', $2, $3) RETURNING match_id`, leagueId, r.teamId, opponent).Scan(&matchId); err != nil {
		t.Fatal(err)
	}
	matchPath := "/match/" + strconv.FormatInt(matchId, 10) + "/lineup"
	r.router.PUT("/match/:"+ID+"/lineup", SetMatchLineupHandler(r.dbCtx))
	r.router.GET("/match/:"+ID+"/lineup", GetMatchLineupHandler(r.dbCtx))

	var friendlyId, cupId, tieId int64
	if err := r.db.QueryRow(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '3 days') RETURNING match_id`,
		r.teamId, opponent).Scan(&friendlyId); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow(`INSERT INTO cups (country, season, name) SELECT country, 1, 'Mentality Cup' FROM teams WHERE team_id = $1 RETURNING cup_id`, r.teamId).Scan(&cupId); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date) VALUES ($1, 1, 1, $2, $3, now() + interval '3 days') RETURNING tie_id`,
		cupId, r.teamId, opponent).Scan(&tieId); err != nil {
		t.Fatal(err)
	}
	friendlyPath := "/friendly/" + strconv.FormatInt(friendlyId, 10) + "/lineup"
	tiePath := "/ties/" + strconv.FormatInt(tieId, 10) + "/lineup"
	r.router.PUT("/friendly/:"+ID+"/lineup", SetFriendlyLineupHandler(r.dbCtx))
	r.router.PUT("/ties/:"+ID+"/lineup", SetCupTieLineupHandler(r.dbCtx))
	r.router.GET("/ties/:"+ID+"/lineup", GetCupTieLineupHandler(r.dbCtx))

	for _, path := range []string{"/lineup", "/lineup/friendly", matchPath, friendlyPath, tiePath} {
		if code := r.putRaw(path, intPtr(common.MENTALITY_ATTACKING)); code != http.StatusOK {
			t.Fatalf("%s: save Attacking = %d", path, code)
		}
		if got := r.get(path).Mentality; got != common.MENTALITY_ATTACKING {
			t.Errorf("%s: mentality = %d, want Attacking", path, got)
		}
		if code := r.putRaw(path, nil); code != http.StatusOK {
			t.Fatalf("%s: save without a mentality = %d", path, code)
		}
		if got := r.get(path).Mentality; got != common.MENTALITY_BALANCED {
			t.Errorf("%s: mentality after a save without one = %d, want Balanced", path, got)
		}
		if code := r.putRaw(path, intPtr(2)); code != http.StatusBadRequest {
			t.Errorf("%s: save mentality 2 = %d, want 400", path, code)
		}
	}

	// Each standing default keeps its own mentality.
	if code := r.putRaw("/lineup", intPtr(common.MENTALITY_DEFENSIVE)); code != http.StatusOK {
		t.Fatal(code)
	}
	if got := r.get("/lineup/friendly").Mentality; got != common.MENTALITY_BALANCED {
		t.Errorf("friendly default followed the league/cup default: %d", got)
	}
}
