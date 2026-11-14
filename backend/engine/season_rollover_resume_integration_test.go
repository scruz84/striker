package engine

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/scruz84/striker/common"
)

// A rollover that stops halfway resumes at the next step on the following tick instead of running
// again from the start (issue #401, openspec/specs/engine-replicas "Resumable season rollover").
// Needs STRIKER_TEST_DB_DSN.

const resumeTeamsPerLeague = 6

type resumeFixture struct {
	db      *sql.DB
	dbCtx   *common.DatabaseContext
	country string
	leagues []int64
	players map[int64]int // player id -> contract years left before the rollover
}

func newResumeFixture(t *testing.T) *resumeFixture {
	t.Helper()
	db, dbCtx := openReplicaTestDb(t)
	f := &resumeFixture{db: db, dbCtx: dbCtx, country: uniqueCountryCode(), players: map[int64]int{}}
	f.exec(t, `INSERT INTO countries (name, current_season) VALUES ($1, 1)`, f.country)
	for tier := 1; tier <= 2; tier++ {
		league := f.scan(t, `INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, $2, $3) RETURNING league_id`,
			f.country, fmt.Sprintf("D%d", tier), tier)
		f.leagues = append(f.leagues, league)
		teamIds := make([]int64, resumeTeamsPerLeague)
		for i := range teamIds {
			teamIds[i] = f.scan(t, `INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`,
				fmt.Sprintf("T%d-%d", league, i), f.country, league)
			f.exec(t, `INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 1, $3)`, league, teamIds[i], 30-i)
		}
		f.exec(t, `INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
			VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true)`, league, teamIds[0], teamIds[1])
		if tier == 1 {
			common.CreatePlayers(dbCtx, teamIds[0], f.country)
			// Young enough that nobody retires, contracts long enough that nobody leaves. A season
			// takes one year off every contract (age moves on birthdays, not at the rollover).
			f.exec(t, `UPDATE players SET age = 24, contract_years_remaining = 3 WHERE team_id = $1`, teamIds[0])
			rows, err := db.Query(`SELECT player_id, contract_years_remaining FROM players WHERE team_id = $1`, teamIds[0])
			if err != nil {
				t.Fatal(err)
			}
			for rows.Next() {
				var id int64
				var years int
				if err := rows.Scan(&id, &years); err != nil {
					t.Fatal(err)
				}
				f.players[id] = years
			}
			rows.Close()
		}
	}
	return f
}

func (f *resumeFixture) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(query, args...); err != nil {
		t.Fatalf("%s: %s", query, err)
	}
}

func (f *resumeFixture) scan(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var v int64
	if err := f.db.QueryRow(query, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %s", query, err)
	}
	return v
}

// rolloverStoppingAfter runs the rollover once, stopped right after step.
func (f *resumeFixture) rolloverStoppingAfter(t *testing.T, step int) {
	t.Helper()
	rolloverInterruptAfter = func(done int) bool { return done == step }
	defer func() { rolloverInterruptAfter = nil }()
	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
}

