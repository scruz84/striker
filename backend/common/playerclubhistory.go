package common

import "database/sql"

// sqlExecutor is satisfied by both *sql.DB and *sql.Tx - lets RecordPlayerClubHistory run either
// directly or inside an already-open transaction (most of its callers are mid-transfer/signing),
// without needing two near-identical functions.
type sqlExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
}

const InsertPlayerClubHistoryStmtSql = "INSERT INTO player_club_history (player_id, team_id) VALUES ($1, $2)"

// RecordPlayerClubHistory logs playerId permanently joining teamId's senior squad - called
// whenever that actually happens (initial squad generation, a transfer, an accepted offer, or a
// free agent signing - see each caller's own comment), never for a loan (temporary) or an academy
// promotion (same club throughout, see PromoteFromAcademy).
func RecordPlayerClubHistory(ex sqlExecutor, playerId int64, teamId int64) error {
	_, err := ex.Exec(InsertPlayerClubHistoryStmtSql, playerId, teamId)
	return err
}

// PlayerClubHistoryEntry is one club a player has belonged to - see GetPlayerClubHistory.
type PlayerClubHistoryEntry struct {
	TeamId   int64  `json:"teamId"`
	TeamName string `json:"teamName"`
	JoinedAt string `json:"joinedAt"`
}

const GetPlayerClubHistoryStmtSql = "SELECT h.team_id, t.name, h.joined_at FROM player_club_history h " +
	"JOIN teams t ON t.team_id = h.team_id WHERE h.player_id = $1 ORDER BY h.joined_at DESC"

// GetPlayerClubHistory lists every club playerId has permanently joined, most recent first.
func GetPlayerClubHistory(dbCtx *DatabaseContext, playerId int64) ([]PlayerClubHistoryEntry, error) {
	rows, err := dbCtx.Db.Query(GetPlayerClubHistoryStmtSql, playerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]PlayerClubHistoryEntry, 0)
	for rows.Next() {
		var e PlayerClubHistoryEntry
		if err := rows.Scan(&e.TeamId, &e.TeamName, &e.JoinedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
