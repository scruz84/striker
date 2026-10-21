package common

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// queryCounter is a pgx tracer counting every statement the connection runs, so a test can assert
// how many queries a function issues.
type queryCounter struct{ n atomic.Int64 }

func (q *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	q.n.Add(1)
	return ctx
}

func (q *queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func openCountingTestDB(t *testing.T) (*DatabaseContext, *queryCounter) {
	t.Helper()
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parsing DSN: %s", err)
	}
	counter := &queryCounter{}
	cfg.Tracer = counter
	db := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { db.Close() })
	return &DatabaseContext{Db: db}, counter
}

// TestMarketListings_ConstantQueryCount checks GetActiveTransferListings/GetActiveLoanListings
// return each listing with the same player GetPlayerById would, newest first, in a number of
// queries that doesn't grow with the number of listings (issue #289).
func TestMarketListings_ConstantQueryCount(t *testing.T) {
	db, _ := openTransfersTestDB(t)
	dbCtx, counter := openCountingTestDB(t)
	seedTransfersCountry(t, db, "mkl")
	team := seedTransfersTeam(t, db, "Market Listings FC", "mkl")

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %s", q, err)
		}
	}
	exec(`DELETE FROM transfer_listings`)
	exec(`DELETE FROM loan_listings`)

	listAll := func(n int) (transfer, loan []int64) {
		for i := 0; i < n; i++ {
			transferred := seedTransfersPlayer(t, db, team, "mkl")
			exec(`INSERT INTO transfer_listings (player_id, selling_team_id, asking_price, listed_at) VALUES ($1, $2, $3, now() + make_interval(secs => $4))`,
				transferred, team, 1000+i, i)
			transfer = append(transfer, transferred)
			loaned := seedTransfersPlayer(t, db, team, "mkl")
			exec(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, buy_option_fee, listed_at) VALUES ($1, $2, 1, $3, 0, now() + make_interval(secs => $4))`,
				loaned, team, 100+i, i)
			loan = append(loan, loaned)
		}
		return
	}

	measure := func(fn func()) int64 {
		before := counter.n.Load()
		fn()
		return counter.n.Load() - before
	}

	transferIds, loanIds := listAll(3)
	var small int64
	var transfers []TransferListing
	var loans []LoanListing
	small = measure(func() {
		var err error
		if transfers, err = GetActiveTransferListings(dbCtx); err != nil {
			t.Fatalf("GetActiveTransferListings: %s", err)
		}
		if loans, err = GetActiveLoanListings(dbCtx); err != nil {
			t.Fatalf("GetActiveLoanListings: %s", err)
		}
	})

	if len(transfers) != 3 || len(loans) != 3 {
		t.Fatalf("expected 3 transfer and 3 loan listings, got %d and %d", len(transfers), len(loans))
	}
	// Newest listed_at first - the last player seeded was listed last.
	for i, l := range transfers {
		wantId := transferIds[len(transferIds)-1-i]
		want, err := GetPlayerById(dbCtx, wantId)
		if err != nil {
			t.Fatalf("GetPlayerById: %s", err)
		}
		if !reflect.DeepEqual(l.Player, *want) {
			t.Errorf("transfer listing %d player = %+v, want %+v", i, l.Player, *want)
		}
		if l.SellingTeamId != team || l.SellingTeamName != "Market Listings FC" || l.AskingPrice != int64(1000+len(transferIds)-1-i) {
			t.Errorf("transfer listing %d = %+v", i, l)
		}
	}
	for i, l := range loans {
		wantId := loanIds[len(loanIds)-1-i]
		want, err := GetPlayerById(dbCtx, wantId)
		if err != nil {
			t.Fatalf("GetPlayerById: %s", err)
		}
		if !reflect.DeepEqual(l.Player, *want) {
			t.Errorf("loan listing %d player = %+v, want %+v", i, l.Player, *want)
		}
		if l.LendingTeamId != team || l.LoanFee != int64(100+len(loanIds)-1-i) {
			t.Errorf("loan listing %d = %+v", i, l)
		}
	}

	listAll(10)
	large := measure(func() {
		if _, err := GetActiveTransferListings(dbCtx); err != nil {
			t.Fatalf("GetActiveTransferListings: %s", err)
		}
		if _, err := GetActiveLoanListings(dbCtx); err != nil {
			t.Fatalf("GetActiveLoanListings: %s", err)
		}
	})
	if large != small {
		t.Errorf("query count grew with the number of listings: %d for 3 listings, %d for 13", small, large)
	}
	if small > 4 {
		t.Errorf("expected at most 4 queries (listings + caps, twice), got %d", small)
	}
}

// TestTransferMarketStats_TopTransferFeeFromPlayerTransfers checks the biggest fee on record is
// read from player_transfers, counting only the kinds that pay a club for a player.
func TestTransferMarketStats_TopTransferFeeFromPlayerTransfers(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "mkf")
	buyer := seedTransfersTeam(t, db, "Top Fee Buyer", "mkf")
	seller := seedTransfersTeam(t, db, "Top Fee Seller", "mkf")
	cheap := seedTransfersPlayer(t, db, buyer, "mkf")
	pricey := seedTransfersPlayer(t, db, buyer, "mkf")
	loaned := seedTransfersPlayer(t, db, buyer, "mkf")

	if _, err := db.Exec(`DELETE FROM player_transfers`); err != nil {
		t.Fatalf("clearing player_transfers: %s", err)
	}
	record := func(player int64, fee int64, kind int) {
		t.Helper()
		if err := RecordPlayerTransfer(db, player, seller, buyer, fee, kind); err != nil {
			t.Fatalf("RecordPlayerTransfer: %s", err)
		}
	}
	record(cheap, 1_000, PLAYER_TRANSFER_LISTING)
	record(pricey, 5_000, PLAYER_TRANSFER_OFFER)
	// A loan fee is a rental, not a transfer fee - it must not win even though it's the largest.
	record(loaned, 9_000, PLAYER_TRANSFER_LOAN)

	stats, err := GetTransferMarketStats(dbCtx)
	if err != nil {
		t.Fatalf("GetTransferMarketStats: %s", err)
	}
	top := stats.TopTransferFee
	if top == nil {
		t.Fatal("TopTransferFee is nil")
	}
	if top.Player.Id != pricey || top.Amount != 5_000 || top.BuyerTeamId != buyer || top.BuyerTeamName != "Top Fee Buyer" ||
		top.SellerTeamId != seller || top.SellerTeamName != "Top Fee Seller" || top.OccurredAt.IsZero() {
		t.Errorf("TopTransferFee = %+v", top)
	}
}

// memoryCache is a map-backed Cache, so a test can exercise cached code paths without Redis.
type memoryCache struct{ values map[string]string }

func (m *memoryCache) GetValue(key string) (string, error) {
	v, ok := m.values[key]
	if !ok {
		return "", sql.ErrNoRows
	}
	return v, nil
}
func (m *memoryCache) SetValue(key string, value string)             { m.values[key] = value }
func (m *memoryCache) SetValueLongDuration(key string, value string) { m.values[key] = value }
func (m *memoryCache) DeleteValue(key string)                        { delete(m.values, key) }

// TestGetPlayerCapsBatch_CachesResult checks a second call for the same set of players (in any
// order) is served from ObjectCache without touching the database.
func TestGetPlayerCapsBatch_CachesResult(t *testing.T) {
	openTransfersTestDB(t)
	dbCtx, counter := openCountingTestDB(t)
	dbCtx.ObjectCache = &memoryCache{values: map[string]string{}}

	if _, err := GetPlayerCapsBatch(dbCtx, []int64{3, 1, 2}); err != nil {
		t.Fatalf("GetPlayerCapsBatch: %s", err)
	}
	before := counter.n.Load()
	if _, err := GetPlayerCapsBatch(dbCtx, []int64{1, 2, 3}); err != nil {
		t.Fatalf("GetPlayerCapsBatch: %s", err)
	}
	if n := counter.n.Load() - before; n != 0 {
		t.Errorf("second call ran %d queries, want 0 (cached)", n)
	}
}
