package common

// GetCupTieLineupSubstitutionsStmtSql/DeleteCupTieLineupSubstitutionsStmtSql/
// InsertCupTieLineupSubstitutionStmtSql back cup_tie_lineup_substitutions (see
// init63_match_lineup_substitutions.sql) - the cup-tie equivalent of
// lineup_substitutions/friendly_lineup_substitutions, scoped to one tie since a cup tie's
// starting XI can itself be overridden per tie (see cup_tie_lineup/cup_tie_lineup_positions)
// independently of the team's standing default lineup.
const GetCupTieLineupSubstitutionsStmtSql = "SELECT player_out_id, player_in_id, minute FROM cup_tie_lineup_substitutions " +
	"WHERE tie_id = $1 AND team_id = $2 ORDER BY slot"
const DeleteCupTieLineupSubstitutionsStmtSql = "DELETE FROM cup_tie_lineup_substitutions WHERE tie_id = $1 AND team_id = $2"
const InsertCupTieLineupSubstitutionStmtSql = "INSERT INTO cup_tie_lineup_substitutions " +
	"(tie_id, team_id, slot, player_out_id, player_in_id, minute) VALUES ($1, $2, $3, $4, $5, $6)"

// CupTieLineupOverrideExistsStmtSql checks whether teamId has saved a per-tie lineup override for
// tieId at all (see SetCupTieLineup) - used by GetSubstitutionsForCupTie the same way
// MatchLineupOverrideExistsStmtSql is used by GetSubstitutionsForMatch.
const CupTieLineupOverrideExistsStmtSql = "SELECT EXISTS(SELECT 1 FROM cup_tie_lineup WHERE tie_id = $1 AND team_id = $2)"

// GetCupTieLineupSubstitutions returns teamId's planned substitutions for one specific cup tie
// override (see SetCupTieLineup), in save order.
func GetCupTieLineupSubstitutions(dbCtx *DatabaseContext, tieId int64, teamId int64) ([]LineupSubstitution, error) {
	rows, err := dbCtx.Db.Query(GetCupTieLineupSubstitutionsStmtSql, tieId, teamId)
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

// GetSubstitutionsForCupTie resolves teamId's scheduled substitutions for one cup tie, the
// GetSubstitutionsForMatch equivalent for cup ties: its own per-tie override if it has saved a
// lineup override for this tie at all, falling back to its team-wide default-lineup substitutions
// otherwise.
func GetSubstitutionsForCupTie(dbCtx *DatabaseContext, tieId int64, teamId int64) ([]LineupSubstitution, error) {
	var hasOverride bool
	if err := dbCtx.Db.QueryRow(CupTieLineupOverrideExistsStmtSql, tieId, teamId).Scan(&hasOverride); err != nil {
		return nil, err
	}
	if hasOverride {
		return GetCupTieLineupSubstitutions(dbCtx, tieId, teamId)
	}
	return GetDefaultLineupSubstitutions(dbCtx, teamId)
}
