package engine

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// Integration tests of the league pyramid (openspec/specs/league-pyramid): the promotion playoff,
// the rollover that waits for it, a restructure, and the staggered league kickoffs. They run
// against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset), each in its own fresh country.

type pyramidFixture struct {
	t       *testing.T
	db      *sql.DB
	dbCtx   *common.DatabaseContext
	country string
	// leagues[tier-1][group-1] is a league id; teams[leagueId] its teams best to worst.
	leagues [][]int64
	teams   map[int64][]int64
}

func openPyramidDb(t *testing.T) (*sql.DB, *common.DatabaseContext) {
	t.Helper()
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	dbCtx := &common.DatabaseContext{Db: db, ObjectCache: common.NewRedisCache()}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	return db, dbCtx
}

func (f *pyramidFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(query, args...); err != nil {
		f.t.Fatalf("%s: %s", query, err)
	}
}

func (f *pyramidFixture) scan(query string, args ...any) int64 {
	f.t.Helper()
	var n int64
	if err := f.db.QueryRow(query, args...).Scan(&n); err != nil {
		f.t.Fatalf("%s: %s", query, err)
	}
	return n
}

// newPyramidCountry lays out a fresh country with groupsPerTier groups of teamsPerGroup teams,
// season 1 finished: every team has its final points (position p of group g on 100 - 5p - g, so
// groups rank 1 above 2 on equal positions) and every league one played match outside the
// live-broadcast window.
func newPyramidCountry(t *testing.T, teamsPerGroup int, groupsPerTier ...int) *pyramidFixture {
	db, dbCtx := openPyramidDb(t)
	f := &pyramidFixture{t: t, db: db, dbCtx: dbCtx, country: uniqueCountryCode(), teams: make(map[int64][]int64)}
	f.exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`, f.country)
	names := []string{"I", "II", "III", "IV", "V"}
	for i, groups := range groupsPerTier {
		tier := i + 1
		row := make([]int64, 0, groups)
		for g := 1; g <= groups; g++ {
			league := f.scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, $2, $3, $4) RETURNING league_id`,
				f.country, g, fmt.Sprintf("%s.%d", names[i], g), tier)
			row = append(row, league)
			ids := make([]int64, teamsPerGroup)
			for p := 1; p <= teamsPerGroup; p++ {
				ids[p-1] = f.scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`,
					fmt.Sprintf("T%d.%d.%d", tier, g, p), f.country, league)
				f.exec(`INSERT INTO classification (league_id, team_id, season, matches, points, goals_scored, goals_received)
					VALUES ($1, $2, 1, 14, $3, 20, 10)`, league, ids[p-1], 100-5*p-g)
			}
			f.teams[league] = ids
			f.exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
				VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true)`, league, ids[0], ids[1])
		}
		f.leagues = append(f.leagues, row)
	}
	return f
}

func (f *pyramidFixture) season() int {
	f.t.Helper()
	season, err := common.GetCurrentSeasonForCountry(f.dbCtx, f.country)
	if err != nil {
		f.t.Fatal(err)
	}
	return season
}

func (f *pyramidFixture) leagueOf(teamId int64) int64 {
	return f.scan(`SELECT league_id FROM teams WHERE team_id = $1`, teamId)
}

func (f *pyramidFixture) playoffTies() []common.CupTie {
	f.t.Helper()
	state, err := common.GetCountryPlayoff(f.dbCtx, f.country, 1)
	if err != nil || !state.Exists {
		f.t.Fatalf("no playoff for %s: %+v %v", f.country, state, err)
	}
	ties, err := common.GetCupTies(f.dbCtx, state.CupId)
	if err != nil {
		f.t.Fatal(err)
	}
	return ties
}

// settleTie records a tie's result as if it had been simulated and broadcast ago long ago.
func (f *pyramidFixture) settleTie(tie common.CupTie, winner int64, broadcastAgo time.Duration) {
	f.exec(`UPDATE cup_ties SET played = true, home_goals = 1, away_goals = 0, winner_team_id = $2, match_date = $3 WHERE tie_id = $1`,
		tie.TieId, winner, time.Now().Add(-broadcastAgo))
}

