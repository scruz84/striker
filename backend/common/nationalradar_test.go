package common

import "testing"

func poolOf(firstId int64, n int) []*Player {
	pool := make([]*Player, n)
	for i := range pool {
		pool[i] = &Player{Id: firstId + int64(i), Midfield: 90 - i, Pass: 90 - i}
	}
	return pool
}

func TestBuildNationalRadar_LookupAndRanks(t *testing.T) {
	radar := buildNationalRadar(map[string][]*Player{"es": poolOf(1, 8), "it": poolOf(100, 3)}, map[int64]int{1: 12, 101: 3})

	best, ok := radar.Lookup(1)
	if !ok || best.Country != "es" || best.Rank != 0 || best.Caps != 12 {
		t.Errorf("lookup of Spain's best = %+v, %v", best, ok)
	}
	second, ok := radar.Lookup(101)
	if !ok || second.Country != "it" || second.Rank != 1 || second.Caps != 3 {
		t.Errorf("lookup of Italy's second = %+v, %v", second, ok)
	}
	if entry, ok := radar.Lookup(999); ok {
		t.Errorf("an unknown player is not an international, got %+v", entry)
	}
}

func TestNationalRadar_BestTakesTheTopOfEachCountry(t *testing.T) {
	radar := buildNationalRadar(map[string][]*Player{"es": poolOf(1, 8), "it": poolOf(100, 3)}, nil)
	best := radar.Best(2)
	if len(best) != 4 {
		t.Fatalf("got %d entries, want 2 per country", len(best))
	}
	perCountry := map[string]int{}
	for _, e := range best {
		perCountry[e.Country]++
		if e.Rank >= 2 {
			t.Errorf("entry %d rank %d is outside the top 2", e.Player.Id, e.Rank)
		}
	}
	if perCountry["es"] != 2 || perCountry["it"] != 2 {
		t.Errorf("per country = %v, want 2 each", perCountry)
	}
	if got := radar.Best(NationalRadarTopPerCountry); len(got) != NationalRadarTopPerCountry+3 {
		t.Errorf("top %d of an 8-player and a 3-player pool = %d entries, want %d", NationalRadarTopPerCountry, len(got), NationalRadarTopPerCountry+3)
	}
}

func TestNationalRadar_NilIsSafe(t *testing.T) {
	var radar *NationalRadar
	if _, ok := radar.Lookup(1); ok {
		t.Error("a nil radar knows no internationals")
	}
	if radar.Best(5) != nil {
		t.Error("a nil radar has no best players")
	}
}
