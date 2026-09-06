package common

const COOKIE = "jwt"
const UserId = "id"
const PictureUrl = "pictureUrl"

const TeamsPerLeague = 10

// Positions
const GOALKEEPER = 0
const DEFENSE = 1
const LATERAL_DEFENSE_LEFT = 2
const LATERAL_DEFENSE_RIGHT = 3
const MIDFIELD = 4
const LATERAL_MIDFIELD_LEFT = 5
const LATERAL_MIDFIELD_RIGHT = 6
const STRIKER = 7
const LATERAL_STRIKER_LEFT = 8
const LATERAL_STRIKER_RIGHT = 9

// ALL_OUTFIELD_POSITIONS is every tactical position except the goalkeeper.
var ALL_OUTFIELD_POSITIONS = []int{
	DEFENSE, LATERAL_DEFENSE_LEFT, LATERAL_DEFENSE_RIGHT,
	MIDFIELD, LATERAL_MIDFIELD_LEFT, LATERAL_MIDFIELD_RIGHT,
	STRIKER, LATERAL_STRIKER_LEFT, LATERAL_STRIKER_RIGHT,
}

// Events
const GOAL = 0
const NO_GOAL = 1
const YELLOW_CARD = 2
const RED_CARD = 3
const INJURY = 4

// player_unavailability.reason
const UNAVAILABLE_SUSPENSION = 0
const UNAVAILABLE_INJURY = 1

// team_transactions.reason
const TRANSACTION_TRANSFER_IN = 0      // sold a player - income
const TRANSACTION_TRANSFER_OUT = 1     // bought a player - expense
const TRANSACTION_WAGES = 2            // wage bill for a played match - expense
const TRANSACTION_ADMIN_ADJUSTMENT = 3 // a commissioner's manual budget correction
const TRANSACTION_TICKET_REVENUE = 4   // matchday ticket sales for a home match - income
const TRANSACTION_PRIZE_MONEY = 5      // season league-position or cup prize money - income
const TRANSACTION_SPONSORSHIP = 6      // per-match sponsorship income - income
const TRANSACTION_STADIUM_UPGRADE = 7  // capacity expansion - expense
const TRANSACTION_CONTRACT_RELEASE = 8 // severance paid to release a player early - expense

// Stadium capacity is picked once at team creation, scaled by the league tier it starts in
// (see the initializer's createTeam and common.RandomStadiumCapacity) - bigger clubs get
// bigger stadiums, and promotion/relegation carries a real financial stake beyond prestige.
const StadiumCapacityTier1Min = 3000
const StadiumCapacityTier1Max = 6000
const StadiumCapacityTier2Min = 1500
const StadiumCapacityTier2Max = 3500
const StadiumCapacityTier3Min = 500
const StadiumCapacityTier3Max = 2000

// DefaultTicketPrice is what a new team starts charging per seat; MinTicketPrice/MaxTicketPrice
// bound what an owner can set it to (see handlers.SetTicketPriceHandler).
const DefaultTicketPrice = 10
const MinTicketPrice = 1
const MaxTicketPrice = 100

// A stadium upgrade adds a fixed number of seats at a cost proportional to the *current*
// capacity, so expanding an already-big stadium costs more in absolute terms each time -
// see UpgradeStadium.
const StadiumUpgradeSeats = 500
const StadiumUpgradeCostPerCurrentSeat = 10

// notifications.notification_type
const NOTIFICATION_MATCH_RESULT = 0
const NOTIFICATION_PROMOTION = 1
const NOTIFICATION_RELEGATION = 2
const NOTIFICATION_TRANSFER = 3
const NOTIFICATION_CUP = 4