// A 1/2-group country of 8-team groups: the league ends, the playoff is created once, the
// rollover waits for it, and its outcome moves teams (spec scenarios "Playoff phase", "Rollover
// waits for the playoff", "Playoff outcome", "Season history marks playoff moves").
func TestPyramid_PlayoffFullFlow(t *testing.T) {
	f := newPyramidCountry(t, 8, 1, 2)
	top, groupA, groupB := f.leagues[0][0], f.leagues[1][0], f.leagues[1][1]
	topTeams := f.teams[top]

	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 1 {
		t.Fatal("the first tick after the league should create the playoff, not roll over")
	}
	ties := f.playoffTies()
	if len(ties) != 2 {
		t.Fatalf("expected 2 playoff ties (tier 1's 5th and 6th vs the two runners-up), got %d", len(ties))
	}
	// Worst upper team (6th) against the best runner-up (group A's, on more points), at home.
	if *ties[0].HomeTeamId != topTeams[5] || *ties[0].AwayTeamId != f.teams[groupA][1] ||
		*ties[1].HomeTeamId != topTeams[4] || *ties[1].AwayTeamId != f.teams[groupB][1] {
		t.Fatalf("pairings %d-%d, %d-%d", *ties[0].HomeTeamId, *ties[0].AwayTeamId, *ties[1].HomeTeamId, *ties[1].AwayTeamId)
	}
	// The day after the last league match, at tier 1's kickoff (14:00 local; UTC for a test code).
	for _, tie := range ties {
		if tie.MatchDate == nil || tie.MatchDate.UTC().Format("15:04") != "14:00" || !tie.MatchDate.After(time.Now()) {
			t.Errorf("tie %d kicks off at %v, want a future 14:00", tie.TieId, tie.MatchDate)
		}
	}

	// A second tick neither duplicates the playoff nor rolls over while the ties are unplayed.
	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if n := f.scan(`SELECT count(*) FROM cups WHERE country = $1 AND kind = 'playoff'`, f.country); n != 1 || f.season() != 1 {
		t.Fatalf("%d playoffs, season %d; want 1 and 1", n, f.season())
	}
	// The cup job never advances or crowns a playoff.
	AdvanceCups(f.dbCtx)
	if n := f.scan(`SELECT count(*) FROM cup_ties ct JOIN cups c ON c.cup_id = ct.cup_id WHERE c.country = $1 AND c.kind = 'playoff'`, f.country); n != 2 {
		t.Fatalf("the cup job touched the playoff: %d ties", n)
	}

	// Tie 1: the lower team wins. Tie 2: the upper team wins but is still broadcasting.
	f.settleTie(ties[0], *ties[0].AwayTeamId, 3*time.Hour)
	f.settleTie(ties[1], *ties[1].HomeTeamId, 30*time.Minute)
	// The league page lists each league's ties, with a result only once its broadcast is over
	// (league-page "Playoff section").
	details, err := common.GetDivisionDetails(f.dbCtx, top)
	if err != nil {
		t.Fatal(err)
	}
	if len(details.Playoff) != 2 {
		t.Fatalf("tier 1's page lists %d playoff ties, want 2", len(details.Playoff))
	}
	for _, tie := range details.Playoff {
		switch tie.TieId {
		case ties[0].TieId:
			if !tie.Played || tie.WinnerTeamId == nil {
				t.Error("a tie whose broadcast is over should show its result")
			}
		case ties[1].TieId:
			if tie.Played || tie.WinnerTeamId != nil || tie.HomeGoals != 0 {
				t.Error("a tie still broadcasting must not show its result")
			}
		}
	}
	if groupDetails, err := common.GetDivisionDetails(f.dbCtx, groupB); err != nil || len(groupDetails.Playoff) != 1 {
		t.Errorf("group B's page should list its one tie, got %v (%v)", groupDetails, err)
	}
	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 1 {
		t.Fatal("the season rolled over while a playoff tie was still broadcasting")
	}
	f.settleTie(ties[1], *ties[1].HomeTeamId, 3*time.Hour)
	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	if f.season() != 2 {
		t.Fatalf("season %d after the playoff, want 2", f.season())
	}

	// Direct moves: both group winners up, tier 1's 7th and 8th down.
	for _, id := range []int64{f.teams[groupA][0], f.teams[groupB][0]} {
		if f.leagueOf(id) != top {
			t.Errorf("group winner %d not promoted", id)
		}
	}
	for _, id := range topTeams[6:] {
		if f.leagueOf(id) == top {
			t.Errorf("tier-1 team %d (7th/8th) not relegated", id)
		}
	}
	// Playoff: tie 1's teams swapped, tie 2's stayed.
	if f.leagueOf(*ties[0].AwayTeamId) != top || f.leagueOf(*ties[0].HomeTeamId) != groupA {
		t.Errorf("tie 1 (away win) did not swap the teams")
	}
	if f.leagueOf(*ties[1].HomeTeamId) != top || f.leagueOf(*ties[1].AwayTeamId) != groupB {
		t.Errorf("tie 2 (home win) moved a team")
	}
	if n := f.scan(`SELECT count(*) FROM teams WHERE league_id = $1`, top); n != 8 {
		t.Errorf("tier 1 has %d teams, want 8", n)
	}
	if details, err := common.GetDivisionDetails(f.dbCtx, top); err != nil || len(details.Playoff) != 0 {
		t.Errorf("the new season's page should show no playoff, got %d ties (%v)", len(details.Playoff), err)
	}
	if n := f.scan(`SELECT count(*) FROM cups WHERE country = $1 AND kind = 'playoff' AND completed`, f.country); n != 1 {
		t.Error("the playoff was not completed at rollover")
	}
	viaPlayoff := func(teamId int64) (bool, bool, bool) {
		var promoted, relegated, via bool
		if err := f.db.QueryRow(`SELECT promoted, relegated, via_playoff FROM season_history WHERE team_id = $1 AND season = 1`, teamId).
			Scan(&promoted, &relegated, &via); err != nil {
			t.Fatal(err)
		}
		return promoted, relegated, via
	}
	if p, r, v := viaPlayoff(*ties[0].AwayTeamId); !p || r || !v {
		t.Errorf("tie 1 winner history = promoted %v relegated %v via %v", p, r, v)
	}
	if p, r, v := viaPlayoff(*ties[0].HomeTeamId); p || !r || !v {
		t.Errorf("tie 1 loser history = promoted %v relegated %v via %v", p, r, v)
	}
	if p, r, v := viaPlayoff(*ties[1].HomeTeamId); p || r || v {
		t.Errorf("tie 2 winner (stayed) history = promoted %v relegated %v via %v", p, r, v)
	}
	if p, r, v := viaPlayoff(f.teams[groupA][0]); !p || r || v {
		t.Errorf("direct promotion history = promoted %v relegated %v via %v", p, r, v)
	}
}

