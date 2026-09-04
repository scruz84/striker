// These types mirror the JSON shape of backend/common/fmstructs.go. Field names follow the
// backend's `json:"..."` tags, not the Go field names, since that's what actually goes over
// the wire.

export interface LoggedUser {
  id: string
  name: string
  provider: string
  email: string
  country: string
  teamId: string
  teamName: string
  teamLeague: string
  pictureUrl: string
  isAdmin: boolean
}

export interface PlayerAvatar {
  accessoriesType: string
  avatarStyle: string
  clotheColor: string
  clotheType: string
  eyeType: string
  eyebrowType: string
  facialHairColor: string
  facialHairType: string
  graphicType: string
  hairColor: string
  hatColor: string
  mouthType: string
  skinColor: string
  topType: string
}

export interface Player {
  id: number
  teamId: number
  firstName: string
  lastName: string
  birthDay: string
  goalKeeper: number
  defense: number
  lateral: number
  pass: number
  midfield: number
  dribbling: number
  shot: number
  header: boolean
  quick: boolean
  creative: boolean
  technical: boolean
  wage: number
  age: number
  avatar: PlayerAvatar
}

export interface Team {
  id: number
  ownerId: number
  ownerName: string
  country: string
  name: string
  divisionId: number
  divisionName: string
  classificationPosition: number
}

export interface TeamDetails {
  team: Team
  players: Player[]
  lastMatches: WeekMatch[] | null
  nextMatches: WeekMatch[] | null
}

export interface LineupPosition {
  playerId: number
  position: number
}

export interface Lineup {
  matchId: number
  teamId: number
  positions: LineupPosition[]
}

export interface MatchEvent {
  eventType: number
  minute: number
  teamOwner: number
  teamOwnerPlayers: number[] | null
  teamOwnerPlayerNames: string[] | null
  otherTeam: number
  otherTeamPlayers: number[] | null
  otherTeamPlayerNames: string[] | null
  homeGoals: number
  awayGoals: number
}

// EventType values from backend/common/fmconstants.go
export const EVENT_TYPE_GOAL = 0
export const EVENT_TYPE_NO_GOAL = 1
export const EVENT_TYPE_YELLOW_CARD = 2
export const EVENT_TYPE_RED_CARD = 3
export const EVENT_TYPE_INJURY = 4

export interface MatchInstant {
  matchId: number
  minute: number
  homeGoals: number
  awayGoals: number
  midfieldDiff: number
  homeAttackRightDiff: number
  homeAttackLeftDiff: number
  homeAttackCenterDiff: number
  awayAttackRightDiff: number
  awayAttackLeftDiff: number
  awayAttackCenterDiff: number
}

export interface WeekMatch {
  matchId: number
  divisionId: number
  season: number
  weekNumber: number
  matchDate: string
  played: boolean
  inSimulation: boolean
  home: number
  homeName: string
  away: number
  awayName: string
  homeGoals: number
  awayGoals: number
  events: MatchEvent[] | null
  matchInstants: MatchInstant[] | null
  homeLineup: Lineup | null
  awayLineup: Lineup | null
  homePlayers: Record<string, Player> | null
  awayPlayers: Record<string, Player> | null
}

export interface ClassificationPosition {
  position: number
  teamId: number
  teamName: string
  points: number
  matches: number
  wins: number
  draws: number
  losses: number
  goalsScored: number
  goalsReceived: number
}

export interface Classification {
  positions: ClassificationPosition[]
}

export interface DivisionDetails {
  id: number
  season: number
  name: string
  classification: Classification
  previousWeekMatches: WeekMatch[]
  nextWeekMatches: WeekMatch[]
}

// Tactical position ids from backend/common/fmconstants.go - shared by Lineup and
// the lineup editor.
export const POSITION_GOALKEEPER = 0
export const POSITION_DEFENSE = 1
export const POSITION_LATERAL_DEFENSE_LEFT = 2
export const POSITION_LATERAL_DEFENSE_RIGHT = 3
export const POSITION_MIDFIELD = 4
export const POSITION_LATERAL_MIDFIELD_LEFT = 5
export const POSITION_LATERAL_MIDFIELD_RIGHT = 6
export const POSITION_STRIKER = 7
export const POSITION_LATERAL_STRIKER_LEFT = 8
export const POSITION_LATERAL_STRIKER_RIGHT = 9