// checkRolledOverOnce asserts that every effect of the rollover happened exactly once.
func (f *resumeFixture) checkRolledOverOnce(t *testing.T) {
	t.Helper()
	if season := f.scan(t, `SELECT current_season FROM countries WHERE name = $1`, f.country); season != 2 {
		t.Errorf("current season %d, want 2", season)
	}
	if step := f.scan(t, `SELECT rollover_step FROM countries WHERE name = $1`, f.country); step != 0 {
		t.Errorf("rollover marker still at step %d, want cleared", step)
	}
	if len(f.players) == 0 {
		t.Fatal("the fixture has no players")
	}
	for id, before := range f.players {
		var after int
		if err := f.db.QueryRow(`SELECT contract_years_remaining FROM players WHERE player_id = $1`, id).Scan(&after); err != nil {
			t.Fatalf("player %d: %s", id, err)
		}
		if after != before-1 {
			t.Errorf("player %d contract went from %d to %d years, want one season off", id, before, after)
		}
	}
	if prizes := f.scan(t, `SELECT count(*) FROM team_transactions tt JOIN teams t ON t.team_id = tt.team_id
		WHERE t.country = $1 AND tt.reason = $2`, f.country, common.TRANSACTION_PRIZE_MONEY); prizes == 0 {
		t.Fatal("no prize money was paid; the test needs prizes to mean anything")
	}
	if dup := f.scan(t, `SELECT count(*) FROM (SELECT tt.team_id FROM team_transactions tt JOIN teams t ON t.team_id = tt.team_id
		WHERE t.country = $1 GROUP BY tt.team_id, tt.reason, tt.description HAVING count(*) > 1) d`, f.country); dup != 0 {
		t.Errorf("%d team/ledger lines were booked twice", dup)
	}
	if cups := f.scan(t, `SELECT count(*) FROM cups WHERE country = $1 AND season = 2`, f.country); cups != 1 {
		t.Errorf("%d cups for season 2, want 1", cups)
	}
	if history := f.scan(t, `SELECT count(*) FROM season_history sh JOIN teams t ON t.team_id = sh.team_id WHERE t.country = $1 AND sh.season = 1`, f.country); history != 2*resumeTeamsPerLeague {
		t.Errorf("%d season-1 history rows, want %d", history, 2*resumeTeamsPerLeague)
	}
	want := int64(resumeTeamsPerLeague * (resumeTeamsPerLeague - 1))
	for _, league := range f.leagues {
		if got := f.scan(t, `SELECT count(*) FROM matches WHERE league_id = $1 AND season = 2`, league); got != want {
			t.Errorf("league %d season 2 has %d fixtures, want %d", league, got, want)
		}
	}
}

func TestSeasonRollover_ResumesAfterThePrizes(t *testing.T) {
	f := newResumeFixture(t)

	f.rolloverStoppingAfter(t, rolloverStepPromotionApplied)
	if season := f.scan(t, `SELECT current_season FROM countries WHERE name = $1`, f.country); season != 1 {
		t.Fatalf("season %d after the stop, want still 1", season)
	}
	if step := f.scan(t, `SELECT rollover_step FROM countries WHERE name = $1`, f.country); step != rolloverStepPromotionApplied {
		t.Fatalf("marker at step %d after the stop, want %d", step, rolloverStepPromotionApplied)
	}

	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	f.checkRolledOverOnce(t)
	// The starting order was built from the stored standings and moves: every team has a seed.
	if seeded := f.scan(t, `SELECT count(*) FROM classification c JOIN teams t ON t.team_id = c.team_id
		WHERE t.country = $1 AND c.season = 2 AND c.seed IS NOT NULL`, f.country); seeded != 2*resumeTeamsPerLeague {
		t.Errorf("%d teams seeded for season 2, want %d", seeded, 2*resumeTeamsPerLeague)
	}
}

func TestSeasonRollover_GeneratesFixturesAfterAStopPastTheSeasonBump(t *testing.T) {
	f := newResumeFixture(t)

	f.rolloverStoppingAfter(t, rolloverStepSeasonAdvanced)
	if season := f.scan(t, `SELECT current_season FROM countries WHERE name = $1`, f.country); season != 2 {
		t.Fatalf("season %d after the stop, want 2", season)
	}
	if fixtures := f.scan(t, `SELECT count(*) FROM matches WHERE league_id = ANY($1) AND season = 2`, f.leagues); fixtures != 0 {
		t.Fatalf("%d season-2 fixtures before the resume, want none", fixtures)
	}

	advanceCountrySeasonIfComplete(f.dbCtx, f.country)
	f.checkRolledOverOnce(t)
}

// Every step boundary: wherever the rollover stops, the next run finishes it with each effect once.
func TestSeasonRollover_ResumesFromEveryStep(t *testing.T) {
	for step := rolloverStepPlayersProgressed; step <= rolloverStepSeasonAdvanced; step++ {
		t.Run(fmt.Sprintf("after step %d", step), func(t *testing.T) {
			f := newResumeFixture(t)
			f.rolloverStoppingAfter(t, step)
			advanceCountrySeasonIfComplete(f.dbCtx, f.country)
			f.checkRolledOverOnce(t)
		})
	}
}