// Three replicas see the league end at once: exactly one playoff with one set of ties, and each
// owned team gets one qualification notification (engine-replicas "Playoff created once").
func TestPyramid_PlayoffCreatedOnceAcrossReplicas(t *testing.T) {
	f := newPyramidCountry(t, 8, 1, 2)
	runnerUp := f.teams[f.leagues[1][0]][1]
	user := f.scan(`INSERT INTO users (name, provider, email) VALUES ('playoff', 'dev', $1) RETURNING user_id`,
		fmt.Sprintf("playoff-%d@example.test", time.Now().UnixNano()))
	f.exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, user, runnerUp)

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			advanceCountrySeasonIfComplete(f.dbCtx, f.country)
		}()
	}
	wg.Wait()

	if n := f.scan(`SELECT count(*) FROM cups WHERE country = $1 AND kind = 'playoff'`, f.country); n != 1 {
		t.Fatalf("%d playoffs, want 1", n)
	}
	if len(f.playoffTies()) != 2 {
		t.Fatal("want one set of 2 ties")
	}
	if n := f.scan(`SELECT count(*) FROM notifications WHERE user_id = $1 AND message_key = 'notifications.messages.playoffQualified'`, user); n != 1 {
		t.Fatalf("the runner-up's owner got %d qualification notifications, want 1", n)
	}
}

