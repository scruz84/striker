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
  // True right after a first-time sign-in, before the onboarding country picker has run - every
  // other field involving a team is meaningless until this is false (teamId is "0").
  pendingCountrySelection: boolean
}

// A country the engine can generate a world for (see backend common.KnownCountries) - used by
// the onboarding country picker (only ever the currently-active ones, see
// useOnboardingCountries) and reused as the base shape of admin's CountryStatus below.
export interface CountryInfo {
  code: string
  name: string
}

// Mirrors backend/common.DevUser - only ever non-empty when the backend has DEV_LOGIN_ENABLED
// set, see useDevLoginUsers.
export interface DevUser {
  email: string
  name: string
  isAdmin: boolean
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
  contractYearsRemaining: number
  // The country whose name pool generated this player - independent of which team/country they
  // currently play for, since the transfer market has no country restriction. Empty for anyone
  // created before this field existed. See COUNTRY_NAME_KEYS.
  birthCountry?: string
}

// Every country code the game can generate players/teams for (mirrors backend
// common.KnownCountries) - maps to an i18n key rather than a hardcoded name so a player's birth
// country renders in whichever language the UI is already in.
export const COUNTRY_NAME_KEYS: Record<string, string> = {
  es: 'countries.es',
  en: 'countries.en',
  fr: 'countries.fr',
  de: 'countries.de',
  it: 'countries.it',
  pt: 'countries.pt',
}

// Mirrors backend/common.ContractRenewalOffer - the outcome of proposing new contract terms to
// one of the team's own players (see useRenewContract). expectedWage/maxAcceptableYears are
// always present, accepted or not, so the UI can guide a retry after a decline.
export interface ContractRenewalOffer {
  accepted: boolean
  reason?: string
  expectedWage: number
  maxAcceptableYears: number
  contractYearsRemaining?: number
  wage?: number
}

// Mirrors backend/common.MaxContractYearsRemaining - the server enforces the real cap, this
// just keeps the UI (renewal offer years, the renew button's disabled state) in step with it.
export const MAX_CONTRACT_YEARS_REMAINING = 3

// Mirrors backend/handlers.AcademyOverview - the caller's own team's youth academy ("cantera")
// pool, plus whether a promotion is available right now (see useAcademy). nextAvailableAt is a
// unix-millis timestamp, only present while canPromote is false.
export interface AcademyOverview {
  players: Player[]
  canPromote: boolean
  nextAvailableAt?: number
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
  // Only meaningfully populated on the caller's own team (GET /v1/user/team) - see
  // useRenameTeam. Elsewhere it's just the zero value, false.
  nameChanged: boolean
  // Only populated by GET /v1/teams/team/:id/ - 0/undefined for an owned team (meaningless
  // there), or 1-3 for an unowned one. See AI_LEVEL_* constants.
  aiLevel?: number
}

export const AI_LEVEL_CARETAKER = 1
export const AI_LEVEL_MANAGER = 2
export const AI_LEVEL_DIRECTOR = 3

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

// LineupSubstitution is one planned in-match change to a team's default lineup: bring
// playerInId on for playerOutId at minute. At most MAX_SCHEDULED_SUBSTITUTIONS may be set.
export interface LineupSubstitution {
  playerOutId: number
  playerInId: number
  minute: number
}

export const MAX_SCHEDULED_SUBSTITUTIONS = 3

