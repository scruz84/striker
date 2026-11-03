package engine

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/common/metrictest"
)

// These tests run against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset). Each one
// fires the same engine step from several goroutines at once - what two or three --engine
// replicas ticking the same cron trigger do in production - and asserts the data it generates is
// what a single run would have produced. See openspec/specs/engine-replicas.

const replicas = 3

func openReplicaTestDb(t *testing.T) (*sql.DB, *common.DatabaseContext) {
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
	dbCtx := &common.DatabaseContext{Db: db, ObjectCache: common.NewRedisCache(), LiveHub: common.NewLiveMatchHub()}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}
	return db, dbCtx
}

// uniqueCountryCode returns a made-up four-letter country code (countries.name is varchar(4)), so
// re-running a test against the same database never trips over a previous run's rows.
func uniqueCountryCode() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := []byte("z___")
	for i := 1; i < 4; i++ {
		code[i] = byte('a' + r.Intn(26))
	}
	return string(code)
}

func runConcurrently(n int, fn func()) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}
	wg.Wait()
}

// TestAdvanceCountrySeason_ConcurrentReplicasFixturesFollowPromotionRelegation is the production
// report behind this suite: a manager played a league match this season against a team that had
// been relegated from their division at the end of the previous one. The new season's fixtures
// must be generated from the leagues as they stand after promotion/relegation, exactly once, no
// matter how many replicas run the rollover at the same instant.
func TestAdvanceCountrySeason_ConcurrentReplicasFixturesFollowPromotionRelegation(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}

	country := uniqueCountryCode()
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1)`, country)
	tier1 := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'D1', 1) RETURNING league_id`, country)
	tier2 := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'D2', 2) RETURNING league_id`, country)

	const teamsPerLeague = 6
	for _, leagueId := range []int64{tier1, tier2} {
		teamIds := make([]int64, teamsPerLeague)
		for i := range teamIds {
			teamIds[i] = scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`,
				fmt.Sprintf("T%d-%d", leagueId, i), country, leagueId)
			exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 1, $3)`, leagueId, teamIds[i], 30-i)
		}
		// The season's last match, played and well outside the live-broadcast window.
		exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
			VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true)`, leagueId, teamIds[0], teamIds[1])
	}

	runConcurrently(replicas, func() { advanceCountrySeasonIfComplete(dbCtx, country) })

	if season := scan(`SELECT current_season FROM countries WHERE name = $1`, country); season != 2 {
		t.Fatalf("current season after concurrent rollovers: got %d, want 2 (advanced exactly once)", season)
	}
	if moved := scan(`SELECT count(*) FROM season_history sh JOIN teams t ON t.team_id = sh.team_id
		WHERE t.country = $1 AND sh.season = 1 AND (sh.promoted OR sh.relegated)`, country); moved == 0 {
		t.Fatalf("no team was promoted or relegated; the test needs moves to mean anything")
	}

	// Every new-season fixture is between two teams that play in that fixture's league now.
	if bad := scan(`SELECT count(*) FROM matches m
		JOIN teams h ON h.team_id = m.home_team_id JOIN teams a ON a.team_id = m.away_team_id
		WHERE m.league_id IN ($1, $2) AND m.season = 2 AND (h.league_id <> m.league_id OR a.league_id <> m.league_id)`,
		tier1, tier2); bad != 0 {
		t.Errorf("%d season-2 fixtures involve a team that no longer plays in that league", bad)
	}
	// A double round robin of teamsPerLeague teams, generated once per league - no duplicate schedule.
	want := int64(teamsPerLeague * (teamsPerLeague - 1))
	for _, leagueId := range []int64{tier1, tier2} {
		if got := scan(`SELECT count(*) FROM matches WHERE league_id = $1 AND season = 2`, leagueId); got != want {
			t.Errorf("league %d season 2 has %d fixtures, want %d", leagueId, got, want)
		}
	}
	if cups := scan(`SELECT count(*) FROM cups WHERE country = $1 AND season = 2`, country); cups != 1 {
		t.Errorf("concurrent rollovers created %d cups for season 2, want 1", cups)
	}
}