// A legacy 1/4 country of 8-team groups rolls over into the 1/2/4 x 8 pyramid: no playoff, the
// legacy swap (tier 1's 7th and 8th for the best 2 of the 4 winners), then the restructure - groups
// II.3 and II.4 archived, III.1..III.4 created, 16 CPU teams added, fixtures only for active
// leagues, each tier kicking off 15 minutes after the one above (spec "Legacy pyramid season",
// "Restructure ordering", "Restructure fills groups and tops up", "Restructure keeps league
// history"; match-scheduling "League created by a restructure"). A second run is a no-op, and
// three concurrent rollovers restructure once.
func TestPyramid_RestructureLegacyCountry(t *testing.T) {
	target := []common.TierConfig{
		{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8},
		{Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 8},
		{Tier: 3, Name: "III", Groups: 4, TeamsPerLeague: 8},
	}
	previous := restructureTarget
	restructureTarget = func(*common.DatabaseContext, string) []common.TierConfig { return target }
	t.Cleanup(func() { restructureTarget = previous })

	f := newPyramidCountry(t, 8, 1, 4)
	top := f.leagues[0][0]
	oldTier2 := f.leagues[1]

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			advanceCountrySeasonIfComplete(f.dbCtx, f.country)
		}()
	}
	wg.Wait()
	if f.season() != 2 {
		t.Fatalf("season %d, want 2 (a legacy country plays no playoff)", f.season())
	}
	if n := f.scan(`SELECT count(*) FROM cups WHERE country = $1 AND kind = 'playoff'`, f.country); n != 0 {
		t.Errorf("a legacy country got %d playoffs", n)
	}

	// Legacy swap: the two best winners (groups 1 and 2 on equal positions) up, 7th and 8th down.
	if f.leagueOf(f.teams[oldTier2[0]][0]) != top || f.leagueOf(f.teams[oldTier2[1]][0]) != top {
		t.Error("the best two tier-2 winners were not promoted")
	}
	if f.leagueOf(f.teams[oldTier2[2]][0]) == top {
		t.Error("the third-best winner was promoted")
	}

	shape, err := common.GetCountryPyramidShape(f.dbCtx, f.country)
	if err != nil {
		t.Fatal(err)
	}
	if !shape.MatchesConfig(target) {
		t.Fatalf("after the restructure the country is %+v", shape.Leagues)
	}
	if n := f.scan(`SELECT count(*) FROM leagues WHERE country = $1 AND archived_season = 1`, f.country); n != 2 {
		t.Errorf("%d leagues archived, want II.3 and II.4", n)
	}
	for _, archived := range oldTier2[2:] {
		if n := f.scan(`SELECT count(*) FROM leagues WHERE league_id = $1 AND archived_season = 1`, archived); n != 1 {
			t.Errorf("league %d (old II.3/II.4) not archived", archived)
		}
	}
	if n := f.scan(`SELECT count(*) FROM teams WHERE country = $1 AND league_id IS NOT NULL`, f.country); n != 56 {
		t.Errorf("%d teams, want 40 + 16 new", n)
	}
	if n := f.scan(`SELECT count(*) FROM leagues WHERE country = $1 AND tier = 3 AND archived_season IS NULL`, f.country); n != 4 {
		t.Errorf("%d tier-3 leagues, want 4", n)
	}
	// Fixtures only for active leagues, staggered by tier.
	if n := f.scan(`SELECT count(*) FROM matches m JOIN leagues l ON l.league_id = m.league_id
		WHERE l.country = $1 AND m.season = 2 AND l.archived_season IS NOT NULL`, f.country); n != 0 {
		t.Errorf("archived leagues got %d new fixtures", n)
	}
	for tier, want := range map[int]string{1: "14:00", 2: "14:15", 3: "14:30"} {
		var kickoff time.Time
		if err := f.db.QueryRow(`SELECT min(m.match_date) FROM matches m JOIN leagues l ON l.league_id = m.league_id
			WHERE l.country = $1 AND l.tier = $2 AND m.season = 2`, f.country, tier).Scan(&kickoff); err != nil {
			t.Fatalf("tier %d has no season-2 fixtures: %s", tier, err)
		}
		if got := kickoff.UTC().Format("15:04"); got != want {
			t.Errorf("tier %d kicks off at %s, want %s", tier, got, want)
		}
	}

	// A rerun after a crash before the top-up finished creates only what is still missing: take
	// three teams out of one tier-3 group and the next pass puts the country back in shape with
	// exactly three new teams.
	shortGroup := f.scan(`SELECT league_id FROM leagues WHERE country = $1 AND tier = 3 AND archived_season IS NULL ORDER BY group_number DESC LIMIT 1`, f.country)
	f.exec(`UPDATE teams SET league_id = NULL WHERE team_id IN (SELECT team_id FROM teams WHERE league_id = $1 ORDER BY team_id DESC LIMIT 3)`, shortGroup)
	_, _ = restructureIfNeeded(f.dbCtx, f.country, 2, nil)
	if n := f.scan(`SELECT count(*) FROM teams WHERE league_id = $1`, shortGroup); n != 8 {
		t.Errorf("the short group has %d teams after the rerun, want 8", n)
	}
	if n := f.scan(`SELECT count(*) FROM teams WHERE country = $1 AND league_id IS NOT NULL`, f.country); n != 56 {
		t.Errorf("%d teams in the pyramid after the rerun, want 56", n)
	}

	// Already in shape: a second restructure changes nothing.
	before := f.scan(`SELECT count(*) FROM teams WHERE country = $1`, f.country)
	if moves, _ := restructureIfNeeded(f.dbCtx, f.country, 2, nil); moves != nil {
		t.Errorf("a country already in shape was restructured again: %v", moves)
	}
	if after := f.scan(`SELECT count(*) FROM teams WHERE country = $1`, f.country); after != before {
		t.Errorf("teams %d -> %d on a no-op restructure", before, after)
	}
}

