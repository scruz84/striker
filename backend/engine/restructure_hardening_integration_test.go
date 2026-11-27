package engine

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// Integration tests of a restructure that hits failures part-way (openspec/specs/league-pyramid
// "Restructure leaves no team behind", "Restructure result is checked"). They run against a real
// Postgres (STRIKER_TEST_DB_DSN; skipped when unset), each in its own fresh country.

// useEffectiveStructure makes restructures follow each country's effective structure, as in
// production (engine/main_test.go switches them off by default).
func useEffectiveStructure(t *testing.T) {
	previous := restructureTarget
	restructureTarget = func(dbCtx *common.DatabaseContext, country string) []common.TierConfig {
		return common.EffectiveStructure(dbCtx, country)
	}
	t.Cleanup(func() { restructureTarget = previous })
}

// withDissolveTeam swaps the restructure's dissolution for hook for the rest of the test.
func withDissolveTeam(t *testing.T, hook func(dbCtx *common.DatabaseContext, teamId int64, season int) (bool, []common.PlayerLoan, error)) {
	previous := dissolveTeam
	dissolveTeam = hook
	t.Cleanup(func() { dissolveTeam = previous })
}

// withCreateLeague swaps the restructure's league creation for hook for the rest of the test.
func withCreateLeague(t *testing.T, hook func(dbCtx *common.DatabaseContext, country string, tierName string, tier int, group int) (int64, error)) {
	previous := createLeague
	createLeague = hook
	t.Cleanup(func() { createLeague = previous })
}

// assertCountryInShape fails unless the country's active leagues are exactly target.
func (f *pyramidFixture) assertCountryInShape(target []common.TierConfig) {
	f.t.Helper()
	shape, err := common.GetCountryPyramidShape(f.dbCtx, f.country)
	if err != nil {
		f.t.Fatal(err)
	}
	if diffs := shape.Differences(target); len(diffs) != 0 {
		f.t.Fatalf("the country differs from its structure: %+v (leagues %+v)", diffs, shape.Leagues)
	}
}

func (f *pyramidFixture) tierOf(teamId int64) int64 {
	return f.scan(`SELECT l.tier FROM teams t JOIN leagues l ON l.league_id = t.league_id WHERE t.team_id = $1`, teamId)
}

func (f *pyramidFixture) dissolvedCount() int64 {
	return f.scan(`SELECT count(*) FROM teams WHERE country = $1 AND dissolved_season IS NOT NULL`, f.country)
}

var structure12x8 = []common.TierConfig{
	{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8},
	{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 8},
}

var structure124x8 = []common.TierConfig{
	{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8},
	{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 8},
	{Tier: 3, Name: "III", Groups: 4, TeamsPerLeague: 8},
}

// Spec scenario "Club claimed during the restructure", on a 1/4 x 8 country (40 teams) shrinking
// to its own 1/2 x 8 structure (24 places): the first CPU club the plan dissolves gets a user just
// before its dissolution, so it plays next season in tier 2, another CPU club is dissolved in its
// place, and every group has 8 teams.
func TestRestructure_ClubClaimedDuringTheRestructure(t *testing.T) {
	useEffectiveStructure(t)
	f := newPyramidCountry(t, 8, 1, 4)
	if err := common.SetCountryStructure(f.dbCtx, f.country, structure12x8); err != nil {
		t.Fatal(err)
	}
	user := f.scan(`INSERT INTO users (name, provider, email) VALUES ('claim', 'dev', $1) RETURNING user_id`,
		fmt.Sprintf("claim-%d@test", time.Now().UnixNano()))
	claimed := int64(-1)
	withDissolveTeam(t, func(dbCtx *common.DatabaseContext, teamId int64, season int) (bool, []common.PlayerLoan, error) {
		if claimed < 0 {
			claimed = teamId
			f.exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, user, teamId)
		}
		return common.DissolveTeam(dbCtx, teamId, season)
	})

	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 2 {
		t.Fatalf("season %d, want 2", f.season())
	}
	if claimed < 0 {
		t.Fatal("the restructure dissolved nothing")
	}
	if tier := f.tierOf(claimed); tier != 2 {
		t.Errorf("the claimed club plays in tier %d, want 2", tier)
	}
	if n := f.dissolvedCount(); n != 16 {
		t.Errorf("%d clubs dissolved, want 16", n)
	}
	f.assertCountryInShape(structure12x8)
}

