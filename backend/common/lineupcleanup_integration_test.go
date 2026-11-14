package common

import (
	"testing"
	"time"
)

// A player who leaves his club leaves every lineup that club has still to play - league, cup,
// friendly and the standing defaults - while played matches keep theirs (issue #399). Needs
// STRIKER_TEST_DB_DSN.
func TestReleasePlayer_TakesHimOutOfUnplayedLineups(t *testing.T) {
	dbCtx, club, other, _, player := marketCleanupRig(t)
	db := dbCtx.Db
	league := accInt(t, db, `SELECT league_id FROM teams WHERE team_id = $1`, club)
	bench := accPlayerIds(t, db, club, false)[1]
	exec := func(what, query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		accMust(t, err, what)
	}
	future := time.Now().Add(48 * time.Hour)
	past := time.Now().Add(-48 * time.Hour)

	upcoming := accInt(t, db, `INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id) VALUES ($1, 1, $2, $3, $4) RETURNING match_id`,
		league, future, club, other)
	played := accInt(t, db, `INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id, played) VALUES ($1, 1, $2, $3, $4, true) RETURNING match_id`,
		league, past, club, other)
	for _, match := range []int64{upcoming, played} {
		lineup := accInt(t, db, `INSERT INTO lineup (match_id, team_id, captain_id) VALUES ($1, $2, $3) RETURNING lineup_id`, match, club, player)
		exec("league lineup", `INSERT INTO lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)`, lineup, player, MIDFIELD)
	}
	exec("league substitution", `INSERT INTO lineup_substitutions (match_id, team_id, slot, player_out_id, player_in_id, minute) VALUES ($1, $2, 1, $3, $4, 60)`,
		upcoming, club, bench, player)

	cup := accInt(t, db, `INSERT INTO cups (country, season, name) VALUES ($1, 1, 'Cup') RETURNING cup_id`, accTestCountry)
	tie := accInt(t, db, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date) VALUES ($1, 1, 0, $2, $3, $4) RETURNING tie_id`,
		cup, club, other, future)
	cupLineup := accInt(t, db, `INSERT INTO cup_tie_lineup (tie_id, team_id) VALUES ($1, $2) RETURNING lineup_id`, tie, club)
	exec("cup lineup", `INSERT INTO cup_tie_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)`, cupLineup, player, MIDFIELD)

	friendly := accInt(t, db, `INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, $3) RETURNING match_id`, club, other, future)
	friendlyLineup := accInt(t, db, `INSERT INTO friendly_lineup (match_id, team_id) VALUES ($1, $2) RETURNING lineup_id`, friendly, club)
	exec("friendly lineup", `INSERT INTO friendly_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)`, friendlyLineup, player, MIDFIELD)
	exec("friendly substitution", `INSERT INTO friendly_lineup_substitutions (match_id, team_id, slot, player_out_id, player_in_id, minute) VALUES ($1, $2, 1, $3, $4, 60)`,
		friendly, club, player, bench)

	exec("default lineup", `INSERT INTO default_lineup_positions (team_id, player_id, position) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, club, player, MIDFIELD)
	exec("default friendly lineup", `INSERT INTO default_friendly_lineup_positions (team_id, player_id, position) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, club, player, MIDFIELD)
	exec("default substitution", `INSERT INTO default_lineup_substitutions (team_id, slot, player_out_id, player_in_id, minute) VALUES ($1, 1, $2, $3, 70)`, club, player, bench)
	exec("default order", `INSERT INTO default_lineup_orders (team_id, slot, from_minute, condition, action, player_out_id, player_in_id) VALUES ($1, 1, 60, 0, 1, $2, $3)`,
		club, bench, player)
	exec("default roles", `INSERT INTO default_lineup_roles (team_id, friendly, captain_id, penalty_taker_id) VALUES ($1, false, $2, $3)`, club, player, bench)

	accMust(t, ReleasePlayer(dbCtx, player, club), "release")

	for _, check := range []struct {
		what  string
		query string
		args  []any
		want  int64
	}{
		{"upcoming league lineup", `SELECT count(*) FROM lineup_positions lp JOIN lineup l USING (lineup_id) WHERE l.match_id = $1 AND lp.player_id = $2`, []any{upcoming, player}, 0},
		{"played league lineup", `SELECT count(*) FROM lineup_positions lp JOIN lineup l USING (lineup_id) WHERE l.match_id = $1 AND lp.player_id = $2`, []any{played, player}, 1},
		{"upcoming league captain", `SELECT count(*) FROM lineup WHERE match_id = $1 AND captain_id = $2`, []any{upcoming, player}, 0},
		{"played league captain", `SELECT count(*) FROM lineup WHERE match_id = $1 AND captain_id = $2`, []any{played, player}, 1},
		{"league substitution", `SELECT count(*) FROM lineup_substitutions WHERE match_id = $1`, []any{upcoming}, 0},
		{"cup lineup", `SELECT count(*) FROM cup_tie_lineup_positions WHERE lineup_id = $1`, []any{cupLineup}, 0},
		{"friendly lineup", `SELECT count(*) FROM friendly_lineup_positions WHERE lineup_id = $1`, []any{friendlyLineup}, 0},
		{"friendly substitution", `SELECT count(*) FROM friendly_lineup_substitutions WHERE match_id = $1`, []any{friendly}, 0},
		{"default lineup", `SELECT count(*) FROM default_lineup_positions WHERE player_id = $1`, []any{player}, 0},
		{"default friendly lineup", `SELECT count(*) FROM default_friendly_lineup_positions WHERE player_id = $1`, []any{player}, 0},
		{"default substitution", `SELECT count(*) FROM default_lineup_substitutions WHERE team_id = $1`, []any{club}, 0},
		{"default order", `SELECT count(*) FROM default_lineup_orders WHERE team_id = $1`, []any{club}, 0},
		{"default captain", `SELECT count(*) FROM default_lineup_roles WHERE team_id = $1 AND captain_id IS NULL AND penalty_taker_id = $2`, []any{club, bench}, 1},
	} {
		if got := accInt(t, db, check.query, check.args...); got != check.want {
			t.Errorf("%s: %d rows, want %d", check.what, got, check.want)
		}
	}
}