// Spec scenario (match-scheduling) "Admin changes a country's kickoff": moving a country's kickoff
// re-dates its unplayed matches without an override, tier by tier, and leaves an override alone.
func TestPyramid_CountryKickoffChangeReschedules(t *testing.T) {
	f := newPyramidCountry(t, 4, 1, 2)
	f.exec(`DELETE FROM matches WHERE league_id IN (SELECT league_id FROM leagues WHERE country = $1)`, f.country)
	leagueIds, err := common.GetLeagueIdsForCountry(f.dbCtx, f.country)
	if err != nil {
		t.Fatal(err)
	}
	override := f.leagues[1][1]
	if err := common.SetLeagueSchedule(f.dbCtx, override, common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 18, Minute: 0}); err != nil {
		t.Fatal(err)
	}
	if err := GenerateMatchesForLeagues(f.dbCtx, leagueIds); err != nil {
		t.Fatal(err)
	}
	utcKickoff := func(league int64) string {
		var d time.Time
		if err := f.db.QueryRow(`SELECT min(match_date) FROM matches WHERE league_id = $1 AND played = false`, league).Scan(&d); err != nil {
			t.Fatalf("league %d: %s", league, err)
		}
		return d.UTC().Format("15:04")
	}
	if got := utcKickoff(f.leagues[0][0]); got != "14:00" {
		t.Errorf("tier 1 at %s, want 14:00", got)
	}
	if got := utcKickoff(f.leagues[1][0]); got != "14:15" {
		t.Errorf("tier 2 at %s, want 14:15", got)
	}
	overrideBefore := utcKickoff(override)

	if err := common.SetCountryLeagueKickoff(f.dbCtx, f.country, 15*60, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := RescheduleUnplayedMatchesForCountry(f.dbCtx, f.country); err != nil {
		t.Fatal(err)
	}
	if got := utcKickoff(f.leagues[0][0]); got != "15:00" {
		t.Errorf("tier 1 at %s after the change, want 15:00", got)
	}
	if got := utcKickoff(f.leagues[1][0]); got != "15:15" {
		t.Errorf("tier 2 at %s after the change, want 15:15", got)
	}
	if got := utcKickoff(override); got != overrideBefore {
		t.Errorf("the league with an override moved from %s to %s", overrideBefore, got)
	}
}

