package common

import (
	"testing"
)

// TestGetRecentLeagueTransfers_ReadsPlayerTransfers covers the recent-transfers panel's content
// (needs STRIKER_TEST_DB_DSN, see openspec/specs/league-page "Recent transfers content"): a free
// loan shows in both divisions, a loan fee is carried, a bought listing and an exercised buy option
// appear, releases and premium-free free-agent signings do not, and the list is newest first and
// capped by the limit.
func TestGetRecentLeagueTransfers_ReadsPlayerTransfers(t *testing.T) {
	db, dbCtx, lender, borrower, loanOut := loanSeed(t, "zn")
	outsider := seedTransfersTeam(t, db, "Recent Transfers Outsider", "zn")
	leagueOf := func(teamId int64) int64 {
		var id int64
		if err := db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, teamId).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	lenderLeague, borrowerLeague, outsiderLeague := leagueOf(lender), leagueOf(borrower), leagueOf(outsider)

	// Nothing yet.
	if got, err := GetRecentLeagueTransfers(dbCtx, lenderLeague, 10); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty panel = %v, %v, want a non-nil empty slice", got, err)
	}

	// A free loan with a buy option, then a buy option exercised on a second loan.
	freeLoan := seedTransfersPlayer(t, db, lender, "zn")
	loanOut(freeLoan, 1, 0, 0)
	optioned := seedTransfersPlayer(t, db, lender, "zn")
	loanOut(optioned, 2, 0, 900000)
	accMust(t, ExerciseLoanBuyOption(dbCtx, optioned, borrower), "ExerciseLoanBuyOption")

	// A listing bought by the lender from the borrower.
	listed := seedTransfersPlayer(t, db, borrower, "zn")
	if _, err := db.Exec(`INSERT INTO transfer_listings (player_id, selling_team_id, asking_price) VALUES ($1, $2, 2000000)`, listed, borrower); err != nil {
		t.Fatal(err)
	}
	var listingId int64
	accMust(t, db.QueryRow(`SELECT listing_id FROM transfer_listings WHERE player_id = $1`, listed).Scan(&listingId), "reading listing")
	accMust(t, ExecuteTransfer(dbCtx, listingId, lender), "ExecuteTransfer")

	// Moves that are not between clubs: a release, and a free agent signed without a premium.
	released := seedTransfersPlayer(t, db, lender, "zn")
	accMust(t, RecordPlayerTransfer(db, released, lender, 0, 0, PLAYER_TRANSFER_RELEASE), "recording release")
	agent := seedTransfersPlayer(t, db, lender, "zn")
	accMust(t, RecordPlayerTransfer(db, agent, 0, lender, 0, PLAYER_TRANSFER_FREE_AGENT), "recording free agent")
	// ... and a free agent that did pay a premium to his previous club.
	premium := seedTransfersPlayer(t, db, lender, "zn")
	accMust(t, RecordPlayerTransfer(db, premium, borrower, lender, 55000, PLAYER_TRANSFER_FREE_AGENT), "recording premium signing")

	got, err := GetRecentLeagueTransfers(dbCtx, lenderLeague, 10)
	accMust(t, err, "GetRecentLeagueTransfers")
	// Newest first: premium signing, listing, buy option, loan (optioned), loan (free).
	want := []struct {
		player int64
		kind   int
		amount int64
		buyer  int64
		seller int64
	}{
		{premium, PLAYER_TRANSFER_FREE_AGENT, 55000, lender, borrower},
		{listed, PLAYER_TRANSFER_LISTING, 2000000, lender, borrower},
		{optioned, PLAYER_TRANSFER_BUY_OPTION, 900000, borrower, lender},
		{optioned, PLAYER_TRANSFER_LOAN, 0, borrower, lender},
		{freeLoan, PLAYER_TRANSFER_LOAN, 0, borrower, lender},
	}
	if len(got) != len(want) {
		t.Fatalf("lender division panel has %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Player.Id != w.player || g.Kind != w.kind || g.Amount != w.amount || g.BuyerTeamId != w.buyer || g.SellerTeamId != w.seller {
			t.Errorf("row %d = player %d kind %d amount %d buyer %d seller %d, want %+v", i, g.Player.Id, g.Kind, g.Amount, g.BuyerTeamId, g.SellerTeamId, w)
		}
		if g.BuyerTeamName == "" || g.SellerTeamName == "" || g.OccurredAt.IsZero() {
			t.Errorf("row %d lacks team names or time: %+v", i, g)
		}
	}

	// The borrower's division sees the same moves (it is the other side of each).
	if other, err := GetRecentLeagueTransfers(dbCtx, borrowerLeague, 10); err != nil || len(other) != len(want) {
		t.Errorf("borrower division panel = %d rows (%v), want %d", len(other), err, len(want))
	}
	// A division with no involved club sees nothing.
	if other, _ := GetRecentLeagueTransfers(dbCtx, outsiderLeague, 10); len(other) != 0 {
		t.Errorf("uninvolved division panel = %+v, want none", other)
	}
	// The limit caps the list, keeping the newest.
	if capped, _ := GetRecentLeagueTransfers(dbCtx, lenderLeague, 2); len(capped) != 2 || capped[0].Player.Id != premium || capped[1].Player.Id != listed {
		t.Errorf("limit 2 = %+v", capped)
	}
}

// TestGetRecentLeagueTransfers_LoanFee checks a loan that carried a fee reports it as the amount.
func TestGetRecentLeagueTransfers_LoanFee(t *testing.T) {
	db, dbCtx, lender, borrower, _ := loanSeed(t, "zo")
	player := seedTransfersPlayer(t, db, lender, "zo")
	var listingId int64
	accMust(t, db.QueryRow(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, wage_contribution_percent, buy_option_fee)
		VALUES ($1, $2, 1, 150000, 0, 0) RETURNING listing_id`, player, lender).Scan(&listingId), "listing")
	accMust(t, AcceptLoan(dbCtx, listingId, borrower), "AcceptLoan")
	var leagueId int64
	accMust(t, db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, borrower).Scan(&leagueId), "league")
	got, err := GetRecentLeagueTransfers(dbCtx, leagueId, 10)
	accMust(t, err, "GetRecentLeagueTransfers")
	if len(got) != 1 || got[0].Kind != PLAYER_TRANSFER_LOAN || got[0].Amount != 150000 {
		t.Errorf("loan with a fee = %+v, want one loan row of 150000", got)
	}
}
