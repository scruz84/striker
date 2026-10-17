package engine

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// TestRescheduleUnplayedMatchesForCountry runs against a real Postgres (STRIKER_TEST_DB_DSN;
// skipped when unset). Fixtures dated far further out than they should be (e.g. generated while
// the fixture-start date was computed from another country's schedule) must be pullable forward
// for one country without touching any other: the target country's unplayed rounds land on its
// league's own schedule starting from the next slot after now, keeping their round order, while a
// second country's far-future fixtures stay exactly where they were.
func TestRescheduleUnplayedMatchesForCountry(t *testing.T) {
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	defer db.Close()
	dbCtx := &common.DatabaseContext{Db: db}
	if err := common.EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}

	seed := func(country string) (leagueId int64, firstWeekDate, secondWeekDate time.Time) {
		exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO UPDATE SET current_season = 1`, country)
		leagueId = scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, country)
		teamA := scan(`INSERT INTO teams (name, country, league_id) VALUES ('A', $1, $2) RETURNING team_id`, country, leagueId)
		teamB := scan(`INSERT INTO teams (name, country, league_id) VALUES ('B', $1, $2) RETURNING team_id`, country, leagueId)
		// Both rounds dated 11 and 12 days out, on the daily 16:00 slot.
		base := time.Now().UTC().Truncate(24 * time.Hour).Add(16 * time.Hour)
		firstWeekDate = base.Add(11 * 24 * time.Hour)
		secondWeekDate = base.Add(12 * 24 * time.Hour)
		exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3, $4)`,
			leagueId, firstWeekDate, teamA, teamB)
		exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id) VALUES ($1, 1, 2, $2, $3, $4)`,
			leagueId, secondWeekDate, teamB, teamA)
		// Its own schedule, so the result doesn't depend on whatever app-wide default the shared
		// test database happens to hold.
		if err := common.SetLeagueSchedule(dbCtx, leagueId, common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}); err != nil {
			t.Fatalf("SetLeagueSchedule: %s", err)
		}
		return leagueId, firstWeekDate, secondWeekDate
	}
	leagueA, _, _ := seed("zr1")
	leagueB, firstB, secondB := seed("zr2")

	leagues, err := RescheduleUnplayedMatchesForCountry(dbCtx, "zr1")
	if err != nil || leagues != 1 {
		t.Fatalf("RescheduleUnplayedMatchesForCountry: got %d leagues, %v; want 1, nil", leagues, err)
	}

	weekDate := func(leagueId int64, week int) time.Time {
		t.Helper()
		var d time.Time
		if err := db.QueryRow(`SELECT match_date FROM matches WHERE league_id = $1 AND week_number = $2`, leagueId, week).Scan(&d); err != nil {
			t.Fatalf("reading week %d: %s", week, err)
		}
		return d
	}

	now := time.Now()
	first, second := weekDate(leagueA, 1), weekDate(leagueA, 2)
	if first.Before(now) || first.After(now.Add(25*time.Hour)) {
		t.Errorf("zr1 week 1 rescheduled to %s, want the next daily 16:00 slot within a day of now (%s)", first, now)
	}
	if !second.Equal(first.Add(24 * time.Hour)) {
		t.Errorf("zr1 week 2 rescheduled to %s, want one day after week 1 (%s)", second, first.Add(24*time.Hour))
	}

	if got := weekDate(leagueB, 1); !got.Equal(firstB) {
		t.Errorf("zr2 week 1 changed to %s, want untouched %s", got, firstB)
	}
	if got := weekDate(leagueB, 2); !got.Equal(secondB) {
		t.Errorf("zr2 week 2 changed to %s, want untouched %s", got, secondB)
	}
}
