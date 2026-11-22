package common

import (
	"testing"
	"time"
)

// A player who retires is deleted, and the lineups he played in keep a row whose player_id is now
// NULL (lineup_positions and its cup/friendly twins are ON DELETE SET NULL). Those lineups must
// still read back - without him - and his retirement must not be blocked by a standing friendly
// default (default_friendly_lineup_positions has player_id in its primary key, so it cannot be set
// to NULL). A lineup for a fixture still to be played loses him like any departure. Needs
// STRIKER_TEST_DB_DSN.
func TestRetirePlayer_LineupsStillReadWithoutHim(t *testing.T) {
	dbCtx, club, other, _, player := marketCleanupRig(t)
	db := dbCtx.Db
	league := accInt(t, db, `SELECT league_id FROM teams WHERE team_id = $1`, club)
	teammate := accPlayerIds(t, db, club, false)[1]
	if teammate == player {
		teammate = accPlayerIds(t, db, club, false)[2]
	}
	exec := func(what, query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		accMust(t, err, what)
	}
	past := time.Now().Add(-48 * time.Hour)
	future := time.Now().Add(48 * time.Hour)
	starters := []struct {
		id       int64
		position int
	}{{player, MIDFIELD}, {teammate, DEFENSE}}

	played := accInt(t, db, `INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id, played) VALUES ($1, 1, $2, $3, $4, true) RETURNING match_id`,
		league, past, club, other)
	lineup := accInt(t, db, `INSERT INTO lineup (match_id, team_id, captain_id) VALUES ($1, $2, $3) RETURNING lineup_id`, played, club, player)

	cup := accInt(t, db, `INSERT INTO cups (country, season, name) VALUES ($1, 1, 'Cup') RETURNING cup_id`, accTestCountry)
	tie := accInt(t, db, `INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date, played) VALUES ($1, 1, 0, $2, $3, $4, true) RETURNING tie_id`,
		cup, club, other, past)
	cupLineup := accInt(t, db, `INSERT INTO cup_tie_lineup (tie_id, team_id) VALUES ($1, $2) RETURNING lineup_id`, tie, club)

	playedFriendly := accInt(t, db, `INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, played) VALUES ($1, $2, $3, true) RETURNING match_id`, club, other, past)
	playedFriendlyLineup := accInt(t, db, `INSERT INTO friendly_lineup (match_id, team_id) VALUES ($1, $2) RETURNING lineup_id`, playedFriendly, club)
	upcomingFriendly := accInt(t, db, `INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, $3) RETURNING match_id`, club, other, future)
	upcomingFriendlyLineup := accInt(t, db, `INSERT INTO friendly_lineup (match_id, team_id) VALUES ($1, $2) RETURNING lineup_id`, upcomingFriendly, club)

	for _, s := range starters {
		exec("league lineup", `INSERT INTO lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)`, lineup, s.id, s.position)
		exec("cup lineup", `INSERT INTO cup_tie_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)`, cupLineup, s.id, s.position)
		exec("played friendly lineup", `INSERT INTO friendly_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)`, playedFriendlyLineup, s.id, s.position)
		exec("upcoming friendly lineup", `INSERT INTO friendly_lineup_positions (lineup_id, player_id, position) VALUES ($1, $2, $3)`, upcomingFriendlyLineup, s.id, s.position)
	}
	exec("default friendly lineup", `INSERT INTO default_friendly_lineup_positions (team_id, player_id, position) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, club, player, MIDFIELD)

	accMust(t, RetirePlayer(dbCtx, player), "retire")

	if n := accInt(t, db, `SELECT count(*) FROM players WHERE player_id = $1`, player); n != 0 {
		t.Fatalf("the retired player is still in the game")
	}
	onlyTeammate := func(what string, positions []LineupPosition) {
		t.Helper()
		if len(positions) != 1 || positions[0].PlayerId != teammate {
			t.Errorf("%s: expected only the teammate %d left, got %+v", what, teammate, positions)
		}
	}

	lineups, err := GetLineupsForMatch(dbCtx, played, &[]int64{club, other})
	accMust(t, err, "GetLineupsForMatch")
	onlyTeammate("league match", (*lineups)[club].Positions)
	override, has, err := GetMatchLineupOverride(dbCtx, played, club)
	accMust(t, err, "GetMatchLineupOverride")
	if !has {
		t.Fatal("the played match's lineup is gone")
	}
	onlyTeammate("league override", override.Positions)

	cupTie, has, err := GetCupTieLineup(dbCtx, tie, club)
	accMust(t, err, "GetCupTieLineup")
	if !has {
		t.Fatal("the played tie's lineup is gone")
	}
	onlyTeammate("cup tie", cupTie.Positions)

	friendly, has, err := GetFriendlyLineup(dbCtx, playedFriendly, club)
	accMust(t, err, "GetFriendlyLineup")
	if !has {
		t.Fatal("the played friendly's lineup is gone")
	}
	onlyTeammate("played friendly", friendly.Positions)

	// The opponent preview reads the latest finished match's kickoff XI: it used to fail with a
	// NULL scan error once one of its starters had retired.
	preview, err := GetOpponentPreview(dbCtx, club)
	accMust(t, err, "GetOpponentPreview")
	if preview.Likely == nil || len(preview.Likely.Players) != 1 || preview.Likely.Players[0].PlayerId != teammate {
		t.Errorf("expected the likely XI to show only the teammate, got %+v", preview.Likely)
	}

	if n := accInt(t, db, `SELECT count(*) FROM friendly_lineup_positions WHERE lineup_id = $1 AND player_id IS NULL`, upcomingFriendlyLineup); n != 0 {
		t.Errorf("the upcoming friendly kept an empty slot for the retired player")
	}
	if n := accInt(t, db, `SELECT count(*) FROM default_friendly_lineup_positions WHERE team_id = $1 AND player_id IS NULL`, club); n != 0 {
		t.Errorf("the friendly default kept an empty slot for the retired player")
	}
}
