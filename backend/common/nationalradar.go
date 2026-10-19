package common

import (
	"sort"
	"sync"
	"time"
)

const (
	// NationalRadarTTL is how long a snapshot of every country's national-team pool is reused
	// before it is rebuilt: the pools shift slowly (a player drops out of a country's 40 best only
	// when others outgrow him), and a CPU team consults them on every shopping pass.
	NationalRadarTTL = 15 * time.Minute
	// NationalRadarTopPerCountry is how many of each country's best players a Director's radar
	// looks at - "the best of each national team".
	NationalRadarTopPerCountry = 5
)

// NationalRadarEntry is one player of some country's national-team pool (see
// GetPlayersByBirthCountry) as of the snapshot: the player themself, his country, how many
// international caps he has, and his rank in that country's pool (0 = the country's best).
type NationalRadarEntry struct {
	Player  Player
	Country string
	Caps    int
	Rank    int
}

// NationalRadar is a snapshot of every active country's national-team pool. It answers two
// questions for CPU teams: is this player an international (and with how many caps), and who are
// the best players of every national team - including those playing in other leagues, listed or
// not - which is what a Director scouts.
type NationalRadar struct {
	entries []NationalRadarEntry
	byId    map[int64]NationalRadarEntry
}

// buildNationalRadar assembles a radar from each country's pool (in the best-first order
// GetPlayersByBirthCountry returns it) and the caps count of every player in them. Pure.
func buildNationalRadar(pools map[string][]*Player, caps map[int64]int) *NationalRadar {
	radar := &NationalRadar{byId: make(map[int64]NationalRadarEntry)}
	countries := make([]string, 0, len(pools))
	for country := range pools {
		countries = append(countries, country)
	}
	sort.Strings(countries)
	for _, country := range countries {
		for rank, p := range pools[country] {
			entry := NationalRadarEntry{Player: *p, Country: country, Caps: caps[p.Id], Rank: rank}
			radar.entries = append(radar.entries, entry)
			radar.byId[p.Id] = entry
		}
	}
	return radar
}

// NewNationalRadar builds a radar straight from country pools (best first) and caps by player
// id - what GetNationalRadar does after reading them from the database, exposed so code that
// consults a radar can be tested without one.
func NewNationalRadar(pools map[string][]*Player, caps map[int64]int) *NationalRadar {
	return buildNationalRadar(pools, caps)
}

// Lookup reports whether playerId is in a national-team pool and, if so, his entry. Safe on a nil
// radar (a CPU team that couldn't load one simply sees no internationals).
func (r *NationalRadar) Lookup(playerId int64) (NationalRadarEntry, bool) {
	if r == nil {
		return NationalRadarEntry{}, false
	}
	entry, ok := r.byId[playerId]
	return entry, ok
}

// Best returns each country's perCountry best pool players, best first within a country and
// countries in name order.
func (r *NationalRadar) Best(perCountry int) []NationalRadarEntry {
	if r == nil {
		return nil
	}
	best := make([]NationalRadarEntry, 0)
	for _, e := range r.entries {
		if e.Rank < perCountry {
			best = append(best, e)
		}
	}
	return best
}

var nationalRadarCache struct {
	sync.Mutex
	radar   *NationalRadar
	builtAt time.Time
}

// ResetNationalRadarCache drops the cached radar so the next GetNationalRadar rebuilds it - for
// tests, which change pools underneath it.
func ResetNationalRadarCache() {
	nationalRadarCache.Lock()
	defer nationalRadarCache.Unlock()
	nationalRadarCache.radar = nil
}

// GetNationalRadar returns the current radar, rebuilding it from the database when the cached one
// is older than NationalRadarTTL (see GetPlayersByBirthCountry/GetPlayerCapsBatch for what goes
// into it).
func GetNationalRadar(dbCtx *DatabaseContext) (*NationalRadar, error) {
	nationalRadarCache.Lock()
	defer nationalRadarCache.Unlock()
	if nationalRadarCache.radar != nil && time.Since(nationalRadarCache.builtAt) < NationalRadarTTL {
		return nationalRadarCache.radar, nil
	}

	countries, err := GetActiveCountries(dbCtx)
	if err != nil {
		return nil, err
	}
	pools := make(map[string][]*Player, len(countries))
	ids := make([]int64, 0)
	for _, country := range countries {
		pool, err := GetPlayersByBirthCountry(dbCtx, country)
		if err != nil {
			return nil, err
		}
		pools[country] = pool
		for _, p := range pool {
			ids = append(ids, p.Id)
		}
	}
	caps, err := GetPlayerCapsBatch(dbCtx, ids)
	if err != nil {
		return nil, err
	}
	nationalRadarCache.radar = buildNationalRadar(pools, caps)
	nationalRadarCache.builtAt = time.Now()
	return nationalRadarCache.radar, nil
}
