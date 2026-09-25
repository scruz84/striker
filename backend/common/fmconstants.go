package common

import "time"

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
const SUBSTITUTION = 5

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
const TRANSACTION_TRAINER_WAGES = 9    // wage bill for an employed head coach, paid alongside player wages - expense
const TRANSACTION_SEASON_TICKETS = 10  // once-a-season lump sum paid for stadium capacity, at rollover - income
const TRANSACTION_MERCHANDISING = 11   // once-a-season shirt/merchandise income by league position, at rollover - income

// DefaultStadiumCapacity is every new team's starting stadium capacity (see the initializer's
// createTeam), regardless of the league tier it starts in - flat and fixed rather than random, so
// every manager starts with the same-size ground. From here on it only ever changes through
// common.UpgradeStadium.
const DefaultStadiumCapacity = 10000

// DefaultTicketPrice is what a new team starts charging per seat; MinTicketPrice/MaxTicketPrice
// bound what an owner can set it to (see handlers.SetTicketPriceHandler).
const DefaultTicketPrice = 10
const MinTicketPrice = 1
const MaxTicketPrice = 100

// A stadium upgrade adds an owner-chosen number of seats (see UpgradeStadium and
// MinStadiumUpgradeSeats/MaxStadiumUpgradeSeats for the range they can pick from), at a cost
// proportional to both the *current* capacity and the number of seats requested - so expanding
// an already-big stadium, or requesting more seats at once, both cost more in absolute terms.
// StadiumUpgradeSeats is the fixed amount AI-managed teams request (see
// engine.upgradeStadiumIfAffordable) and the baseline the cost formula is scaled against.
const StadiumUpgradeSeats = 500
const MinStadiumUpgradeSeats = 100
const MaxStadiumUpgradeSeats = 5000
const StadiumUpgradeCostPerCurrentSeat = 10

// StadiumUpgradeDuration is how long a paid-for stadium upgrade takes to finish construction
// before UpgradeStadium's extra seats actually become usable - a real week, not tied to match
// simulation, the same way weekly wages are (see engine.PayWeeklyWages).
const StadiumUpgradeDuration = 7 * 24 * time.Hour

// MaxDebt is how far a team's budget may go negative before IsTeamSolvent starts blocking new
// spending (buying, signing, borrowing on loan) - half the 400,000 a team starts with, generous
// enough that a genuine cash crunch from a bad transfer window or a run of empty stadiums isn't
// an immediate hard stop, but real enough that budget mismanagement eventually has a consequence
// beyond an ever-more-negative number nobody has to answer for. Selling, releasing, and cancelling
// are never blocked by this - those are exactly how a team digs back out.
const MaxDebt = 200000

// notifications.notification_type
const NOTIFICATION_MATCH_RESULT = 0
const NOTIFICATION_PROMOTION = 1
const NOTIFICATION_RELEGATION = 2
const NOTIFICATION_TRANSFER = 3
const NOTIFICATION_CUP = 4
const NOTIFICATION_TRAINER = 5
const NOTIFICATION_FRIENDLY = 6
const NOTIFICATION_ACADEMY_PROMOTION = 7
const NOTIFICATION_PROGRESSION = 8

// Golden Boot and Player of the Season share this one type, same as NOTIFICATION_CUP covers both
// a tie result and the champion - distinguished by NotificationMessage.Key, not by type.
const NOTIFICATION_AWARD = 9
