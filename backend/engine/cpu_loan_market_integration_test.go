package engine

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// loanWorld is a small throwaway world in its own country for the CPU loan-market tests (they need
// STRIKER_TEST_DB_DSN and are skipped without it): a league, an open transfer window and helpers to
// add teams, owners and players.
type loanWorld struct {
	t      *testing.T
	db     *sql.DB
	dbCtx  *common.DatabaseContext
	league int64
}

func newLoanWorld(t *testing.T, country string) *loanWorld {
	t.Helper()
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	w := &loanWorld{t: t, db: db, dbCtx: &common.DatabaseContext{Db: db}}
	if err := common.EnsureSchema(w.dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	common.ResetNationalRadarCache()
	// Start from a market empty of everything a CPU team could prefer to the fixtures below:
	// earlier tests in the package leave transfer and loan listings and free agents in the shared
	// database, and a Manager makes only one move per pass. Free agents are put out of reach by
	// age (the CPU ignores anyone over aiMaxPlayerAge) rather than deleted.
	w.exec(`DELETE FROM loan_listings`)
	w.exec(`DELETE FROM transfer_listings`)
	w.exec(`UPDATE players SET age = 50 WHERE team_id IS NULL`)
	w.exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`)
	w.exec(`UPDATE season SET transfer_window_open = true, offer_negotiation_window_minutes = 0`)
	w.exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country)
	w.league = w.scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, country)
	return w
}

func (w *loanWorld) exec(query string, args ...any) {
	w.t.Helper()
	if _, err := w.db.Exec(query, args...); err != nil {
		w.t.Fatalf("%s: %s", query, err)
	}
}

func (w *loanWorld) scan(query string, args ...any) int64 {
	w.t.Helper()
	var id int64
	if err := w.db.QueryRow(query, args...).Scan(&id); err != nil {
		w.t.Fatalf("%s: %s", query, err)
	}
	return id
}

// user inserts a real user row (notifications reference one) and returns its id.
func (w *loanWorld) user(name string) int64 {
	return w.scan(`INSERT INTO users (name, provider, email) VALUES ($1, 'dev', $2) RETURNING user_id`,
		name, fmt.Sprintf("%s-%d@example.test", name, time.Now().UnixNano()))
}

// team inserts a team of country at AI level (0 and an owner = a human club) with the given budget.
func (w *loanWorld) team(country, name string, level int, owner any, budget int64) int64 {
	return w.scan(`INSERT INTO teams (name, country, league_id, budget, ai_level, owner_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING team_id`,
		name, country, w.league, budget, level, owner)
}

// player inserts a player of the given position and flat skill (nil team = free agent).
func (w *loanWorld) player(country string, teamId any, position int, skill int, age int) int64 {
	var gk, def, pass, mid, dri, shot int
	switch position {
	case common.GOALKEEPER:
		gk = skill
	case common.DEFENSE:
		def = skill
	case common.MIDFIELD:
		mid, pass = skill, skill
	default:
		shot, pass, dri = skill, skill, skill
	}
	wage := common.PlayerWageFromSkills(gk, def, 0, pass, mid, dri, shot)
	shirt := any(nil)
	if teamId != nil {
		shirt = w.scan(`SELECT COALESCE(MAX(shirt_number), 0) + 1 FROM players WHERE team_id = $1`, teamId)
	}
	return w.scan(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield,
		dribbling, shot, "header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country, shirt_number,
		goal_keeper_max, defense_max, lateral_max, pass_max, midfield_max, dribbling_max, shot_max)
		VALUES ($1, 'Loan', 'Player', 0, $2, $3, 0, $4, $5, $6, $7, false, false, false, false, '', $8, $9, 4, $10, $11,
		99, 99, 10, 99, 99, 99, 99) RETURNING player_id`,
		teamId, gk, def, pass, mid, dri, shot, wage, age, country, shirt)
}

// squad fills a team with a balanced, ordinary 18 (2 GK, 5 D, 6 M, 5 S at skill 35, age 27).
func (w *loanWorld) squad(country string, teamId int64) {
	for _, n := range []struct{ position, count int }{
		{common.GOALKEEPER, 2}, {common.DEFENSE, 5}, {common.MIDFIELD, 6}, {common.STRIKER, 5},
	} {
		for i := 0; i < n.count; i++ {
			w.player(country, teamId, n.position, 35, 27)
		}
	}
}

func (w *loanWorld) teamOf(teamId int64) *common.Team {
	w.t.Helper()
	team, err := common.GetTeamById(teamId, w.dbCtx)
	if err != nil {
		w.t.Fatal(err)
	}
	return team
}

func (w *loanWorld) playerTeam(playerId int64) int64 {
	return w.scan(`SELECT COALESCE(team_id, 0) FROM players WHERE player_id = $1`, playerId)
}

func (w *loanWorld) count(query string, args ...any) int {
	return int(w.scan(query, args...))
}

// TestCpuLoanMarket_ManagerTakesAHumansLoanListing: a Manager whose strikers are weak accepts a
// human's free 1-season loan of a far better striker through the same AcceptLoan a human uses -
// the player moves, the loan is recorded with its return season and the human is told.
func TestCpuLoanMarket_ManagerTakesAHumansLoanListing(t *testing.T) {
	const country = "zl"
	w := newLoanWorld(t, country)
	human := w.user("loan-lender")
	lender := w.team(country, "Loan Lender FC", 0, human, 5_000_000)
	w.squad(country, lender)
	manager := w.team(country, "Loan Manager FC", common.AiLevelManager, nil, 5_000_000)
	w.squad(country, manager)
	star := w.player(country, lender, common.STRIKER, 99, 27)
	w.exec(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee)
		VALUES ($1, $2, 1, 0, 0, 0)`, star, lender)

	runAiTransferPass(w.dbCtx, w.teamOf(manager))

	if got := w.playerTeam(star); got != manager {
		t.Fatalf("the Manager should have taken the loan: star plays for team %d, want %d", got, manager)
	}
	var from, to int64
	var returnSeason int
	if err := w.db.QueryRow(`SELECT from_team_id, to_team_id, return_season FROM player_loans WHERE player_id = $1`, star).Scan(&from, &to, &returnSeason); err != nil {
		t.Fatalf("the loan should be recorded: %s", err)
	}
	if from != lender || to != manager || returnSeason != 2 {
		t.Errorf("loan = from %d to %d returning in season %d, want %d -> %d returning in season 2", from, to, returnSeason, lender, manager)
	}
	if n := w.count(`SELECT count(*) FROM loan_listings WHERE player_id = $1`, star); n != 0 {
		t.Errorf("the accepted listing should be gone, found %d", n)
	}
	notifications, err := common.GetNotifications(w.dbCtx, human, 10)
	if err != nil {
		t.Fatal(err)
	}
	told := false
	for _, n := range notifications {
		told = told || n.MessageKey == "notifications.messages.loanedOut"
	}
	if !told {
		t.Errorf("the human lender should be told the player was loaned out, got %+v", notifications)
	}
}

// TestCpuLoanMarket_ManagerExercisesABuyOptionInTheLastSeason: a CPU team with a star on loan in
// the last season of the loan, and an option at his market value, pays it: the loan row goes, the
// player stays, the fee moves to the lender and a buy-option transfer is recorded. (A Manager, so
// that the result doesn't depend on what a Director's market-wide search finds in a shared DB.)
func TestCpuLoanMarket_ManagerExercisesABuyOptionInTheLastSeason(t *testing.T) {
	const country = "zm"
	w := newLoanWorld(t, country)
	lender := w.team(country, "Option Lender FC", common.AiLevelCaretaker, nil, 5_000_000)
	w.squad(country, lender)
	director := w.team(country, "Option Manager FC", common.AiLevelManager, nil, 5_000_000)
	w.squad(country, director)
	star := w.player(country, lender, common.MIDFIELD, 99, 27)
	p, err := common.GetPlayerById(w.dbCtx, star)
	if err != nil {
		t.Fatal(err)
	}
	fee := common.MarketValue(*p)
	w.exec(`UPDATE players SET team_id = $1, shirt_number = 99 WHERE player_id = $2`, director, star)
	w.exec(`INSERT INTO player_loans (player_id, from_team_id, to_team_id, return_season, wage_contribution_percent, buy_option_fee)
		VALUES ($1, $2, $3, 2, 0, $4)`, star, lender, director, fee) // current season is 1: this is the last one

	directorBefore, lenderBefore := w.scan(`SELECT budget FROM teams WHERE team_id = $1`, director), w.scan(`SELECT budget FROM teams WHERE team_id = $1`, lender)
	runAiTransferPass(w.dbCtx, w.teamOf(director))

	if n := w.count(`SELECT count(*) FROM player_loans WHERE player_id = $1`, star); n != 0 {
		t.Fatalf("the team should have exercised the option and closed the loan, %d loan rows remain", n)
	}
	if got := w.playerTeam(star); got != director {
		t.Errorf("the player should stay at the borrower, plays for team %d", got)
	}
	if n := w.count(`SELECT count(*) FROM player_transfers WHERE player_id = $1 AND kind = $2 AND fee = $3`, star, common.PLAYER_TRANSFER_BUY_OPTION, fee); n != 1 {
		t.Errorf("want one buy-option transfer of %d recorded, found %d", fee, n)
	}
	// The pass may also make other moves, so check the fee moved rather than exact balances.
	if got := w.scan(`SELECT budget FROM teams WHERE team_id = $1`, lender); got != lenderBefore+fee {
		t.Errorf("lender budget = %d, want %d + the option fee %d", got, lenderBefore, fee)
	}
	if got := w.scan(`SELECT budget FROM teams WHERE team_id = $1`, director); got > directorBefore-fee {
		t.Errorf("borrower budget = %d, want at most %d after paying the option", got, directorBefore-fee)
	}
}

// TestCpuLoanMarket_TwoCpuTeamsRacingForOneListingProduceOneLoan: two Managers run their passes at
// the same instant against one listing (two engine replicas, or two teams in one tick) - the row
// lock inside AcceptLoan lets exactly one win; the other's attempt is an expected, silent failure.
func TestCpuLoanMarket_TwoCpuTeamsRacingForOneListingProduceOneLoan(t *testing.T) {
	const country = "zn"
	w := newLoanWorld(t, country)
	human := w.user("race-lender")
	lender := w.team(country, "Race Lender FC", 0, human, 5_000_000)
	w.squad(country, lender)
	a := w.team(country, "Race Manager A", common.AiLevelManager, nil, 5_000_000)
	w.squad(country, a)
	b := w.team(country, "Race Manager B", common.AiLevelManager, nil, 5_000_000)
	w.squad(country, b)
	star := w.player(country, lender, common.STRIKER, 99, 27)
	w.exec(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee)
		VALUES ($1, $2, 1, 0, 0, 0)`, star, lender)

	teamA, teamB := w.teamOf(a), w.teamOf(b)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, team := range []*common.Team{teamA, teamB} {
		wg.Add(1)
		go func(team *common.Team) {
			defer wg.Done()
			<-start
			runAiTransferPass(w.dbCtx, team)
		}(team)
	}
	close(start)
	wg.Wait()

	if n := w.count(`SELECT count(*) FROM player_loans WHERE player_id = $1`, star); n != 1 {
		t.Fatalf("exactly one loan must result from two simultaneous accepts, found %d", n)
	}
	if got := w.playerTeam(star); got != a && got != b {
		t.Errorf("the star should play for one of the two Managers, plays for team %d", got)
	}
	if n := w.count(`SELECT count(*) FROM loan_listings WHERE player_id = $1`, star); n != 0 {
		t.Errorf("the listing should be consumed, found %d", n)
	}
}