// A legacy 1/4 country of 8-team groups (40 teams) with its own 1/2 x 8 structure (24 places)
// rolls over: the 16 worst-ranked CPU teams are dissolved, a user's team that would have been cut
// keeps a place in tier 2 instead, no CPU team is created, and the season-2 fixtures leave the
// dissolved teams out (spec "Removing the bottom tier", "User team at the bottom", "Country with
// its own structure"). Five of them were already dissolved by an interrupted rollover, three
// replicas roll over at once, and the total is still 16.
func TestPyramid_RestructureDissolvesSurplus(t *testing.T) {
	previous := restructureTarget
	restructureTarget = func(dbCtx *common.DatabaseContext, country string) []common.TierConfig {
		return common.EffectiveStructure(dbCtx, country)
	}
	t.Cleanup(func() { restructureTarget = previous })

	f := newPyramidCountry(t, 8, 1, 4)
	small := []common.TierConfig{{Tier: 1, Name: "I", Groups: 1, TeamsPerLeague: 8}, {Tier: 2, Name: "II", Groups: 2, TeamsPerLeague: 8}}
	if err := common.SetCountryStructure(f.dbCtx, f.country, small); err != nil {
		t.Fatal(err)
	}
	// The last team of the last tier-2 group belongs to a user.
	lastGroup := f.leagues[1][3]
	userTeam := f.teams[lastGroup][7]
	user := f.scan(`INSERT INTO users (name, provider, email) VALUES ('shrink', 'dev', $1) RETURNING user_id`,
		fmt.Sprintf("shrink-%d@test", time.Now().UnixNano()))
	f.exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, user, userTeam)
	// An interrupted earlier attempt already dissolved five of the worst teams of that group.
	for _, teamId := range f.teams[lastGroup][2:7] {
		if ok, _, err := common.DissolveTeam(f.dbCtx, teamId, 1); err != nil || !ok {
			t.Fatalf("pre-dissolving %d: %v %v", teamId, ok, err)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			advanceCountrySeasonIfComplete(f.dbCtx, f.country)
		}()
	}
	wg.Wait()
	if f.season() != 2 {
		t.Fatalf("season %d, want 2", f.season())
	}
	if n := f.scan(`SELECT count(*) FROM teams WHERE country = $1 AND dissolved_season = 1 AND league_id = -1`, f.country); n != 16 {
		t.Errorf("%d teams dissolved, want 16", n)
	}
	if n := f.scan(`SELECT count(*) FROM teams t JOIN leagues l ON l.league_id = t.league_id WHERE t.country = $1 AND l.archived_season IS NULL`, f.country); n != 24 {
		t.Errorf("%d teams left in the active leagues, want 24", n)
	}
	if n := f.scan(`SELECT count(*) FROM teams WHERE country = $1`, f.country); n != 40 {
		t.Errorf("%d team rows, want the 40 kept (no CPU team created, none deleted)", n)
	}
	if tier := f.scan(`SELECT l.tier FROM teams t JOIN leagues l ON l.league_id = t.league_id WHERE t.team_id = $1`, userTeam); tier != 2 {
		t.Errorf("the user's team plays in tier %d, want 2", tier)
	}
	shape, err := common.GetCountryPyramidShape(f.dbCtx, f.country)
	if err != nil {
		t.Fatal(err)
	}
	if !shape.MatchesConfig(small) {
		t.Fatalf("after the restructure the country is %+v", shape.Leagues)
	}
	if n := f.scan(`SELECT count(*) FROM matches m JOIN teams t ON t.team_id IN (m.home_team_id, m.away_team_id)
		WHERE m.season = 2 AND t.country = $1 AND t.dissolved_season IS NOT NULL`, f.country); n != 0 {
		t.Errorf("dissolved teams got %d season-2 fixtures", n)
	}
	// Already in shape: another pass dissolves nothing more.
	if moves, _ := restructureIfNeeded(f.dbCtx, f.country, 2, nil); moves != nil {
		t.Errorf("a country already in shape was restructured again: %v", moves)
	}
}
