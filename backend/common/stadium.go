package common

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

var ErrInvalidTicketPrice = errors.New("ticket price out of range")
var ErrStadiumUpgradeInProgress = errors.New("stadium upgrade already in progress")

// RandomStadiumCapacity picks a new team's stadium capacity for the tier it starts in (see the
// StadiumCapacityTier*Min/Max constants). An unrecognized tier falls back to the tier-3
// (smallest) range rather than panicking, since a tier the seed data doesn't create yet
// shouldn't be able to crash team creation.
func RandomStadiumCapacity(tier int, r *rand.Rand) int {
	min, max := StadiumCapacityTier3Min, StadiumCapacityTier3Max
	switch tier {
	case 1:
		min, max = StadiumCapacityTier1Min, StadiumCapacityTier1Max
	case 2:
		min, max = StadiumCapacityTier2Min, StadiumCapacityTier2Max
	}
	return min + r.Intn(max-min+1)
}

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

// UpgradeStadium charges a team's budget StadiumUpgradeCostPerCurrentSeat times its *current*
// capacity - so the same fixed expansion costs more in absolute terms each time the stadium
// grows - and schedules StadiumUpgradeSeats more capacity to go live after StadiumUpgradeDuration
// (construction isn't instant; see resolveDueStadiumUpgrade for how that's later applied). Both
// the capacity and budget rows are locked for the duration (SELECT ... FOR UPDATE), so two
// upgrades racing each other can't both succeed off the same starting capacity/budget. Returns
// ErrStadiumUpgradeInProgress if one is already under way, or ErrInsufficientBudget if the team
// can't afford it.
func UpgradeStadium(dbCtx *DatabaseContext, teamId int64) (*Stadium, error) {
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

	cost := int64(capacity) * StadiumUpgradeCostPerCurrentSeat

	var budget int64
	if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, teamId).Scan(&budget); err != nil {
		return nil, err
	}
	if budget < cost {
		return nil, ErrInsufficientBudget
	}

	newCapacity := capacity + StadiumUpgradeSeats
	completesAt := time.Now().Add(StadiumUpgradeDuration)
	if _, err := tx.Exec(SetStadiumUpgradeStmtSql, newCapacity, completesAt, teamId); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -cost, teamId); err != nil {
		return nil, err
	}
	description := fmt.Sprintf("Stadium expansion to %d seats under construction", newCapacity)
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, teamId, -cost, TRANSACTION_STADIUM_UPGRADE, description, nil); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return GetStadium(dbCtx, teamId)
}
