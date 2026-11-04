package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// TestAiContractRenewals_ManagerMeetsAnImprovingPlayersRaise: a player who has just improved
// expects more than his skill-based wage, so the Manager's first offer is declined - it must
// retry on the player's own terms instead of letting him walk at season end.
func TestAiContractRenewals_ManagerMeetsAnImprovingPlayersRaise(t *testing.T) {
	const country = "zr"
	w := newLoanWorld(t, country)
	team := w.team(country, "Renewers", common.AiLevelManager, nil, 10_000_000)
	w.squad(country, team)
	improver := w.player(country, team, common.STRIKER, 50, 21)
	w.exec(`UPDATE players SET contract_years_remaining = 1, intelligence = 5 WHERE player_id = $1`, improver)
	w.exec(`INSERT INTO player_progression_log (player_id, season, age_before, age_after, goal_keeper_before, goal_keeper_after,
		defense_before, defense_after, lateral_before, lateral_after, pass_before, pass_after, midfield_before, midfield_after,
		dribbling_before, dribbling_after, shot_before, shot_after)
		VALUES ($1, 1, 21, 21, 0, 0, 0, 0, 0, 0, 40, 50, 0, 0, 40, 50, 40, 50)`, improver)
	baseWage := common.PlayerWageFromSkills(0, 0, 0, 50, 0, 50, 50)

	renewExpiringContracts(w.dbCtx, w.teamOf(team))

	if years := w.count(`SELECT contract_years_remaining FROM players WHERE player_id = $1`, improver); years <= 1 {
		t.Fatalf("improving player still has %d season(s) left, want a renewal", years)
	}
	if wage := w.count(`SELECT wage FROM players WHERE player_id = $1`, improver); wage <= baseWage {
		t.Errorf("renewed wage %d, want the raise above the skill-based %d the player asked for", wage, baseWage)
	}
}

// TestAiContractRenewals_CaretakerRenewsOnlyItsStarters: a Caretaker keeps its current XI under
// contract but lets the bench run down to free agency.
func TestAiContractRenewals_CaretakerRenewsOnlyItsStarters(t *testing.T) {
	const country = "zs"
	w := newLoanWorld(t, country)
	team := w.team(country, "Caretakers", common.AiLevelCaretaker, nil, 1_000_000)
	w.squad(country, team)
	w.exec(`UPDATE players SET contract_years_remaining = 1 WHERE team_id = $1`, team)
	refreshAiTeamLineup(w.dbCtx, team, common.AiLevelCaretaker)
	lineup, err := common.GetDefaultLineup(w.dbCtx, team)
	if err != nil {
		t.Fatal(err)
	}
	if len(lineup.Positions) != 11 {
		t.Fatalf("default lineup has %d players, want 11", len(lineup.Positions))
	}

	renewExpiringContracts(w.dbCtx, w.teamOf(team))

	renewedStarters := w.count(`SELECT COUNT(*) FROM players p JOIN default_lineup_positions d ON d.player_id = p.player_id AND d.team_id = p.team_id
		WHERE p.team_id = $1 AND p.contract_years_remaining > 1`, team)
	if renewedStarters != 11 {
		t.Errorf("%d starters renewed, want all 11", renewedStarters)
	}
	renewed := w.count(`SELECT COUNT(*) FROM players WHERE team_id = $1 AND contract_years_remaining > 1`, team)
	if renewed != 11 {
		t.Errorf("%d players renewed, want only the 11 starters (bench left to expire)", renewed)
	}
}
