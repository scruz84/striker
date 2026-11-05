package engine

import (
	"math"
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "CPU financial crisis": "In debt" and "Two listings already" - a Director in
// debt signs nobody (a cheap, far better free agent is left alone) and lists its best-paid sellable
// player at 80% of the usual asking price, once per pass, up to 2 open listings.
func TestAiCrisis_DirectorInDebtSellsAndSignsNobody(t *testing.T) {
	const country = "zv"
	w := newLoanWorld(t, country)
	team := w.team(country, "Debtors", common.AiLevelDirector, nil, -50_000)
	w.squad(country, team)
	// The earner is as good as his team-mates (so not a key player) but paid far more; a weaker
	// midfielder is the one the surplus rule lists at the full price.
	earner := w.player(country, team, common.MIDFIELD, 35, 27)
	w.exec(`UPDATE players SET wage = 9000 WHERE player_id = $1`, earner)
	w.player(country, team, common.MIDFIELD, 30, 27)
	freeAgent := w.player(country, nil, common.STRIKER, 90, 24)

	RunAITeamManagement(w.dbCtx, team)

	if got := w.playerTeam(freeAgent); got != 0 {
		t.Errorf("the free agent joined team %d, want nobody signing in a crisis", got)
	}
	asking := w.count(`SELECT COALESCE((SELECT asking_price FROM transfer_listings WHERE player_id = $1), 0)`, earner)
	if asking == 0 {
		t.Fatalf("the best-paid sellable player %d was not listed", earner)
	}
	p, err := common.GetPlayerById(w.dbCtx, earner)
	if err != nil {
		t.Fatal(err)
	}
	usual := common.ListingAskingPrice(*p, common.AiTransferProfileFor(common.AiLevelDirector), false, 0)
	if want := int(math.Round(float64(usual) * 0.8)); asking != want {
		t.Errorf("asking price %d, want 80%% of %d = %d", asking, usual, want)
	}

	// Two listings are open now (the surplus one and the crisis one): no further crisis sale.
	before := w.count(`SELECT COUNT(*) FROM transfer_listings WHERE player_id IN (SELECT player_id FROM players WHERE team_id = $1)`, team)
	if before < 2 {
		t.Fatalf("%d listings open, want at least 2", before)
	}
	listForFundsIfInCrisis(w.dbCtx, w.teamOf(team), w.squadOf(team), true)
	if n := w.count(`SELECT COUNT(*) FROM transfer_listings WHERE player_id IN (SELECT player_id FROM players WHERE team_id = $1)`, team); n != before {
		t.Errorf("%d listings after another crisis sale attempt, want %d (the cap is 2 open)", n, before)
	}
}

// cpu-team-management "Releasing unsold veterans": "Nobody wanted him".
func TestAiRelease_ManagerReleasesAnUnsoldVeteran(t *testing.T) {
	const country = "zw"
	w := newLoanWorld(t, country)
	team := w.team(country, "Veterans", common.AiLevelManager, nil, 200_000)
	w.squad(country, team)
	w.player(country, team, common.MIDFIELD, 35, 27)
	veteran := w.player(country, team, common.MIDFIELD, 30, 32) // 20 owned players
	w.exec(`UPDATE players SET wage = 3000, contract_years_remaining = 2 WHERE player_id = $1`, veteran)
	w.exec(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, listed_at) VALUES ($1, $2, 1, now() - interval '15 days')`, veteran, team)

	releaseUnsoldVeteranIfManager(w.dbCtx, w.teamOf(team), w.squadOf(team), 200_000)

	if got := w.playerTeam(veteran); got != 0 {
		t.Fatalf("the veteran is still at team %d, want him released", got)
	}
	if n := w.count(`SELECT COUNT(*) FROM loan_listings WHERE player_id = $1`, veteran); n != 0 {
		t.Errorf("his loan listing is still open")
	}
	if budget := w.count(`SELECT budget FROM teams WHERE team_id = $1`, team); budget != 200_000-30_000 {
		t.Errorf("budget %d, want 170,000 after the 30,000 severance", budget)
	}
}
