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
const TRANSACTION_TRANSFER_IN = 0  // sold a player - income
const TRANSACTION_TRANSFER_OUT = 1 // bought a player - expense
const TRANSACTION_WAGES = 2        // wage bill for a played match - expense

// notifications.notification_type
const NOTIFICATION_MATCH_RESULT = 0
const NOTIFICATION_PROMOTION = 1
const NOTIFICATION_RELEGATION = 2
const NOTIFICATION_TRANSFER = 3
const NOTIFICATION_CUP = 4
