package common

import (
	"testing"
	"time"
)

// TestCpuLoanMarketHelpers_ReadLoanAndListingState covers the readers a CPU team uses to decide on
// loans (needs STRIKER_TEST_DB_DSN): which of its players are out on loan, which players are on
// loan to it with what terms, and how long and at what price its transfer listings have been up.
func TestCpuLoanMarketHelpers_ReadLoanAndListingState(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zq")
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatalf("seeding season: %s", err)
	}
	if _, err := db.Exec(`UPDATE season SET transfer_window_open = true`); err != nil {
		t.Fatalf("opening the window: %s", err)
	}
	lender := seedTransfersTeam(t, db, "Loan Market Lender", "zq")
	borrower := seedTransfersTeam(t, db, "Loan Market Borrower", "zq")
	loaned := seedTransfersPlayer(t, db, lender, "zq")
	listed := seedTransfersPlayer(t, db, lender, "zq")
	kept := seedTransfersPlayer(t, db, lender, "zq")

	var listingId int64
	if err := db.QueryRow(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee)
		VALUES ($1, $2, 2, 0, 40, 750000) RETURNING listing_id`, loaned, lender).Scan(&listingId); err != nil {
		t.Fatal(err)
	}
	if err := AcceptLoan(dbCtx, listingId, borrower); err != nil {
		t.Fatalf("AcceptLoan: %s", err)
	}
	if _, err := db.Exec(`INSERT INTO transfer_listings (player_id, selling_team_id, asking_price, listed_at) VALUES ($1, $2, 123000, now() - interval '8 days')`,
		listed, lender); err != nil {
		t.Fatal(err)
	}

	// The lender sees its loaned player as out on loan, the borrower as loaned in.
	lenderCommitments, err := GetTeamMarketCommitments(dbCtx, lender)
	if err != nil {
		t.Fatal(err)
	}
	if !lenderCommitments.LoanedOut[loaned] || lenderCommitments.LoanedOut[kept] || len(lenderCommitments.LoanedOut) != 1 {
		t.Errorf("lender LoanedOut = %v, want only %d", lenderCommitments.LoanedOut, loaned)
	}
	if len(lenderCommitments.LoanedIn) != 0 {
		t.Errorf("lender LoanedIn = %v, want none", lenderCommitments.LoanedIn)
	}
	borrowerCommitments, err := GetTeamMarketCommitments(dbCtx, borrower)
	if err != nil {
		t.Fatal(err)
	}
	if !borrowerCommitments.LoanedIn[loaned] || len(borrowerCommitments.LoanedOut) != 0 {
		t.Errorf("borrower commitments = %+v, want %d loaned in and nobody out", borrowerCommitments, loaned)
	}

	// The borrower can read the terms of the loans to it; the lender has none to it.
	loans, err := GetLoansToTeam(dbCtx, borrower)
	if err != nil || len(loans) != 1 {
		t.Fatalf("GetLoansToTeam(borrower) = %v, %v; want one loan", loans, err)
	}
	season, err := GetCurrentSeasonForTeam(dbCtx, borrower)
	if err != nil {
		t.Fatal(err)
	}
	loan := loans[0]
	if loan.PlayerId != loaned || loan.FromTeamId != lender || loan.ToTeamId != borrower ||
		loan.ReturnSeason != season+2 || loan.WageContributionPercent != 40 || loan.BuyOptionFee != 750000 {
		t.Errorf("loan = %+v, want player %d from %d to %d returning in season %d with 40%% wages and a 750000 option",
			loan, loaned, lender, borrower, season+2)
	}
	if none, err := GetLoansToTeam(dbCtx, lender); err != nil || len(none) != 0 {
		t.Errorf("GetLoansToTeam(lender) = %v, %v; want none", none, err)
	}

	// Own transfer listings carry their price and age.
	listings, err := GetTeamTransferListings(dbCtx, lender)
	if err != nil {
		t.Fatal(err)
	}
	l, ok := listings[listed]
	if !ok || len(listings) != 1 {
		t.Fatalf("GetTeamTransferListings = %v, want only %d", listings, listed)
	}
	if l.AskingPrice != 123000 || time.Since(l.ListedAt) < 7*24*time.Hour || time.Since(l.ListedAt) > 9*24*time.Hour {
		t.Errorf("listing = %+v, want price 123000 listed about 8 days ago", l)
	}
}
