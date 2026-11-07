package engine

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/resources"
)

// regionalCupFixture seeds made-up countries for the regional international cup tests (see
// openspec/specs/international-cup). Every test uses a region name of its own, passed straight to
// startRegionalCups, so countries and cups left by earlier runs in the persistent test database
// never take part.
type regionalCupFixture struct {
	t      *testing.T
	db     *sql.DB
	dbCtx  *common.DatabaseContext
	region string
	tier1  map[string]int64
}

func newRegionalCupFixture(t *testing.T) *regionalCupFixture {
	db, dbCtx := openReplicaTestDb(t)
	return &regionalCupFixture{t: t, db: db, dbCtx: dbCtx, region: fmt.Sprintf("test-%d", time.Now().UnixNano()%1e12), tier1: map[string]int64{}}
}

func (f *regionalCupFixture) scan(query string, args ...any) int64 {
	f.t.Helper()
	var v int64
	if err := f.db.QueryRow(query, args...).Scan(&v); err != nil {
		f.t.Fatalf("%s: %s", query, err)
	}
	return v
}

func (f *regionalCupFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(query, args...); err != nil {
		f.t.Fatalf("%s: %s", query, err)
	}
}

// country creates a fresh country with one tier-1 league and returns its code.
func (f *regionalCupFixture) country() string {
	f.t.Helper()
	code := uniqueCountryCode()
	for f.tier1[code] != 0 {
		code = uniqueCountryCode()
	}
	f.exec(`INSERT INTO countries (name) VALUES ($1) ON CONFLICT DO NOTHING`, code)
	f.tier1[code] = f.scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, code)
	return code
}

// team adds a team to country's tier-1 league.
func (f *regionalCupFixture) team(country, name string) int64 {
	f.t.Helper()
	return f.scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, country+" "+name, country, f.tier1[country])
}

// champion records teamId as country's tier-1 champion of season.
func (f *regionalCupFixture) champion(country string, season int, teamId int64) {
	f.t.Helper()
	f.exec(`INSERT INTO season_history (team_id, season, league_id, final_position) VALUES ($1, $2, $3, 1)`, teamId, season, f.tier1[country])
}

func (f *regionalCupFixture) run(countries ...string) {
	startRegionalCups(f.dbCtx, func() (map[string][]string, error) {
		return map[string][]string{f.region: countries}, nil
	})
}

func (f *regionalCupFixture) cupCount() int64 {
	return f.scan(`SELECT count(*) FROM cups WHERE region = $1`, f.region)
}

func (f *regionalCupFixture) latestCup() (cupId int64, edition int64) {
	f.t.Helper()
	cupId = f.scan(`SELECT cup_id FROM cups WHERE region = $1 ORDER BY cup_id DESC LIMIT 1`, f.region)
	return cupId, f.scan(`SELECT season FROM cups WHERE cup_id = $1`, cupId)
}

func (f *regionalCupFixture) entrants(cupId int64) map[int64]bool {
	f.t.Helper()
	ties, err := common.GetCupTies(f.dbCtx, cupId)
	if err != nil {
		f.t.Fatalf("GetCupTies: %s", err)
	}
	teams := make(map[int64]bool)
	for _, tie := range ties {
		if tie.Round != 1 {
			continue
		}
		if tie.HomeTeamId != nil {
			teams[*tie.HomeTeamId] = true
		}
		if tie.AwayTeamId != nil {
			teams[*tie.AwayTeamId] = true
		}
	}
	return teams
}

func (f *regionalCupFixture) entries(cupId int64) map[string]common.InternationalCupEntry {
	f.t.Helper()
	list, err := common.GetInternationalCupEntries(f.dbCtx, cupId)
	if err != nil {
		f.t.Fatalf("GetInternationalCupEntries: %s", err)
	}
	byCountry := make(map[string]common.InternationalCupEntry, len(list))
	for _, e := range list {
		byCountry[e.Country] = e
	}
	return byCountry
}

