package common

import (
	"fmt"
	"testing"
)

// national-teams "Call-up shortly before kickoff": "Injured star" - an injured player is left out
// of his country's XI, his club is not notified, and the saved lineup carries the planned
// formation, captain and orders. Runs against a real Postgres (STRIKER_TEST_DB_DSN).
func TestSetNationalTeamLineup_SkipsInjuredPlayers(t *testing.T) {
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
	const country = "nq"
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1), ('nr', 1) ON CONFLICT (name) DO NOTHING`, country)
	exec(`UPDATE players SET birth_country = NULL WHERE birth_country = $1`, country)
	league := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'NQ1', 1) RETURNING league_id`, country)
	userId := scan(`INSERT INTO users (name, provider, email) VALUES ('nq-owner', 'dev', 'nq-owner-' || extract(epoch from now())::text || '@example.test') RETURNING user_id`)
	club := scan(`INSERT INTO teams (name, country, league_id, budget, owner_id) VALUES ('NQ Club', $1, $2, 100000, $3) RETURNING team_id`, country, league, userId)
	national, err := ensureNationalTeam(dbCtx, country)
	if err != nil {
		t.Fatal(err)
	}
	opponent, err := ensureNationalTeam(dbCtx, "nr")
	if err != nil {
		t.Fatal(err)
	}
	player := func(gk, def, mid, shot int) int64 {
		return scan(`INSERT INTO players (team_id, first_name, last_name, birth_day, goal_keeper, defense, "lateral", pass, midfield,
			dribbling, shot, "header", quick, creative, technical, avatar, wage, age, contract_years_remaining, birth_country)
			VALUES ($1, 'Nq', 'Player', 0, $2, $3, 30, $4, $4, $5, $5, false, false, false, false, '', 100, 27, 3, $6) RETURNING player_id`,
			club, gk, def, mid, shot, country)
	}
	player(80, 10, 10, 10)
	for i := 0; i < 5; i++ {
		player(5, 60, 30, 20)
	}
	for i := 0; i < 5; i++ {
		player(5, 30, 60, 30)
	}
	for i := 0; i < 3; i++ {
		player(5, 20, 40, 60)
	}
	star := player(5, 20, 50, 95)
	exec(`INSERT INTO player_unavailability (player_id, reason, matches_remaining) VALUES ($1, $2, 2)`, star, UNAVAILABLE_INJURY)
	matchId := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() + interval '1 day') RETURNING match_id`, national, opponent)

	if err := SetNationalTeamLineup(dbCtx, matchId, national, country); err != nil {
		t.Fatalf("SetNationalTeamLineup: %s", err)
	}
	lineup, ok, err := GetFriendlyLineup(dbCtx, matchId, national)
	if err != nil || !ok {
		t.Fatalf("GetFriendlyLineup: ok=%v err=%v", ok, err)
	}
	if len(lineup.Positions) != 11 {
		t.Fatalf("%d starters, want 11", len(lineup.Positions))
	}
	for _, p := range lineup.Positions {
		if p.PlayerId == star {
			t.Errorf("the injured star %d was called up", star)
		}
	}
	if lineup.CaptainId == nil {
		t.Error("no captain named")
	}
	orders, err := GetFriendlyLineupOrders(dbCtx, matchId, national)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 2 {
		t.Errorf("orders %+v, want the Director's two mentality orders", orders)
	}
	notified := scan(`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND notification_type = $2 AND link = $3`, userId, NOTIFICATION_CALL_UP, fmt.Sprintf("/friendlies/%d", matchId))
	if notified != 11 {
		t.Errorf("%d call-up notifications, want 11 (the injured star's club is not told about him)", notified)
	}
}