// crowdPool fills a country's national-team pool (its NationalSquadPoolSize best players) with
// strong players on a filler club, so that the ordinary fixture squads of that country are not
// "internationals" - whom a CPU team won't lend.
func (w *loanWorld) crowdPool(country string) {
	filler := w.team(country, country+" Pool Filler FC", common.AiLevelCaretaker, nil, 5_000_000)
	for i := 0; i < common.NationalSquadPoolSize+2; i++ {
		w.player(country, filler, common.MIDFIELD, 90, 27)
	}
}

func (w *loanWorld) squadOf(teamId int64) []*common.Player {
	w.t.Helper()
	players, err := common.GetPlayersFromTeamId(w.dbCtx, teamId)
	if err != nil {
		w.t.Fatal(err)
	}
	return *players
}

// listForTransfer puts a player on the transfer list as if listed the given number of days ago.
func (w *loanWorld) listForTransfer(teamId, playerId int64, daysAgo int) int64 {
	p, err := common.GetPlayerById(w.dbCtx, playerId)
	if err != nil {
		w.t.Fatal(err)
	}
	asking := common.MarketValue(*p)
	w.exec(`INSERT INTO transfer_listings (player_id, selling_team_id, asking_price, listed_at) VALUES ($1, $2, $3, now() - make_interval(days => $4))`,
		playerId, teamId, asking, daysAgo)
	return asking
}