// Spec scenario "Dissolution error": dissolving one planned CPU club fails, so it is kept like a
// protected one, another CPU club is dissolved instead, and the old league it was in is archived.
func TestRestructure_DissolutionErrorDissolvesAnother(t *testing.T) {
	useEffectiveStructure(t)
	f := newPyramidCountry(t, 8, 1, 4)
	if err := common.SetCountryStructure(f.dbCtx, f.country, structure12x8); err != nil {
		t.Fatal(err)
	}
	failing := int64(-1)
	var failingLeague int64
	withDissolveTeam(t, func(dbCtx *common.DatabaseContext, teamId int64, season int) (bool, []common.PlayerLoan, error) {
		if failing < 0 {
			failing = teamId
			failingLeague = f.leagueOf(teamId)
		}
		if teamId == failing {
			return false, nil, errors.New("injected dissolution failure")
		}
		return common.DissolveTeam(dbCtx, teamId, season)
	})

	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 2 {
		t.Fatalf("season %d, want 2", f.season())
	}
	if failing < 0 {
		t.Fatal("the restructure dissolved nothing")
	}
	if n := f.scan(`SELECT count(*) FROM teams WHERE team_id = $1 AND dissolved_season IS NULL`, failing); n != 1 {
		t.Error("the club whose dissolution failed was dissolved anyway")
	}
	if tier := f.tierOf(failing); tier != 2 {
		t.Errorf("the club whose dissolution failed plays in tier %d, want 2", tier)
	}
	if n := f.dissolvedCount(); n != 16 {
		t.Errorf("%d clubs dissolved, want 16", n)
	}
	if n := f.scan(`SELECT count(*) FROM leagues WHERE league_id = $1 AND archived_season IS NOT NULL`, failingLeague); n != 1 {
		t.Errorf("league %d the failing club played in is still active", failingLeague)
	}
	f.assertCountryInShape(structure12x8)
}

// Spec scenario "Leftover groups from an earlier restructure", with groups of 8: a 1/2/3 country
// whose structure is 1/2/4 plays legacy (no playoff), and its rollover creates III.4, deals the 24
// tier-3 teams into four groups, adds 8 new CPU teams and leaves 1/2/4 groups of 8.
func TestRestructure_RepairsLeftoverGroups(t *testing.T) {
	useEffectiveStructure(t)
	f := newPyramidCountry(t, 8, 1, 2, 3)
	if err := common.SetCountryStructure(f.dbCtx, f.country, structure124x8); err != nil {
		t.Fatal(err)
	}
	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if n := f.scan(`SELECT count(*) FROM cups WHERE country = $1 AND kind = 'playoff'`, f.country); n != 0 {
		t.Errorf("a 1/2/3 country got %d playoffs, want none", n)
	}
	if f.season() != 2 {
		t.Fatalf("season %d, want 2", f.season())
	}
	f.assertCountryInShape(structure124x8)
	if n := f.scan(`SELECT count(*) FROM teams WHERE country = $1 AND dissolved_season IS NULL`, f.country); n != 56 {
		t.Errorf("%d teams, want 56 (48 + 8 new CPU teams)", n)
	}
	if n := f.scan(`SELECT count(*) FROM leagues WHERE country = $1 AND name = 'III.4' AND archived_season IS NULL`, f.country); n != 1 {
		t.Error("III.4 was not created")
	}
}

// withPlayers gives the tier-1 champion a squad young enough that nobody retires, with 3 years
// left on every contract, and returns the squad's player ids. A rollover takes one year off.
func (f *pyramidFixture) withPlayers() []int64 {
	f.t.Helper()
	team := f.teams[f.leagues[0][0]][0]
	common.CreatePlayers(f.dbCtx, team, f.country)
	f.exec(`UPDATE players SET age = 24, contract_years_remaining = 3 WHERE team_id = $1`, team)
	rows, err := f.db.Query(`SELECT player_id FROM players WHERE team_id = $1`, team)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			f.t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		f.t.Fatal("no players created")
	}
	return ids
}

// assertOneSeasonOff fails unless every player lost exactly one contract year (aged once).
func (f *pyramidFixture) assertOneSeasonOff(players []int64) {
	f.t.Helper()
	for _, id := range players {
		if years := f.scan(`SELECT contract_years_remaining FROM players WHERE player_id = $1`, id); years != 2 {
			f.t.Errorf("player %d has %d contract years left, want 2 (one rollover)", id, years)
		}
	}
}

func (f *pyramidFixture) rolloverStep() int64 {
	return f.scan(`SELECT rollover_step FROM countries WHERE name = $1`, f.country)
}

// failingCreateLeague makes creating a league fail the first failures times, then work.
func failingCreateLeague(t *testing.T, failures int) *int {
	calls := 0
	var mu sync.Mutex
	withCreateLeague(t, func(dbCtx *common.DatabaseContext, country string, tierName string, tier int, group int) (int64, error) {
		mu.Lock()
		calls++
		fail := failures < 0 || calls <= failures
		mu.Unlock()
		if fail {
			return 0, errors.New("injected league creation failure")
		}
		return common.CreateLeague(dbCtx, country, tierName, tier, group)
	})
	return &calls
}

