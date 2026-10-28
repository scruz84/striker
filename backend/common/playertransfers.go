package common

import (
	"database/sql"
	"sort"
	"time"
)

const InsertPlayerTransferStmtSql = "INSERT INTO player_transfers (player_id, from_team_id, to_team_id, fee, kind, position, age, average_skill) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"

// GetPlayerTransferSnapshotStmtSql reads what RecordPlayerTransfer freezes about the moved player:
// the seven ratings BestCorePosition and AverageSkill are computed from, and his age.
const GetPlayerTransferSnapshotStmtSql = `SELECT goal_keeper, defense, "lateral", pass, midfield, dribbling, shot, age FROM players WHERE player_id = $1`

// sqlQueryExecutor is a sqlExecutor that can also read a single row - satisfied by *sql.DB and
// *sql.Tx, so RecordPlayerTransfer reads the player inside its caller's transaction.
type sqlQueryExecutor interface {
	sqlExecutor
	QueryRow(query string, args ...any) *sql.Row
}

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
//
// The row also stores a snapshot of the player (best core position, age, average skill) as he is
// right now, read through the same executor, so a later comparison by GetTransferComparables
// judges him by what he was when moved. Every kind gets one; only the eligible kinds ever use it.
func RecordPlayerTransfer(ex sqlQueryExecutor, playerId int64, fromTeamId int64, toTeamId int64, fee int64, kind int) error {
	var p Player
	if err := ex.QueryRow(GetPlayerTransferSnapshotStmtSql, playerId).Scan(
		&p.GoalKeeper, &p.Defense, &p.Lateral, &p.Pass, &p.Midfield, &p.Dribbling, &p.Shot, &p.Age); err != nil {
		return err
	}
	_, err := ex.Exec(InsertPlayerTransferStmtSql, playerId, teamIdRef(fromTeamId), teamIdRef(toTeamId), fee, kind,
		BestCorePosition(p), p.Age, AverageSkill(p))
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

const getPlayerIdsAcquiredSinceStmtSql = "SELECT DISTINCT player_id FROM player_transfers WHERE to_team_id = $1 AND created_at > $2"

// GetPlayerIdsAcquiredSince is the set of players teamId took in (bought, signed, loaned or
// made permanent) after the given time - what a CPU team won't turn around and sell straight
// away, so a signing doesn't bounce between clubs.
func GetPlayerIdsAcquiredSince(dbCtx *DatabaseContext, teamId int64, since time.Time) (map[int64]bool, error) {
	rows, err := dbCtx.Db.Query(getPlayerIdsAcquiredSinceStmtSql, teamId, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// Transfer comparables: recent sales of players like a target, shown in the offer dialog. They are
// matched and exposed using only public figures (best position, age, average skill) taken from the
// snapshot RecordPlayerTransfer stores - see the transfer-comparables spec.
const (
	ComparablesWindowDays = 60 // keep in sync with the interval literal in getTransferComparablesStmtSql
	ComparablesLimit      = 8
	ComparablesMinNarrow  = 3 // fewer narrow-band results than this broadens the search

	ComparablesNarrowAge     = 2
	ComparablesNarrowAverage = 4
	ComparablesWideAge       = 4
	ComparablesWideAverage   = 8
)

// getTransferComparablesStmtSql spells the partial index predicate of player_transfers_comparables_idx
// (init88) as literals - kind IN (PLAYER_TRANSFER_LISTING, _OFFER, _BUY_OPTION) = (0, 1, 4) and
// fee > 0 - because Postgres only uses a partial index when the query provably implies its
// predicate at plan time, which bind parameters would defeat. TestTransferComparablesStatementLiterals
// pins the literals to the constants. The ordered LIMIT sits in a subquery over player_transfers
// alone, so the planner walks the index in ORDER BY order and stops after the limit before any join
// (with the joins in the same level it may trade that for a merge join and a sort); only the few
// survivors are joined to the names. Selected columns are an explicit allow-list: nothing private.
const getTransferComparablesStmtSql = "SELECT t.transfer_id, t.player_id, p.first_name, p.last_name, t.position, t.age, t.average_skill, " +
	"t.fee, t.kind, t.created_at, t.from_team_id, COALESCE(f.name, ''), t.to_team_id, COALESCE(d.name, '') " +
	"FROM (SELECT * FROM player_transfers " +
	"WHERE kind IN (0, 1, 4) AND fee > 0 AND position = $1 " +
	"AND age BETWEEN $2 AND $3 AND average_skill BETWEEN $4 AND $5 " +
	"AND created_at > now() - interval '60 days' AND player_id <> $6 " +
	"ORDER BY created_at DESC, transfer_id DESC LIMIT $7) t " +
	"JOIN players p ON p.player_id = t.player_id " +
	"LEFT JOIN teams f ON f.team_id = t.from_team_id " +
	"LEFT JOIN teams d ON d.team_id = t.to_team_id " +
	"ORDER BY t.created_at DESC, t.transfer_id DESC"

// TransferComparable is one recent sale of a player similar to the target. Position, Age and
// AverageSkill are the snapshot taken when he was sold. It deliberately has no field for any
// individual rating, trait, stamina, form, wage or contract.
type TransferComparable struct {
	TransferId   int64     `json:"transferId"`
	PlayerId     int64     `json:"playerId"`
	PlayerName   string    `json:"playerName"`
	Position     int       `json:"position"`
	Age          int       `json:"age"`
	AverageSkill int       `json:"averageSkill"`
	Fee          int64     `json:"fee"`
	Kind         int       `json:"kind"`
	CreatedAt    time.Time `json:"createdAt"`
	FromTeamId   *int64    `json:"fromTeamId,omitempty"`
	FromTeamName string    `json:"fromTeamName,omitempty"`
	ToTeamId     *int64    `json:"toTeamId,omitempty"`
	ToTeamName   string    `json:"toTeamName,omitempty"`
}

// TransferComparablesResult is the target's reference values, whether the wide band had to be used,
// the summary figures (nil when there are no comparables) and the comparables, newest first.
type TransferComparablesResult struct {
	Position     int                  `json:"position"`
	Age          int                  `json:"age"`
	AverageSkill int                  `json:"averageSkill"`
	Broadened    bool                 `json:"broadened"`
	Count        int                  `json:"count"`
	MedianFee    *int64               `json:"medianFee,omitempty"`
	MinFee       *int64               `json:"minFee,omitempty"`
	MaxFee       *int64               `json:"maxFee,omitempty"`
	Comparables  []TransferComparable `json:"comparables"`
}

// SummarizeComparables fills Count and the fee summary from the comparables. With an even count the
// median is the mean of the two middle fees, rounded down; with none the fee figures stay nil.
func (r *TransferComparablesResult) SummarizeComparables() {
	r.Count = len(r.Comparables)
	r.MedianFee, r.MinFee, r.MaxFee = nil, nil, nil
	if r.Count == 0 {
		return
	}
	fees := make([]int64, r.Count)
	for i, c := range r.Comparables {
		fees[i] = c.Fee
	}
	sort.Slice(fees, func(i, j int) bool { return fees[i] < fees[j] })
	median := fees[r.Count/2]
	if r.Count%2 == 0 {
		median = (fees[r.Count/2-1] + fees[r.Count/2]) / 2
	}
	min, max := fees[0], fees[r.Count-1]
	r.MedianFee, r.MinFee, r.MaxFee = &median, &min, &max
}

func queryTransferComparables(dbCtx *DatabaseContext, target Player, position int, ageBand int, averageBand int) ([]TransferComparable, error) {
	average := AverageSkill(target)
	rows, err := dbCtx.Db.Query(getTransferComparablesStmtSql, position,
		target.Age-ageBand, target.Age+ageBand, average-averageBand, average+averageBand, target.Id, ComparablesLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comparables := make([]TransferComparable, 0)
	for rows.Next() {
		var c TransferComparable
		var first, last string
		if err := rows.Scan(&c.TransferId, &c.PlayerId, &first, &last, &c.Position, &c.Age, &c.AverageSkill,
			&c.Fee, &c.Kind, &c.CreatedAt, &c.FromTeamId, &c.FromTeamName, &c.ToTeamId, &c.ToTeamName); err != nil {
			return nil, err
		}
		c.PlayerName = first + " " + last
		comparables = append(comparables, c)
	}
	return comparables, rows.Err()
}

// GetTransferComparables finds recent sales of players like target: same best core position, and
// age and average skill (at the time of the sale) within a band of the target's current ones. The
// narrow band is tried first; fewer than ComparablesMinNarrow results retry with the wide band and
// flag the result Broadened. Needs the target's full row (target.Id and its ratings), which only
// the server reads - the result echoes just position, age and average skill.
func GetTransferComparables(dbCtx *DatabaseContext, target Player) (*TransferComparablesResult, error) {
	position := BestCorePosition(target)
	result := &TransferComparablesResult{Position: position, Age: target.Age, AverageSkill: AverageSkill(target)}

	comparables, err := queryTransferComparables(dbCtx, target, position, ComparablesNarrowAge, ComparablesNarrowAverage)
	if err != nil {
		return nil, err
	}
	if len(comparables) < ComparablesMinNarrow {
		comparables, err = queryTransferComparables(dbCtx, target, position, ComparablesWideAge, ComparablesWideAverage)
		if err != nil {
			return nil, err
		}
		result.Broadened = true
	}
	result.Comparables = comparables
	result.SummarizeComparables()
	return result, nil
}
