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
