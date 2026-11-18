package common

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"
	"time"
)

func openTransfersTestDB(t *testing.T) (*sql.DB, *DatabaseContext) {
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
	dbCtx := &DatabaseContext{Db: db}
	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	return db, dbCtx
}

// seedTransfersCountry/Team/Player insert the minimum rows the transfer flows read: a country
// (for GetCurrentSeasonForTeam), clubs (each in its own league, which GetTeamById joins) with a generous budget, and a fully populated player
// (GetPlayerById scans every column).
func seedTransfersCountry(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, name); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
}

func seedTransfersTeam(t *testing.T, db *sql.DB, name string, country string) int64 {
	t.Helper()
	var leagueId, id int64
	if err := db.QueryRow(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, country).Scan(&leagueId); err != nil {
		t.Fatalf("seeding league: %s", err)
	}
	if err := db.QueryRow(`INSERT INTO teams (name, country, league_id, budget) VALUES ($1, $2, $3, 50000000) RETURNING team_id`, name, country, leagueId).Scan(&id); err != nil {
		t.Fatalf("seeding team %s: %s", name, err)
	}
	return id
}

func seedTransfersPlayer(t *testing.T, db *sql.DB, teamId int64, country string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield,
		dribbling, shot, "header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country, shirt_number)
		VALUES ($1, 'Test', 'Player', 0, 50, 50, 50, 50, 50, 50, 50, false, false, false, false, '', 1000, 24, 3, $2,
		(SELECT COALESCE(MAX(shirt_number), 0) + 1 FROM players WHERE team_id = $1))
		RETURNING player_id`, teamId, country).Scan(&id)
	if err != nil {
		t.Fatalf("seeding player: %s", err)
	}
	return id
}

type transferRow struct {
	from, to *int64
	fee      int64
	kind     int
}

func ref(id int64) *int64 { return &id }

// TestPlayerTransfers_RecordedByEveryFlow walks one player through every flow that moves a player
// (bought listing, accepted offer, release, free agent signing with and without a premium, loan,
// buy option, contract expiry) and checks each wrote exactly one structured player_transfers row.
func TestPlayerTransfers_RecordedByEveryFlow(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zq")
	// AcceptOffer reads the negotiation window from the (singleton) season row.
	if _, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatalf("seeding season: %s", err)
	}
	a := seedTransfersTeam(t, db, "Transfers A", "zq")
	b := seedTransfersTeam(t, db, "Transfers B", "zq")
	player := seedTransfersPlayer(t, db, a, "zq")

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	var want []transferRow
	check := func(step string) {
		t.Helper()
		got, err := GetPlayerTransfers(dbCtx, player, 50)
		if err != nil {
			t.Fatalf("%s: GetPlayerTransfers: %s", step, err)
		}
		if len(got) != len(want) {
			t.Fatalf("%s: got %d transfers, want %d (%+v)", step, len(got), len(want), got)
		}
		last := got[0]
		w := want[len(want)-1]
		if !reflect.DeepEqual(last.FromTeamId, w.from) || !reflect.DeepEqual(last.ToTeamId, w.to) || last.Fee != w.fee || last.Kind != w.kind {
			t.Fatalf("%s: latest transfer = from %v to %v fee %d kind %d, want from %v to %v fee %d kind %d",
				step, deref(last.FromTeamId), deref(last.ToTeamId), last.Fee, last.Kind, deref(w.from), deref(w.to), w.fee, w.kind)
		}
	}

	// 1. Bought from the transfer list.
	var listingId int64
	if err := db.QueryRow(`INSERT INTO transfer_listings (player_id, selling_team_id, asking_price) VALUES ($1, $2, 500000) RETURNING listing_id`, player, a).Scan(&listingId); err != nil {
		t.Fatal(err)
	}
	if err := ExecuteTransfer(dbCtx, listingId, b); err != nil {
		t.Fatalf("ExecuteTransfer: %s", err)
	}
	want = append(want, transferRow{ref(a), ref(b), 500000, PLAYER_TRANSFER_LISTING})
	check("listing")

	// 2. Accepted offer (backdated past the negotiation window).
	var offerId int64
	if err := db.QueryRow(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount, offered_wage, offered_years, created_at)
		VALUES ($1, $2, $3, 700000, 1000, 2, now() - interval '3 days') RETURNING offer_id`, player, a, b).Scan(&offerId); err != nil {
		t.Fatal(err)
	}
	if err := AcceptOffer(dbCtx, offerId, b); err != nil {
		t.Fatalf("AcceptOffer: %s", err)
	}
	want = append(want, transferRow{ref(b), ref(a), 700000, PLAYER_TRANSFER_OFFER})
	check("offer")

	// 3. Early release: no destination club.
	if err := ReleasePlayer(dbCtx, player, a); err != nil {
		t.Fatalf("ReleasePlayer: %s", err)
	}
	want = append(want, transferRow{ref(a), nil, 0, PLAYER_TRANSFER_RELEASE})
	check("release")

	// 4. Signed within the first season of free agency: the premium goes to the last club, which
	// becomes the origin.
	result, err := SignFreeAgentAtMarketRate(dbCtx, player, b)
	if err != nil || !result.Accepted || result.Premium <= 0 {
		t.Fatalf("SignFreeAgentAtMarketRate: %+v, %v", result, err)
	}
	want = append(want, transferRow{ref(a), ref(b), result.Premium, PLAYER_TRANSFER_FREE_AGENT})
	check("free agent with premium")

	// 5. Released again and signed after a full season unsigned: no premium, so no origin.
	if err := ReleasePlayer(dbCtx, player, b); err != nil {
		t.Fatalf("ReleasePlayer: %s", err)
	}
	want = append(want, transferRow{ref(b), nil, 0, PLAYER_TRANSFER_RELEASE})
	check("second release")
	exec(`UPDATE players SET free_agent_seasons = 1 WHERE player_id = $1`, player)
	result, err = SignFreeAgentAtMarketRate(dbCtx, player, a)
	if err != nil || !result.Accepted || result.Premium != 0 {
		t.Fatalf("SignFreeAgentAtMarketRate without premium: %+v, %v", result, err)
	}
	want = append(want, transferRow{nil, ref(a), 0, PLAYER_TRANSFER_FREE_AGENT})
	check("free agent without premium")

	// 6. Loan with a fee, then made permanent through its buy option.
	var loanListingId int64
	if err := db.QueryRow(`INSERT INTO loan_listings (player_id, lending_team_id, duration_seasons, loan_fee, buy_option_fee)
		VALUES ($1, $2, 1, 25000, 900000) RETURNING listing_id`, player, a).Scan(&loanListingId); err != nil {
		t.Fatal(err)
	}
	if err := AcceptLoan(dbCtx, loanListingId, b); err != nil {
		t.Fatalf("AcceptLoan: %s", err)
	}
	want = append(want, transferRow{ref(a), ref(b), 25000, PLAYER_TRANSFER_LOAN})
	check("loan")
	if err := ExerciseLoanBuyOption(dbCtx, player, b); err != nil {
		t.Fatalf("ExerciseLoanBuyOption: %s", err)
	}
	want = append(want, transferRow{ref(a), ref(b), 900000, PLAYER_TRANSFER_BUY_OPTION})
	check("buy option")

	// 7. Contract running out at season end.
	exec(`UPDATE players SET contract_years_remaining = 1 WHERE player_id = $1`, player)
	released, err := AdvancePlayerContract(dbCtx, player)
	if err != nil || !released {
		t.Fatalf("AdvancePlayerContract: released=%v, %v", released, err)
	}
	want = append(want, transferRow{ref(b), nil, 0, PLAYER_TRANSFER_RELEASE})
	check("contract expiry")

	// Newest first, so the full list is the reverse of what was recorded.
	got, _ := GetPlayerTransfers(dbCtx, player, 50)
	for i, tr := range got {
		w := want[len(want)-1-i]
		if tr.Kind != w.kind {
			t.Errorf("row %d: kind %d, want %d", i, tr.Kind, w.kind)
		}
	}
	if got[len(got)-1].FromTeamName != "Transfers A" || got[len(got)-1].ToTeamName != "Transfers B" {
		t.Errorf("oldest row names = %q -> %q, want Transfers A -> Transfers B", got[len(got)-1].FromTeamName, got[len(got)-1].ToTeamName)
	}
}

