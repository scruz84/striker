package common

import (
	"math"
	"sync"
	"testing"
)

// Rivalry storage end to end (openspec add-emergent-rivalries; needs STRIKER_TEST_DB_DSN).

func rivalryNotifications(t *testing.T, r *watchRig, userId int64) []string {
	t.Helper()
	rows, err := r.db.Query(`SELECT message_key FROM notifications WHERE user_id = $1 AND notification_type = $2 ORDER BY notification_id`, userId, NOTIFICATION_RIVALRY)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			t.Fatal(err)
		}
		keys = append(keys, k)
	}
	return keys
}

func TestRivalries_ScoreCapPairsAndConcurrency(t *testing.T) {
	r := newWatchRig(t, "zr")
	seedTransfersCountry(t, r.db, "zs")
	a := seedTransfersTeam(t, r.db, "Rival A", "zr")
	b := seedTransfersTeam(t, r.db, "Rival B", "zr")
	foreign := seedTransfersTeam(t, r.db, "Rival Abroad", "zs")

	// Two clubs that never met.
	if level, err := GetRivalryLevel(r.dbCtx, a, b); err != nil || level != RIVALRY_NONE {
		t.Fatalf("new pair level = %d (%v), want none", level, err)
	}

	// Concurrent additions to one pair: the sum is exact (40 additions of 0.5 = 20).
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			x, y := a, b
			if i%2 == 0 {
				x, y = b, a // either order is the same pair
			}
			if _, _, err := AddRivalryPoints(r.dbCtx, x, y, "zr", 0.5); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if score, err := GetRivalryScore(r.dbCtx, a, b); err != nil || math.Abs(score-20) > 1e-6 {
		t.Fatalf("score after 40 concurrent additions of 0.5 = %v (%v), want 20", score, err)
	}

	// Cap: 39 + 5 = 40.
	if _, err := r.db.Exec(`UPDATE team_rivalries SET score = 39 WHERE team_low_id = LEAST($1::bigint, $2::bigint) AND team_high_id = GREATEST($1::bigint, $2::bigint)`, a, b); err != nil {
		t.Fatal(err)
	}
	if old, updated, err := AddRivalryPoints(r.dbCtx, a, b, "zr", 5); err != nil || old != 39 || updated != 40 {
		t.Errorf("39 + 5 = %v -> %v (%v), want 39 -> 40", old, updated, err)
	}

	// Clubs of different countries: no score is created.
	AddRivalryEvent(r.dbCtx, a, foreign, 10)
	var rows int
	if err := r.db.QueryRow(`SELECT count(*) FROM team_rivalries WHERE team_low_id IN ($1, $2) AND team_high_id IN ($1, $2)`, a, foreign).Scan(&rows); err != nil || rows != 0 {
		t.Errorf("a cross-country pair has %d rows (%v), want none", rows, err)
	}
}

func TestRivalries_HumanBonusNotificationsAndDecay(t *testing.T) {
	r := newWatchRig(t, "zt")
	home, homeUser := r.ownedTeam("Rival Home", "zt")
	away, awayUser := r.ownedTeam("Rival Away", "zt")
	cpu := seedTransfersTeam(t, r.db, "Rival CPU", "zt")

	// Two human managers in a cup final that ends 2-1: (1 + 5) × 1.5 = 9 - and both are told
	// they are now rivals.
	AddRivalryEvent(r.dbCtx, home, away, 6)
	if score, _ := GetRivalryScore(r.dbCtx, home, away); math.Abs(score-9) > 1e-6 {
		t.Errorf("human pair after a 6-point event = %v, want 9", score)
	}
	for name, user := range map[string]int64{"home": homeUser, "away": awayUser} {
		if got := rivalryNotifications(t, r, user); len(got) != 1 || got[0] != "notifications.messages.rivalryRival" {
			t.Errorf("%s owner notifications = %v, want one rivalryRival", name, got)
		}
	}
	// No bonus against a CPU club, and no notification below a level.
	AddRivalryEvent(r.dbCtx, home, cpu, 6)
	if score, _ := GetRivalryScore(r.dbCtx, home, cpu); math.Abs(score-6) > 1e-6 {
		t.Errorf("human-CPU pair after a 6-point event = %v, want 6", score)
	}
	// New Clásico: 9 -> 18.
	AddRivalryEvent(r.dbCtx, home, away, 6)
	if got := rivalryNotifications(t, r, homeUser); len(got) != 2 || got[1] != "notifications.messages.rivalryClasico" {
		t.Errorf("home notifications after the Clásico = %v", got)
	}

	// Team page: pairs at 22, 9 and 5 -> the Clásico at 22 and the Rival at 9 only.
	other := seedTransfersTeam(t, r.db, "Rival Other", "zt")
	set := func(x, y int64, score float64) {
		t.Helper()
		if _, err := r.db.Exec(`INSERT INTO team_rivalries (team_low_id, team_high_id, country, score) VALUES (LEAST($1::bigint, $2::bigint), GREATEST($1::bigint, $2::bigint), 'zt', $3)
			ON CONFLICT (team_low_id, team_high_id) DO UPDATE SET score = EXCLUDED.score`, x, y, score); err != nil {
			t.Fatal(err)
		}
	}
	set(home, away, 22)
	set(home, cpu, 9)
	set(home, other, 5)
	top, err := GetTopRivalries(r.dbCtx, home, 3)
	if err != nil || len(top) != 2 || top[0].TeamId != away || top[0].Level != RIVALRY_CLASICO || top[1].TeamId != cpu || top[1].Level != RIVALRY_RIVAL {
		t.Errorf("top rivalries = %+v (%v), want the Clásico with Rival Away then the Rival with Rival CPU", top, err)
	}

	// Decay: 22 -> 13.2 drops the Clásico to Rival, silently; 5 -> 3; a pair at 0.8 resets to 0.
	set(cpu, other, 0.8)
	before := len(rivalryNotifications(t, r, homeUser))
	if err := DecayCountryRivalries(r.dbCtx, "zt"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		x, y int64
		want float64
	}{{home, away, 13.2}, {home, other, 3}, {cpu, other, 0}} {
		if score, _ := GetRivalryScore(r.dbCtx, c.x, c.y); math.Abs(score-c.want) > 1e-4 {
			t.Errorf("pair %d-%d after the decay = %v, want %v", c.x, c.y, score, c.want)
		}
	}
	if after := len(rivalryNotifications(t, r, homeUser)); after != before {
		t.Errorf("the decay sent %d notifications", after-before)
	}
}

