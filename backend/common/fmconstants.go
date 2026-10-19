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

// MATCH_ABANDONED marks a match ended early because one side could no longer field enough
// players to continue (see engine.applyForfeitIfUnderstaffed) - TeamOwner is the side left
// short-handed, OtherTeam the side awarded the forfeit win.
const MATCH_ABANDONED = 6

// player_unavailability.reason
const UNAVAILABLE_SUSPENSION = 0
const UNAVAILABLE_INJURY = 1

// team_transactions.reason
const TRANSACTION_TRANSFER_IN = 0             // sold a player - income
const TRANSACTION_TRANSFER_OUT = 1            // bought a player - expense
const TRANSACTION_WAGES = 2                   // wage bill for a played match - expense
const TRANSACTION_ADMIN_ADJUSTMENT = 3        // a commissioner's manual budget correction
const TRANSACTION_TICKET_REVENUE = 4          // matchday ticket sales for a home match - income
const TRANSACTION_PRIZE_MONEY = 5             // season league-position or cup prize money - income
const TRANSACTION_SPONSORSHIP = 6             // per-match sponsorship income - income
const TRANSACTION_STADIUM_UPGRADE = 7         // capacity expansion - expense
const TRANSACTION_CONTRACT_RELEASE = 8        // severance paid to release a player early - expense
const TRANSACTION_TRAINER_WAGES = 9           // wage bill for an employed head coach, paid alongside player wages - expense
const TRANSACTION_SEASON_TICKETS = 10         // once-a-season lump sum paid for stadium capacity, at rollover - income
const TRANSACTION_MERCHANDISING = 11          // once-a-season shirt/merchandise income by league position, at rollover - income
const TRANSACTION_ADMIN_MASS_INJECTION = 12   // a commissioner's budget adjustment applied to every team at once
const TRANSACTION_LOAN_FEE = 13               // one-time fee paid by a borrower to accept a loan - expense for the borrower, income for the lender
const TRANSACTION_LOAN_WAGE_CONTRIBUTION = 14 // a parent club covering part of a loaned-out player's wage - expense for the lender
const TRANSACTION_STADIUM_MAINTENANCE = 15    // upkeep charged once per round, scaled to the stadium's current capacity - expense

// DefaultStadiumCapacity is every new team's starting stadium capacity (see the initializer's
// createTeam), regardless of the league tier it starts in - flat and fixed rather than random, so
// every manager starts with the same-size ground. From here on it only ever changes through
// common.UpgradeStadium.
//
// Lowered from 10000: at that capacity, a single home match's ticket revenue alone (attendance x
// optimalTicketPrice - see engine.optimalTicketPrice) ran upwards of 180,000, several times an
// average 18-player squad's entire ~28,350/week wage bill (see teams.budget's comment in
// resources/sql/init.sql). Combined with wages now being charged once per round instead of once
// per real calendar week (see engine.PayMatchdayWages), a stadium this big no longer also needed
// to dwarf payroll on its own for a well-run team to turn a healthy profit. 1000 keeps a home
// match's expected ticket revenue in the same order of magnitude as the wage bill it's meant to
// help cover, leaving prize money/season tickets/merchandising and match results (sponsorship
// bonuses, league position) as the actual difference between a team that profits and one that
// doesn't, rather than raw stadium size alone.
const DefaultStadiumCapacity = 1000

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

// StadiumMaintenanceCostPerSeat/StadiumMaintenanceCostDivisor price a stadium's upkeep linearly
// in its current capacity - see engine.StadiumMaintenanceCost - so a bigger ground costs more to
// run every round, not just more to build (UpgradeStadium above was previously the only cost tied
// to size, making "upgrade forever" a free lunch). Charged once per round alongside wages (see
// engine.PayMatchdayWages), for the same cadence reason wages are: tying it to the league's own
// match cadence keeps a season's total upkeep proportional to its total match count rather than
// to real calendar time. At DefaultStadiumCapacity (1000) this is 100/round - a real but modest
// bite, roughly in line with a mid-skill trainer's own per-round wage rather than a whole squad's;
// needs balancing/playtesting like every other economy constant here.
const StadiumMaintenanceCostPerSeat = 1
const StadiumMaintenanceCostDivisor = 10

// StadiumUpgradeDuration is how long a paid-for stadium upgrade takes to finish construction
// before UpgradeStadium's extra seats actually become usable - a real week, not tied to match
// simulation cadence.
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

// NOTIFICATION_OFFER_DECLINED tells a buyer their transfer offer was rejected by the selling
// club - see common.RejectOffer, which links it straight back to the player's page so the buyer
// can renegotiate with new terms without hunting for the player again.
const NOTIFICATION_OFFER_DECLINED = 10

// NOTIFICATION_FANBASE tells a team's owner their Fanbase crossed into a new ten-point band
// (grew or shrank) - see engine.notifyFanbaseMilestone.
const NOTIFICATION_FANBASE = 11

// NOTIFICATION_CALL_UP tells a club their player was called up to their national team's starting
// lineup for this week's international friendly - see common.SetNationalTeamLineup.
const NOTIFICATION_CALL_UP = 12

// player_transfers.kind - see RecordPlayerTransfer. Stored as a number so the frontend can
// translate it (playerPage.transfers.kinds.*) instead of showing free-form English text.
const PLAYER_TRANSFER_LISTING = 0    // bought a transfer-listed player at its asking price
const PLAYER_TRANSFER_OFFER = 1      // a negotiated offer the selling club accepted
const PLAYER_TRANSFER_FREE_AGENT = 2 // signed as a free agent - fee is the premium paid to the last club, if any
const PLAYER_TRANSFER_LOAN = 3       // temporary loan - fee is the loan fee, if any
const PLAYER_TRANSFER_BUY_OPTION = 4 // a loan made permanent by exercising its buy option
const PLAYER_TRANSFER_RELEASE = 5    // contract terminated early or expired - no destination club
const PLAYER_TRANSFER_PERMANENT = 6  // a permanent transfer from before this table existed (listing vs offer unknown)
