package common

import "time"

const InsertPlayerTransferStmtSql = "INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind) VALUES ($1, $2, $3, $4, $5)"

// teamIdRef converts a team id into the nullable form player_transfers stores: 0 (no team, as
// Player.LastTeamId uses for a never-released player) becomes NULL.
func teamIdRef(teamId int64) *int64 {
	if teamId == 0 {
		return nil
	}
	return &teamId
}

// RecordPlayerTransfer logs playerId moving from fromTeamId to toTeamId (0 for "no club": a free
// agent being signed, or a player being released) for fee, of one of the PLAYER_TRANSFER_* kinds.
// Called inside the same transaction that moves the player and writes the ledger rows, so the
// two can never disagree - see each caller's own comment.
func RecordPlayerTransfer(ex sqlExecutor, playerId int64, fromTeamId int64, toTeamId int64, fee int64, kind int) error {
	_, err := ex.Exec(InsertPlayerTransferStmtSql, playerId, teamIdRef(fromTeamId), teamIdRef(toTeamId), fee, kind)
	return err
}

// PlayerTransfer is one row of a player's transfer history - see GetPlayerTransfers. A nil
// FromTeamId means the player came from free agency; a nil ToTeamId means they were released.
type PlayerTransfer struct {
	Id           int64     `json:"id"`
	FromTeamId   *int64    `json:"fromTeamId,omitempty"`
	FromTeamName string    `json:"fromTeamName,omitempty"`
	ToTeamId     *int64    `json:"toTeamId,omitempty"`
	ToTeamName   string    `json:"toTeamName,omitempty"`
	Fee          int64     `json:"fee"`
	Kind         int       `json:"kind"`
	CreatedAt    time.Time `json:"createdAt"`
}

const getPlayerTransfersStmtSql = "SELECT t.transfer_id, t.from_team_id, COALESCE(f.name, ''), t.to_team_id, COALESCE(d.name, ''), t.fee, t.kind, t.created_at " +
	"FROM player_transfers t " +
	"LEFT JOIN teams f ON f.team_id = t.from_team_id " +
	"LEFT JOIN teams d ON d.team_id = t.to_team_id " +
	"WHERE t.player_id = $1 ORDER BY t.created_at DESC, t.transfer_id DESC LIMIT $2"

// GetPlayerTransfers returns playerId's most recent transfers, newest first, capped at limit.
// Unlike the accounting ledger (GetPlayerTransactions), this is market information and is shown
// to every visitor, not just the owning team.
func GetPlayerTransfers(dbCtx *DatabaseContext, playerId int64, limit int) ([]PlayerTransfer, error) {
	rows, err := dbCtx.Db.Query(getPlayerTransfersStmtSql, playerId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transfers := make([]PlayerTransfer, 0)
	for rows.Next() {
		var t PlayerTransfer
		if err := rows.Scan(&t.Id, &t.FromTeamId, &t.FromTeamName, &t.ToTeamId, &t.ToTeamName, &t.Fee, &t.Kind, &t.CreatedAt); err != nil {
			return nil, err
		}
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}
