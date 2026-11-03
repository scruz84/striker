package engine

import (
	"reflect"
	"testing"

	"github.com/scruz84/striker/common"
)

// friendlySubsFixture sets teamId's friendly default to its best XI and returns two distinct
// outfield starters and two bench players, so a test can schedule up to two substitutions.
func (e *simEnv) friendlySubsFixture(teamId int64) (starters [2]int64, bench [2]int64) {
	e.t.Helper()
	best, err := common.BestLineupForTeam(e.dbCtx, teamId)
	if err != nil {
		e.t.Fatalf("BestLineupForTeam: %s", err)
	}
	if err := common.SetTeamFriendlyFormation(e.dbCtx, teamId, best.Formation); err != nil {
		e.t.Fatalf("SetTeamFriendlyFormation: %s", err)
	}
	if err := common.SetDefaultFriendlyLineup(e.dbCtx, teamId, best.Positions); err != nil {
		e.t.Fatalf("SetDefaultFriendlyLineup: %s", err)
	}
	started := startersOfLineup(best)
	n := 0
	for _, p := range best.Positions {
		if p.Position != common.GOALKEEPER && n < 2 {
			starters[n] = p.PlayerId
			n++
		}
	}
	rows, err := e.db.Query(`SELECT player_id FROM players WHERE team_id = $1 AND goal_keeper < 50 ORDER BY player_id`, teamId)
	if err != nil {
		e.t.Fatalf("reading the bench: %s", err)
	}
	defer rows.Close()
	n = 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			e.t.Fatalf("scanning: %s", err)
		}
		if !started[id] && n < 2 {
			bench[n] = id
			n++
		}
	}
	if n < 2 {
		e.t.Fatalf("fixture needs 2 bench players, found %d", n)
	}
	return
}

