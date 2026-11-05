package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "Youth minutes when nothing is at stake": a Director's league kickoff gets
// youth starts only with nothing at stake - "Still something to play for" once a tier below makes
// its last place a relegation fight - and never in a cup tie or for a Manager.
func TestAiPlanOptions_YouthStartsOnlyWithNothingAtStake(t *testing.T) {
	const country = "zy"
	w := newLoanWorld(t, country)
	// A tier-2 league left by an earlier run of this test would already make it a relegation fight.
	w.exec(`DELETE FROM leagues WHERE country = $1 AND tier = 2`, country)
	director := w.team(country, "Mid-table", common.AiLevelDirector, nil, 1_000_000)
	manager := w.team(country, "Also mid-table", common.AiLevelManager, nil, 1_000_000)
	leader := w.team(country, "Champions", common.AiLevelDirector, nil, 1_000_000)
	w.exec(`DELETE FROM classification WHERE league_id = $1`, w.league)
	for team, points := range map[int64]int{leader: 70, director: 30, manager: 30} {
		w.exec(`INSERT INTO classification (league_id, team_id, season, matches, points) VALUES ($1, $2, 1, 28, $3)`, w.league, team, points)
	}

	if got := aiPlanOptions(w.dbCtx, kickoffLeague, w.teamOf(director)); got.YouthStarts != aiDirectorYouthStarts {
		t.Errorf("Director with nothing at stake: %+v, want %d youth starts", got, aiDirectorYouthStarts)
	}
	if got := aiPlanOptions(w.dbCtx, kickoffCup, w.teamOf(director)); got.YouthStarts != 0 {
		t.Errorf("cup tie: %+v, want none", got)
	}
	if got := aiPlanOptions(w.dbCtx, kickoffLeague, w.teamOf(manager)); got.YouthStarts != 0 {
		t.Errorf("Manager: %+v, want none", got)
	}

	// A tier below puts the bottom places in a relegation fight.
	w.scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L2', 2) RETURNING league_id`, country)
	if got := aiPlanOptions(w.dbCtx, kickoffLeague, w.teamOf(director)); got.YouthStarts != 0 {
		t.Errorf("Director in a relegation fight: %+v, want its usual XI", got)
	}
}
