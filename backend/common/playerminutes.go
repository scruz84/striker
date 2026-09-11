package common

// FullSeasonMinutesReference is the minutes-played benchmark engine.ProgressPlayers treats as
// "played enough to fully develop this season" (roughly 10 full 90-minute matches) - a player's
// chance of any given relevant skill actually drifting scales linearly with their season's total
// minutes played up to this reference, capping at a certain (not guaranteed) drift rather than
// an all-or-nothing threshold.
const FullSeasonMinutesReference = 900

const AddPlayerPositionMinutesStmtSql = "INSERT INTO player_position_minutes (player_id, position, minutes) VALUES ($1, $2, $3) " +
	"ON CONFLICT (player_id, position) DO UPDATE SET minutes = player_position_minutes.minutes + EXCLUDED.minutes"

// RecordPlayerPositionMinutes adds one match's per-player-per-position minutes (see
// MatchSimulation.HomeMinutesByPosition/AwayMinutesByPosition) onto the running season total in
// player_position_minutes - see GetTeamPlayerPositionMinutes and ClearPlayerPositionMinutes for
// how engine.ProgressPlayers consumes and resets it. Deliberately not called for friendly
// matches (see engine.simulateFriendly) - friendlies have no effect beyond the match itself.
func RecordPlayerPositionMinutes(dbCtx *DatabaseContext, minutesByPosition map[int64]map[int]int) error {
	if len(minutesByPosition) == 0 {
		return nil
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(AddPlayerPositionMinutesStmtSql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for playerId, byPosition := range minutesByPosition {
		for position, minutes := range byPosition {
			if minutes == 0 {
				continue
			}
			if _, err := stmt.Exec(playerId, position, minutes); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

const GetTeamPlayerPositionMinutesStmtSql = "SELECT ppm.player_id, ppm.position, ppm.minutes FROM player_position_minutes ppm " +
	"JOIN players p ON p.player_id = ppm.player_id WHERE p.team_id = $1"

// GetTeamPlayerPositionMinutes returns this season's accumulated minutes-by-position
// (playerId -> position -> minutes) for every player on teamId who has played at least one
// competitive minute so far this season. A player with no entry here hasn't played at all.
func GetTeamPlayerPositionMinutes(dbCtx *DatabaseContext, teamId int64) (map[int64]map[int]int, error) {
	rows, err := dbCtx.Db.Query(GetTeamPlayerPositionMinutesStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]map[int]int)
	for rows.Next() {
		var playerId int64
		var position, minutes int
		if err := rows.Scan(&playerId, &position, &minutes); err != nil {
			return nil, err
		}
		if result[playerId] == nil {
			result[playerId] = make(map[int]int)
		}
		result[playerId][position] = minutes
	}
	return result, nil
}

const ClearPlayerPositionMinutesStmtSql = "DELETE FROM player_position_minutes"

// ClearPlayerPositionMinutes resets the season-long minutes accumulator - called once, after
// every team has been progressed, at the end of engine.ProgressPlayers.
func ClearPlayerPositionMinutes(dbCtx *DatabaseContext) error {
	_, err := dbCtx.Db.Exec(ClearPlayerPositionMinutesStmtSql)
	return err
}
