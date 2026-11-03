package common

import (
	"testing"
)

// Every lineup kind round-trips its mentality (openspec add-team-mentality, needs
// STRIKER_TEST_DB_DSN), and a lineup saved before mentality existed reads Balanced.
func TestLineupMentality_RoundTripsInEveryLineupKind(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zm")
	home := seedTransfersTeam(t, db, "Mentality Home", "zm")
	away := seedTransfersTeam(t, db, "Mentality Away", "zm")
	positions := []LineupPosition{{PlayerId: seedTransfersPlayer(t, db, home, "zm"), Position: GOALKEEPER}}
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	var leagueId int64
	if err := db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, home).Scan(&leagueId); err != nil {
		t.Fatal(err)
	}

	// Standing defaults: a new team reads Balanced, then each keeps its own value.
	if lineup, err := GetDefaultLineup(dbCtx, home); err != nil || lineup.Mentality != MENTALITY_BALANCED {
		t.Fatalf("new team's default mentality = %v (%v), want Balanced", lineup, err)
	}
	if err := SetTeamMentality(dbCtx, home, MENTALITY_ATTACKING); err != nil {
		t.Fatal(err)
	}
	if err := SetTeamFriendlyMentality(dbCtx, home, MENTALITY_DEFENSIVE); err != nil {
		t.Fatal(err)
	}
	if lineup, err := GetDefaultLineup(dbCtx, home); err != nil || lineup.Mentality != MENTALITY_ATTACKING {
		t.Errorf("default mentality = %v (%v), want Attacking", lineup, err)
	}
	if lineup, err := GetDefaultFriendlyLineup(dbCtx, home); err != nil || lineup.Mentality != MENTALITY_DEFENSIVE {
		t.Errorf("friendly default mentality = %v (%v), want Defensive", lineup, err)
	}
	if err := SetTeamMentality(dbCtx, home, 2); err == nil {
		t.Error("SetTeamMentality accepted 2")
	}

	// League match override, and the fallback to the default for a team without one.
	matchId := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, now() + interval '1 day', $2, $3) RETURNING match_id`, leagueId, home, away)
	if err := SetLineupForMatch(dbCtx, matchId, away, FormationBalanced, MENTALITY_DEFENSIVE, positions, nil, nil); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetMatchLineupOverride(dbCtx, matchId, away); err != nil || lineup.Mentality != MENTALITY_DEFENSIVE {
		t.Errorf("match override mentality = %d (%v), want Defensive", lineup.Mentality, err)
	}
	if _, err := db.Exec(`INSERT INTO default_lineup_positions (team_id, player_id, position) VALUES ($1, $2, $3)`, home, positions[0].PlayerId, GOALKEEPER); err != nil {
		t.Fatal(err)
	}
	lineups, err := GetLineupsForMatch(dbCtx, matchId, &[]int64{home, away})
	if err != nil {
		t.Fatal(err)
	}
	if (*lineups)[home].Mentality != MENTALITY_ATTACKING || (*lineups)[away].Mentality != MENTALITY_DEFENSIVE {
		t.Errorf("resolved match mentalities = %d / %d, want Attacking (default) / Defensive (override)",
			(*lineups)[home].Mentality, (*lineups)[away].Mentality)
	}
	if err := SetLineupForMatch(dbCtx, matchId, away, FormationBalanced, -2, positions, nil, nil); err != ErrInvalidMentality {
		t.Errorf("SetLineupForMatch(-2) = %v, want ErrInvalidMentality", err)
	}

	// Cup tie override.
	cupId := scan(`INSERT INTO cups (country, season, name) VALUES ('zm', 1, 'Mentality Cup') RETURNING cup_id`)
	tieId := scan(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3) RETURNING tie_id`, cupId, home, away)
	if err := SetCupTieLineup(dbCtx, tieId, home, FormationBalanced, MENTALITY_ATTACKING, positions, nil, nil); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetCupTieLineup(dbCtx, tieId, home); err != nil || lineup.Mentality != MENTALITY_ATTACKING {
		t.Errorf("cup tie mentality = %d (%v), want Attacking", lineup.Mentality, err)
	}

	// Friendly override.
	friendlyId := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '1 day') RETURNING match_id`, home, away)
	if err := SetFriendlyLineup(dbCtx, friendlyId, home, FormationBalanced, MENTALITY_DEFENSIVE, positions, nil, nil); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetFriendlyLineup(dbCtx, friendlyId, home); err != nil || lineup.Mentality != MENTALITY_DEFENSIVE {
		t.Errorf("friendly mentality = %d (%v), want Defensive", lineup.Mentality, err)
	}

	// A lineup saved before this change (no mentality written) reads Balanced.
	legacy := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 2, now() + interval '8 days', $2, $3) RETURNING match_id`, leagueId, home, away)
	lineupId := scan(`INSERT INTO lineup (match_id, team_id, formation) VALUES ($1, $2, $3) RETURNING lineup_id`, legacy, away, FormationBalanced)
	if _, err := db.Exec(`INSERT INTO lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)`, lineupId, positions[0].PlayerId, GOALKEEPER); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetMatchLineupOverride(dbCtx, legacy, away); err != nil || lineup.Mentality != MENTALITY_BALANCED {
		t.Errorf("legacy lineup mentality = %d (%v), want Balanced", lineup.Mentality, err)
	}
}