export const POSITION_LABELS: Record<number, string> = {
  [POSITION_GOALKEEPER]: 'Goalkeeper',
  [POSITION_DEFENSE]: 'Defense',
  [POSITION_LATERAL_DEFENSE_LEFT]: 'Left back',
  [POSITION_LATERAL_DEFENSE_RIGHT]: 'Right back',
  [POSITION_MIDFIELD]: 'Midfield',
  [POSITION_LATERAL_MIDFIELD_LEFT]: 'Left midfield',
  [POSITION_LATERAL_MIDFIELD_RIGHT]: 'Right midfield',
  [POSITION_STRIKER]: 'Striker',
  [POSITION_LATERAL_STRIKER_LEFT]: 'Left striker',
  [POSITION_LATERAL_STRIKER_RIGHT]: 'Right striker',
}

export const ALL_POSITIONS = [
  POSITION_GOALKEEPER,
  POSITION_DEFENSE,
  POSITION_LATERAL_DEFENSE_LEFT,
  POSITION_LATERAL_DEFENSE_RIGHT,
  POSITION_MIDFIELD,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_LATERAL_MIDFIELD_RIGHT,
  POSITION_STRIKER,
  POSITION_LATERAL_STRIKER_LEFT,
  POSITION_LATERAL_STRIKER_RIGHT,
]

// team_transactions.reason values from backend/common/fmconstants.go
export const TRANSACTION_TRANSFER_IN = 0
export const TRANSACTION_TRANSFER_OUT = 1
export const TRANSACTION_WAGES = 2
export const TRANSACTION_ADMIN_ADJUSTMENT = 3

export const TRANSACTION_TICKET_REVENUE = 4

export const TRANSACTION_LABELS: Record<number, string> = {
  [TRANSACTION_TRANSFER_IN]: 'Player sold',
  [TRANSACTION_TRANSFER_OUT]: 'Player bought',
  [TRANSACTION_WAGES]: 'Wages',
  [TRANSACTION_ADMIN_ADJUSTMENT]: 'Admin adjustment',
  [TRANSACTION_TICKET_REVENUE]: 'Ticket revenue',
}

export interface TeamTransaction {
  id: number
  teamId: number
  amount: number
  reason: number
  description: string
  createdAt: string
}

export interface Stadium {
  capacity: number
  ticketPrice: number
}

export interface FinancesResponse {
  budget: number
  transactions: TeamTransaction[]
  stadium: Stadium | null
}

export interface TransferListing {
  listingId: number
  player: Player
  sellingTeamId: number
  sellingTeamName: string
  askingPrice: number
  listedAt: string
}

export interface Cup {
  cupId: number
  country: string
  season: number
  name: string
  completed: boolean
  championTeamId: number | null
  championTeamName: string
}

export interface CupTie {
  tieId: number
  cupId: number
  round: number
  slotNumber: number
  homeTeamId: number | null
  homeTeamName: string
  awayTeamId: number | null
  awayTeamName: string
  matchDate: string | null
  played: boolean
  homeGoals: number
  awayGoals: number
  wentToPenalties: boolean
  winnerTeamId: number | null
}

export interface CupDetails {
  cup: Cup
  ties: CupTie[]
}

// notifications.notification_type values from backend/common/fmconstants.go
export const NOTIFICATION_MATCH_RESULT = 0
export const NOTIFICATION_PROMOTION = 1
export const NOTIFICATION_RELEGATION = 2
export const NOTIFICATION_TRANSFER = 3
export const NOTIFICATION_CUP = 4

export interface Notification {
  notificationId: number
  userId: number
  notificationType: number
  message: string
  read: boolean
  createdAt: string
}

export interface NotificationsResponse {
  unread: number
  notifications: Notification[]
}

export interface AdminOverview {
  season: number
  userCount: number
  teamCount: number
  leagueCount: number
  matchesTotal: number
  matchesUnplayed: number
  activeCups: number
}

export interface AdminUserSummary {
  userId: number
  name: string
  email: string
  country: string
  teamId: number | null
  teamName: string
  isAdmin: boolean
}
