package common

import (
	"database/sql"
	"regexp"
	"strings"
	"testing"
	"time"
)

// These run against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset), like the other
// transfer integration tests, in their own country "tc" so they never touch other tests' rows.

type comparablesWorld struct {
	db     *sql.DB
	dbCtx  *DatabaseContext
	teamA  int64
	teamB  int64
	target Player // a 26-year-old midfielder
}

// setRatings gives a seeded player the seven ratings and age wanted.
func setRatings(t *testing.T, db *sql.DB, playerId int64, goalKeeper, defense, lateral, pass, midfield, dribbling, shot, age int) {
	t.Helper()
	if _, err := db.Exec(`UPDATE players SET goal_keeper = $2, defense = $3, "lateral" = $4, pass = $5, midfield = $6, dribbling = $7, shot = $8, age = $9
		WHERE player_id = $1`, playerId, goalKeeper, defense, lateral, pass, midfield, dribbling, shot, age); err != nil {
		t.Fatalf("setting ratings: %s", err)
	}
}

func newComparablesWorld(t *testing.T) *comparablesWorld {
	t.Helper()
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "tc")
	clean := func() {
		// Also players released along the way (no team any more), found through their transfers.
		db.Exec(`DELETE FROM players WHERE team_id IN (SELECT team_id FROM teams WHERE country = 'tc')
			OR player_id IN (SELECT player_id FROM player_transfers WHERE from_team_id IN (SELECT team_id FROM teams WHERE country = 'tc')
				OR to_team_id IN (SELECT team_id FROM teams WHERE country = 'tc'))`)
	}
	clean()
	t.Cleanup(clean)
	w := &comparablesWorld{db: db, dbCtx: dbCtx}
	w.teamA = seedTransfersTeam(t, db, "Comparables A", "tc")
	w.teamB = seedTransfersTeam(t, db, "Comparables B", "tc")
	id := seedTransfersPlayer(t, db, w.teamA, "tc")
	setRatings(t, db, id, 40, 40, 40, 58, 58, 40, 40, 26) // best position midfield, average 45
	target, err := GetPlayerById(dbCtx, id)
	if err != nil {
		t.Fatalf("loading target: %s", err)
	}
	w.target = *target
	if BestCorePosition(w.target) != MIDFIELD || AverageSkill(w.target) != 45 {
		t.Fatalf("target is position %d average %d, want midfield / 45", BestCorePosition(w.target), AverageSkill(w.target))
	}
	return w
}

// sale records a transfer of a fresh player with the given snapshot (average < 0 leaves it NULL) and
// returns its transfer id.
func (w *comparablesWorld) sale(t *testing.T, kind int, fee int64, position, age, average int, at time.Time) int64 {
	t.Helper()
	player := seedTransfersPlayer(t, w.db, w.teamB, "tc")
	var avg any
	if average >= 0 {
		avg = average
	}
	var id int64
	if err := w.db.QueryRow(`INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, position, age, average_skill, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING transfer_id`, player, w.teamA, w.teamB, fee, kind, position, age, avg, at).Scan(&id); err != nil {
		t.Fatalf("seeding transfer: %s", err)
	}
	return id
}

func (w *comparablesWorld) comparables(t *testing.T) *TransferComparablesResult {
	t.Helper()
	result, err := GetTransferComparables(w.dbCtx, w.target)
	if err != nil {
		t.Fatalf("GetTransferComparables: %s", err)
	}
	return result
}

func transferIds(r *TransferComparablesResult) []int64 {
	ids := make([]int64, 0, len(r.Comparables))
	for _, c := range r.Comparables {
		ids = append(ids, c.TransferId)
	}
	return ids
}