// TestCpuLoanMarket_UnsoldSurplusIsLoanedOut: a Manager's surplus defender unsold after 8 days
// leaves the transfer list and is offered on a 1-season loan at 10% of his value, no wage
// contribution, with a buy option at the price he was listed at.
func TestCpuLoanMarket_UnsoldSurplusIsLoanedOut(t *testing.T) {
	const country = "zo"
	w := newLoanWorld(t, country)
	w.crowdPool(country)
	manager := w.team(country, "Surplus Manager FC", common.AiLevelManager, nil, 5_000_000)
	w.squad(country, manager)
	surplus := w.player(country, manager, common.DEFENSE, 20, 27) // a sixth defender
	asking := w.listForTransfer(manager, surplus, 8)

	manageLoanListings(w.dbCtx, w.teamOf(manager), w.squadOf(manager))

	if n := w.count(`SELECT count(*) FROM transfer_listings WHERE player_id = $1`, surplus); n != 0 {
		t.Errorf("the unsold transfer listing should be withdrawn, found %d", n)
	}
	var seasons int
	var fee, option int64
	var contribution int
	if err := w.db.QueryRow(`SELECT duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee FROM loan_listings
		WHERE player_id = $1 AND lending_team_id = $2`, surplus, manager).Scan(&seasons, &fee, &contribution, &option); err != nil {
		t.Fatalf("the player should be on the loan market: %s", err)
	}
	if want := (asking + 5) / 10; seasons != 1 || fee < want-1 || fee > want || contribution != 0 || option != asking {
		t.Errorf("loan = %d season(s), fee %d, wages %d%%, option %d; want 1, ~%d, 0%%, %d", seasons, fee, contribution, option, want, asking)
	}
}