// TestSimulateCupTies_ConcurrentReplicasSimulateEachTieOnce: GetSimulableCupTies never claimed
// the ties it returned, so every replica's CupJob simulated the same tie - two sets of events,
// and possibly a different winner written over the first.
func TestSimulateCupTies_ConcurrentReplicasSimulateEachTieOnce(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}

	country := uniqueCountryCode()
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1)`, country); err != nil {
		t.Fatalf("inserting country: %s", err)
	}
	home := scan(`INSERT INTO teams (name, country) VALUES ('Home', $1) RETURNING team_id`, country)
	away := scan(`INSERT INTO teams (name, country) VALUES ('Away', $1) RETURNING team_id`, country)
	common.CreatePlayers(dbCtx, home, country)
	common.CreatePlayers(dbCtx, away, country)
	cupId := scan(`INSERT INTO cups (country, season, name) VALUES ($1, 1, 'Cup') RETURNING cup_id`, country)
	kickoff := time.Now().Add(-2 * time.Hour)
	tieId, err := common.CreateCupTie(dbCtx, cupId, 1, 0, &home, &away, &kickoff)
	if err != nil {
		t.Fatalf("CreateCupTie: %s", err)
	}
	for _, teamId := range []int64{home, away} {
		players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
		if err != nil {
			t.Fatalf("GetPlayersFromTeamId: %s", err)
		}
		positions := common.FlattenLineup(common.SelectBestLineup(*players))
		if err := common.SetCupTieLineup(dbCtx, tieId, teamId, common.FormationBalanced, common.MENTALITY_BALANCED, positions, nil, nil); err != nil {
			t.Fatalf("SetCupTieLineup: %s", err)
		}
	}

	runConcurrently(replicas, func() { SimulateCupTies(dbCtx) })

	if played := scan(`SELECT count(*) FROM cup_ties WHERE tie_id = $1 AND played`, tieId); played != 1 {
		t.Fatalf("tie %d not played after the concurrent passes", tieId)
	}
	// Each simulation pays its winner the round's prize once; a second simulation pays it again.
	if prizes := scan(`SELECT count(*) FROM team_transactions WHERE team_id IN ($1, $2) AND reason = $3`,
		home, away, common.TRANSACTION_PRIZE_MONEY); prizes != 1 {
		t.Errorf("tie %d paid %d round prizes, want 1 (simulated exactly once)", tieId, prizes)
	}
}

// TestAdvanceWorldCup_ConcurrentReplicasStartOneKnockout: advanceWorldCup was an unguarded
// check-then-act, so two replicas could both see the group stage complete and both create a
// knockout cup for the same World Cup.
func TestAdvanceWorldCup_ConcurrentReplicasStartOneKnockout(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}

	worldCupId, err := common.CreateWorldCup(dbCtx, 9000+rand.Intn(100000), "Test World Cup")
	if err != nil {
		t.Fatalf("CreateWorldCup: %s", err)
	}
	groupId, err := common.CreateWorldCupGroup(dbCtx, worldCupId, "A")
	if err != nil {
		t.Fatalf("CreateWorldCupGroup: %s", err)
	}
	teams := make([]int64, 2)
	for i := range teams {
		country := uniqueCountryCode()
		if _, err := db.Exec(`INSERT INTO countries (name) VALUES ($1) ON CONFLICT DO NOTHING`, country); err != nil {
			t.Fatalf("inserting country: %s", err)
		}
		teams[i] = scan(`INSERT INTO teams (name, country, is_national_team) VALUES ($1, $2, true) RETURNING team_id`, "NT "+country, country)
		if err := common.AddWorldCupGroupTeam(dbCtx, groupId, teams[i]); err != nil {
			t.Fatalf("AddWorldCupGroupTeam: %s", err)
		}
	}
	matchId := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, played, home_goals, away_goals)
		VALUES ($1, $2, now() - interval '3 hours', true, 2, 1) RETURNING match_id`, teams[0], teams[1])
	if err := common.LinkWorldCupGroupMatch(dbCtx, groupId, 1, matchId); err != nil {
		t.Fatalf("LinkWorldCupGroupMatch: %s", err)
	}

	runConcurrently(replicas, func() {
		if err := advanceWorldCup(dbCtx, worldCupId); err != nil {
			t.Errorf("advanceWorldCup: %s", err)
		}
	})

	if knockouts := scan(`SELECT count(DISTINCT ct.cup_id) FROM cup_ties ct
		WHERE ct.home_team_id = ANY($1) OR ct.away_team_id = ANY($1)`, teams); knockouts != 1 {
		t.Errorf("concurrent advances created %d knockout cups for World Cup %d, want 1", knockouts, worldCupId)
	}
}

// TestStartInternationalCupsIfDue_ConcurrentReplicasStartOneRound: the due check and the season
// increment were two statements, so every replica firing the daily job at the same instant
// started its own international cup round.
func TestStartInternationalCupsIfDue_ConcurrentReplicasStartOneRound(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	if _, err := db.Exec(`INSERT INTO season SELECT WHERE NOT EXISTS (SELECT 1 FROM season)`); err != nil {
		t.Fatalf("seeding the season row: %s", err)
	}
	var before int
	if err := db.QueryRow(`UPDATE season SET last_international_cup_check = NULL RETURNING international_cup_season`).Scan(&before); err != nil {
		t.Fatalf("resetting the international cup clock: %s", err)
	}

	claimed := make(chan int, replicas)
	runConcurrently(replicas, func() {
		if season, ok, err := common.ClaimInternationalCupSeason(dbCtx, InternationalCupCadence); err != nil {
			t.Errorf("ClaimInternationalCupSeason: %s", err)
		} else if ok {
			claimed <- season
		}
	})
	close(claimed)

	if len(claimed) != 1 {
		t.Fatalf("%d replicas claimed the international cup round, want exactly 1", len(claimed))
	}
	if season := <-claimed; season != before+1 {
		t.Errorf("claimed season %d, want %d", season, before+1)
	}
}

// TestClaimRun_RecordsWonAndLostPerReplica: every replica that fires the same once-per-period job
// records its claim; exactly one wins (striker.engine.job.claims), the others lose.
func TestClaimRun_RecordsWonAndLostPerReplica(t *testing.T) {
	_, dbCtx := openReplicaTestDb(t)
	metrictest.Install()
	jobName := fmt.Sprintf("claims_test_%d", time.Now().UnixNano())
	won := attribute.String("outcome", "won")
	lost := attribute.String("outcome", "lost")
	job := common.EngineJobAttribute(jobName)

	var winners atomic.Int64
	runConcurrently(replicas, func() {
		if claimRun(dbCtx, jobName, 24*time.Hour) {
			winners.Add(1)
		}
	})

	if winners.Load() != 1 {
		t.Fatalf("%d replicas won the claim, want 1", winners.Load())
	}
	if got := metrictest.Sum(t, "striker.engine.job.claims", job, won); got != 1 {
		t.Errorf("claims won = %v, want 1", got)
	}
	if got := metrictest.Sum(t, "striker.engine.job.claims", job, lost); got != replicas-1 {
		t.Errorf("claims lost = %v, want %d", got, replicas-1)
	}
}
