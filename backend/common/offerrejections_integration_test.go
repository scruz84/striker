package common

import (
	"testing"
	"time"
)

// TestOfferRejections_RecordedAndExpired covers the bookkeeping the CPU's offer cooldown rests
// on: a rejected offer leaves a rejection record (what RejectOffer does for a human or a CPU
// seller alike), a stale CPU offer is withdrawn and recorded too, and offers from unowned teams
// are counted per player.
func TestOfferRejections_RecordedAndExpired(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zq")
	seller := seedTransfersTeam(t, db, "Rejections Seller", "zq")
	cpuA := seedTransfersTeam(t, db, "Rejections CPU A", "zq")
	cpuB := seedTransfersTeam(t, db, "Rejections CPU B", "zq")
	humanBuyer := seedTransfersTeam(t, db, "Rejections Human", "zq")
	if _, err := db.Exec(`UPDATE teams SET owner_id = 987654 WHERE team_id = $1`, humanBuyer); err != nil {
		t.Fatal(err)
	}
	player := seedTransfersPlayer(t, db, seller, "zq")

	insertOffer := func(from int64, amount int64, age string) int64 {
		var id int64
		if err := db.QueryRow(`INSERT INTO transfer_offers (player_id, from_team_id, to_team_id, amount, offered_wage, offered_years, created_at)
			VALUES ($1, $2, $3, $4, 100, 1, now() - $5::interval) RETURNING offer_id`, player, from, seller, amount, age).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	freshA := insertOffer(cpuA, 5000, "1 hour")
	staleB := insertOffer(cpuB, 6000, "4 days")
	insertOffer(humanBuyer, 7000, "1 hour")

	if n, err := CountOpenAiOffersOnPlayer(dbCtx, player); err != nil || n != 2 {
		t.Fatalf("CountOpenAiOffersOnPlayer = %d, %v; want 2 (the human's offer doesn't count)", n, err)
	}

	// The seller rejects CPU A's offer: it is remembered.
	if err := RejectOffer(dbCtx, freshA, seller); err != nil {
		t.Fatalf("RejectOffer: %s", err)
	}
	rejections, err := GetRecentOfferRejections(dbCtx, cpuA, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := rejections[player]; !ok || r.Amount != 5000 {
		t.Fatalf("rejection of CPU A = %+v, %v; want the 5000 offer", r, ok)
	}

	// CPU B's offer has sat for four days: its team withdraws it and the withdrawal is remembered.
	expired, err := ExpireStaleOffers(dbCtx, cpuB, time.Now().Add(-AiOfferLifetime))
	if err != nil || expired != 1 {
		t.Fatalf("ExpireStaleOffers = %d, %v; want 1", expired, err)
	}
	var stillThere int
	if err := db.QueryRow(`SELECT count(*) FROM transfer_offers WHERE offer_id = $1`, staleB).Scan(&stillThere); err != nil || stillThere != 0 {
		t.Errorf("stale offer still open (%d rows, %v)", stillThere, err)
	}
	rejections, _ = GetRecentOfferRejections(dbCtx, cpuB, time.Now().Add(-time.Hour))
	if r, ok := rejections[player]; !ok || r.Amount != 6000 {
		t.Errorf("expired offer not remembered: %+v, %v", r, ok)
	}

	// A fresh offer is left alone.
	freshB := insertOffer(cpuB, 9000, "1 hour")
	if expired, err := ExpireStaleOffers(dbCtx, cpuB, time.Now().Add(-AiOfferLifetime)); err != nil || expired != 0 {
		t.Errorf("ExpireStaleOffers withdrew %d recent offers, %v", expired, err)
	}
	sent, err := GetPlayersWithOfferFromTeam(dbCtx, cpuB)
	if err != nil || !sent[player] {
		t.Errorf("open offer %d should still be listed for CPU B: %v, %v", freshB, sent, err)
	}
}