export interface Lineup {
  matchId: number
  teamId: number
  formation: number
  positions: LineupPosition[]
  substitutions: LineupSubstitution[]
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
export const EVENT_TYPE_SUBSTITUTION = 5

// TeamQualification is a team's per-zone strength for one minute of simulation - see
// backend/common/fmstructs.go. These are the actual numbers the simulation compares (a team's
// attack on one flank against the opponent's defense on the mirrored flank) to decide whether a
// scoring chance becomes a goal - not a display-only summary. `goalkeeper` is always 0
// (goalkeeping strength is read from the starting keeper's own rating instead, not tracked here).
export interface TeamQualification {
  goalkeeper: number
  leftDefense: number
  rightDefense: number
  defense: number
  leftMidfield: number
  rightMidfield: number
  midfield: number
  leftAttack: number
  // Spelled to match the backend's own historical (typo'd) field name - see
  // backend/common/fmstructs.go's TeamQualification for why the JSON key stays this way.
  rightAttach: number
  attack: number
}

export interface MatchInstant {
  matchId: number
  minute: number
  homeGoals: number
  awayGoals: number
  homeQua: TeamQualification
  awayQua: TeamQualification
  midfieldDiff: number
  // The four *Left/Right*Diff fields below are always 0 - the backend never actually computes
  // them (only the Center variants and midfieldDiff are real) - see homeAttackCenterDiff/
  // awayAttackCenterDiff instead, or better, homeQua/awayQua directly for the real per-zone
  // numbers used in goal-probability calculations.
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

// Mirrors backend/common.LeagueDirectoryEntry/CountryLeagueDirectory - the browsable list of
// every league in the game world, grouped by country (see useLeagueDirectory). Each entry links
// into DivisionDetails via /division/:leagueId, same as a team's own competitions link.
export interface LeagueDirectoryEntry {
  leagueId: number
  name: string
  tier: number
  groupNumber: number
}

export interface CountryLeagueDirectory {
  country: string
  leagues: LeagueDirectoryEntry[]
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

// Values are i18n keys under "positions.*" (see src/i18n/locales) rather than literal labels -
// look them up with t(POSITION_LABEL_KEYS[position]).
export const POSITION_LABEL_KEYS: Record<number, string> = {
  [POSITION_GOALKEEPER]: 'positions.goalkeeper',
  [POSITION_DEFENSE]: 'positions.defense',
  [POSITION_LATERAL_DEFENSE_LEFT]: 'positions.leftBack',
  [POSITION_LATERAL_DEFENSE_RIGHT]: 'positions.rightBack',
  [POSITION_MIDFIELD]: 'positions.midfield',
  [POSITION_LATERAL_MIDFIELD_LEFT]: 'positions.leftMidfield',
  [POSITION_LATERAL_MIDFIELD_RIGHT]: 'positions.rightMidfield',
  [POSITION_STRIKER]: 'positions.striker',
  [POSITION_LATERAL_STRIKER_LEFT]: 'positions.leftStriker',
  [POSITION_LATERAL_STRIKER_RIGHT]: 'positions.rightStriker',
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

// Formation ids and per-position headcounts from backend/common/formation.go - every formation
// totals 11 players (goalkeeper included).
export const FORMATION_BALANCED = 0
export const FORMATION_DEFENSIVE = 1
export const FORMATION_ATTACKING = 2
export const FORMATION_FOUR_THREE_THREE = 3
export const FORMATION_FOUR_FIVE_ONE = 4

// Values are i18n keys under "formations.*" - look them up with t(FORMATION_LABEL_KEYS[formation]).
export const FORMATION_LABEL_KEYS: Record<number, string> = {
  [FORMATION_BALANCED]: 'formations.balanced',
  [FORMATION_DEFENSIVE]: 'formations.defensive',
  [FORMATION_ATTACKING]: 'formations.attacking',
  [FORMATION_FOUR_THREE_THREE]: 'formations.fourThreeThree',
  [FORMATION_FOUR_FIVE_ONE]: 'formations.fourFiveOne',
}

export const FORMATION_POSITION_COUNTS: Record<number, Partial<Record<number, number>>> = {
  [FORMATION_BALANCED]: {
    [POSITION_GOALKEEPER]: 1,
    [POSITION_DEFENSE]: 2,
    [POSITION_LATERAL_DEFENSE_LEFT]: 1,
    [POSITION_LATERAL_DEFENSE_RIGHT]: 1,
    [POSITION_MIDFIELD]: 2,
    [POSITION_LATERAL_MIDFIELD_LEFT]: 1,
    [POSITION_LATERAL_MIDFIELD_RIGHT]: 1,
    [POSITION_STRIKER]: 2,
  },
  [FORMATION_DEFENSIVE]: {
    [POSITION_GOALKEEPER]: 1,
    [POSITION_DEFENSE]: 3,
    [POSITION_LATERAL_DEFENSE_LEFT]: 1,
    [POSITION_LATERAL_DEFENSE_RIGHT]: 1,
    [POSITION_MIDFIELD]: 1,
    [POSITION_LATERAL_MIDFIELD_LEFT]: 1,
    [POSITION_LATERAL_MIDFIELD_RIGHT]: 1,
    [POSITION_STRIKER]: 2,
  },
  [FORMATION_ATTACKING]: {
    [POSITION_GOALKEEPER]: 1,
    [POSITION_DEFENSE]: 1,
    [POSITION_LATERAL_DEFENSE_LEFT]: 1,
    [POSITION_LATERAL_DEFENSE_RIGHT]: 1,
    [POSITION_MIDFIELD]: 2,
    [POSITION_LATERAL_MIDFIELD_LEFT]: 1,
    [POSITION_LATERAL_MIDFIELD_RIGHT]: 1,
    [POSITION_STRIKER]: 3,
  },
  [FORMATION_FOUR_THREE_THREE]: {
    [POSITION_GOALKEEPER]: 1,
    [POSITION_DEFENSE]: 2,
    [POSITION_LATERAL_DEFENSE_LEFT]: 1,
    [POSITION_LATERAL_DEFENSE_RIGHT]: 1,
    [POSITION_MIDFIELD]: 3,
    [POSITION_STRIKER]: 1,
    [POSITION_LATERAL_STRIKER_LEFT]: 1,
    [POSITION_LATERAL_STRIKER_RIGHT]: 1,
  },
  [FORMATION_FOUR_FIVE_ONE]: {
    [POSITION_GOALKEEPER]: 1,
    [POSITION_DEFENSE]: 2,
    [POSITION_LATERAL_DEFENSE_LEFT]: 1,
    [POSITION_LATERAL_DEFENSE_RIGHT]: 1,
    [POSITION_MIDFIELD]: 3,
    [POSITION_LATERAL_MIDFIELD_LEFT]: 1,
    [POSITION_LATERAL_MIDFIELD_RIGHT]: 1,
    [POSITION_STRIKER]: 1,
  },
}

// team_transactions.reason values from backend/common/fmconstants.go
export const TRANSACTION_TRANSFER_IN = 0
export const TRANSACTION_TRANSFER_OUT = 1
export const TRANSACTION_WAGES = 2
export const TRANSACTION_ADMIN_ADJUSTMENT = 3

export const TRANSACTION_TICKET_REVENUE = 4
export const TRANSACTION_PRIZE_MONEY = 5
export const TRANSACTION_SPONSORSHIP = 6
export const TRANSACTION_STADIUM_UPGRADE = 7
export const TRANSACTION_CONTRACT_RELEASE = 8
export const TRANSACTION_TRAINER_WAGES = 9

// Values are i18n keys under "finances.reasons.*" - look them up with
// t(TRANSACTION_REASON_KEYS[reason] ?? 'finances.reasons.other').
export const TRANSACTION_REASON_KEYS: Record<number, string> = {
  [TRANSACTION_TRANSFER_IN]: 'finances.reasons.transferIn',
  [TRANSACTION_TRANSFER_OUT]: 'finances.reasons.transferOut',
  [TRANSACTION_WAGES]: 'finances.reasons.wages',
  [TRANSACTION_ADMIN_ADJUSTMENT]: 'finances.reasons.adminAdjustment',
  [TRANSACTION_TICKET_REVENUE]: 'finances.reasons.ticketRevenue',
  [TRANSACTION_PRIZE_MONEY]: 'finances.reasons.prizeMoney',
  [TRANSACTION_SPONSORSHIP]: 'finances.reasons.sponsorship',
  [TRANSACTION_STADIUM_UPGRADE]: 'finances.reasons.stadiumUpgrade',
  [TRANSACTION_CONTRACT_RELEASE]: 'finances.reasons.contractRelease',
  [TRANSACTION_TRAINER_WAGES]: 'finances.reasons.trainerWages',
}

export interface TeamTransaction {
  id: number
  teamId: number
  amount: number
  reason: number
  description: string
  createdAt: string
  // Set only for a transaction about one specific player (a transfer, a release) - undefined
  // for team-level transactions (wages, ticket revenue, prize money, sponsorship, stadium
  // upgrades, admin adjustments).
  playerId?: number
}

// A team's head coach - hired from the open market (see useAvailableTrainers/useHireTrainer). A
// hired trainer's skill accelerates player attribute growth each season and raises the youth
// academy's prospect ceiling; a team with none simply gets today's baseline rates.
export interface Trainer {
  id: number
  firstName: string
  lastName: string
  skill: number
  wage: number
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

export interface LoanListing {
  listingId: number
  player: Player
  lendingTeamId: number
  lendingTeamName: string
  durationSeasons: number
  listedAt: string
}

export interface Group {
  groupId: number
  name: string
  ownerId: number
  inviteCode: string
  createdAt: string
}

export interface GroupMember {
  userId: number
  userName: string
  teamId: number | null
  teamName: string
  country: string
  leagueName: string
  budget: number | null
}

export interface GroupDetails {
  group: Group
  leaderboard: GroupMember[]
}

export interface SeasonHistoryEntry {
  teamId: number
  season: number
  leagueName: string | null
  finalPosition: number | null
  promoted: boolean
  relegated: boolean
  cupResult: string | null
}

export interface ScorerStanding {
  playerId: number
  firstName: string
  lastName: string
  teamId: number
  teamName: string
  goals: number
}

// UnavailableReason values from backend/common/fmconstants.go.
export const UNAVAILABLE_SUSPENSION = 0
export const UNAVAILABLE_INJURY = 1

// PlayerDetails is the full profile shown on a player's page - see
// backend/common/playerdetails.go's PlayerDetails and handlers.PlayerDetailsHandler.
export interface PlayerDetails extends Player {
  teamName: string
  divisionId: number
  divisionName: string
  country: string
  bestPosition: number
  goals: number
  yellowCards: number
  redCards: number
  unavailableReason?: number
  unavailableMatchesRemaining?: number
  onLoanFromTeamId?: number
  onLoanFromTeamName?: string
  onLoanReturnSeason?: number
  transferListingAskingPrice?: number
  loanListingDurationSeasons?: number
  history: TeamTransaction[]
}

export interface HeadToHeadMeeting {
  matchId: number
  season: number
  matchDate: string
  homeTeamId: number
  awayTeamId: number
  homeGoals: number
  awayGoals: number
}

// Wins/goals are reported from teamAId's perspective - teamBWins/teamBGoals are the complement,
// so which team you pass as A vs B only matters for how you read the result, not what it means.
export interface HeadToHead {
  teamAId: number
  teamBId: number
  totalMatches: number
  teamAWins: number
  teamBWins: number
  draws: number
  teamAGoals: number
  teamBGoals: number
  meetings: HeadToHeadMeeting[]
}

export interface TransferOffer {
  offerId: number
  player: Player
  fromTeamId: number
  fromTeamName: string
  toTeamId: number
  toTeamName: string
  amount: number
  createdAt: string
}

export interface FriendlyChallenge {
  challengeId: number
  challengerTeamId: number
  challengerTeamName: string
  opponentTeamId: number
  opponentTeamName: string
  createdAt: string
}

export interface FriendlyMatch {
  matchId: number
  homeTeamId: number
  homeTeamName: string
  awayTeamId: number
  awayTeamName: string
  matchDate: string
  played: boolean
  homeGoals: number
  awayGoals: number
}

// The admin-configured weekdays a friendly may kick off on (see backend
// common.GetFriendlyScheduleWeekdays) - same bitmask convention as MatchSchedule.weekdaysMask.
export interface FriendlySchedule {
  weekdaysMask: number
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
export const NOTIFICATION_TRAINER = 5

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

export interface NotificationPreferences {
  emailEnabled: boolean
  pushEnabled: boolean
}

export interface AdminOverview {
  season: number
  userCount: number
  teamCount: number
  leagueCount: number
  matchesTotal: number
  matchesUnplayed: number
  activeCups: number
  transferWindowOpen: boolean
  matchScheduleWeekdaysMask: number
  matchScheduleHour: number
  matchScheduleMinute: number
  maxFriendliesPerWeek: number
  friendlyScheduleWeekdays: number
  promotionSlotsPerGroup: number
  relegationSlotsPerGroup: number
}

// Weekday bitmask values for AdminOverview.matchScheduleWeekdaysMask and
// useSetMatchSchedule's weekdaysMask - matches backend/common/matchschedule.go, bit i
// corresponds to JS Date.getDay() (0 = Sunday ... 6 = Saturday).
export const WEEKDAY_SUNDAY = 1 << 0
export const WEEKDAY_MONDAY = 1 << 1
export const WEEKDAY_TUESDAY = 1 << 2
export const WEEKDAY_WEDNESDAY = 1 << 3
export const WEEKDAY_THURSDAY = 1 << 4
export const WEEKDAY_FRIDAY = 1 << 5
export const WEEKDAY_SATURDAY = 1 << 6
export const ALL_WEEKDAYS_MASK =
  WEEKDAY_SUNDAY | WEEKDAY_MONDAY | WEEKDAY_TUESDAY | WEEKDAY_WEDNESDAY | WEEKDAY_THURSDAY | WEEKDAY_FRIDAY | WEEKDAY_SATURDAY

// Displayed Monday-first regardless of the Sunday-first bit numbering above.
export const WEEKDAY_OPTIONS: { bit: number; key: string }[] = [
  { bit: WEEKDAY_MONDAY, key: 'mon' },
  { bit: WEEKDAY_TUESDAY, key: 'tue' },
  { bit: WEEKDAY_WEDNESDAY, key: 'wed' },
  { bit: WEEKDAY_THURSDAY, key: 'thu' },
  { bit: WEEKDAY_FRIDAY, key: 'fri' },
  { bit: WEEKDAY_SATURDAY, key: 'sat' },
  { bit: WEEKDAY_SUNDAY, key: 'sun' },
]

export interface AdminUserSummary {
  userId: number
  name: string
  email: string
  country: string
  teamId: number | null
  teamName: string
  isAdmin: boolean
}

// One league tier's shape - see backend/common/worldconfig.go's TierConfig.
export interface TierConfig {
  tier: number
  name: string
  groups: number
  teamsPerLeague: number
}

export interface CountryStatus extends CountryInfo {
  active: boolean
}

export interface MatchSchedule {
  weekdaysMask: number
  hour: number
  minute: number
}

export interface AdminLeagueInfo {
  leagueId: number
  country: string
  name: string
  tier: number
  schedule: MatchSchedule
  hasOverride: boolean
}
