package common

// GetFriendlyLineupSubstitutionsStmtSql/DeleteFriendlyLineupSubstitutionsStmtSql/
// InsertFriendlyLineupSubstitutionStmtSql back friendly_lineup_substitutions (see
// init59_friendly_lineup_substitutions.sql) - the friendly equivalent of
// default_lineup_substitutions (see lineupsubstitutions.go), scoped to one match rather than the
// team as a whole since a friendly's starting XI (see GetLineupsForFriendly) can itself differ
// from the team's standing default lineup.
const GetFriendlyLineupSubstitutionsStmtSql = "SELECT player_out_id, player_in_id, minute FROM friendly_lineup_substitutions " +
	"WHERE match_id = $1 AND team_id = $2 ORDER BY slot"
const DeleteFriendlyLineupSubstitutionsStmtSql = "DELETE FROM friendly_lineup_substitutions WHERE match_id = $1 AND team_id = $2"
const InsertFriendlyLineupSubstitutionStmtSql = "INSERT INTO friendly_lineup_substitutions " +
	"(match_id, team_id, slot, player_out_id, player_in_id, minute) VALUES ($1, $2, $3, $4, $5, $6)"

// GetFriendlyLineupSubstitutions returns teamId's planned substitutions for one specific friendly
// (see SetFriendlyLineup), in save order. Unlike GetDefaultLineupSubstitutions, there's no
// fallback to anything else - a friendly with none saved simply plays with none scheduled, since
// its starting XI may not be the same players the team's standing default-lineup substitutions
// were written against (see GetLineupsForFriendly, which can hand back a freshly computed best
// XI rather than the default lineup itself).
func GetFriendlyLineupSubstitutions(dbCtx *DatabaseContext, matchId int64, teamId int64) ([]LineupSubstitution, error) {
	rows, err := dbCtx.Db.Query(GetFriendlyLineupSubstitutionsStmtSql, matchId, teamId)
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
