package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidTicketPrice = errors.New("ticket price out of range")
var ErrStadiumUpgradeInProgress = errors.New("stadium upgrade already in progress")
var ErrInvalidStadiumUpgradeSeats = errors.New("stadium upgrade seats out of range")
var ErrInvalidStadiumName = errors.New("stadium name must be between 2 and 50 characters")

const (
	MinStadiumNameLength = 2
	MaxStadiumNameLength = 50
)

// Stadium is one team's ground and pricing - see GetStadium.
type Stadium struct {
	Name        string `json:"name"`
	Capacity    int    `json:"capacity"`
	TicketPrice int    `json:"ticketPrice"`
	// PendingCapacity/UpgradeCompletesAt describe an upgrade paid for but still under
	// construction (see UpgradeStadium) - both nil when there's none in progress.
	PendingCapacity    *int       `json:"pendingCapacity,omitempty"`
	UpgradeCompletesAt *time.Time `json:"upgradeCompletesAt,omitempty"`
}

// StadiumSummary is the public-facing slice of a team's Stadium - name, capacity, and the most
// recently played home match there - shown on any team's page, unlike Stadium's ticket price and
// pending-upgrade fields, which stay owner-only (see GetFinancesHandler).
type StadiumSummary struct {
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	// LastHomeMatch is nil if this team has never played a completed home match yet.
	LastHomeMatch *StadiumLastMatch `json:"lastHomeMatch,omitempty"`
}

// StadiumLastMatch is the headline facts of one match played at a team's stadium - enough for a
// "last time out at home" summary without loading the full match report (see WeekMatchById).
type StadiumLastMatch struct {
	MatchId      int64     `json:"matchId"`
	MatchDate    time.Time `json:"matchDate"`
	OpponentId   int64     `json:"opponentId"`
	OpponentName string    `json:"opponentName"`
	HomeGoals    int       `json:"homeGoals"`
	AwayGoals    int       `json:"awayGoals"`
	Attendance   int       `json:"attendance"`
}

const GetStadiumStmtSql = "SELECT stadium_name, stadium_capacity, ticket_price, stadium_upgrade_target, stadium_upgrade_completes_at, country " +
	"FROM teams WHERE team_id = $1"
const SetStadiumNameStmtSql = "UPDATE teams SET stadium_name = $1 WHERE team_id = $2"
const SetTicketPriceStmtSql = "UPDATE teams SET ticket_price = $1 WHERE team_id = $2"

// resolveDueStadiumUpgradeStmtSql applies a stadium upgrade once its construction time has
// passed - a no-op (0 rows affected) if there's no upgrade in progress or it isn't due yet.
const resolveDueStadiumUpgradeStmtSql = "UPDATE teams SET stadium_capacity = stadium_upgrade_target, " +
	"stadium_upgrade_target = NULL, stadium_upgrade_completes_at = NULL " +
	"WHERE team_id = $1 AND stadium_upgrade_completes_at IS NOT NULL AND stadium_upgrade_completes_at <= now()"

// resolveDueStadiumUpgrade is the lazy "apply on read" resolution for a stadium upgrade whose
// StadiumUpgradeDuration has elapsed - the same pattern as common.canPromoteFromAcademyAt's
// cooldown check, just for a real-time construction delay instead. Every accessor of
// stadium_capacity in this codebase goes through GetStadium, which always calls this first, so
// there's no separate scheduled job needed to keep capacity up to date.
func resolveDueStadiumUpgrade(dbCtx *DatabaseContext, teamId int64) error {
	_, err := dbCtx.Db.Exec(resolveDueStadiumUpgradeStmtSql, teamId)
	return err
}

func GetStadium(dbCtx *DatabaseContext, teamId int64) (*Stadium, error) {
	if err := resolveDueStadiumUpgrade(dbCtx, teamId); err != nil {
		return nil, err
	}

	var s Stadium
	var name sql.NullString
	var pendingCapacity sql.NullInt64
	var completesAt sql.NullTime
	var country string
	err := dbCtx.Db.QueryRow(GetStadiumStmtSql, teamId).
		Scan(&name, &s.Capacity, &s.TicketPrice, &pendingCapacity, &completesAt, &country)
	if err != nil {
		return nil, err
	}

	if name.Valid && name.String != "" {
		s.Name = name.String
	} else {
		// A team created before stadium_name existed - generate and persist one now rather than
		// showing a blank name forever.
		s.Name = RandomStadiumName(country)
		if _, err := dbCtx.Db.Exec(SetStadiumNameStmtSql, s.Name, teamId); err != nil {
			return nil, err
		}
	}

	if pendingCapacity.Valid {
		capacity := int(pendingCapacity.Int64)
		s.PendingCapacity = &capacity
		s.UpgradeCompletesAt = &completesAt.Time
	}

	return &s, nil
}