// Key player sold to a domestic club: +3 to the pair; a bench player's sale adds nothing.
func TestRivalries_KeyPlayerSale(t *testing.T) {
	r := newWatchRig(t, "zu")
	seller := seedTransfersTeam(t, r.db, "Rival Seller", "zu")
	buyer := seedTransfersTeam(t, r.db, "Rival Buyer", "zu")
	// A thin squad: one keeper and ten outfielders, so every starter is a key player.
	ids := []int64{}
	for i := 0; i < 11; i++ {
		ids = append(ids, seedTransfersPlayer(t, r.db, seller, "zu"))
	}
	if _, err := r.db.Exec(`UPDATE players SET goal_keeper = 90 WHERE player_id = $1`, ids[0]); err != nil {
		t.Fatal(err)
	}
	key := ids[5]
	squad, err := GetPlayersFromTeamId(r.dbCtx, seller)
	if err != nil || !IsKeyPlayer(*squad, key) {
		t.Fatalf("fixture: player %d should be a key player (%v)", key, err)
	}
	if err := CreateTransferListing(r.dbCtx, key, seller, 1000); err != nil {
		t.Fatal(err)
	}
	var listingId int64
	if err := r.db.QueryRow(`SELECT listing_id FROM transfer_listings WHERE player_id = $1`, key).Scan(&listingId); err != nil {
		t.Fatal(err)
	}
	if err := ExecuteTransfer(r.dbCtx, listingId, buyer); err != nil {
		t.Fatal(err)
	}
	if score, _ := GetRivalryScore(r.dbCtx, seller, buyer); math.Abs(score-RivalryKeyPlayerSalePoints) > 1e-6 {
		t.Errorf("pair after a key-player sale = %v, want %v", score, RivalryKeyPlayerSalePoints)
	}

	// A bench player of a deep squad is no key player.
	deep := seedTransfersTeam(t, r.db, "Rival Deep", "zu")
	for i := 0; i < 20; i++ {
		seedTransfersPlayer(t, r.db, deep, "zu")
	}
	var bench int64
	deepSquad, _ := GetPlayersFromTeamId(r.dbCtx, deep)
	for _, p := range *deepSquad {
		if !IsKeyPlayer(*deepSquad, p.Id) {
			bench = p.Id
			break
		}
	}
	if bench == 0 {
		t.Fatal("fixture: no bench player in a 20-player squad")
	}
	if err := CreateTransferListing(r.dbCtx, bench, deep, 1000); err != nil {
		t.Fatal(err)
	}
	if err := r.db.QueryRow(`SELECT listing_id FROM transfer_listings WHERE player_id = $1`, bench).Scan(&listingId); err != nil {
		t.Fatal(err)
	}
	if err := ExecuteTransfer(r.dbCtx, listingId, buyer); err != nil {
		t.Fatal(err)
	}
	if score, _ := GetRivalryScore(r.dbCtx, deep, buyer); score != 0 {
		t.Errorf("pair after a bench player's sale = %v, want 0", score)
	}
}
