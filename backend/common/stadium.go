package common

import (
	"errors"
	"fmt"
	"math/rand"
)

var ErrInvalidTicketPrice = errors.New("ticket price out of range")

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
	Capacity    int `json:"capacity"`
	TicketPrice int `json:"ticketPrice"`
}

const GetStadiumStmtSql = "SELECT stadium_capacity, ticket_price FROM teams WHERE team_id = $1"
const SetTicketPriceStmtSql = "UPDATE teams SET ticket_price = $1 WHERE team_id = $2"

func GetStadium(dbCtx *DatabaseContext, teamId int64) (*Stadium, error) {
	var s Stadium
	err := dbCtx.Db.QueryRow(GetStadiumStmtSql, teamId).Scan(&s.Capacity, &s.TicketPrice)
	if err != nil {
		return nil, err
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

const GetStadiumCapacityForUpdateStmtSql = "SELECT stadium_capacity FROM teams WHERE team_id = $1 " + ForUpdate
const UpdateStadiumCapacityStmtSql = "UPDATE teams SET stadium_capacity = $1 WHERE team_id = $2"

// UpgradeStadium adds StadiumUpgradeSeats to a team's capacity, charging its budget
// StadiumUpgradeCostPerCurrentSeat times the *current* capacity - so the same fixed expansion
// costs more in absolute terms each time the stadium grows. Both the capacity and budget rows
// are locked for the duration (SELECT ... FOR UPDATE), so two upgrades racing each other can't
// both succeed off the same starting capacity/budget. Returns the new capacity on success, or
// ErrInsufficientBudget if the team can't afford it.
func UpgradeStadium(dbCtx *DatabaseContext, teamId int64) (int, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var capacity int
	if err := tx.QueryRow(GetStadiumCapacityForUpdateStmtSql, teamId).Scan(&capacity); err != nil {
		return 0, err
	}

	cost := int64(capacity) * StadiumUpgradeCostPerCurrentSeat

	var budget int64
	if err := tx.QueryRow(GetTeamBudgetStmtSql+" "+ForUpdate, teamId).Scan(&budget); err != nil {
		return 0, err
	}
	if budget < cost {
		return 0, ErrInsufficientBudget
	}

	newCapacity := capacity + StadiumUpgradeSeats
	if _, err := tx.Exec(UpdateStadiumCapacityStmtSql, newCapacity, teamId); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, -cost, teamId); err != nil {
		return 0, err
	}
	description := fmt.Sprintf("Stadium expanded to %d seats", newCapacity)
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, teamId, -cost, TRANSACTION_STADIUM_UPGRADE, description); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return newCapacity, nil
}
