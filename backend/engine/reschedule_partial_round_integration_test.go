package engine

import (
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// A round that is half played when its league is rescheduled: the played match and the one being
// simulated keep their dates, the pending one moves (issue #400). Needs STRIKER_TEST_DB_DSN.
func TestRescheduleUnplayedMatches_LeavesStartedMatchesOfAHalfPlayedRound(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	country := uniqueCountryCode()
	scan(`INSERT INTO countries (name, current_season) VALUES ($1, 1) RETURNING current_season`, country)
	league := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'L1', 1) RETURNING league_id`, country)
	team := func(name string) int64 {
		return scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, name, country, league)
	}
	a, b, c, d, e, f := team("A"), team("B"), team("C"), team("D"), team("E"), team("F")
	kickoff := time.Now().Add(-5 * time.Hour).UTC().Truncate(time.Second)
	match := func(home, away int64, played, inSimulation bool) int64 {
		return scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played, in_simulation)
			VALUES ($1, 1, 1, $2, $3, $4, $5, $6) RETURNING match_id`, league, kickoff, home, away, played, inSimulation)
	}
	played := match(a, b, true, false)
	simulating := match(c, d, false, true)
	pending := match(e, f, false, false)
	if err := common.SetLeagueSchedule(dbCtx, league, common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}); err != nil {
		t.Fatalf("SetLeagueSchedule: %s", err)
	}

	if err := RescheduleUnplayedMatches(dbCtx, league, common.MatchSchedule{WeekdaysMask: common.AllWeekdaysMask, Hour: 16, Minute: 0}); err != nil {
		t.Fatalf("RescheduleUnplayedMatches: %s", err)
	}

	dateOf := func(id int64) time.Time {
		t.Helper()
		var date time.Time
		if err := db.QueryRow(`SELECT match_date FROM matches WHERE match_id = $1`, id).Scan(&date); err != nil {
			t.Fatal(err)
		}
		return date
	}
	if got := dateOf(played); !got.Equal(kickoff) {
		t.Errorf("played match moved from %s to %s", kickoff, got)
	}
	if got := dateOf(simulating); !got.Equal(kickoff) {
		t.Errorf("match in simulation moved from %s to %s", kickoff, got)
	}
	if got := dateOf(pending); !got.After(time.Now()) {
		t.Errorf("pending match dated %s, want a future slot", got)
	}
}
