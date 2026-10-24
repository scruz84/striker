package common

import (
	"testing"
	"time"
)

// TestWeekMatches_OneLeagueWeekAndSeason checks the query behind GET /division/:id/weeks/:week
// returns only the requested matchday of the requested season of one league: a week past the
// last one is empty (never nil), a previous season's row for the same week never leaks in, and
// another league's matches never do either.
func TestWeekMatches_OneLeagueWeekAndSeason(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "lw")
	home := seedTransfersTeam(t, db, "Week Home FC", "lw")
	var leagueId, away, otherHome int64
	if err := db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, home).Scan(&leagueId); err != nil {
		t.Fatalf("reading league: %s", err)
	}
	if err := db.QueryRow(`INSERT INTO teams (name, country, league_id, budget) VALUES ('Week Away FC', 'lw', $1, 1) RETURNING team_id`, leagueId).Scan(&away); err != nil {
		t.Fatalf("seeding away team: %s", err)
	}
	otherHome = seedTransfersTeam(t, db, "Week Other FC", "lw")
	var otherLeagueId int64
	if err := db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, otherHome).Scan(&otherLeagueId); err != nil {
		t.Fatalf("reading other league: %s", err)
	}

	// A season number far from the other integration tests' data (those use season 1).
	const season = 700
	date := time.Now().Truncate(time.Second)
	insert := func(league, h, a int64, s, week int) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING match_id`, league, s, week, date, h, a).Scan(&id); err != nil {
			t.Fatalf("inserting match: %s", err)
		}
		return id
	}
	wanted := insert(leagueId, home, away, season, 1)
	insert(leagueId, away, home, season, 2)
	insert(leagueId, home, away, season-1, 1) // previous season, same week
	insert(otherLeagueId, otherHome, otherHome, season, 1)

	got, err := WeekMatches(dbCtx, leagueId, season, 1)
	if err != nil {
		t.Fatalf("WeekMatches: %s", err)
	}
	if got == nil || len(*got) != 1 || (*got)[0].MatchId != wanted {
		t.Errorf("week 1 of season %d: got %+v, want only match %d", season, got, wanted)
	}

	past, err := WeekMatches(dbCtx, leagueId, season, 31)
	if err != nil || past == nil || len(*past) != 0 {
		t.Errorf("week past the last one: got %v, %v; want an empty non-nil slice", past, err)
	}
}
