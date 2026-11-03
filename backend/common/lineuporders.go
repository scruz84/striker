package common

import (
	"database/sql"
	"fmt"
)

// Conditional match orders (see openspec/specs/match-simulation "Conditional match orders"): up to
// MaxConditionalOrders per lineup, each armed from its FromMinute and fired once, in slot order,
// at the first minute whose starting score meets its Condition (from the side's own point of
// view), with one Action - set the mentality, or substitute a starter for a bench player.
const (
	ORDER_CONDITION_LOSING       = 0
	ORDER_CONDITION_LOSING_BY_2  = 1
	ORDER_CONDITION_DRAWING      = 2
	ORDER_CONDITION_WINNING      = 3
	ORDER_CONDITION_WINNING_BY_2 = 4

	ORDER_ACTION_MENTALITY    = 0
	ORDER_ACTION_SUBSTITUTION = 1

	MaxConditionalOrders = 3
	// OrderMinFromMinute / OrderMaxFromMinute bound an order's from-minute.
	OrderMinFromMinute = 1
	OrderMaxFromMinute = 89
)

// LineupOrder is one conditional order. Mentality is meaningful for ORDER_ACTION_MENTALITY,
// PlayerOutId/PlayerInId for ORDER_ACTION_SUBSTITUTION (0 otherwise).
type LineupOrder struct {
	FromMinute  int   `json:"fromMinute"`
	Condition   int   `json:"condition"`
	Action      int   `json:"action"`
	Mentality   int   `json:"mentality"`
	PlayerOutId int64 `json:"playerOutId,omitempty"`
	PlayerInId  int64 `json:"playerInId,omitempty"`
}

// OrderConditionHolds reports whether condition holds for a side with goalsFor and goalsAgainst.
func OrderConditionHolds(condition int, goalsFor int, goalsAgainst int) bool {
	diff := goalsFor - goalsAgainst
	switch condition {
	case ORDER_CONDITION_LOSING:
		return diff < 0
	case ORDER_CONDITION_LOSING_BY_2:
		return diff <= -2
	case ORDER_CONDITION_DRAWING:
		return diff == 0
	case ORDER_CONDITION_WINNING:
		return diff > 0
	case ORDER_CONDITION_WINNING_BY_2:
		return diff >= 2
	}
	return false
}

// ValidateOrders checks a lineup's conditional orders against the XI saved with them (positions):
// at most MaxConditionalOrders, a from-minute from 1 to 89, a known condition and action, a valid
// mentality for a mentality action, and for a substitution action the same player checks as a
// scheduled substitution (the outgoing player starts, the incoming one is an available squad
// player on the bench). The same player may be the outgoing player of several orders - the first
// to fire takes him off and the others are spent.
func ValidateOrders(positions []LineupPosition, orders []LineupOrder, teamPlayers *[]*Player, unavailable map[int64]bool) error {
	if len(orders) > MaxConditionalOrders {
		return fmt.Errorf("at most %d conditional orders may be set, got %d", MaxConditionalOrders, len(orders))
	}
	starters := make(map[int64]bool, len(positions))
	for _, pos := range positions {
		starters[pos.PlayerId] = true
	}
	roster := make(map[int64]bool, len(*teamPlayers))
	for _, p := range *teamPlayers {
		roster[p.Id] = true
	}
	for i, o := range orders {
		if o.FromMinute < OrderMinFromMinute || o.FromMinute > OrderMaxFromMinute {
			return fmt.Errorf("order %d: from-minute must be between %d and %d, got %d", i+1, OrderMinFromMinute, OrderMaxFromMinute, o.FromMinute)
		}
		if o.Condition < ORDER_CONDITION_LOSING || o.Condition > ORDER_CONDITION_WINNING_BY_2 {
			return fmt.Errorf("order %d: unknown condition %d", i+1, o.Condition)
		}
		switch o.Action {
		case ORDER_ACTION_MENTALITY:
			if err := ValidateMentality(o.Mentality); err != nil {
				return fmt.Errorf("order %d: %w", i+1, err)
			}
		case ORDER_ACTION_SUBSTITUTION:
			if !starters[o.PlayerOutId] {
				return fmt.Errorf("order %d: player %d being substituted out is not in the starting lineup", i+1, o.PlayerOutId)
			}
			if !roster[o.PlayerInId] {
				return fmt.Errorf("order %d: player %d does not belong to this team", i+1, o.PlayerInId)
			}
			if starters[o.PlayerInId] {
				return fmt.Errorf("order %d: player %d being substituted in is already in the starting lineup", i+1, o.PlayerInId)
			}
			if unavailable[o.PlayerInId] {
				return fmt.Errorf("order %d: player %d is suspended or injured and unavailable", i+1, o.PlayerInId)
			}
		default:
			return fmt.Errorf("order %d: unknown action %d", i+1, o.Action)
		}
	}
	return nil
}

// Each lineup kind keeps its orders in its own table, keyed like its scheduled substitutions.
const orderColumns = "from_minute, condition, action, mentality, player_out_id, player_in_id"

const GetDefaultLineupOrdersStmtSql = "SELECT " + orderColumns + " FROM default_lineup_orders WHERE team_id = $1 ORDER BY slot"
const DeleteDefaultLineupOrdersStmtSql = "DELETE FROM default_lineup_orders WHERE team_id = $1"
const InsertDefaultLineupOrderStmtSql = "INSERT INTO default_lineup_orders (team_id, slot, " + orderColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"

const GetMatchLineupOrdersStmtSql = "SELECT " + orderColumns + " FROM lineup_orders WHERE match_id = $1 AND team_id = $2 ORDER BY slot"
const DeleteMatchLineupOrdersStmtSql = "DELETE FROM lineup_orders WHERE match_id = $1 AND team_id = $2"
const InsertMatchLineupOrderStmtSql = "INSERT INTO lineup_orders (match_id, team_id, slot, " + orderColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"

