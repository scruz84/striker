package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// A CPU side's kickoff plan carries the mentality common.PickAiMentality gives it against the
// opponent's XI (openspec/specs/cpu-team-management "CPU mentality"); friendlies stay Balanced.
// Needs STRIKER_TEST_DB_DSN.
func TestPlanAiKickoffs_PicksTheMentality(t *testing.T) {
	const country = "zh"
	w := newLoanWorld(t, country)
	strong := w.team(country, "Mentality Strong", common.AiLevelManager, nil, 1000000)
	weak := w.team(country, "Mentality Weak", common.AiLevelManager, nil, 1000000)
	for _, n := range []struct{ position, count int }{
		{common.GOALKEEPER, 2}, {common.DEFENSE, 5}, {common.MIDFIELD, 6}, {common.STRIKER, 5},
	} {
		for i := 0; i < n.count; i++ {
			w.player(country, strong, n.position, 70, 27)
			w.player(country, weak, n.position, 40, 27)
		}
	}
	matchId := w.scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, now(), $2, $3) RETURNING match_id`, w.league, strong, weak)
	lineups, err := common.GetLineupsForMatch(w.dbCtx, matchId, &[]int64{strong, weak})
	if err != nil {
		t.Fatal(err)
	}
	planAiKickoffs(w.dbCtx, kickoffLeague, matchId, strong, weak, lineups, nil, nil)
	if got := (*lineups)[strong].Mentality; got != common.MENTALITY_ATTACKING {
		t.Errorf("the much stronger Manager plays %d, want Attacking", got)
	}
	if got := (*lineups)[weak].Mentality; got != common.MENTALITY_DEFENSIVE {
		t.Errorf("the much weaker Manager plays %d, want Defensive", got)
	}

	friendlyId := w.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now()) RETURNING match_id`, strong, weak)
	friendlyLineups, err := common.GetLineupsForFriendly(w.dbCtx, friendlyId, []int64{strong, weak})
	if err != nil {
		t.Fatal(err)
	}
	planAiKickoffs(w.dbCtx, kickoffFriendly, friendlyId, strong, weak, friendlyLineups, nil, nil)
	if (*friendlyLineups)[strong].Mentality != common.MENTALITY_BALANCED || (*friendlyLineups)[weak].Mentality != common.MENTALITY_BALANCED {
		t.Errorf("friendly mentalities %d / %d, want both Balanced", (*friendlyLineups)[strong].Mentality, (*friendlyLineups)[weak].Mentality)
	}
}