func (e *simEnv) friendlyMatch() (*common.FriendlyMatch, int64) {
	e.t.Helper()
	friendlyId := e.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() - interval '3 hours') RETURNING match_id`, e.home, e.away)
	friendly, err := common.GetFriendlyMatchById(e.dbCtx, friendlyId)
	if err != nil {
		e.t.Fatalf("GetFriendlyMatchById: %s", err)
	}
	return friendly, friendlyId
}

func (e *simEnv) storedFriendlySubs(friendlyId, teamId int64) []common.LineupSubstitution {
	e.t.Helper()
	subs, err := common.GetFriendlyLineupSubstitutions(e.dbCtx, friendlyId, teamId)
	if err != nil {
		e.t.Fatalf("GetFriendlyLineupSubstitutions: %s", err)
	}
	return subs
}

// A friendly without an override plays the friendly default's own substitutions, and the stored
// snapshot lists them - never the league/cup set (spec team-lineups, "Friendly falls back to the
// friendly default").
func TestSimulateFriendly_FallsBackToTheFriendlyDefaultSubstitutions(t *testing.T) {
	e := newSimEnv(t, "fd1")
	starters, bench := e.friendlySubsFixture(e.home)
	friendlySubs := []common.LineupSubstitution{
		{PlayerOutId: starters[0], PlayerInId: bench[0], Minute: 60},
		{PlayerOutId: starters[1], PlayerInId: bench[1], Minute: 70},
	}
	leagueSubs := []common.LineupSubstitution{{PlayerOutId: starters[1], PlayerInId: bench[0], Minute: 80}}
	if err := common.SetDefaultFriendlyLineupSubstitutions(e.dbCtx, e.home, friendlySubs); err != nil {
		t.Fatalf("SetDefaultFriendlyLineupSubstitutions: %s", err)
	}
	if err := common.SetDefaultLineupSubstitutions(e.dbCtx, e.home, leagueSubs); err != nil {
		t.Fatalf("SetDefaultLineupSubstitutions: %s", err)
	}
	friendly, friendlyId := e.friendlyMatch()

	if err := simulateFriendly(e.dbCtx, friendly); err != nil {
		t.Fatalf("simulateFriendly: %s", err)
	}

	if got := e.storedFriendlySubs(friendlyId, e.home); !reflect.DeepEqual(got, friendlySubs) {
		t.Fatalf("snapshot substitutions = %v, want the friendly default's %v", got, friendlySubs)
	}
}

// A per-friendly override plays only its own substitutions.
func TestSimulateFriendly_OverridePlaysOnlyItsOwnSubstitutions(t *testing.T) {
	e := newSimEnv(t, "fd2")
	starters, bench := e.friendlySubsFixture(e.home)
	if err := common.SetDefaultFriendlyLineupSubstitutions(e.dbCtx, e.home, []common.LineupSubstitution{
		{PlayerOutId: starters[0], PlayerInId: bench[0], Minute: 60},
		{PlayerOutId: starters[1], PlayerInId: bench[1], Minute: 70},
	}); err != nil {
		t.Fatalf("SetDefaultFriendlyLineupSubstitutions: %s", err)
	}
	friendly, friendlyId := e.friendlyMatch()
	best, err := common.BestLineupForTeam(e.dbCtx, e.home)
	if err != nil {
		t.Fatalf("BestLineupForTeam: %s", err)
	}
	own := []common.LineupSubstitution{{PlayerOutId: starters[1], PlayerInId: bench[1], Minute: 75}}
	if err := common.SetFriendlyLineup(e.dbCtx, friendlyId, e.home, best.Formation, common.MENTALITY_BALANCED, common.LineupRoles{}, best.Positions, own, nil); err != nil {
		t.Fatalf("SetFriendlyLineup: %s", err)
	}

	if err := simulateFriendly(e.dbCtx, friendly); err != nil {
		t.Fatalf("simulateFriendly: %s", err)
	}

	if got := e.storedFriendlySubs(friendlyId, e.home); !reflect.DeepEqual(got, own) {
		t.Fatalf("snapshot substitutions = %v, want the override's %v", got, own)
	}
}

// With no friendly default the friendly starts the computed best XI with no substitutions, even if
// the team has league/cup ones ("Friendly with no friendly default").
func TestSimulateFriendly_BestXiFallbackPlaysNoSubstitutions(t *testing.T) {
	e := newSimEnv(t, "fd3")
	best, err := common.BestLineupForTeam(e.dbCtx, e.home)
	if err != nil {
		t.Fatalf("BestLineupForTeam: %s", err)
	}
	var out, in int64
	started := startersOfLineup(best)
	for _, p := range best.Positions {
		if p.Position != common.GOALKEEPER {
			out = p.PlayerId
			break
		}
	}
	e.db.QueryRow(`SELECT player_id FROM players WHERE team_id = $1 AND goal_keeper < 50 AND NOT (player_id = ANY($2)) LIMIT 1`, e.home, pqIds(started)).Scan(&in)
	if out == 0 || in == 0 {
		t.Fatalf("fixture: starter %d bench %d", out, in)
	}
	if err := common.SetDefaultLineupSubstitutions(e.dbCtx, e.home, []common.LineupSubstitution{
		{PlayerOutId: out, PlayerInId: in, Minute: 60},
		{PlayerOutId: out, PlayerInId: in, Minute: 70},
		{PlayerOutId: out, PlayerInId: in, Minute: 80},
	}); err != nil {
		t.Fatalf("SetDefaultLineupSubstitutions: %s", err)
	}
	friendly, friendlyId := e.friendlyMatch()

	if err := simulateFriendly(e.dbCtx, friendly); err != nil {
		t.Fatalf("simulateFriendly: %s", err)
	}

	if got := e.storedFriendlySubs(friendlyId, e.home); len(got) != 0 {
		t.Fatalf("expected no substitutions on the best-XI fallback, got %v", got)
	}
}

// A league match without an override plays the league/cup set and ignores the friendly one.
func TestLeagueMatchFallback_IgnoresTheFriendlyDefaultSubstitutions(t *testing.T) {
	e := newSimEnv(t, "fd4")
	starters, bench := e.friendlySubsFixture(e.home)
	leagueSubs := []common.LineupSubstitution{{PlayerOutId: starters[0], PlayerInId: bench[0], Minute: 65}}
	if err := common.SetDefaultLineupSubstitutions(e.dbCtx, e.home, leagueSubs); err != nil {
		t.Fatalf("SetDefaultLineupSubstitutions: %s", err)
	}
	if err := common.SetDefaultFriendlyLineupSubstitutions(e.dbCtx, e.home, []common.LineupSubstitution{
		{PlayerOutId: starters[0], PlayerInId: bench[0], Minute: 50},
		{PlayerOutId: starters[1], PlayerInId: bench[1], Minute: 60},
		{PlayerOutId: starters[1], PlayerInId: bench[0], Minute: 70},
	}); err != nil {
		t.Fatalf("SetDefaultFriendlyLineupSubstitutions: %s", err)
	}
	matchId := e.leagueMatch(0)

	got, err := common.GetSubstitutionsForMatch(e.dbCtx, matchId, e.home)
	if err != nil {
		t.Fatalf("GetSubstitutionsForMatch: %s", err)
	}
	if !reflect.DeepEqual(got, leagueSubs) {
		t.Fatalf("league substitutions = %v, want %v", got, leagueSubs)
	}
}

func pqIds(set map[int64]bool) []int64 {
	ids := make([]int64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}
