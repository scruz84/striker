package common

import (
	"fmt"
	"testing"
)

// Every lineup kind round-trips its roles, a sold player's role reads automatic and the public
// match views never show the set-piece takers (openspec add-captain-and-set-piece-takers, needs
// STRIKER_TEST_DB_DSN).
func TestLineupRoles_RoundTripInEveryLineupKind(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zw")
	home := seedTransfersTeam(t, db, "Roles Home", "zw")
	away := seedTransfersTeam(t, db, "Roles Away", "zw")
	keeper := seedTransfersPlayer(t, db, home, "zw")
	centreBack := seedTransfersPlayer(t, db, home, "zw")
	number10 := seedTransfersPlayer(t, db, home, "zw")
	positions := []LineupPosition{{PlayerId: keeper, Position: GOALKEEPER}, {PlayerId: centreBack, Position: DEFENSE}, {PlayerId: number10, Position: MIDFIELD}}
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
	ptr := func(v int64) *int64 { return &v }
	// Roles saved: the number 10 takes free kicks, the centre-back is captain, penalties automatic.
	roles := LineupRoles{CaptainId: ptr(centreBack), FreeKickTakerId: ptr(number10)}
	same := func(name string, got LineupRoles, want LineupRoles) {
		t.Helper()
		eq := func(a, b *int64) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
		if !eq(got.CaptainId, want.CaptainId) || !eq(got.PenaltyTakerId, want.PenaltyTakerId) || !eq(got.FreeKickTakerId, want.FreeKickTakerId) {
			t.Errorf("%s roles = %s, want %s", name, describeRoles(got), describeRoles(want))
		}
	}

	// Standing defaults: automatic until saved, each default keeps its own.
	if lineup, err := GetDefaultLineup(dbCtx, home); err != nil {
		t.Fatal(err)
	} else {
		same("new default", lineup.LineupRoles, LineupRoles{})
	}
	if err := SetDefaultLineupRoles(dbCtx, home, false, roles); err != nil {
		t.Fatal(err)
	}
	if err := SetDefaultLineupRoles(dbCtx, home, true, LineupRoles{PenaltyTakerId: ptr(centreBack)}); err != nil {
		t.Fatal(err)
	}
	if lineup, err := GetDefaultLineup(dbCtx, home); err != nil {
		t.Fatal(err)
	} else {
		same("default", lineup.LineupRoles, roles)
	}
	if lineup, err := GetDefaultFriendlyLineup(dbCtx, home); err != nil {
		t.Fatal(err)
	} else {
		same("friendly default", lineup.LineupRoles, LineupRoles{PenaltyTakerId: ptr(centreBack)})
	}

	// League match: the override's roles, and the default's for a side without one.
	matchId := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, now() + interval '1 day', $2, $3) RETURNING match_id`, leagueId, home, away)
	matchRoles := LineupRoles{CaptainId: ptr(number10), PenaltyTakerId: ptr(number10)}
	if err := SetLineupForMatch(dbCtx, matchId, home, FormationBalanced, MENTALITY_BALANCED, matchRoles, positions, nil, nil); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetMatchLineupOverride(dbCtx, matchId, home); err != nil {
		t.Fatal(err)
	} else {
		same("match override", lineup.LineupRoles, matchRoles)
	}
	for _, p := range positions {
		if _, err := db.Exec(`INSERT INTO default_lineup_positions (team_id, player_id, position) VALUES ($1, $2, $3)`, home, p.PlayerId, p.Position); err != nil {
			t.Fatal(err)
		}
	}
	lineups, err := GetLineupsForMatch(dbCtx, matchId, &[]int64{home, away})
	if err != nil {
		t.Fatal(err)
	}
	same("resolved match", (*lineups)[home].LineupRoles, matchRoles)
	otherMatch := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 2, now() + interval '8 days', $2, $3) RETURNING match_id`, leagueId, home, away)
	if lineups, err = GetLineupsForMatch(dbCtx, otherMatch, &[]int64{home, away}); err != nil {
		t.Fatal(err)
	}
	same("resolved match from the default", (*lineups)[home].LineupRoles, roles)

	// The public match view keeps the armband and hides the takers.
	if match, err := WeekMatchById(dbCtx, matchId); err != nil || match == nil {
		t.Fatalf("WeekMatchById: %v", err)
	} else {
		same("public match view", match.HomeLineup.LineupRoles, LineupRoles{CaptainId: ptr(number10)})
	}

	// Cup tie and friendly overrides.
	cupId := scan(`INSERT INTO cups (country, season, name) VALUES ('zw', 1, 'Roles Cup') RETURNING cup_id`)
	tieId := scan(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3) RETURNING tie_id`, cupId, home, away)
	if err := SetCupTieLineup(dbCtx, tieId, home, FormationBalanced, MENTALITY_BALANCED, roles, positions, nil, nil); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetCupTieLineup(dbCtx, tieId, home); err != nil {
		t.Fatal(err)
	} else {
		same("cup tie", lineup.LineupRoles, roles)
	}
	friendlyId := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '1 day') RETURNING match_id`, home, away)
	if err := SetFriendlyLineup(dbCtx, friendlyId, home, FormationBalanced, MENTALITY_BALANCED, matchRoles, positions, nil, nil); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetFriendlyLineup(dbCtx, friendlyId, home); err != nil {
		t.Fatal(err)
	} else {
		same("friendly", lineup.LineupRoles, matchRoles)
	}

	// Taker sold: once the number 10 belongs to another club, every role he held reads automatic.
	if _, err := db.Exec(`UPDATE players SET team_id = $1 WHERE player_id = $2`, away, number10); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetMatchLineupOverride(dbCtx, matchId, home); err != nil {
		t.Fatal(err)
	} else {
		same("match override after the sale", lineup.LineupRoles, LineupRoles{})
	}
	if lineup, err := GetDefaultLineup(dbCtx, home); err != nil {
		t.Fatal(err)
	} else {
		same("default after the sale", lineup.LineupRoles, LineupRoles{CaptainId: ptr(centreBack)})
	}
	if lineup, _, err := GetCupTieLineup(dbCtx, tieId, home); err != nil {
		t.Fatal(err)
	} else {
		same("cup tie after the sale", lineup.LineupRoles, LineupRoles{CaptainId: ptr(centreBack)})
	}
	if lineup, _, err := GetFriendlyLineup(dbCtx, friendlyId, home); err != nil {
		t.Fatal(err)
	} else {
		same("friendly after the sale", lineup.LineupRoles, LineupRoles{})
	}

	// A played match keeps the roles it kicked off with, sold player or not, so its report still
	// shows the captain it was played with.
	if _, err := db.Exec(`UPDATE matches SET played = true WHERE match_id = $1`, matchId); err != nil {
		t.Fatal(err)
	}
	if lineup, _, err := GetMatchLineupOverride(dbCtx, matchId, home); err != nil {
		t.Fatal(err)
	} else {
		same("played match after the sale", lineup.LineupRoles, matchRoles)
	}
	if match, err := WeekMatchById(dbCtx, matchId); err != nil || match == nil {
		t.Fatalf("WeekMatchById: %v", err)
	} else {
		same("played match's public view after the sale", match.HomeLineup.LineupRoles, LineupRoles{CaptainId: ptr(number10)})
	}
}

func describeRoles(r LineupRoles) string {
	v := func(id *int64) any {
		if id == nil {
			return "auto"
		}
		return *id
	}
	return fmt.Sprintf("captain=%v penalties=%v freeKicks=%v", v(r.CaptainId), v(r.PenaltyTakerId), v(r.FreeKickTakerId))
}