// TestCpuLoanMarket_WindowClosedMeansNoLoanListing: with the transfer window shut a CPU team lends
// nobody, and leaves its transfer listing alone.
func TestCpuLoanMarket_WindowClosedMeansNoLoanListing(t *testing.T) {
	const country = "zp"
	w := newLoanWorld(t, country)
	w.crowdPool(country)
	manager := w.team(country, "Closed Window FC", common.AiLevelManager, nil, 5_000_000)
	w.squad(country, manager)
	surplus := w.player(country, manager, common.DEFENSE, 20, 27)
	w.listForTransfer(manager, surplus, 8)
	w.exec(`UPDATE season SET transfer_window_open = false`)
	t.Cleanup(func() { w.exec(`UPDATE season SET transfer_window_open = true`) })

	manageLoanListings(w.dbCtx, w.teamOf(manager), w.squadOf(manager))

	if n := w.count(`SELECT count(*) FROM loan_listings WHERE lending_team_id = $1`, manager); n != 0 {
		t.Errorf("no loan listing while the window is closed, found %d", n)
	}
	if n := w.count(`SELECT count(*) FROM transfer_listings WHERE player_id = $1`, surplus); n != 1 {
		t.Errorf("the transfer listing should stay, found %d", n)
	}
}

// TestCpuLoanMarket_DirectorLoansOutABenchedTeenagerButAManagerDoesNot: development loans are a
// Director trait - free, half the wage kept, no buy option.
func TestCpuLoanMarket_DirectorLoansOutABenchedTeenagerButAManagerDoesNot(t *testing.T) {
	const country = "zq2"
	w := newLoanWorld(t, country)
	w.crowdPool(country)
	build := func(name string, level int) (int64, int64) {
		team := w.team(country, name, level, nil, 5_000_000)
		w.squad(country, team)
		teen := w.player(country, team, common.STRIKER, 20, 19)
		return team, teen
	}
	director, directorTeen := build("Youth Director FC", common.AiLevelDirector)
	manager, managerTeen := build("Youth Manager FC", common.AiLevelManager)

	manageLoanListings(w.dbCtx, w.teamOf(director), w.squadOf(director))
	manageLoanListings(w.dbCtx, w.teamOf(manager), w.squadOf(manager))

	var fee, option int64
	var contribution int
	if err := w.db.QueryRow(`SELECT loan_fee, wage_contribution_percent, buy_option_fee FROM loan_listings WHERE player_id = $1`, directorTeen).
		Scan(&fee, &contribution, &option); err != nil {
		t.Fatalf("the Director should list its benched teenager for loan: %s", err)
	}
	if fee != 0 || contribution != 50 || option != 0 {
		t.Errorf("development loan = fee %d, wages %d%%, option %d; want 0, 50%%, 0", fee, contribution, option)
	}
	if n := w.count(`SELECT count(*) FROM loan_listings WHERE player_id = $1`, managerTeen); n != 0 {
		t.Errorf("a Manager keeps its youngsters, found %d loan listings", n)
	}
	if n := w.count(`SELECT count(*) FROM loan_listings WHERE lending_team_id IN ($1, $2)`, director, manager); n != 1 {
		t.Errorf("exactly one loan listing expected across both teams, found %d", n)
	}
}

// TestCpuLoanMarket_LoanListingOfAPlayerWhoNowStartsIsWithdrawn: a loan-listed player who has
// become a starter is taken back off the market.
func TestCpuLoanMarket_LoanListingOfAPlayerWhoNowStartsIsWithdrawn(t *testing.T) {
	const country = "zr"
	w := newLoanWorld(t, country)
	w.crowdPool(country)
	director := w.team(country, "Withdraw Director FC", common.AiLevelDirector, nil, 5_000_000)
	w.squad(country, director)
	star := w.player(country, director, common.STRIKER, 99, 19)
	w.exec(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons) VALUES ($1, $2, 1)`, star, director)

	manageLoanListings(w.dbCtx, w.teamOf(director), w.squadOf(director))

	if n := w.count(`SELECT count(*) FROM loan_listings WHERE player_id = $1`, star); n != 0 {
		t.Errorf("a starter's loan listing should be withdrawn, found %d", n)
	}
}
