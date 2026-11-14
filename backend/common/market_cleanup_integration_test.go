package common

import (
	"database/sql"
	"errors"
	"testing"
)

// A player who leaves his club takes his listings and pending offers with him, and a stale
// listing never moves money (issue #397). Needs STRIKER_TEST_DB_DSN.

func marketCleanupRig(t *testing.T) (dbCtx *DatabaseContext, seller, other, buyer, player int64) {
	t.Helper()
	dbCtx = accDb(t)
	db := dbCtx.Db
	accMust(t, func() error {
		_, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`)
		return err
	}(), "seeding season")
	accMust(t, func() error { _, err := db.Exec(`UPDATE season SET transfer_window_open = true`); return err }(), "opening window")
	league := accSeedLeague(t, db)
	seller = accSeedTeam(t, dbCtx, league, nil, 0)
	other = accSeedTeam(t, dbCtx, league, nil, 0)
	buyer = accSeedTeam(t, dbCtx, league, nil, 0)
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE teams SET budget = 100000000 WHERE team_id IN ($1, $2, $3)`, seller, other, buyer)
		return err
	}(), "funding teams")
	player = accPlayerIds(t, db, seller, false)[0]
	return
}

func marketRowCount(t *testing.T, db *sql.DB, player int64) int64 {
	t.Helper()
	return accInt(t, db, `SELECT (SELECT count(*) FROM transfer_listings WHERE player_id = $1)
		+ (SELECT count(*) FROM loan_listings WHERE player_id = $1)
		+ (SELECT count(*) FROM transfer_offers WHERE player_id = $1)`, player)
}

func TestReleasePlayer_ClearsListingAndOffers(t *testing.T) {
	dbCtx, seller, other, _, player := marketCleanupRig(t)
	db := dbCtx.Db
	accMust(t, CreateTransferListing(dbCtx, player, seller, 1000000), "listing")
	accMust(t, func() error {
		_, err := db.Exec(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount) VALUES ($1, $2, $3, 500)`, player, other, seller)
		return err
	}(), "seeding offer")

	accMust(t, ReleasePlayer(dbCtx, player, seller), "release")
	if n := marketRowCount(t, db, player); n != 0 {
		t.Errorf("market rows after release = %d, want 0", n)
	}
}

func TestReleasePlayer_ClearsLoanListing(t *testing.T) {
	dbCtx, seller, _, _, player := marketCleanupRig(t)
	accMust(t, CreateLoanListing(dbCtx, player, seller, 1, 0, 50, 0), "loan listing")
	accMust(t, ReleasePlayer(dbCtx, player, seller), "release")
	if n := marketRowCount(t, dbCtx.Db, player); n != 0 {
		t.Errorf("market rows after release = %d, want 0", n)
	}
}

func TestAdvancePlayerContract_ExpiryClearsListing(t *testing.T) {
	dbCtx, seller, _, _, player := marketCleanupRig(t)
	db := dbCtx.Db
	accMust(t, CreateTransferListing(dbCtx, player, seller, 1000000), "listing")
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE players SET contract_years_remaining = 1 WHERE player_id = $1`, player)
		return err
	}(), "last contract year")
	released, err := AdvancePlayerContract(dbCtx, player)
	accMust(t, err, "advance contract")
	if !released {
		t.Fatal("contract did not expire")
	}
	if n := marketRowCount(t, db, player); n != 0 {
		t.Errorf("market rows after expiry = %d, want 0", n)
	}
}

// The reproduction from the issue, with the listing left behind as an older build would have:
// the purchase fails, the stale listing goes and no money moves.
func TestExecuteTransfer_StaleListingMovesNoMoney(t *testing.T) {
	dbCtx, seller, other, buyer, player := marketCleanupRig(t)
	db := dbCtx.Db
	var listing int64
	accMust(t, db.QueryRow(`INSERT INTO transfer_listings (player_id, selling_team_id, asking_price) VALUES ($1, $2, 1000000) RETURNING listing_id`,
		player, seller).Scan(&listing), "seeding listing")
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE players SET team_id = $1, shirt_number = NULL WHERE player_id = $2`, other, player)
		return err
	}(), "moving player")
	budgets := func() (int64, int64) {
		return accInt(t, db, `SELECT budget FROM teams WHERE team_id = $1`, seller), accInt(t, db, `SELECT budget FROM teams WHERE team_id = $1`, buyer)
	}
	sellerBefore, buyerBefore := budgets()

	if err := ExecuteTransfer(dbCtx, listing, buyer); !errors.Is(err, ErrListingNotFound) {
		t.Fatalf("ExecuteTransfer on a stale listing = %v, want ErrListingNotFound", err)
	}
	if sellerAfter, buyerAfter := budgets(); sellerAfter != sellerBefore || buyerAfter != buyerBefore {
		t.Errorf("budgets moved: seller %d -> %d, buyer %d -> %d", sellerBefore, sellerAfter, buyerBefore, buyerAfter)
	}
	if team := accInt(t, db, `SELECT team_id FROM players WHERE player_id = $1`, player); team != other {
		t.Errorf("player belongs to %d, want %d", team, other)
	}
	if n := accInt(t, db, `SELECT count(*) FROM transfer_listings WHERE listing_id = $1`, listing); n != 0 {
		t.Error("stale listing was kept")
	}
}

func TestAcceptLoan_StaleListingStartsNoLoan(t *testing.T) {
	dbCtx, seller, other, buyer, player := marketCleanupRig(t)
	db := dbCtx.Db
	var listing int64
	accMust(t, db.QueryRow(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons) VALUES ($1, $2, 1) RETURNING listing_id`,
		player, seller).Scan(&listing), "seeding loan listing")
	accMust(t, func() error {
		_, err := db.Exec(`UPDATE players SET team_id = $1, shirt_number = NULL WHERE player_id = $2`, other, player)
		return err
	}(), "moving player")

	if err := AcceptLoan(dbCtx, listing, buyer); !errors.Is(err, ErrLoanListingNotFound) {
		t.Fatalf("AcceptLoan on a stale listing = %v, want ErrLoanListingNotFound", err)
	}
	if n := accInt(t, db, `SELECT count(*) FROM player_loans WHERE player_id = $1`, player); n != 0 {
		t.Error("a loan started from a stale listing")
	}
	if team := accInt(t, db, `SELECT team_id FROM players WHERE player_id = $1`, player); team != other {
		t.Errorf("player belongs to %d, want %d", team, other)
	}
	if n := accInt(t, db, `SELECT count(*) FROM loan_listings WHERE listing_id = $1`, listing); n != 0 {
		t.Error("stale loan listing was kept")
	}
}

// A buy clears the player's other market rows too, not only the listing that was bought.
func TestExecuteTransfer_ClearsPendingOffers(t *testing.T) {
	dbCtx, seller, other, buyer, player := marketCleanupRig(t)
	db := dbCtx.Db
	accMust(t, CreateTransferListing(dbCtx, player, seller, 1000000), "listing")
	accMust(t, func() error {
		_, err := db.Exec(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount) VALUES ($1, $2, $3, 500)`, player, other, seller)
		return err
	}(), "seeding offer")
	listing := accInt(t, db, `SELECT listing_id FROM transfer_listings WHERE player_id = $1`, player)
	accMust(t, ExecuteTransfer(dbCtx, listing, buyer), "buy")
	if n := marketRowCount(t, db, player); n != 0 {
		t.Errorf("market rows after the sale = %d, want 0", n)
	}
}
