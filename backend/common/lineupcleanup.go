package common

// A player who leaves a club must not stay in any lineup that club has yet to play (issue #399):
// the kickoff reads a match's saved lineup as it stands, so a sold player would start for his old
// club. removePlayerFromTeamLineupsTx takes him out of the club's standing lineups (league/cup and
// friendly defaults, their substitutions, orders and roles) and out of every lineup the club saved
// for a match, cup tie or friendly that has not been played or picked up by a simulation yet.
// Lineups of played matches are the record of who played and are never touched; a national team
// is a team of its own, so a club move leaves his national team lineups alone.
//
// Every statement takes $1 = player id and $2 = the club he is leaving.
var removePlayerFromTeamLineupsStmts = []string{
	// Standing lineups.
	"DELETE FROM default_lineup_positions WHERE player_id = $1 AND team_id = $2",
	"DELETE FROM default_friendly_lineup_positions WHERE player_id = $1 AND team_id = $2",
	"DELETE FROM default_lineup_substitutions WHERE team_id = $2 AND $1 IN (player_out_id, player_in_id)",
	"DELETE FROM default_friendly_lineup_substitutions WHERE team_id = $2 AND $1 IN (player_out_id, player_in_id)",
	"DELETE FROM default_lineup_orders WHERE team_id = $2 AND $1 IN (player_out_id, player_in_id)",
	"UPDATE default_lineup_roles SET captain_id = NULLIF(captain_id, $1), penalty_taker_id = NULLIF(penalty_taker_id, $1), " +
		"free_kick_taker_id = NULLIF(free_kick_taker_id, $1) WHERE team_id = $2 AND $1 IN (captain_id, penalty_taker_id, free_kick_taker_id)",

	// League matches not played yet.
	"DELETE FROM lineup_positions lp USING lineup l, matches m WHERE lp.lineup_id = l.lineup_id AND m.match_id = l.match_id " +
		"AND lp.player_id = $1 AND l.team_id = $2 AND m.played = false AND m.in_simulation = false",
	"DELETE FROM lineup_substitutions s USING matches m WHERE m.match_id = s.match_id " +
		"AND s.team_id = $2 AND $1 IN (s.player_out_id, s.player_in_id) AND m.played = false AND m.in_simulation = false",
	"DELETE FROM lineup_orders o USING matches m WHERE m.match_id = o.match_id " +
		"AND o.team_id = $2 AND $1 IN (o.player_out_id, o.player_in_id) AND m.played = false AND m.in_simulation = false",
	"UPDATE lineup l SET captain_id = NULLIF(l.captain_id, $1), penalty_taker_id = NULLIF(l.penalty_taker_id, $1), " +
		"free_kick_taker_id = NULLIF(l.free_kick_taker_id, $1) FROM matches m WHERE m.match_id = l.match_id " +
		"AND l.team_id = $2 AND $1 IN (l.captain_id, l.penalty_taker_id, l.free_kick_taker_id) AND m.played = false AND m.in_simulation = false",

	// Cup ties not played yet.
	"DELETE FROM cup_tie_lineup_positions lp USING cup_tie_lineup l, cup_ties ct WHERE lp.lineup_id = l.lineup_id AND ct.tie_id = l.tie_id " +
		"AND lp.player_id = $1 AND l.team_id = $2 AND ct.played = false AND ct.in_simulation = false",
	"DELETE FROM cup_tie_lineup_substitutions s USING cup_ties ct WHERE ct.tie_id = s.tie_id " +
		"AND s.team_id = $2 AND $1 IN (s.player_out_id, s.player_in_id) AND ct.played = false AND ct.in_simulation = false",
	"DELETE FROM cup_tie_lineup_orders o USING cup_ties ct WHERE ct.tie_id = o.tie_id " +
		"AND o.team_id = $2 AND $1 IN (o.player_out_id, o.player_in_id) AND ct.played = false AND ct.in_simulation = false",
	"UPDATE cup_tie_lineup l SET captain_id = NULLIF(l.captain_id, $1), penalty_taker_id = NULLIF(l.penalty_taker_id, $1), " +
		"free_kick_taker_id = NULLIF(l.free_kick_taker_id, $1) FROM cup_ties ct WHERE ct.tie_id = l.tie_id " +
		"AND l.team_id = $2 AND $1 IN (l.captain_id, l.penalty_taker_id, l.free_kick_taker_id) AND ct.played = false AND ct.in_simulation = false",

	// Friendlies not played yet.
	"DELETE FROM friendly_lineup_positions lp USING friendly_lineup l, friendly_matches f WHERE lp.lineup_id = l.lineup_id AND f.match_id = l.match_id " +
		"AND lp.player_id = $1 AND l.team_id = $2 AND f.played = false AND f.in_simulation = false",
	"DELETE FROM friendly_lineup_substitutions s USING friendly_matches f WHERE f.match_id = s.match_id " +
		"AND s.team_id = $2 AND $1 IN (s.player_out_id, s.player_in_id) AND f.played = false AND f.in_simulation = false",
	"DELETE FROM friendly_lineup_orders o USING friendly_matches f WHERE f.match_id = o.match_id " +
		"AND o.team_id = $2 AND $1 IN (o.player_out_id, o.player_in_id) AND f.played = false AND f.in_simulation = false",
	"UPDATE friendly_lineup l SET captain_id = NULLIF(l.captain_id, $1), penalty_taker_id = NULLIF(l.penalty_taker_id, $1), " +
		"free_kick_taker_id = NULLIF(l.free_kick_taker_id, $1) FROM friendly_matches f WHERE f.match_id = l.match_id " +
		"AND l.team_id = $2 AND $1 IN (l.captain_id, l.penalty_taker_id, l.free_kick_taker_id) AND f.played = false AND f.in_simulation = false",
}

// removePlayerFromTeamLineupsTx runs removePlayerFromTeamLineupsStmts for playerId leaving teamId,
// inside the caller's transaction (see the comment on removePlayerFromTeamLineupsStmts).
func removePlayerFromTeamLineupsTx(ex sqlExecutor, playerId, teamId int64) error {
	for _, stmt := range removePlayerFromTeamLineupsStmts {
		if _, err := ex.Exec(stmt, playerId, teamId); err != nil {
			return err
		}
	}
	return nil
}
