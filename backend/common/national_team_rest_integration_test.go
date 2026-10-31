package common

import (
	"testing"
	"time"
)

// TestNationalTeamFriendlyEarliestKickoff: the later of now and each side's previous kickoff plus
// NationalTeamFriendlyRest; a side that never played imposes nothing.
func TestNationalTeamFriendlyEarliestKickoff(t *testing.T) {
	now := time.Date(2026, 10, 31, 20, 50, 0, 0, time.UTC)
	if got := nationalTeamFriendlyEarliestKickoff(now, time.Time{}, time.Time{}); !got.Equal(now) {
		t.Errorf("no previous kickoffs: got %v, want now", got)
	}
	nigeria := time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC)
	saoTome := time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	if got, want := nationalTeamFriendlyEarliestKickoff(now, saoTome, nigeria), saoTome.Add(NationalTeamFriendlyRest); !got.Equal(want) {
		t.Errorf("got %v, want %v (the later side's rest boundary)", got, want)
	}
	// With São Tomé's home timezone (UTC+0) that boundary lands on the next Saturday, a week out.
	kickoff := NextLocalSaturdayKickoffAfter("st", 21, 0, nationalTeamFriendlyEarliestKickoff(now, saoTome, nigeria))
	if want := time.Date(2026, 11, 7, 21, 0, 0, 0, time.UTC); !kickoff.Equal(want) {
		t.Errorf("re-paired kickoff = %v, want %v", kickoff, want)
	}
}

// TestScheduleNationalTeamFriendlies_RespectsRest reproduces the reported double-booking: two
// countries whose friendly is simulated (played = true) 14 minutes before its kickoff are free
// again for the pairing pass, which used to give them a second friendly the same evening. After
// the pass, any new fixture of theirs must start at least NationalTeamFriendlyRest after that
// kickoff. Runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when unset).
func TestScheduleNationalTeamFriendlies_RespectsRest(t *testing.T) {
	db, dbCtx := openEngineLocksTestDb(t)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	// Friendlies are not scheduled while a World Cup runs; one left active by another test in this
	// shared database is paused for the duration of this test and restored afterwards.
	activeWorldCups, err := GetActiveWorldCupIds(dbCtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(activeWorldCups) > 0 {
		exec(`UPDATE world_cups SET completed = true WHERE world_cup_id = ANY($1)`, activeWorldCups)
		t.Cleanup(func() {
			db.Exec(`UPDATE world_cups SET completed = false WHERE world_cup_id = ANY($1)`, activeWorldCups)
		})
	}

	countries := []string{"rz1", "rz2"}
	teamIds := make([]int64, 0, 2)
	for _, country := range countries {
		exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country)
		teamId, err := ensureNationalTeam(dbCtx, country)
		if err != nil {
			t.Fatal(err)
		}
		teamIds = append(teamIds, teamId)
	}
	exec(`DELETE FROM friendly_matches WHERE home_team_id = ANY($1) OR away_team_id = ANY($1)`, teamIds)

	kickoff := time.Now().Add(14 * time.Minute).Truncate(time.Second)
	exec(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, played, home_goals, away_goals) VALUES ($1, $2, $3, true, 1, 0)`,
		teamIds[0], teamIds[1], kickoff)

	if err := ScheduleNationalTeamFriendlies(dbCtx); err != nil {
		t.Fatalf("ScheduleNationalTeamFriendlies: %s", err)
	}

	rows, err := db.Query(`SELECT match_date FROM friendly_matches WHERE played = false AND (home_team_id = ANY($1) OR away_team_id = ANY($1))`, teamIds)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	scheduled := 0
	for rows.Next() {
		var next time.Time
		if err := rows.Scan(&next); err != nil {
			t.Fatal(err)
		}
		scheduled++
		if next.Before(kickoff.Add(NationalTeamFriendlyRest)) {
			t.Errorf("new friendly kicks off %v, only %v after the previous one (want at least %v)", next, next.Sub(kickoff), NationalTeamFriendlyRest)
		}
	}
	if scheduled == 0 {
		t.Errorf("expected the pass to re-pair at least one of %v", countries)
	}
}
