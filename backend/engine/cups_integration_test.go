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

// runnerUp records teamId as country's tier-1 runner-up of season.
func (f *regionalCupFixture) runnerUp(country string, season int, teamId int64) {
	f.t.Helper()
	f.exec(`INSERT INTO season_history (team_id, season, league_id, final_position) VALUES ($1, $2, $3, 2)`, teamId, season, f.tier1[country])
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

// entries returns cupId's recorded entrants by team id.
func (f *regionalCupFixture) entries(cupId int64) map[int64]common.InternationalCupEntry {
	f.t.Helper()
	list, err := common.GetInternationalCupEntries(f.dbCtx, cupId)
	if err != nil {
		f.t.Fatalf("GetInternationalCupEntries: %s", err)
	}
	byTeam := make(map[int64]common.InternationalCupEntry, len(list))
	for _, e := range list {
		byTeam[e.TeamId] = e
	}
	return byTeam
}

// firstRoundTies returns a cup's round-1 ties.
func (f *regionalCupFixture) firstRoundTies(cupId int64) []common.CupTie {
	f.t.Helper()
	ties, err := common.GetCupTies(f.dbCtx, cupId)
	if err != nil {
		f.t.Fatalf("GetCupTies: %s", err)
	}
	round1 := make([]common.CupTie, 0, len(ties))
	for _, tie := range ties {
		if tie.Round == 1 {
			round1 = append(round1, tie)
		}
	}
	return round1
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
	if e := f.entries(cupId); len(e) != 3 || e[esChampion].Season != 1 || e[esChampion].Country != es || e[esChampion].FinalPosition != 1 {
		t.Errorf("first edition entries %v, want the es/fr/it season-1 champions (no runner-up recorded)", e)
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
	if e := f.entries(cupId); len(e) != 2 || e[esChampion].Season != 3 || e[frChampion].Season != 2 {
		t.Errorf("second edition entries %v, want es season 3 and fr season 2", e)
	}
}

// TestRegionalCup_TopTwoOfEachCountry covers international-cup "Top two of each country" and
// "Two countries": each fresh country sends its latest season's champion and runner-up (not an
// older season's runner-up), both are recorded with their position, and round 1 pairs each
// champion with the other country's runner-up.
func TestRegionalCup_TopTwoOfEachCountry(t *testing.T) {
	f := newRegionalCupFixture(t)
	es, fr := f.country(), f.country()
	esChampion, esRunnerUp, esOldRunnerUp := f.team(es, "champion"), f.team(es, "runner-up"), f.team(es, "old runner-up")
	frChampion, frRunnerUp := f.team(fr, "champion"), f.team(fr, "runner-up")
	f.champion(es, 1, esChampion)
	f.runnerUp(es, 1, esOldRunnerUp)
	f.champion(es, 2, esChampion)
	f.runnerUp(es, 2, esRunnerUp)
	f.champion(fr, 1, frChampion)
	f.runnerUp(fr, 1, frRunnerUp)

	f.run(es, fr)
	cupId, _ := f.latestCup()
	e := f.entries(cupId)
	want := map[int64]common.InternationalCupEntry{
		esChampion: {Country: es, Season: 2, TeamId: esChampion, FinalPosition: 1},
		esRunnerUp: {Country: es, Season: 2, TeamId: esRunnerUp, FinalPosition: 2},
		frChampion: {Country: fr, Season: 1, TeamId: frChampion, FinalPosition: 1},
		frRunnerUp: {Country: fr, Season: 1, TeamId: frRunnerUp, FinalPosition: 2},
	}
	if len(e) != len(want) {
		t.Fatalf("entries %v, want the top two of es and fr", e)
	}
	for team, entry := range want {
		if e[team] != entry {
			t.Errorf("entry of team %d = %+v, want %+v", team, e[team], entry)
		}
	}

	ties := f.firstRoundTies(cupId)
	if len(ties) != 2 {
		t.Fatalf("%d first-round ties, want 2", len(ties))
	}
	opponents := map[int64]int64{}
	for _, tie := range ties {
		if tie.HomeTeamId == nil || tie.AwayTeamId == nil {
			t.Fatalf("first-round tie %+v has a bye, want none with 4 teams", tie)
		}
		opponents[*tie.HomeTeamId] = *tie.AwayTeamId
	}
	if opponents[esChampion] != frRunnerUp || opponents[frChampion] != esRunnerUp {
		t.Errorf("first round %v, want es champion vs fr runner-up and fr champion vs es runner-up", opponents)
	}
}

// TestCreateRegionalCup_FailureLeavesNothing: a write that fails part-way (here an entrant whose
// team does not exist) must not leave a cup row behind, or the region would look busy for ever.
func TestCreateRegionalCup_FailureLeavesNothing(t *testing.T) {
	f := newRegionalCupFixture(t)
	es := f.country()
	team := f.team(es, "champion")
	entrants := []common.RegionalCupEntrant{
		{Country: es, Season: 1, TeamId: team, FinalPosition: 1},
		{Country: "zzzz", Season: 1, TeamId: -1, FinalPosition: 1},
	}
	ties := []common.RegionalCupTieSeed{{HomeTeamId: team}}
	if _, _, err := common.CreateRegionalCup(f.dbCtx, f.region, "Test Cup", entrants, ties); err == nil {
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

	migration, err := resources.SQLFiles.ReadFile("sql/history/init103_international_cup_entries.sql")
	if err != nil {
		t.Fatalf("reading migration: %s", err)
	}
	runnersUp, err := resources.SQLFiles.ReadFile("sql/history/init112_international_cup_runners_up.sql")
	if err != nil {
		t.Fatalf("reading migration: %s", err)
	}
	// Every startup re-applies both files, in order.
	for i := 0; i < 2; i++ {
		f.exec(string(migration))
		f.exec(string(runnersUp))
	}

	e := f.entries(cupId)
	if len(e) != 2 || e[esChampion].Season != 3 || e[esChampion].Country != es || e[frChampion].Season != 1 || e[frChampion].Country != fr {
		t.Fatalf("backfilled entries %v, want es season 3 and fr season 1", e)
	}
	for team, entry := range e {
		if entry.FinalPosition != 1 {
			t.Errorf("backfilled entry of team %d has final position %d, want 1 (a champion)", team, entry.FinalPosition)
		}
	}
	if n := f.scan(`SELECT count(*) FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
		WHERE c.conrelid = 'international_cup_entries'::regclass AND c.contype = 'p' AND a.attname = 'team_id'`); n != 1 {
		t.Errorf("international_cup_entries primary key does not include team_id")
	}
}