// Spec scenario "Transient failure": creating III.4 fails on the first attempt, so the rollover
// stops before the cup with the restructure still to do; the next tick finishes it, and the season
// advances once with 1/2/4 groups.
func TestRestructure_TransientFailureIsRetried(t *testing.T) {
	useEffectiveStructure(t)
	f := newPyramidCountry(t, 8, 1, 2, 3)
	if err := common.SetCountryStructure(f.dbCtx, f.country, structure124x8); err != nil {
		t.Fatal(err)
	}
	players := f.withPlayers()
	failingCreateLeague(t, 1)

	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 1 {
		t.Fatalf("season %d after a failed restructure, want still 1", f.season())
	}
	if step := f.rolloverStep(); step != rolloverStepPromotionApplied {
		t.Fatalf("rollover marker at step %d, want %d (restructure still to do)", step, rolloverStepPromotionApplied)
	}
	if n := f.scan(`SELECT count(*) FROM cups WHERE country = $1 AND season = 2`, f.country); n != 0 {
		t.Errorf("%d season-2 cups created before the restructure finished", n)
	}

	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 2 {
		t.Fatalf("season %d after the retry, want 2", f.season())
	}
	f.assertCountryInShape(structure124x8)
	f.assertOneSeasonOff(players)
	if step := f.rolloverStep(); step != 0 {
		t.Errorf("rollover marker still at step %d, want cleared", step)
	}
	// A further tick changes nothing.
	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 2 {
		t.Errorf("season %d after a further tick, want 2", f.season())
	}
}

// Spec scenario "Persistent failure": creating III.4 always fails. The first two ticks keep the
// restructure pending; the third gives up, the season advances, every league gets its fixtures,
// and the country is still listed as pending.
func TestRestructure_PersistentFailureAdvancesAfterThreeAttempts(t *testing.T) {
	useEffectiveStructure(t)
	f := newPyramidCountry(t, 8, 1, 2, 3)
	if err := common.SetCountryStructure(f.dbCtx, f.country, structure124x8); err != nil {
		t.Fatal(err)
	}
	players := f.withPlayers()
	failingCreateLeague(t, -1)

	for tick := 1; tick < maxRestructureAttempts; tick++ {
		advanceCountrySeasonIfComplete(f.dbCtx, f.country)
		if f.season() != 1 {
			t.Fatalf("season %d after failed attempt %d, want still 1", f.season(), tick)
		}
	}
	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 2 {
		t.Fatalf("season %d after %d failed attempts, want 2", f.season(), maxRestructureAttempts)
	}
	f.assertOneSeasonOff(players)
	if n := f.scan(`SELECT count(*) FROM leagues l WHERE l.country = $1 AND l.archived_season IS NULL
		AND NOT EXISTS (SELECT 1 FROM matches m WHERE m.league_id = l.league_id AND m.season = 2)`, f.country); n != 0 {
		t.Errorf("%d active leagues have no season-2 fixtures", n)
	}
	pending, err := common.GetPendingRestructureCountries(f.dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range pending {
		if p.Country == f.country {
			found = true
		}
	}
	if !found {
		t.Errorf("%s is not listed as pending after giving up", f.country)
	}
}

// Three replicas roll over at once on the transient-failure path, then again on the next tick:
// the restructure is finished once, the season advances by exactly 1 and one cup is created
// (openspec/specs/engine-replicas; league-pyramid "Restructure result is checked").
func TestRestructure_RetryIsSafeWithConcurrentReplicas(t *testing.T) {
	useEffectiveStructure(t)
	f := newPyramidCountry(t, 8, 1, 2, 3)
	if err := common.SetCountryStructure(f.dbCtx, f.country, structure124x8); err != nil {
		t.Fatal(err)
	}
	players := f.withPlayers()
	failingCreateLeague(t, 1)

	for tick := 1; tick <= 2; tick++ {
		var wg sync.WaitGroup
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				advanceCountrySeasonIfComplete(f.dbCtx, f.country)
			}()
		}
		wg.Wait()
	}
	if f.season() != 2 {
		t.Fatalf("season %d, want 2", f.season())
	}
	f.assertCountryInShape(structure124x8)
	f.assertOneSeasonOff(players)
	if n := f.scan(`SELECT count(*) FROM cups WHERE country = $1 AND season = 2 AND kind = 'cup'`, f.country); n != 1 {
		t.Errorf("%d season-2 cups, want 1", n)
	}
	if n := f.scan(`SELECT count(*) FROM leagues WHERE country = $1 AND archived_season IS NULL AND tier = 3`, f.country); n != 4 {
		t.Errorf("%d active tier-3 groups, want 4", n)
	}
}
