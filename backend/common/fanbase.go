package common

// DefaultFanbase is the value every new team starts with, and what every pre-existing team was
// backfilled to (see resources/sql/history/init66_fanbase.sql) - the same neutral-midpoint convention
// DefaultForm uses for a 0-99 attribute with no real history to derive a starting value from.
const DefaultFanbase = 50

const GetFanbaseStmtSql = "SELECT fanbase FROM teams WHERE team_id = $1"
const SetFanbaseStmtSql = "UPDATE teams SET fanbase = $1 WHERE team_id = $2"

// GetFanbase returns teamId's current Fanbase index (0-99) - see engine/fanbase.go for how it
// moves match by match, and what it feeds into (matchday attendance, sponsorship, season tickets/
// merchandising).
func GetFanbase(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var fanbase int
	err := dbCtx.Db.QueryRow(GetFanbaseStmtSql, teamId).Scan(&fanbase)
	return fanbase, err
}

// SetFanbase persists a new Fanbase value - callers (engine.applyFanbaseDrift) are responsible
// for clamping it to [1, 99] first, same convention every other 0-99 attribute's setter leaves to
// its own caller.
func SetFanbase(dbCtx *DatabaseContext, teamId int64, fanbase int) error {
	_, err := dbCtx.Db.Exec(SetFanbaseStmtSql, fanbase, teamId)
	return err
}
