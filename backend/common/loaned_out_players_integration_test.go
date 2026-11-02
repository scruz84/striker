package common

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

// loanSeed gives a lender and a borrower club (each in its own league) with an open transfer
// window, and a helper that loans a player from the lender to the borrower through AcceptLoan.
func loanSeed(t *testing.T, country string) (db *sql.DB, dbCtx *DatabaseContext, lender, borrower int64, loanOut func(playerId int64, seasons, wagePercent int, buyOption int64)) {
	t.Helper()
	db, dbCtx = openTransfersTestDB(t)
	seedTransfersCountry(t, db, country)
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatalf("seeding season: %s", err)
	}
	if _, err := db.Exec(`UPDATE season SET transfer_window_open = true`); err != nil {
		t.Fatalf("opening the window: %s", err)
	}
	lender = seedTransfersTeam(t, db, "Loaned Out Lender "+country, country)
	borrower = seedTransfersTeam(t, db, "Loaned Out Borrower "+country, country)
	loanOut = func(playerId int64, seasons, wagePercent int, buyOption int64) {
		t.Helper()
		var listingId int64
		if err := db.QueryRow(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee)
			VALUES ($1, $2, $3, 0, $4, $5) RETURNING listing_id`, playerId, lender, seasons, wagePercent, buyOption).Scan(&listingId); err != nil {
			t.Fatal(err)
		}
		if err := AcceptLoan(dbCtx, listingId, borrower); err != nil {
			t.Fatalf("AcceptLoan: %s", err)
		}
	}
	return
}

// TestGetLoanedOutPlayers_FollowsTheLoansLife covers the lender's view of his loans (needs
// STRIKER_TEST_DB_DSN): the player shows from the accept with the borrower, its division, the
// return season and the terms, and leaves on return, on a buy option and for a release-on-return loan.
func TestGetLoanedOutPlayers_FollowsTheLoansLife(t *testing.T) {
	db, dbCtx, lender, borrower, loanOut := loanSeed(t, "zl")
	later := seedTransfersPlayer(t, db, lender, "zl")
	soon := seedTransfersPlayer(t, db, lender, "zl")
	kept := seedTransfersPlayer(t, db, lender, "zl")

	out, err := GetLoanedOutPlayers(dbCtx, lender)
	accMust(t, err, "GetLoanedOutPlayers")
	if out == nil || len(out) != 0 {
		t.Fatalf("before any loan: %v, want a non-nil empty slice", out)
	}

	loanOut(later, 3, 0, 0)
	loanOut(soon, 1, 50, 1200000)

	out, err = GetLoanedOutPlayers(dbCtx, lender)
	accMust(t, err, "GetLoanedOutPlayers")
	if len(out) != 2 {
		t.Fatalf("loaned out = %d, want 2", len(out))
	}
	// Soonest return first.
	first := out[0]
	if first.Player.Id != soon || first.BorrowingTeamId != borrower || first.BorrowingTeamName != "Loaned Out Borrower zl" ||
		first.BorrowingDivisionName != "L1" || first.BorrowingDivisionId == 0 ||
		first.ReturnSeason != 2 || first.WageContributionPercent != 50 || first.BuyOptionFee != 1200000 {
		t.Errorf("first row = %+v", first)
	}
	if out[1].Player.Id != later || out[1].ReturnSeason != 4 || out[1].BuyOptionFee != 0 {
		t.Errorf("second row = %+v", out[1])
	}
	for _, o := range out {
		if o.Player.Id == kept {
			t.Errorf("kept player %d listed as loaned out", kept)
		}
	}

	// The borrower sees both as loaned in, with the lender as the parent.
	in, err := GetLoanedInPlayers(dbCtx, borrower)
	accMust(t, err, "GetLoanedInPlayers")
	if len(in) != 2 || in[0].ParentTeamId != lender || in[0].ParentTeamName != "Loaned Out Lender zl" {
		t.Errorf("loaned in = %+v", in)
	}
	if lenderIn, _ := GetLoanedInPlayers(dbCtx, lender); lenderIn == nil || len(lenderIn) != 0 {
		t.Errorf("lender loaned in = %v, want an empty slice", lenderIn)
	}

	// Buy option: the former lender no longer lists him.
	accMust(t, ExerciseLoanBuyOption(dbCtx, soon, borrower), "ExerciseLoanBuyOption")
	out, _ = GetLoanedOutPlayers(dbCtx, lender)
	if len(out) != 1 || out[0].Player.Id != later {
		t.Errorf("after the buy option = %+v, want only %d", out, later)
	}

	// Release-on-return loans never show.
	if _, err := db.Exec(`UPDATE player_loans SET release_on_return = true WHERE player_id = $1`, later); err != nil {
		t.Fatal(err)
	}
	out, _ = GetLoanedOutPlayers(dbCtx, lender)
	if len(out) != 0 {
		t.Errorf("release-on-return loan listed: %+v", out)
	}
	if _, err := db.Exec(`UPDATE player_loans SET release_on_return = false WHERE player_id = $1`, later); err != nil {
		t.Fatal(err)
	}

	// Return: back in the squad, gone from the section.
	loans, err := GetLoansToTeam(dbCtx, borrower)
	accMust(t, err, "GetLoansToTeam")
	if len(loans) != 1 {
		t.Fatalf("loans to borrower = %d, want 1", len(loans))
	}
	accMust(t, ReturnLoan(dbCtx, loans[0]), "ReturnLoan")
	out, _ = GetLoanedOutPlayers(dbCtx, lender)
	if len(out) != 0 {
		t.Errorf("after the return = %+v, want none", out)
	}
}

// TestTeamDetails_CarryLoans checks both team-details builders expose loanedOut / loanedIn, always
// as arrays, and that another club's view redacts the loaned player's skills like its own squad's.
func TestTeamDetails_CarryLoans(t *testing.T) {
	db, dbCtx, lender, borrower, loanOut := loanSeed(t, "zm")
	var owner int64
	if err := db.QueryRow(`INSERT INTO users (name, provider, email, country) VALUES ('Lender Owner', 'google', $1, 'zm') RETURNING user_id`,
		"loaned-out-"+accUnique()+"@example.com").Scan(&owner); err != nil {
		t.Fatalf("seeding user: %s", err)
	}
	if _, err := db.Exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, owner, lender); err != nil {
		t.Fatal(err)
	}
	loaned := seedTransfersPlayer(t, db, lender, "zm")
	seedTransfersPlayer(t, db, lender, "zm")
	loanOut(loaned, 1, 25, 0)

	own, err := GetFullTeamDetailsByUser(owner, true, dbCtx)
	accMust(t, err, "GetFullTeamDetailsByUser")
	if len(own.LoanedOut) != 1 || own.LoanedOut[0].Player.Id != loaned || own.LoanedOut[0].Player.Defense != 50 {
		t.Errorf("own LoanedOut = %+v", own.LoanedOut)
	}
	if own.LoanedIn == nil || len(own.LoanedIn) != 0 {
		t.Errorf("own LoanedIn = %v, want an empty slice", own.LoanedIn)
	}

	public, err := GetTeamDetailsById(lender, false, dbCtx)
	accMust(t, err, "GetTeamDetailsById lender")
	if len(public.LoanedOut) != 1 || public.LoanedOut[0].WageContributionPercent != 25 {
		t.Fatalf("public LoanedOut = %+v", public.LoanedOut)
	}
	if public.LoanedOut[0].Player.Defense != 0 {
		t.Errorf("public view leaked the loaned player's skills: %+v", public.LoanedOut[0].Player)
	}
	for _, p := range public.Players {
		if p.Id == loaned {
			t.Errorf("loaned player still in the lender's squad")
		}
	}

	bDetails, err := GetTeamDetailsById(borrower, false, dbCtx)
	accMust(t, err, "GetTeamDetailsById borrower")
	if len(bDetails.LoanedIn) != 1 || bDetails.LoanedIn[0].PlayerId != loaned || bDetails.LoanedIn[0].ParentTeamId != lender {
		t.Errorf("borrower LoanedIn = %+v", bDetails.LoanedIn)
	}
	if bDetails.LoanedOut == nil || len(bDetails.LoanedOut) != 0 {
		t.Errorf("borrower LoanedOut = %v, want an empty slice", bDetails.LoanedOut)
	}

	// The JSON keys the client reads, and empty arrays rather than null.
	raw, err := json.Marshal(public)
	accMust(t, err, "marshal")
	for _, key := range []string{`"loanedOut":[{`, `"borrowingTeamId"`, `"borrowingTeamName"`, `"borrowingDivisionId"`,
		`"borrowingDivisionName"`, `"returnSeason"`, `"wageContributionPercent"`, `"buyOptionFee"`, `"goalsForBorrower"`, `"loanedIn":[]`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("JSON lacks %s: %s", key, raw)
		}
	}
}