// TestRegionalCup_EditionsFollowNewChampions walks one region through two editions: it waits for
// its last country, takes each country's season_history champion (not whoever now leads the
// reset live table), does not start a second edition while the first is running, and then stops
// waiting for a slow country once another has finished two seasons since it last entered.
func TestRegionalCup_EditionsFollowNewChampions(t *testing.T) {
	f := newRegionalCupFixture(t)
	es, fr, it := f.country(), f.country(), f.country()
	esChampion, esStale := f.team(es, "champion"), f.team(es, "stale-leader")
	frChampion := f.team(fr, "champion")
	itChampion := f.team(it, "champion")
	f.champion(es, 1, esChampion)
	f.champion(fr, 1, frChampion)
	// es has moved on to season 2: its live table shows another team on top.
	f.exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 2, 30)`, f.tier1[es], esStale)
	f.exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 2, 10)`, f.tier1[es], esChampion)

	// Waiting for the last country: it is still playing its first season.
	f.run(es, fr, it)
	if n := f.cupCount(); n != 0 {
		t.Fatalf("%d editions started while %s had no champion yet, want 0", n, it)
	}

	// Every country has a new champion.
	f.champion(it, 1, itChampion)
	f.run(es, fr, it)
	cupId, edition := f.latestCup()
	if edition != 1 {
		t.Errorf("first edition numbered %d, want 1", edition)
	}
	got := f.entrants(cupId)
	for _, team := range []int64{esChampion, frChampion, itChampion} {
		if !got[team] {
			t.Errorf("champion %d missing from the first edition's entrants %v", team, got)
		}
	}
	if got[esStale] {
		t.Errorf("live-table leader %d must not enter (entrants %v)", esStale, got)
	}
	if e := f.entries(cupId); len(e) != 3 || e[es].Season != 1 || e[es].TeamId != esChampion {
		t.Errorf("first edition entries %v, want es/fr/it season 1", e)
	}

	// Edition still running: nothing new however many champions appear.
	esSecond := f.team(es, "second")
	f.champion(es, 2, esSecond)
	f.champion(es, 3, esChampion)
	f.champion(fr, 2, frChampion)
	f.run(es, fr, it)
	if n := f.cupCount(); n != 1 {
		t.Fatalf("%d editions while the first is running, want 1", n)
	}

	// Once it is over, es (two titles since it entered) does not wait for it, which has not finished
	// season 2: es enters with its season-3 champion, fr with its season-2 one, it stays out.
	f.exec(`UPDATE cups SET completed = true WHERE cup_id = $1`, cupId)
	f.run(es, fr, it)
	cupId, edition = f.latestCup()
	if edition != 2 {
		t.Fatalf("second edition numbered %d, want 2", edition)
	}
	got = f.entrants(cupId)
	if len(got) != 2 || !got[esChampion] || !got[frChampion] || got[esSecond] || got[itChampion] {
		t.Errorf("second edition entrants %v, want es champion %d and fr champion %d", got, esChampion, frChampion)
	}
	if e := f.entries(cupId); len(e) != 2 || e[es].Season != 3 || e[fr].Season != 2 {
		t.Errorf("second edition entries %v, want es season 3 and fr season 2", e)
	}
}

// TestCreateRegionalCup_FailureLeavesNothing: a write that fails part-way (here an entrant whose
// team does not exist) must not leave a cup row behind, or the region would look busy for ever.
func TestCreateRegionalCup_FailureLeavesNothing(t *testing.T) {
	f := newRegionalCupFixture(t)
	es := f.country()
	team := f.team(es, "champion")
	entrants := []common.RegionalCupCountryState{
		{Country: es, ChampionSeason: 1, ChampionTeamId: team},
		{Country: "zzzz", ChampionSeason: 1, ChampionTeamId: -1},
	}
	ties := []common.RegionalCupTieSeed{{HomeTeamId: team}}
	if _, _, err := common.CreateRegionalCup(f.dbCtx, f.region, "Test Cup", entrants, ties, time.Now()); err == nil {
		t.Fatalf("CreateRegionalCup with a missing team succeeded, want an error")
	}
	if n := f.cupCount(); n != 0 {
		t.Fatalf("%d cups left behind after a failed write, want 0", n)
	}
}

// TestInternationalCupEntriesMigration_BackfillsExistingEditions: an edition created before
// international_cup_entries existed gets its entrants recorded with the season each one last won
// its league, and running the migration again changes nothing.
func TestInternationalCupEntriesMigration_BackfillsExistingEditions(t *testing.T) {
	f := newRegionalCupFixture(t)
	es, fr := f.country(), f.country()
	esChampion, frChampion := f.team(es, "champion"), f.team(fr, "champion")
	f.champion(es, 2, esChampion)
	f.champion(es, 3, esChampion)
	f.champion(fr, 1, frChampion)
	cupId := f.scan(`INSERT INTO cups (country, region, season, name) VALUES (NULL, $1, 5, 'Old Cup') RETURNING cup_id`, f.region)
	f.exec(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, played) VALUES ($1, 1, 1, $2, $3, false)`, cupId, esChampion, frChampion)

	migration, err := resources.SQLFiles.ReadFile("sql/init103_international_cup_entries.sql")
	if err != nil {
		t.Fatalf("reading migration: %s", err)
	}
	f.exec(string(migration))
	f.exec(string(migration))

	e := f.entries(cupId)
	if len(e) != 2 || e[es].Season != 3 || e[es].TeamId != esChampion || e[fr].Season != 1 || e[fr].TeamId != frChampion {
		t.Fatalf("backfilled entries %v, want es season 3 and fr season 1", e)
	}
}
