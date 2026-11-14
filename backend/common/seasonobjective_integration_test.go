package common

import (
	"fmt"
	"testing"
	"time"
)

// GetTeamSeasonSituation against a real Postgres (STRIKER_TEST_DB_DSN): a 4-team tier-2 league of a
// 3-tier country, its table and the club's unplayed league matches.
func TestGetTeamSeasonSituation(t *testing.T) {
	db, dbCtx := openEngineLocksTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	country := fmt.Sprintf("s%03d", time.Now().UnixNano()%1000)
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 3) ON CONFLICT (name) DO UPDATE SET current_season = 3`, country)
	scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'Top', 1) RETURNING league_id`, country)
	league := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'Second', 2) RETURNING league_id`, country)
	scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'Third', 3) RETURNING league_id`, country)
	teams := make([]int64, 4)
	for i, points := range []int{30, 25, 20, 10} {
		teams[i] = scan(`INSERT INTO teams (name, country, league_id, ai_level) VALUES ($1, $2, $3, 3) RETURNING team_id`, fmt.Sprintf("T%d", i), country, league)
		exec(`INSERT INTO classification (league_id, team_id, season, matches, points) VALUES ($1, $2, 3, 4, $3)`, league, teams[i], points)
	}
	for _, opp := range teams[1:] {
		exec(`INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id) VALUES ($1, 3, now(), $2, $3)`, league, teams[0], opp)
	}
	exec(`INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id) VALUES ($1, 3, true, now(), $2, $3)`, league, teams[1], teams[0])
	exec(`INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id) VALUES ($1, 2, now(), $2, $3)`, league, teams[2], teams[0])

	s, err := GetTeamSeasonSituation(dbCtx, teams[0])
	if err != nil {
		t.Fatal(err)
	}
	// One league per tier breaks the ratio rule, so the tier plays the legacy zones: 1 candidate
	// place and 2 relegation places.
	zones := PyramidZones(2, 3, 4, true)
	if s.Index != 0 || len(s.Points) != 4 || s.Points[0] != 30 || s.Points[3] != 10 {
		t.Errorf("table %+v at index %d, want the 4 clubs best to worst with the club first", s.Points, s.Index)
	}
	if s.Played != 4 || s.Remaining != 3 {
		t.Errorf("played %d remaining %d, want 4 and the 3 unplayed matches of this season", s.Played, s.Remaining)
	}
	if s.PromotionSlots != zones.PromotionPlaces() || s.RelegationSlots != zones.RelegationPlaces() {
		t.Errorf("slots %d/%d, want the pyramid's %d/%d in a middle tier", s.PromotionSlots, s.RelegationSlots, zones.PromotionPlaces(), zones.RelegationPlaces())
	}
	if got := TeamSeasonObjective(dbCtx, teams[0]); got != SeasonObjectiveFor(s) {
		t.Errorf("TeamSeasonObjective %q, want %q", got, SeasonObjectiveFor(s))
	}
}
