package engine

import "testing"

func TestPrizeMoneyForPosition_FirstPlaceEarnsMostLastPlaceEarnsLeast(t *testing.T) {
	pool := int64(100000)
	teamCount := 10

	first := prizeMoneyForPosition(pool, 1, teamCount)
	last := prizeMoneyForPosition(pool, teamCount, teamCount)

	if first <= last {
		t.Fatalf("expected 1st place (%d) to earn more than last place (%d)", first, last)
	}
	if last < 0 {
		t.Fatalf("expected non-negative prize money, got %d", last)
	}
}

func TestPrizeMoneyForPosition_MonotonicallyDecreasing(t *testing.T) {
	pool := int64(100000)
	teamCount := 8
	prev := prizeMoneyForPosition(pool, 1, teamCount)
	for position := 2; position <= teamCount; position++ {
		cur := prizeMoneyForPosition(pool, position, teamCount)
		if cur > prev {
			t.Fatalf("expected prize money to never increase as position worsens: position %d (%d) > position %d (%d)",
				position, cur, position-1, prev)
		}
		prev = cur
	}
}

func TestPrizeMoneyForPosition_SingleTeamGetsWholePool(t *testing.T) {
	if got := prizeMoneyForPosition(100000, 1, 1); got != 100000 {
		t.Fatalf("expected the whole pool for a 1-team league, got %d", got)
	}
}

func TestPrizePoolForTier_HigherTiersPayMore(t *testing.T) {
	if prizePoolForTier(1) <= prizePoolForTier(2) {
		t.Fatal("expected tier 1's pool to exceed tier 2's")
	}
	if prizePoolForTier(2) <= prizePoolForTier(3) {
		t.Fatal("expected tier 2's pool to exceed tier 3's")
	}
	// An unrecognized tier falls back to the smallest pool rather than paying nothing or
	// panicking.
	if prizePoolForTier(99) != PrizeMoneyPoolTier3 {
		t.Fatalf("expected an unknown tier to fall back to the tier-3 pool")
	}
}

func TestOrdinal(t *testing.T) {
	cases := map[int]string{
		1: "1st", 2: "2nd", 3: "3rd", 4: "4th",
		11: "11th", 12: "12th", 13: "13th",
		21: "21st", 22: "22nd", 23: "23rd",
		101: "101st", 111: "111th",
	}
	for n, want := range cases {
		if got := ordinal(n); got != want {
			t.Errorf("ordinal(%d) = %q, want %q", n, got, want)
		}
	}
}
