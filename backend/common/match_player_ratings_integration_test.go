package common

import (
	"errors"
	"testing"
)

// TestMatchPlayerRatings_StoredWithTheResult runs against a real Postgres (STRIKER_TEST_DB_DSN, see
// openTransfersTestDB; skipped when unset). For a league match, a cup tie and a friendly, the
// result save stores the per-minute contributions on the instants and the ratings in
// match_player_ratings, replacing any rows a failed earlier attempt left, and a save rejected with
// ErrAlreadyPlayed stores nothing.
func TestMatchPlayerRatings_StoredWithTheResult(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)

	mustScan := func(dest *int64, query string, args ...any) {
		t.Helper()
		if err := db.QueryRow(query, args...).Scan(dest); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}

	const country = "zmpr"
	var leagueId, teamA, teamB, matchId, cupId, tieId, friendlyId int64
	mustExec(`INSERT INTO countries (name, current_season) VALUES ($1, 1) ON CONFLICT (name) DO NOTHING`, country)
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ($1, 1, 'L') RETURNING league_id`, country)
	mustScan(&teamA, `INSERT INTO teams (name, country) VALUES ('RatingsA', $1) RETURNING team_id`, country)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('RatingsB', $1) RETURNING team_id`, country)
	mustScan(&matchId, `INSERT INTO matches (league_id, season, played, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, false, now(), $2, $3) RETURNING match_id`, leagueId, teamA, teamB)
	mustScan(&cupId, InsertCupStmtSql, country, 1, "Cup")
	mustScan(&tieId, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3) RETURNING tie_id`,
		cupId, teamA, teamB)
	mustScan(&friendlyId, InsertNationalTeamFriendlyMatchStmtSql, teamA, teamB, "2026-01-01T00:00:00Z")

	instants := []MatchInstant{
		{Minute: 1, PlayerContributions: &InstantContributions{
			Home: map[int64]PlayerContribution{7: {Position: STRIKER, Own: 52.3, Rival: 47.1}},
			Away: map[int64]PlayerContribution{8: {Position: DEFENSE, Own: 47.1, Rival: 52.3}},
		}},
		{Minute: 2},
	}
	events := []Event{{EventType: GOAL, Minute: 1, TeamOwner: teamA, OtherTeam: teamB, TeamOwnerPlayers: []int64{7}, HomeGoals: 1}}
	ratings := ComputePlayerRatings(teamA, teamB, nil, nil, instants, events, 90)
	result := MatchResult{HomeGoals: 1, Events: events, Instants: instants, PlayerRatings: ratings}

	cases := []struct {
		kind     string
		id       int64
		save     func() error
		instants func() []MatchInstant
	}{
		{SimulationKindLeague, matchId, func() error { return SaveLeagueMatchResultTx(dbCtx, matchId, leagueId, teamA, teamB, result) },
			func() []MatchInstant { return *GetMatchInstants(dbCtx, matchId) }},
		{SimulationKindCup, tieId, func() error { return SaveCupTieResultTx(dbCtx, tieId, false, teamA, result) },
			func() []MatchInstant {
				got, err := GetCupTieInstants(dbCtx, tieId)
				if err != nil {
					t.Fatalf("GetCupTieInstants: %s", err)
				}
				return *got
			}},
		{SimulationKindFriendly, friendlyId, func() error { return SaveFriendlyResultTx(dbCtx, friendlyId, teamA, teamB, result) },
			func() []MatchInstant { return *GetFriendlyInstants(dbCtx, friendlyId) }},
	}
	for _, c := range cases {
		// A row left behind by an earlier, failed attempt must not survive the save.
		mustExec(InsertMatchPlayerRatingStmtSql, c.kind, c.id, 999, teamA, true, 6.0, 90, MIDFIELD, nil, nil, nil, nil, 0, 0, 0, 0, 0)
		if err := c.save(); err != nil {
			t.Fatalf("%s save: %s", c.kind, err)
		}

		got, err := GetMatchPlayerRatings(dbCtx, c.kind, c.id)
		if err != nil {
			t.Fatalf("%s GetMatchPlayerRatings: %s", c.kind, err)
		}
		if len(got) != 2 || got[0].PlayerId != 7 || got[1].PlayerId != 8 {
			t.Fatalf("%s ratings: got %+v", c.kind, got)
		}
		want := ratingOf(t, ratings, 7)
		scorer := got[0]
		if scorer.Rating != want.Rating || scorer.Goals != 1.4 || scorer.Zone != want.Zone || scorer.Minutes != 1 ||
			scorer.Own == nil || *scorer.Own != 52.3 || scorer.Rival == nil || *scorer.Rival != 47.1 || scorer.TeamId != teamA {
			t.Fatalf("%s scorer: got %+v, want %+v", c.kind, scorer, want)
		}

		readInstants := c.instants()
		if len(readInstants) != 2 || readInstants[0].PlayerContributions == nil || readInstants[1].PlayerContributions != nil {
			t.Fatalf("%s instants: %+v", c.kind, readInstants)
		}
		if contribution := readInstants[0].PlayerContributions.Home[7]; contribution != (PlayerContribution{Position: STRIKER, Own: 52.3, Rival: 47.1}) {
			t.Fatalf("%s contribution: %+v", c.kind, contribution)
		}

		// A second save of the same, now played fixture is rejected and changes nothing.
		if err := c.save(); !errors.Is(err, ErrAlreadyPlayed) {
			t.Fatalf("%s second save: got %v, want ErrAlreadyPlayed", c.kind, err)
		}
		if after, _ := GetMatchPlayerRatings(dbCtx, c.kind, c.id); len(after) != 2 {
			t.Fatalf("%s ratings after a rejected save: %+v", c.kind, after)
		}
	}
}
