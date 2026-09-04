package common

import (
	"errors"
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
