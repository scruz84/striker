package common

// GetMatchLineupSubstitutionsStmtSql/DeleteMatchLineupSubstitutionsStmtSql/
// InsertMatchLineupSubstitutionStmtSql back lineup_substitutions (see
// init63_match_lineup_substitutions.sql) - the league-match equivalent of
// friendly_lineup_substitutions (see friendlylineupsubstitutions.go), scoped to one match rather
// than the team as a whole since a league match's starting XI can itself be overridden per match
// (see lineup/lineup_positions) independently of the team's standing default lineup.
const GetMatchLineupSubstitutionsStmtSql = "SELECT player_out_id, player_in_id, minute FROM lineup_substitutions " +
	"WHERE match_id = $1 AND team_id = $2 ORDER BY slot"
const DeleteMatchLineupSubstitutionsStmtSql = "DELETE FROM lineup_substitutions WHERE match_id = $1 AND team_id = $2"
const InsertMatchLineupSubstitutionStmtSql = "INSERT INTO lineup_substitutions " +
	"(match_id, team_id, slot, player_out_id, player_in_id, minute) VALUES ($1, $2, $3, $4, $5, $6)"

// MatchLineupOverrideExistsStmtSql checks whether teamId has saved a per-match lineup override
// for matchId at all (see SetLineupForMatch) - used by GetSubstitutionsForMatch to decide whether
// this match's own substitutions table governs (even if empty) or the team-wide default does,
// mirroring the all-or-nothing override/fallback GetLineupsForMatch already applies to positions.
const MatchLineupOverrideExistsStmtSql = "SELECT EXISTS(SELECT 1 FROM lineup WHERE match_id = $1 AND team_id = $2)"

// GetMatchLineupSubstitutions returns teamId's planned substitutions for one specific league
// match override (see SetLineupForMatch), in save order.
func GetMatchLineupSubstitutions(dbCtx *DatabaseContext, matchId int64, teamId int64) ([]LineupSubstitution, error) {
	rows, err := dbCtx.Db.Query(GetMatchLineupSubstitutionsStmtSql, matchId, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subs := make([]LineupSubstitution, 0, MaxScheduledSubstitutions)
	for rows.Next() {
		var s LineupSubstitution
		if err := rows.Scan(&s.PlayerOutId, &s.PlayerInId, &s.Minute); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

// GetSubstitutionsForMatch resolves teamId's scheduled substitutions for one league match: its
// own per-match override if it has saved a lineup override for this match at all (even one with
// zero substitutions scheduled), falling back to its team-wide default-lineup substitutions
// otherwise - the same all-or-nothing override/fallback GetLineupsForMatch already applies to
// positions, rather than silently mixing the two.
func GetSubstitutionsForMatch(dbCtx *DatabaseContext, matchId int64, teamId int64) ([]LineupSubstitution, error) {
	var hasOverride bool
	if err := dbCtx.Db.QueryRow(MatchLineupOverrideExistsStmtSql, matchId, teamId).Scan(&hasOverride); err != nil {
		return nil, err
	}
	if hasOverride {
		return GetMatchLineupSubstitutions(dbCtx, matchId, teamId)
	}
	return GetDefaultLineupSubstitutions(dbCtx, teamId)
}
