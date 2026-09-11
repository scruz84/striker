package common

// MaxScheduledSubstitutions is the most substitutions a manager may pre-plan on their default
// lineup - matches the real-football limit of 3, and keeps engine.SubstitutionSimulation's
// per-minute scan cheap regardless of squad size.
const MaxScheduledSubstitutions = 3

// LineupSubstitution is one planned in-match change to a team's default lineup: bring PlayerInId
// on for PlayerOutId at Minute. Persisted alongside default_lineup_positions (see
// SetDefaultLineupSubstitutions) and applied during simulation by engine.SubstitutionSimulation.
type LineupSubstitution struct {
	PlayerOutId int64 `json:"playerOutId"`
	PlayerInId  int64 `json:"playerInId"`
	Minute      int   `json:"minute"`
}

const GetDefaultLineupSubstitutionsStmtSql = "SELECT player_out_id, player_in_id, minute FROM default_lineup_substitutions " +
	"WHERE team_id = $1 ORDER BY slot"
const DeleteDefaultLineupSubstitutionsStmtSql = "DELETE FROM default_lineup_substitutions WHERE team_id = $1"
const InsertDefaultLineupSubstitutionStmtSql = "INSERT INTO default_lineup_substitutions " +
	"(team_id, slot, player_out_id, player_in_id, minute) VALUES ($1, $2, $3, $4, $5)"

// GetDefaultLineupSubstitutions returns a team's planned substitutions, in save order.
func GetDefaultLineupSubstitutions(dbCtx *DatabaseContext, teamId int64) ([]LineupSubstitution, error) {
	rows, err := dbCtx.Db.Query(GetDefaultLineupSubstitutionsStmtSql, teamId)
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

// SetDefaultLineupSubstitutions atomically replaces a team's planned substitutions. Callers must
// have already validated each entry (see handlers.validateSubstitutions) - slots are assigned by
// save order (1-based), not taken from the caller.
func SetDefaultLineupSubstitutions(dbCtx *DatabaseContext, teamId int64, subs []LineupSubstitution) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Same non-blocking per-team guard as SetDefaultLineup, and for the same reason: two
	// concurrent saves for the same team racing this delete-then-insert could otherwise hit a
	// duplicate-key violation on default_lineup_substitutions_pkey.
	var acquiredLock bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock($1)", teamId).Scan(&acquiredLock); err != nil {
		return err
	}
	if !acquiredLock {
		return nil
	}

	if _, err := tx.Exec(DeleteDefaultLineupSubstitutionsStmtSql, teamId); err != nil {
		return err
	}

	insertStmt, err := tx.Prepare(InsertDefaultLineupSubstitutionStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for i, s := range subs {
		if _, err := insertStmt.Exec(teamId, i+1, s.PlayerOutId, s.PlayerInId, s.Minute); err != nil {
			return err
		}
	}

	return tx.Commit()
}
