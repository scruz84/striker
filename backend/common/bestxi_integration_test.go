package common

import (
	"sync"
	"testing"
	"time"
)

// TestBestXi_MatchdayCompletesOnceAcrossReplicas runs against a real Postgres (STRIKER_TEST_DB_DSN,
// skipped when unset) and pins openspec/specs/best-xi: a matchday is complete only when every
// match is played and past its replay, and replicas storing its team of the week at once store
// it exactly once.
func TestBestXi_MatchdayCompletesOnceAcrossReplicas(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	mustScan := func(dest any, query string, args ...any) {
		t.Helper()
		if err := db.QueryRow(query, args...).Scan(dest); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	country := unusedTestCountryCode(t, db, 'x')
	seedTransfersCountry(t, db, country)
	var leagueId, teamA, teamB int64
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'XiL', 1) RETURNING league_id`, country)
	mustScan(&teamA, `INSERT INTO teams (name, country, league_id) VALUES ('Xi A', $1, $2) RETURNING team_id`, country, leagueId)
	mustScan(&teamB, `INSERT INTO teams (name, country, league_id) VALUES ('Xi B', $1, $2) RETURNING team_id`, country, leagueId)

	var played, pending int64
	mustScan(&played, `INSERT INTO matches (league_id, season, week_number, played, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, false, $2, $3, $4) RETURNING match_id`, leagueId, time.Now().Add(-3*time.Hour), teamA, teamB)
	mustScan(&pending, `INSERT INTO matches (league_id, season, week_number, played, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, false, $2, $3, $4) RETURNING match_id`, leagueId, time.Now().Add(-1*time.Hour), teamB, teamA)

	// A full side each: 1 keeper, 4 defenders, 4 midfielders, 2 forwards, every player rated alike.
	positions := []int{GOALKEEPER, DEFENSE, LATERAL_DEFENSE_LEFT, LATERAL_DEFENSE_RIGHT, DEFENSE, MIDFIELD, MIDFIELD,
		LATERAL_MIDFIELD_LEFT, LATERAL_MIDFIELD_RIGHT, STRIKER, STRIKER}
	side := func(teamId int64, bonus float64) []PlayerRating {
		ratings := make([]PlayerRating, 0, len(positions))
		for i, position := range positions {
			id := seedTransfersPlayer(t, db, teamId, country)
			ratings = append(ratings, PlayerRating{PlayerId: id, TeamId: teamId, Rated: true, Rating: 6 + bonus, Minutes: 90 - i, Position: position})
		}
		return ratings
	}
	sideA, sideB := side(teamA, 0.5), side(teamB, 0)
	result := MatchResult{PlayerRatings: append(append([]PlayerRating{}, sideA...), sideB...),
		Effects: &PostMatchPayload{Home: teamA, Away: teamB}}
	if err := SaveLeagueMatchResultTx(dbCtx, played, leagueId, teamA, teamB, result); err != nil {
		t.Fatalf("saving the first match: %s", err)
	}

	complete := func() bool {
		t.Helper()
		ok, err := IsLeagueWeekComplete(dbCtx, leagueId, 1, 1)
		if err != nil {
			t.Fatalf("IsLeagueWeekComplete: %s", err)
		}
		return ok
	}
	if complete() {
		t.Fatal("matchday complete with a match unplayed")
	}
	if err := SaveLeagueMatchResultTx(dbCtx, pending, leagueId, teamB, teamA, MatchResult{Effects: &PostMatchPayload{Home: teamB, Away: teamA}}); err != nil {
		t.Fatalf("saving the second match: %s", err)
	}
	if complete() {
		t.Fatal("matchday complete while matches still wait for their post-match effects")
	}
	if _, err := db.Exec(`DELETE FROM pending_post_match_effects WHERE kind = 'league' AND match_id IN ($1, $2)`, played, pending); err != nil {
		t.Fatalf("applying effects: %s", err)
	}
	if !complete() {
		t.Fatal("matchday not complete once every replay ended")
	}

	candidates, err := GetWeekBestXiCandidates(dbCtx, leagueId, 1, 1)
	if err != nil || len(candidates) != 22 {
		t.Fatalf("candidates = %d, %v; want 22", len(candidates), err)
	}
	picks := SelectBestXi(candidates)
	var wg sync.WaitGroup
	var mu sync.Mutex
	stored := 0
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := SaveBestXi(dbCtx, leagueId, 1, 1, picks)
			if err != nil {
				t.Errorf("SaveBestXi: %s", err)
			}
			if ok {
				mu.Lock()
				stored++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if stored != 1 {
		t.Fatalf("SaveBestXi stored %d times, want once", stored)
	}

	xi, err := GetBestXi(dbCtx, leagueId, 1, -1)
	if err != nil {
		t.Fatalf("GetBestXi: %s", err)
	}
	if xi.Week != 1 || len(xi.Weeks) != 1 || len(xi.Players) != 11 {
		t.Fatalf("best XI = week %d weeks %v players %d", xi.Week, xi.Weeks, len(xi.Players))
	}
	// Team A's players are rated half a point higher: they fill the XI.
	for _, p := range xi.Players {
		if p.TeamId != teamA || p.TeamName != "Xi A" || p.LastName != "Player" {
			t.Errorf("slot %d = %+v, want a team A player", p.Slot, p)
		}
	}
	if xi.Players[0].Line != BestXiLineGoalkeeper || xi.Players[0].PlayerId != sideA[0].PlayerId {
		t.Errorf("goalkeeper = %+v", xi.Players[0])
	}
	counts, err := GetPlayerBestXiCounts(dbCtx, sideA[0].PlayerId)
	if err != nil || counts.Weeks != 1 || counts.Seasons != 0 {
		t.Errorf("counts = %+v, %v", counts, err)
	}

	season, err := GetSeasonBestXiCandidates(dbCtx, leagueId, 1)
	if err != nil || len(season) != 22 {
		t.Errorf("season candidates = %d, %v; want 22 (90 minutes of 1 match each)", len(season), err)
	}
}
