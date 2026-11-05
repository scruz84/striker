package engine

import (
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "AI level after a division change": a CPU club moved to another tier changes
// by at most one step, and an owned club is untouched.
func TestRedrawAiLevelAfterMove(t *testing.T) {
	const country = "zz"
	w := newLoanWorld(t, country)
	top := w.scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'Top', 1) RETURNING league_id`, country)
	moved := 0
	for seed := int64(0); seed < 20; seed++ {
		team := w.team(country, "Climbers", common.AiLevelCaretaker, nil, 100_000)
		redrawAiLevelAfterMove(w.dbCtx, team, top, rand.New(rand.NewSource(seed)))
		level := w.count(`SELECT ai_level FROM teams WHERE team_id = $1`, team)
		if level != common.AiLevelCaretaker && level != common.AiLevelManager {
			t.Fatalf("a promoted Caretaker became level %d, want at most one step up", level)
		}
		if level == common.AiLevelManager {
			moved++
		}
	}
	if moved == 0 {
		t.Error("no promoted Caretaker stepped up in 20 draws with tier-1 weights (90% above Caretaker)")
	}
	owner := w.user("redraw-owner")
	owned := w.team(country, "Owned", 0, owner, 100_000)
	redrawAiLevelAfterMove(w.dbCtx, owned, top, rand.New(rand.NewSource(1)))
	if level := w.count(`SELECT ai_level FROM teams WHERE team_id = $1`, owned); level != 0 {
		t.Errorf("an owned club's level became %d", level)
	}
}

// cpu-team-management "Club style": the stored style is what GetTeamById reads, NULL reads as
// balanced, and BackfillAiStyles gives a club with none its derived style without touching one that
// has one.
func TestAiStyle_StoredReadAndBackfilled(t *testing.T) {
	const country = "zq"
	w := newLoanWorld(t, country)
	team := w.team(country, "Styled", common.AiLevelManager, nil, 100_000)
	if style := w.teamOf(team).AiStyle; style != common.AiStyleBalanced {
		t.Errorf("a club with no stored style reads %q, want balanced", style)
	}
	w.exec(`UPDATE teams SET ai_style = $1 WHERE team_id = $2`, common.AiStyleCode(common.AiStyleFrugal), team)
	if style := w.teamOf(team).AiStyle; style != common.AiStyleFrugal {
		t.Errorf("stored frugal reads %q", style)
	}
	if reserve := aiProfile(w.teamOf(team)).BudgetReserve; reserve < 0.649 || reserve > 0.651 {
		t.Errorf("a frugal Manager's reserve %.2f, want 0.65", reserve)
	}

	fresh := w.team(country, "Unstyled", common.AiLevelManager, nil, 100_000)
	if err := common.BackfillAiStyles(w.dbCtx); err != nil {
		t.Fatal(err)
	}
	if style := w.teamOf(fresh).AiStyle; style != common.AiClubStyleFor(fresh) {
		t.Errorf("backfilled style %q, want %q", style, common.AiClubStyleFor(fresh))
	}
	if style := w.teamOf(team).AiStyle; style != common.AiStyleFrugal {
		t.Errorf("the backfill changed a stored style to %q", style)
	}
}
