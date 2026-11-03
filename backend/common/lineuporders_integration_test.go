package common

import (
	"reflect"
	"testing"
)

// Conditional orders round-trip in every lineup kind that holds them (openspec
// add-conditional-match-orders; needs STRIKER_TEST_DB_DSN), and a fixture without its own override
// plays the default's.
func TestLineupOrders_RoundTripInEveryLineupKind(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zo")
	home := seedTransfersTeam(t, db, "Orders Home", "zo")
	away := seedTransfersTeam(t, db, "Orders Away", "zo")
	starter, bench := seedTransfersPlayer(t, db, home, "zo"), seedTransfersPlayer(t, db, home, "zo")
	positions := []LineupPosition{{PlayerId: starter, Position: MIDFIELD}}
	orders := []LineupOrder{
		{FromMinute: 60, Condition: ORDER_CONDITION_LOSING, Action: ORDER_ACTION_MENTALITY, Mentality: MENTALITY_ATTACKING},
		{FromMinute: 75, Condition: ORDER_CONDITION_WINNING, Action: ORDER_ACTION_SUBSTITUTION, PlayerOutId: starter, PlayerInId: bench},
	}
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(query, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return id
	}
	check := func(what string, got []LineupOrder, err error, want []LineupOrder) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %s", what, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %+v, want %+v", what, got, want)
		}
	}
	leagueId := scan(`SELECT league_id FROM teams WHERE team_id = $1`, home)

	if err := SetDefaultLineupOrders(dbCtx, home, orders); err != nil {
		t.Fatal(err)
	}
	got, err := GetDefaultLineupOrders(dbCtx, home)
	check("default", got, err, orders)

	matchId := scan(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id)
		VALUES ($1, 1, 1, now() + interval '1 day', $2, $3) RETURNING match_id`, leagueId, home, away)
	got, err = GetOrdersForMatch(dbCtx, matchId, home)
	check("match without an override (the default's)", got, err, orders)
	if err := SetLineupForMatch(dbCtx, matchId, home, FormationBalanced, MENTALITY_BALANCED, LineupRoles{}, positions, nil, orders[:1]); err != nil {
		t.Fatal(err)
	}
	got, err = GetOrdersForMatch(dbCtx, matchId, home)
	check("match override", got, err, orders[:1])

	cupId := scan(`INSERT INTO cups (country, season, name) VALUES ('zo', 1, 'Orders Cup') RETURNING cup_id`)
	tieId := scan(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id) VALUES ($1, 1, 1, $2, $3) RETURNING tie_id`, cupId, home, away)
	if err := SetCupTieLineup(dbCtx, tieId, home, FormationBalanced, MENTALITY_BALANCED, LineupRoles{}, positions, nil, orders[1:]); err != nil {
		t.Fatal(err)
	}
	got, err = GetOrdersForCupTie(dbCtx, tieId, home)
	check("cup tie override", got, err, orders[1:])

	friendlyId := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '1 day') RETURNING match_id`, home, away)
	if err := SetFriendlyLineup(dbCtx, friendlyId, home, FormationBalanced, MENTALITY_BALANCED, LineupRoles{}, positions, nil, orders); err != nil {
		t.Fatal(err)
	}
	got, err = GetFriendlyLineupOrders(dbCtx, friendlyId, home)
	check("friendly override", got, err, orders)

	// Re-saving replaces them; saving none clears them.
	if err := SetFriendlyLineup(dbCtx, friendlyId, home, FormationBalanced, MENTALITY_BALANCED, LineupRoles{}, positions, nil, nil); err != nil {
		t.Fatal(err)
	}
	got, err = GetFriendlyLineupOrders(dbCtx, friendlyId, home)
	check("friendly override cleared", got, err, []LineupOrder{})
}