// RenameStadium lets a team's owner rename their own stadium at any time - unlike the team name
// itself (see RenameTeam), there's no one-shot restriction here.
func RenameStadium(dbCtx *DatabaseContext, teamId int64, name string) error {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) < MinStadiumNameLength || len(trimmed) > MaxStadiumNameLength {
		return ErrInvalidStadiumName
	}
	_, err := dbCtx.Db.Exec(SetStadiumNameStmtSql, trimmed, teamId)
	return err
}

// SetTicketPrice lets a team's owner set what they charge per seat, within
// [MinTicketPrice, MaxTicketPrice].
func SetTicketPrice(dbCtx *DatabaseContext, teamId int64, price int) error {
	if price < MinTicketPrice || price > MaxTicketPrice {
		return ErrInvalidTicketPrice
	}
	_, err := dbCtx.Db.Exec(SetTicketPriceStmtSql, price, teamId)
	return err
}

const GetStadiumCapacityForUpdateStmtSql = "SELECT stadium_capacity, stadium_upgrade_target FROM teams WHERE team_id = $1 " + ForUpdate
const SetStadiumUpgradeStmtSql = "UPDATE teams SET stadium_upgrade_target = $1, stadium_upgrade_completes_at = $2 WHERE team_id = $3"

// UpgradeStadium charges a team's budget for adding seats new seats to its *current* capacity -
// cost scales with both the current capacity and the number of seats requested (relative to the
// StadiumUpgradeSeats baseline), so a bigger stadium or a bigger expansion both cost more in
// absolute terms - and schedules those seats to go live after StadiumUpgradeDuration
// (construction isn't instant; see resolveDueStadiumUpgrade for how that's later applied). Both
// the capacity and budget rows are locked for the duration (SELECT ... FOR UPDATE), so two
// upgrades racing each other can't both succeed off the same starting capacity/budget. Returns
// ErrInvalidStadiumUpgradeSeats if seats is outside [MinStadiumUpgradeSeats,
// MaxStadiumUpgradeSeats], ErrStadiumUpgradeInProgress if one is already under way, or
// ErrInsufficientBudget if the team can't afford it.
func UpgradeStadium(dbCtx *DatabaseContext, teamId int64, seats int) (*Stadium, error) {
	if seats < MinStadiumUpgradeSeats || seats > MaxStadiumUpgradeSeats {
		return nil, ErrInvalidStadiumUpgradeSeats
	}

	if err := resolveDueStadiumUpgrade(dbCtx, teamId); err != nil {
		return nil, err
	}

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var capacity int
	var pendingTarget sql.NullInt64
	if err := tx.QueryRow(GetStadiumCapacityForUpdateStmtSql, teamId).Scan(&capacity, &pendingTarget); err != nil {
		return nil, err
	}
	if pendingTarget.Valid {
		return nil, ErrStadiumUpgradeInProgress
	}

	cost := int64(capacity) * StadiumUpgradeCostPerCurrentSeat * int64(seats) / StadiumUpgradeSeats

	var budget int64
	if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, teamId).Scan(&budget); err != nil {
		return nil, err
	}
	if budget < cost {
		return nil, ErrInsufficientBudget
	}

	newCapacity := capacity + seats
	completesAt := time.Now().Add(StadiumUpgradeDuration)
	if _, err := tx.Exec(SetStadiumUpgradeStmtSql, newCapacity, completesAt, teamId); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -cost, teamId); err != nil {
		return nil, err
	}
	description := fmt.Sprintf("Stadium expansion to %d seats under construction", newCapacity)
	var flows LedgerFlows
	if err := insertLedgerRow(tx, &flows, teamId, -cost, TRANSACTION_STADIUM_UPGRADE, description, nil); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	flows.Commit(context.Background())
	return GetStadium(dbCtx, teamId)
}

const getLastHomeMatchStmtSql = "SELECT m.match_id, m.match_date, m.away_team_id, t.name, m.home_goals, m.away_goals, m.attendance " +
	"FROM matches m JOIN teams t ON t.team_id = m.away_team_id " +
	"WHERE m.home_team_id = $1 AND m.played = true AND m.match_date <= $2 " +
	"ORDER BY m.match_date DESC LIMIT 1"

// GetLastHomeMatch reports the most recent completed league match teamId played at home, or nil
// if it's never played one yet. Restricted to league matches, like GetPlayerRecentMatches - cup
// ties and friendlies live in their own tables (cup_ties, friendly_challenges) and never generate
// gate receipts/attendance to begin with. Excludes a match still inside its live-broadcast window
// (see LiveBroadcastCutoff), so a stadium summary can't show a match as finished before its own
// report stops treating it as "LIVE".
func GetLastHomeMatch(dbCtx *DatabaseContext, teamId int64) (*StadiumLastMatch, error) {
	var m StadiumLastMatch
	var attendance sql.NullInt64
	err := dbCtx.Db.QueryRow(getLastHomeMatchStmtSql, teamId, LiveBroadcastCutoff()).
		Scan(&m.MatchId, &m.MatchDate, &m.OpponentId, &m.OpponentName, &m.HomeGoals, &m.AwayGoals, &attendance)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Attendance = int(attendance.Int64)
	return &m, nil
}
