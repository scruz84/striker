package engine

import "testing"

func TestMerchandisingForPosition_FirstPlaceEarnsMoreThanLast(t *testing.T) {
	pool := int64(100000)
	teamCount := 10

	first := merchandisingForPosition(pool, 1, teamCount)
	last := merchandisingForPosition(pool, teamCount, teamCount)

	if first <= last {
		t.Fatalf("expected 1st place (%d) to earn more than last place (%d)", first, last)
	}
	if last <= 0 {
		t.Fatalf("expected last place to still earn something (a struggling team's shirts still sell), got %d", last)
	}
}

func TestMerchandisingForPosition_SingleTeamGetsWholePool(t *testing.T) {
	if got := merchandisingForPosition(100000, 1, 1); got != 100000 {
		t.Fatalf("expected the whole pool for a 1-team league, got %d", got)
	}
}

func TestMerchandisingPoolForTier_HigherTiersPayMore(t *testing.T) {
	if merchandisingPoolForTier(1) <= merchandisingPoolForTier(2) {
		t.Fatal("expected tier 1's pool to exceed tier 2's")
	}
	if merchandisingPoolForTier(2) <= merchandisingPoolForTier(3) {
		t.Fatal("expected tier 2's pool to exceed tier 3's")
	}
	if merchandisingPoolForTier(99) != MerchandisingPoolTier3 {
		t.Fatalf("expected an unknown tier to fall back to the tier-3 pool")
	}
}