func equalIds(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTransferComparables_OnlyEligibleTransfers(t *testing.T) {
	w := newComparablesWorld(t)
	now := time.Now()
	day := 24 * time.Hour

	// Three eligible kinds (listing, offer, buy option), all inside the narrow band (age 24-28, average 41-49).
	listing := w.sale(t, PLAYER_TRANSFER_LISTING, 100000, MIDFIELD, 26, 45, now.Add(-1*day))
	offer := w.sale(t, PLAYER_TRANSFER_OFFER, 120000, MIDFIELD, 24, 41, now.Add(-2*day))
	option := w.sale(t, PLAYER_TRANSFER_BUY_OPTION, 150000, MIDFIELD, 28, 49, now.Add(-3*day))

	// Everything below must stay out, even of the wide band.
	w.sale(t, PLAYER_TRANSFER_FREE_AGENT, 40000, MIDFIELD, 26, 45, now.Add(-1*day)) // free-agent premium
	w.sale(t, PLAYER_TRANSFER_LOAN, 20000, MIDFIELD, 26, 45, now.Add(-1*day))
	w.sale(t, PLAYER_TRANSFER_RELEASE, 0, MIDFIELD, 26, 45, now.Add(-1*day))
	w.sale(t, PLAYER_TRANSFER_PERMANENT, 90000, MIDFIELD, 26, 45, now.Add(-1*day)) // pre-table, unknown kind
	w.sale(t, PLAYER_TRANSFER_LISTING, 0, MIDFIELD, 26, 45, now.Add(-1*day))       // fee 0
	w.sale(t, PLAYER_TRANSFER_LISTING, 100000, MIDFIELD, 26, 45, now.Add(-61*day)) // too old
	w.sale(t, PLAYER_TRANSFER_LISTING, 100000, DEFENSE, 26, 45, now.Add(-1*day))   // other position
	w.sale(t, PLAYER_TRANSFER_OFFER, 100000, MIDFIELD, 26, -1, now.Add(-1*day))    // no snapshot
	w.sale(t, PLAYER_TRANSFER_OFFER, 100000, MIDFIELD, 40, 45, now.Add(-1*day))    // far too old a player
	w.sale(t, PLAYER_TRANSFER_OFFER, 100000, MIDFIELD, 26, 80, now.Add(-1*day))    // far too good
	// The target's own earlier sale.
	if _, err := w.db.Exec(`INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, position, age, average_skill, created_at)
		VALUES ($1, $2, $3, 100000, $4, $5, 26, 45, $6)`, w.target.Id, w.teamB, w.teamA, PLAYER_TRANSFER_LISTING, MIDFIELD, now.Add(-10*day)); err != nil {
		t.Fatal(err)
	}

	got := w.comparables(t)
	if want := []int64{listing, offer, option}; !equalIds(transferIds(got), want) {
		t.Fatalf("comparables = %v, want %v", transferIds(got), want)
	}
	if got.Broadened {
		t.Error("three narrow-band results must not be flagged broadened")
	}
	if got.Position != MIDFIELD || got.Age != 26 || got.AverageSkill != 45 {
		t.Errorf("reference = position %d age %d average %d, want %d / 26 / 45", got.Position, got.Age, got.AverageSkill, MIDFIELD)
	}
	if got.Count != 3 || *got.MedianFee != 120000 || *got.MinFee != 100000 || *got.MaxFee != 150000 {
		t.Errorf("summary = count %d median %d min %d max %d, want 3 / 120000 / 100000 / 150000", got.Count, *got.MedianFee, *got.MinFee, *got.MaxFee)
	}
	first := got.Comparables[0]
	if first.Fee != 100000 || first.Kind != PLAYER_TRANSFER_LISTING || first.Position != MIDFIELD || first.Age != 26 || first.AverageSkill != 45 ||
		first.FromTeamName != "Comparables A" || first.ToTeamName != "Comparables B" || first.PlayerName != "Test Player" {
		t.Errorf("first comparable = %+v", first)
	}
}

func TestTransferComparables_BroadensBelowThreeNarrowResults(t *testing.T) {
	w := newComparablesWorld(t)
	now := time.Now()
	day := 24 * time.Hour

	narrowA := w.sale(t, PLAYER_TRANSFER_LISTING, 100000, MIDFIELD, 25, 44, now.Add(-1*day))
	narrowB := w.sale(t, PLAYER_TRANSFER_OFFER, 110000, MIDFIELD, 27, 46, now.Add(-2*day))
	// Age 29 (+3) and average 40 (-5): inside the wide band (±4 / ±8) only.
	wideOnly := w.sale(t, PLAYER_TRANSFER_OFFER, 130000, MIDFIELD, 29, 40, now.Add(-3*day))
	// Outside even the wide band.
	w.sale(t, PLAYER_TRANSFER_OFFER, 130000, MIDFIELD, 31, 45, now.Add(-3*day))

	got := w.comparables(t)
	if !got.Broadened {
		t.Error("two narrow-band results must broaden the search")
	}
	if want := []int64{narrowA, narrowB, wideOnly}; !equalIds(transferIds(got), want) {
		t.Fatalf("comparables = %v, want %v", transferIds(got), want)
	}

	// With a third narrow-band result the wide-only sale drops out and the flag clears.
	narrowC := w.sale(t, PLAYER_TRANSFER_LISTING, 105000, MIDFIELD, 26, 45, now.Add(-4*day))
	got = w.comparables(t)
	if got.Broadened {
		t.Error("three narrow-band results must not broaden")
	}
	if want := []int64{narrowA, narrowB, narrowC}; !equalIds(transferIds(got), want) {
		t.Fatalf("comparables = %v, want %v", transferIds(got), want)
	}
}

func TestTransferComparables_NewestFirstCappedAtEight(t *testing.T) {
	w := newComparablesWorld(t)
	now := time.Now().Truncate(time.Second)

	var ids []int64
	for i := 0; i < 10; i++ {
		ids = append(ids, w.sale(t, PLAYER_TRANSFER_LISTING, int64(100000+i), MIDFIELD, 26, 45, now.Add(-time.Duration(i/2)*24*time.Hour)))
	}
	// Pairs share a timestamp, so the higher transfer id comes first within each pair.
	want := []int64{ids[1], ids[0], ids[3], ids[2], ids[5], ids[4], ids[7], ids[6]}
	got := w.comparables(t)
	if !equalIds(transferIds(got), want) {
		t.Fatalf("comparables = %v, want %v", transferIds(got), want)
	}
	if got.Count != ComparablesLimit {
		t.Errorf("count = %d, want %d", got.Count, ComparablesLimit)
	}
}

func TestTransferComparables_NoneFound(t *testing.T) {
	w := newComparablesWorld(t)
	got := w.comparables(t)
	if got.Count != 0 || got.MedianFee != nil || len(got.Comparables) != 0 || !got.Broadened {
		t.Errorf("empty result = %+v, want no comparables, no summary and the wide band tried", got)
	}
}

// RecordPlayerTransfer freezes position, age and average skill as they are at the transfer.
func TestRecordPlayerTransfer_StoresSnapshotTakenAtTheMove(t *testing.T) {
	w := newComparablesWorld(t)
	if _, err := w.db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatal(err)
	}
	player := w.target.Id // 26-year-old midfielder, average 45
	snapshot := func(step string, wantPosition, wantAge, wantAverage int) {
		t.Helper()
		var position, age, average int
		if err := w.db.QueryRow(`SELECT position, age, average_skill FROM player_transfers WHERE player_id = $1 ORDER BY transfer_id DESC LIMIT 1`, player).
			Scan(&position, &age, &average); err != nil {
			t.Fatalf("%s: reading snapshot: %s", step, err)
		}
		if position != wantPosition || age != wantAge || average != wantAverage {
			t.Errorf("%s: snapshot = position %d age %d average %d, want %d / %d / %d", step, position, age, average, wantPosition, wantAge, wantAverage)
		}
	}

	// A bought listing.
	var listingId int64
	if err := w.db.QueryRow(`INSERT INTO transfer_listings (player_id, selling_team_id, asking_price) VALUES ($1, $2, 500000) RETURNING listing_id`, player, w.teamA).Scan(&listingId); err != nil {
		t.Fatal(err)
	}
	if err := ExecuteTransfer(w.dbCtx, listingId, w.teamB); err != nil {
		t.Fatalf("ExecuteTransfer: %s", err)
	}
	snapshot("listing", MIDFIELD, 26, 45)

	// He then improves and ages; an accepted offer snapshots the new figures, the first row keeps its own.
	setRatings(t, w.db, player, 40, 40, 40, 70, 70, 40, 40, 27) // midfield 70, average 49
	var offerId int64
	if err := w.db.QueryRow(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount, offered_wage, offered_years, created_at)
		VALUES ($1, $2, $3, 700000, 1000, 2, now() - interval '3 days') RETURNING offer_id`, player, w.teamA, w.teamB).Scan(&offerId); err != nil {
		t.Fatal(err)
	}
	if err := AcceptOffer(w.dbCtx, offerId, w.teamB); err != nil {
		t.Fatalf("AcceptOffer: %s", err)
	}
	snapshot("offer", MIDFIELD, 27, 49)

	// A release snapshots too, and is never a comparable anyway.
	setRatings(t, w.db, player, 10, 10, 10, 20, 20, 10, 10, 28)
	if err := ReleasePlayer(w.dbCtx, player, w.teamA); err != nil {
		t.Fatalf("ReleasePlayer: %s", err)
	}
	snapshot("release", MIDFIELD, 28, 13)

	var firstAge, firstAverage int
	if err := w.db.QueryRow(`SELECT age, average_skill FROM player_transfers WHERE player_id = $1 ORDER BY transfer_id LIMIT 1`, player).Scan(&firstAge, &firstAverage); err != nil {
		t.Fatal(err)
	}
	if firstAge != 26 || firstAverage != 45 {
		t.Errorf("the first transfer's snapshot changed to age %d average %d; it must stay 26 / 45", firstAge, firstAverage)
	}
}

// The comparables statement must be served by player_transfers_comparables_idx: walked in ORDER BY
// order and stopped at the LIMIT, with no sequential scan or sort over the transfers.
func TestTransferComparables_UsesIndex(t *testing.T) {
	w := newComparablesWorld(t)
	tx, err := w.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	// Bitmap scans are off too: on a selective age/average band the planner may legitimately gather
	// the index range with a bitmap scan and sort the few survivors, which is just as index-served.
	// What this test pins is that the index can also be walked in ORDER BY order (no sort node) and
	// that nothing falls back to scanning the table.
	if _, err := tx.Exec(`SET LOCAL enable_seqscan = off; SET LOCAL enable_bitmapscan = off`); err != nil {
		t.Fatalf("disabling seq and bitmap scans: %s", err)
	}
	// The planner's choice depends on table statistics, which an almost empty test table does not
	// have: seed a realistic history (rolled back with the transaction) - sales spread over many
	// players, positions, kinds, ages and dates - and analyze, so the test checks the plan the
	// statement gets on a real table, not an accident of whatever other tests left behind.
	if _, err := tx.Exec(`INSERT INTO players (first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield, dribbling, shot,
			"header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country, shirt_number)
		SELECT 'Idx', 'Seed', 0, 50, 50, 50, 50, 50, 50, 50, false, false, false, false, '', 1000, 24, 3, 'tc', g FROM generate_series(1, 500) g`); err != nil {
		t.Fatalf("seeding players: %s", err)
	}
	if _, err := tx.Exec(`WITH seeded AS (
			SELECT player_id, row_number() OVER (ORDER BY player_id) - 1 AS rn FROM players WHERE first_name = 'Idx' AND last_name = 'Seed'
		)
		INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, position, age, average_skill, created_at)
		SELECT s.player_id, $1::bigint, $2::bigint, 1000 + g, (ARRAY[0, 1, 2, 3, 4, 5])[1 + g % 6], (ARRAY[0, 1, 4, 7])[1 + g % 4], 18 + g % 20, 30 + g % 50,
		       now() - (g % 400) * interval '1 day'
		FROM generate_series(1, 20000) g JOIN seeded s ON s.rn = g % 500`, w.teamA, w.teamB); err != nil {
		t.Fatalf("seeding a transfer history: %s", err)
	}
	if _, err := tx.Exec(`ANALYZE player_transfers`); err != nil {
		t.Fatalf("analyzing: %s", err)
	}
	for _, band := range []struct {
		name         string
		age, average int
	}{{"narrow", ComparablesNarrowAge, ComparablesNarrowAverage}, {"wide", ComparablesWideAge, ComparablesWideAverage}} {
		rows, err := tx.Query(`EXPLAIN `+getTransferComparablesStmtSql, MIDFIELD, 26-band.age, 26+band.age, 45-band.average, 45+band.average, w.target.Id, ComparablesLimit)
		if err != nil {
			t.Fatalf("%s: EXPLAIN: %s", band.name, err)
		}
		var plan strings.Builder
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				t.Fatalf("reading plan: %s", err)
			}
			plan.WriteString(line + "\n")
		}
		rows.Close()
		if !strings.Contains(plan.String(), "player_transfers_comparables_idx") {
			t.Errorf("%s: plan does not use player_transfers_comparables_idx:\n%s", band.name, plan.String())
		}
		if strings.Contains(plan.String(), "Seq Scan on player_transfers ") {
			t.Errorf("%s: plan scans player_transfers sequentially:\n%s", band.name, plan.String())
		}
		// The limit sits directly on the ordered index walk: no sort of the transfers (the outer
		// ORDER BY only orders the few joined survivors).
		if !regexp.MustCompile(`Limit[^\n]*\n\s+->\s+Index Scan using player_transfers_comparables_idx`).MatchString(plan.String()) {
			t.Errorf("%s: the limit is not applied straight on the ordered index walk:\n%s", band.name, plan.String())
		}
	}
}