const GetCupTieLineupOrdersStmtSql = "SELECT " + orderColumns + " FROM cup_tie_lineup_orders WHERE tie_id = $1 AND team_id = $2 ORDER BY slot"
const DeleteCupTieLineupOrdersStmtSql = "DELETE FROM cup_tie_lineup_orders WHERE tie_id = $1 AND team_id = $2"
const InsertCupTieLineupOrderStmtSql = "INSERT INTO cup_tie_lineup_orders (tie_id, team_id, slot, " + orderColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"

const GetFriendlyLineupOrdersStmtSql = "SELECT " + orderColumns + " FROM friendly_lineup_orders WHERE match_id = $1 AND team_id = $2 ORDER BY slot"
const DeleteFriendlyLineupOrdersStmtSql = "DELETE FROM friendly_lineup_orders WHERE match_id = $1 AND team_id = $2"
const InsertFriendlyLineupOrderStmtSql = "INSERT INTO friendly_lineup_orders (match_id, team_id, slot, " + orderColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"

type sqlQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func queryOrders(q sqlQueryer, query string, args ...any) ([]LineupOrder, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := make([]LineupOrder, 0, MaxConditionalOrders)
	for rows.Next() {
		var o LineupOrder
		var mentality sql.NullInt64
		var out, in sql.NullInt64
		if err := rows.Scan(&o.FromMinute, &o.Condition, &o.Action, &mentality, &out, &in); err != nil {
			return nil, err
		}
		o.Mentality, o.PlayerOutId, o.PlayerInId = int(mentality.Int64), out.Int64, in.Int64
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// orderValues are an order's stored columns: the fields that don't apply to its action are NULL.
func orderValues(o LineupOrder) []any {
	var mentality, out, in any
	if o.Action == ORDER_ACTION_MENTALITY {
		mentality = o.Mentality
	} else {
		out, in = o.PlayerOutId, o.PlayerInId
	}
	return []any{o.FromMinute, o.Condition, o.Action, mentality, out, in}
}

// replaceOrders replaces the orders under key (the owner columns) inside ex, in slot order.
func replaceOrders(ex sqlExecutor, deleteSql string, insertSql string, key []any, orders []LineupOrder) error {
	if _, err := ex.Exec(deleteSql, key...); err != nil {
		return err
	}
	for i, o := range orders {
		args := append(append(append([]any{}, key...), i+1), orderValues(o)...)
		if _, err := ex.Exec(insertSql, args...); err != nil {
			return err
		}
	}
	return nil
}

// GetDefaultLineupOrders / SetDefaultLineupOrders read and replace the orders of a team's standing
// league/cup default lineup. (The friendly default carries none.)
func GetDefaultLineupOrders(dbCtx *DatabaseContext, teamId int64) ([]LineupOrder, error) {
	return queryOrders(dbCtx.Db, GetDefaultLineupOrdersStmtSql, teamId)
}

func SetDefaultLineupOrders(dbCtx *DatabaseContext, teamId int64, orders []LineupOrder) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := replaceOrders(tx, DeleteDefaultLineupOrdersStmtSql, InsertDefaultLineupOrderStmtSql, []any{teamId}, orders); err != nil {
		return err
	}
	return tx.Commit()
}

// GetMatchLineupOrders, GetCupTieLineupOrders and GetFriendlyLineupOrders read the orders saved
// with one fixture's lineup override (see SetLineupForMatch, SetCupTieLineup, SetFriendlyLineup).
func GetMatchLineupOrders(dbCtx *DatabaseContext, matchId int64, teamId int64) ([]LineupOrder, error) {
	return queryOrders(dbCtx.Db, GetMatchLineupOrdersStmtSql, matchId, teamId)
}

func GetCupTieLineupOrders(dbCtx *DatabaseContext, tieId int64, teamId int64) ([]LineupOrder, error) {
	return queryOrders(dbCtx.Db, GetCupTieLineupOrdersStmtSql, tieId, teamId)
}

func GetFriendlyLineupOrders(dbCtx *DatabaseContext, matchId int64, teamId int64) ([]LineupOrder, error) {
	return queryOrders(dbCtx.Db, GetFriendlyLineupOrdersStmtSql, matchId, teamId)
}

// GetOrdersForMatch resolves the orders a team plays one league match with: its override's if it
// saved one for this match at all, else its standing default's - the same all-or-nothing rule as
// GetSubstitutionsForMatch.
func GetOrdersForMatch(dbCtx *DatabaseContext, matchId int64, teamId int64) ([]LineupOrder, error) {
	var hasOverride bool
	if err := dbCtx.Db.QueryRow(MatchLineupOverrideExistsStmtSql, matchId, teamId).Scan(&hasOverride); err != nil {
		return nil, err
	}
	if hasOverride {
		return GetMatchLineupOrders(dbCtx, matchId, teamId)
	}
	return GetDefaultLineupOrders(dbCtx, teamId)
}

// GetOrdersForCupTie is GetOrdersForMatch for a cup tie.
func GetOrdersForCupTie(dbCtx *DatabaseContext, tieId int64, teamId int64) ([]LineupOrder, error) {
	var hasOverride bool
	if err := dbCtx.Db.QueryRow(CupTieLineupOverrideExistsStmtSql, tieId, teamId).Scan(&hasOverride); err != nil {
		return nil, err
	}
	if hasOverride {
		return GetCupTieLineupOrders(dbCtx, tieId, teamId)
	}
	return GetDefaultLineupOrders(dbCtx, teamId)
}

// nullableInt is a nullable integer column as *int (nil for NULL) - Event.Mentality.
func nullableInt(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int64)
	return &i
}
