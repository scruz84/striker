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

// Every lineup endpoint takes and gives back the lineup roles (openspec
// add-captain-and-set-piece-takers "Lineup roles"); a goalkeeper may be captain but never a
// set-piece taker. Needs STRIKER_TEST_DB_DSN.

func (r *friendlySubsRig) putRoles(path string, roles common.LineupRoles) (int, string) {
	r.t.Helper()
	body, _ := json.Marshal(SetLineupRequest{Formation: r.lineup.Formation, Positions: r.lineup.Positions, LineupRoles: roles})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// fixtureLineupPaths registers the per-fixture lineup endpoints on the rig and returns their
// paths for one upcoming league match, friendly and cup tie of its team.
func (r *friendlySubsRig) fixtureLineupPaths(name string) []string {
	r.t.Helper()
	t := r.t
	var opponent, matchId, friendlyId, cupId, tieId int64
	if err := r.db.QueryRow(`INSERT INTO teams (name, country, league_id, budget) SELECT $2, country, league_id, 0 FROM teams WHERE team_id = $1 RETURNING team_id`,
		r.teamId, name).Scan(&opponent); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		SELECT league_id, 1, 1, now() + interval '3 days', $1, $2 FROM teams WHERE team_id = $1 RETURNING match_id`, r.teamId, opponent).Scan(&matchId); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '3 days') RETURNING match_id`,
		r.teamId, opponent).Scan(&friendlyId); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow(`INSERT INTO cups (country, season, name) SELECT country, 1, $2 FROM teams WHERE team_id = $1 RETURNING cup_id`, r.teamId, name+" Cup").Scan(&cupId); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date) VALUES ($1, 1, 1, $2, $3, now() + interval '3 days') RETURNING tie_id`,
		cupId, r.teamId, opponent).Scan(&tieId); err != nil {
		t.Fatal(err)
	}
	r.router.PUT("/match/:"+ID+"/lineup", SetMatchLineupHandler(r.dbCtx))
	r.router.GET("/match/:"+ID+"/lineup", GetMatchLineupHandler(r.dbCtx))
	r.router.PUT("/friendly/:"+ID+"/lineup", SetFriendlyLineupHandler(r.dbCtx))
	r.router.PUT("/ties/:"+ID+"/lineup", SetCupTieLineupHandler(r.dbCtx))
	r.router.GET("/ties/:"+ID+"/lineup", GetCupTieLineupHandler(r.dbCtx))
	return []string{
		"/match/" + strconv.FormatInt(matchId, 10) + "/lineup",
		"/friendly/" + strconv.FormatInt(friendlyId, 10) + "/lineup",
		"/ties/" + strconv.FormatInt(tieId, 10) + "/lineup",
	}
}

func TestLineupHandlers_Roles(t *testing.T) {
	r := newFriendlySubsRig(t)
	var keeper int64
	for _, p := range r.lineup.Positions {
		if p.Position == common.GOALKEEPER {
			keeper = p.PlayerId
		}
	}
	ptr := func(v int64) *int64 { return &v }
	// A starter as captain, another as free-kick taker, a bench player on penalties.
	roles := common.LineupRoles{CaptainId: ptr(r.starter[0]), PenaltyTakerId: ptr(r.bench[0]), FreeKickTakerId: ptr(r.starter[1])}

	paths := append([]string{"/lineup", "/lineup/friendly"}, r.fixtureLineupPaths("Roles Opponent")...)
	for _, path := range paths {
		if code, body := r.putRoles(path, roles); code != http.StatusOK {
			t.Fatalf("%s: save roles = %d: %s", path, code, body)
		}
		got := r.get(path).LineupRoles
		if got.CaptainId == nil || *got.CaptainId != r.starter[0] || got.PenaltyTakerId == nil || *got.PenaltyTakerId != r.bench[0] ||
			got.FreeKickTakerId == nil || *got.FreeKickTakerId != r.starter[1] {
			t.Errorf("%s: stored roles = %+v", path, got)
		}

		// Goalkeeper as taker: rejected. As captain: fine.
		if code, _ := r.putRoles(path, common.LineupRoles{PenaltyTakerId: ptr(keeper)}); code != http.StatusBadRequest {
			t.Errorf("%s: goalkeeper as penalty taker = %d, want 400", path, code)
		}
		if code, _ := r.putRoles(path, common.LineupRoles{FreeKickTakerId: ptr(keeper)}); code != http.StatusBadRequest {
			t.Errorf("%s: goalkeeper as free-kick taker = %d, want 400", path, code)
		}
		if code, _ := r.putRoles(path, common.LineupRoles{CaptainId: ptr(999999999)}); code != http.StatusBadRequest {
			t.Errorf("%s: captain from outside the squad = %d, want 400", path, code)
		}
		if code, body := r.putRoles(path, common.LineupRoles{CaptainId: ptr(keeper)}); code != http.StatusOK {
			t.Errorf("%s: goalkeeper as captain = %d: %s", path, code, body)
		}

		// Saving without roles leaves all three automatic.
		if code, _ := r.putRoles(path, common.LineupRoles{}); code != http.StatusOK {
			t.Fatal(code)
		}
		if got := r.get(path).LineupRoles; got.CaptainId != nil || got.PenaltyTakerId != nil || got.FreeKickTakerId != nil {
			t.Errorf("%s: roles after a save without any = %+v", path, got)
		}
	}
}
