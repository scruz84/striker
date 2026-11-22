package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// A club whose saved lineup lost players who left it (their rows are deleted when they go) and
// that has no CPU plan at kickoff (here an unowned club with no AI level yet) used to kick off
// with whoever was left and forfeit 0-3 at minute 0 despite a full squad. The kickoff refill now
// tops the XI back up to its formation from the rest of the squad.
func TestSimulateMatch_ThinnedSavedLineupIsToppedUpAtKickoff(t *testing.T) {
	country := uniqueCountryCode()
	w := loanRolloverWorld(t, country)
	w.dbCtx.LiveHub = common.NewLiveMatchHub()
	club := w.team(country, "Thinned FC", 0, nil, 5_000_000)
	opponent := w.team(country, "Opponent FC", common.AiLevelManager, nil, 5_000_000)
	w.squad(country, club)
	w.player(country, club, common.STRIKER, 40, 22)
	w.squad(country, opponent)
	refreshAiTeamLineup(w.dbCtx, club, 0)
	w.exec(`DELETE FROM default_lineup_positions WHERE team_id = $1 AND player_id IN
		(SELECT player_id FROM default_lineup_positions WHERE team_id = $1 AND position <> $2 LIMIT 5)`, club, common.GOALKEEPER)
	if n := w.count(`SELECT count(*) FROM default_lineup_positions WHERE team_id = $1`, club); n != 6 {
		t.Fatalf("setup: expected a saved lineup of 6, got %d", n)
	}
	season, err := common.GetCurrentSeasonForCountry(w.dbCtx, country)
	if err != nil {
		t.Fatal(err)
	}
	matchId := w.scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, $2, 1, now() - interval '3 hours', $3, $4) RETURNING match_id`, w.league, season, club, opponent)
	match, err := common.WeekMatchById(w.dbCtx, matchId)
	if err != nil || match == nil {
		t.Fatalf("reading match %d: %v", matchId, err)
	}

	sim, err := simulateMatch(w.dbCtx, match)
	if err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}

	if got := len(sim.HomeLineup.Positions); got != 11 {
		t.Fatalf("expected the thinned club to kick off with 11, got %d", got)
	}
	for _, e := range *sim.Events {
		if e.EventType == common.MATCH_ABANDONED {
			t.Fatalf("the match was abandoned at minute %d (%d-%d)", e.Minute, sim.HomeGoals, sim.AwayGoals)
		}
	}
}
