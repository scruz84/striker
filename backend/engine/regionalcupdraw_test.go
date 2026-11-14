package engine

import (
	"math/rand"
	"testing"

	"github.com/scruz84/striker/common"
)

// topTwo builds a country's champion (team id base+1) and runner-up (base+2).
func topTwo(country string, base int64) []common.RegionalCupEntrant {
	return []common.RegionalCupEntrant{
		{Country: country, Season: 1, TeamId: base + 1, FinalPosition: 1},
		{Country: country, Season: 1, TeamId: base + 2, FinalPosition: 2},
	}
}

func drawEntrants(countries ...string) []common.RegionalCupEntrant {
	entrants := make([]common.RegionalCupEntrant, 0, 2*len(countries))
	for i, country := range countries {
		entrants = append(entrants, topTwo(country, int64(i+1)*100)...)
	}
	return entrants
}

// checkDraw verifies the invariants every draw keeps: each entrant appears exactly once, no tie has
// two teams of one country, and the bracket has the expected byes.
func checkDraw(t *testing.T, entrants []common.RegionalCupEntrant, pairs [][2]int64, wantByes int) {
	t.Helper()
	country := map[int64]string{}
	for _, e := range entrants {
		country[e.TeamId] = e.Country
	}
	seen := map[int64]int{}
	byes := 0
	for _, pair := range pairs {
		seen[pair[0]]++
		if pair[1] == ByeTeamId {
			byes++
			continue
		}
		seen[pair[1]]++
		if country[pair[0]] == country[pair[1]] {
			t.Errorf("tie %v pairs two teams of %s", pair, country[pair[0]])
		}
	}
	if byes != wantByes {
		t.Errorf("%d byes, want %d (draw %v)", byes, wantByes, pairs)
	}
	for _, e := range entrants {
		if seen[e.TeamId] != 1 {
			t.Errorf("team %d appears %d times in the draw %v", e.TeamId, seen[e.TeamId], pairs)
		}
	}
}

func TestRegionalCupFirstRound_TwoCountries(t *testing.T) {
	entrants := drawEntrants("es", "fr")
	for seed := int64(0); seed < 50; seed++ {
		pairs := regionalCupFirstRound(entrants, nil, rand.New(rand.NewSource(seed)))
		checkDraw(t, entrants, pairs, 0)
		opponents := map[int64]int64{}
		for _, pair := range pairs {
			opponents[pair[0]] = pair[1]
		}
		// es: 101 champion, 102 runner-up; fr: 201, 202. Each champion hosts the other's runner-up.
		if opponents[101] != 202 || opponents[201] != 102 {
			t.Fatalf("seed %d: draw %v, want 101 vs 202 and 201 vs 102", seed, pairs)
		}
	}
}

func TestRegionalCupFirstRound_ThreeCountries(t *testing.T) {
	entrants := drawEntrants("es", "fr", "it")
	// The es and fr champions lead their leagues; the it champion has fewer points.
	points := map[int64]int{101: 12, 201: 9, 301: 3, 102: 20, 202: 20, 302: 20}
	for seed := int64(0); seed < 50; seed++ {
		pairs := regionalCupFirstRound(entrants, points, rand.New(rand.NewSource(seed)))
		checkDraw(t, entrants, pairs, 2)
		if len(pairs) != 4 {
			t.Fatalf("seed %d: %d slots, want 4", seed, len(pairs))
		}
		// Byes in slots 1 and 3 (bye, tie, bye, tie), to the two champions with the most points, so
		// they meet round-1 winners in round 2, not each other.
		if pairs[0] != [2]int64{101, ByeTeamId} || pairs[2] != [2]int64{201, ByeTeamId} {
			t.Fatalf("seed %d: draw %v, want byes for 101 then 201 in slots 1 and 3", seed, pairs)
		}
		for _, slot := range []int{1, 3} {
			if pairs[slot][1] == ByeTeamId {
				t.Fatalf("seed %d: slot %d is a bye in %v, want a real tie", seed, slot+1, pairs)
			}
		}
		// The it champion plays a runner-up of another country.
		for _, pair := range pairs {
			if pair[0] == 301 || pair[1] == 301 {
				other := pair[0] + pair[1] - 301
				if other != 102 && other != 202 {
					t.Fatalf("seed %d: it champion plays %d, want the es or fr runner-up", seed, other)
				}
			}
		}
	}
}

func TestRegionalCupFirstRound_FourCountries(t *testing.T) {
	entrants := drawEntrants("es", "fr", "it", "pt")
	for seed := int64(0); seed < 50; seed++ {
		pairs := regionalCupFirstRound(entrants, nil, rand.New(rand.NewSource(seed)))
		checkDraw(t, entrants, pairs, 0)
		for _, pair := range pairs {
			if pair[0]%100 != 1 || pair[1]%100 != 2 {
				t.Fatalf("seed %d: tie %v is not a champion at home against a runner-up", seed, pair)
			}
		}
	}
}

func TestRegionalCupFirstRound_MissingRunnerUp(t *testing.T) {
	entrants := append(topTwo("es", 100), common.RegionalCupEntrant{Country: "fr", Season: 1, TeamId: 201, FinalPosition: 1})
	points := map[int64]int{101: 5, 201: 8}
	for seed := int64(0); seed < 50; seed++ {
		pairs := regionalCupFirstRound(entrants, points, rand.New(rand.NewSource(seed)))
		checkDraw(t, entrants, pairs, 1)
		// 201 has more points, but its bye would leave 101 against its own runner-up: the bye goes
		// to 101 and 201 plays the es runner-up.
		if pairs[0] != [2]int64{101, ByeTeamId} || pairs[1] != [2]int64{201, 102} {
			t.Fatalf("seed %d: draw %v, want a bye for 101 and 201 vs 102", seed, pairs)
		}
	}
}

// With one country holding most of the field the draw still never pairs it with itself while
// another pairing exists.
func TestRegionalCupFirstRound_NoSameCountryTieWhenAvoidable(t *testing.T) {
	cases := [][]string{
		{"es", "fr", "it", "pt", "de"},
		{"es", "fr", "it", "pt", "de", "nl"},
		{"es", "fr", "it", "pt", "de", "nl", "be", "gr"},
	}
	for _, countries := range cases {
		entrants := drawEntrants(countries...)
		n := len(entrants)
		wantByes := 0
		if p := largestPowerOfTwoAtMost(n); p != n {
			wantByes = 2*p - n
		}
		for seed := int64(0); seed < 100; seed++ {
			points := map[int64]int{}
			r := rand.New(rand.NewSource(seed + 1000))
			for _, e := range entrants {
				points[e.TeamId] = r.Intn(30)
			}
			checkDraw(t, entrants, regionalCupFirstRound(entrants, points, rand.New(rand.NewSource(seed))), wantByes)
		}
	}
}