func deref(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// TestPlayerTransfersMigration_BackfillsFromLedger replays init78's backfill over legacy
// team_transactions/player_club_history rows and checks it pairs transfers up, recognizes the
// special cases, skips the initial squad, and is a no-op once the table has rows.
func TestPlayerTransfersMigration_BackfillsFromLedger(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zq")
	a := seedTransfersTeam(t, db, "Backfill A", "zq")
	b := seedTransfersTeam(t, db, "Backfill B", "zq")
	bought := seedTransfersPlayer(t, db, b, "zq")   // initial squad at A, bought by B
	signed := seedTransfersPlayer(t, db, b, "zq")   // initial squad at A, released, signed free by B
	released := seedTransfersPlayer(t, db, a, "zq") // released early by A
	ctx := context.Background()

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	at := func(day int) time.Time { return time.Date(2026, 1, day, 12, 0, 0, 0, time.UTC) }

	exec(`DELETE FROM player_transfers`)
	// bought: initial squad at A (day 1), then transferred to B for 400000 (day 5).
	exec(`INSERT INTO player_club_history (player_id, team_id, joined_at) VALUES ($1, $2, $3), ($1, $4, $5)`, bought, a, at(1), b, at(5))
	exec(`INSERT INTO team_transactions (team_id, amount, reason, description, created_at, player_id) VALUES
		($1, -400000, 1, 'Bought X from Backfill A', $3, $4), ($2, 400000, 0, 'Sold X to Backfill B', $3, $4)`, b, a, at(5), bought)
	// signed: initial squad at A (day 1), then a free agent signed by B without a premium (day 9).
	exec(`INSERT INTO player_club_history (player_id, team_id, joined_at) VALUES ($1, $2, $3), ($1, $4, $5)`, signed, a, at(1), b, at(9))
	// released: early release by A with severance (day 7).
	exec(`INSERT INTO team_transactions (team_id, amount, reason, description, created_at, player_id) VALUES ($1, -3000, 8, 'Released X', $2, $3)`, a, at(7), released)

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := applySchemaFile(ctx, conn, "history/init78_player_transfers.sql"); err != nil {
		t.Fatalf("applying migration: %s", err)
	}

	byPlayer := func(id int64) []PlayerTransfer {
		rows, err := GetPlayerTransfers(dbCtx, id, 50)
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	if rows := byPlayer(bought); len(rows) != 1 || *rows[0].FromTeamId != a || *rows[0].ToTeamId != b || rows[0].Fee != 400000 || rows[0].Kind != PLAYER_TRANSFER_PERMANENT {
		t.Errorf("bought player: got %+v, want one permanent transfer A->B for 400000", rows)
	}
	if rows := byPlayer(signed); len(rows) != 1 || rows[0].FromTeamId != nil || *rows[0].ToTeamId != b || rows[0].Fee != 0 || rows[0].Kind != PLAYER_TRANSFER_FREE_AGENT {
		t.Errorf("signed player: got %+v, want one free agent signing by B (initial squad row skipped)", rows)
	}
	if rows := byPlayer(released); len(rows) != 1 || *rows[0].FromTeamId != a || rows[0].ToTeamId != nil || rows[0].Kind != PLAYER_TRANSFER_RELEASE {
		t.Errorf("released player: got %+v, want one release from A", rows)
	}

	// Re-applying (every startup does) must not duplicate anything.
	if err := applySchemaFile(ctx, conn, "history/init78_player_transfers.sql"); err != nil {
		t.Fatalf("re-applying migration: %s", err)
	}
	if rows := byPlayer(bought); len(rows) != 1 {
		t.Errorf("after re-applying: got %d rows for bought player, want 1", len(rows))
	}
}
