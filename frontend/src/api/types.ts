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
  // Stamina (0-99) fatigues this player's in-match skills minute by minute - unlike the seven
  // core skills it's deliberately left out of SKILL_FIELDS/overallRating (see playerRating.ts),
  // since it measures physical conditioning rather than footballing quality. See backend
  // Player.Stamina/engine.StaminaSimulation.
  stamina: number
  // Form (0-99) nudges this player's match ratings up or down and moves match-by-match like the
  // other skills (unlike Stamina, not minute-by-minute) - also excluded from SKILL_FIELDS for the
  // same reason. 50 is neutral (backend common.DefaultForm). See backend Player.Form.
  form: number
  wage: number
  age: number
  contractYearsRemaining: number
  // The country whose name pool generated this player - independent of which team/country they
  // currently play for, since the transfer market has no country restriction. Empty for anyone
  // created before this field existed. See COUNTRY_NAME_KEYS.
  birthCountry?: string
  // Only meaningful for a free agent (teamId === 0) - see backend Player.FreeAgentSeasons/
  // LastTeamId. freeAgentSeasons === 0 and lastTeamId set means signing them owes lastTeamId a
  // transfer premium (see backend common.FreeAgentPremiumWageMultiplier).
  freeAgentSeasons?: number
  lastTeamId?: number
  // 0/absent for a player with none assigned yet (an academy prospect not yet promoted, or a
  // free agent) - see backend common.nextFreeShirtNumber.
  shirtNumber?: number
  // A single 0-99 aggregate of the seven core skills - only populated where the backend
  // specifically computes it (currently TeamDetails.players), unlike those underlying fields
  // (zeroed for someone else's rostered, unlisted player), so a squad list can show a rough
  // scouting figure without the full breakdown. See backend common.AverageSkill.
  averageSkill?: number
}

// FreeAgentListing is what the free agents tab of the transfer market actually gets back - a
// Player plus their last club's name (see backend common.FreeAgentListing), resolved server-side
// so the market page doesn't have to look up lastTeamId one team at a time.
export interface FreeAgentListing extends Player {
  lastTeamName?: string
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
  nl: 'countries.nl',
  se: 'countries.se',
  pl: 'countries.pl',
  gr: 'countries.gr',
  br: 'countries.br',
  ar: 'countries.ar',
  mx: 'countries.mx',
  co: 'countries.co',
  cl: 'countries.cl',
  pe: 'countries.pe',
  uy: 'countries.uy',
  us: 'countries.us',
  ca: 'countries.ca',
  ec: 'countries.ec',
  jp: 'countries.jp',
  kr: 'countries.kr',
  cn: 'countries.cn',
  in: 'countries.in',
  th: 'countries.th',
  vn: 'countries.vn',
  id: 'countries.id',
  ph: 'countries.ph',
  sa: 'countries.sa',
  ir: 'countries.ir',
  ng: 'countries.ng',
  eg: 'countries.eg',
  gh: 'countries.gh',
  ci: 'countries.ci',
  cm: 'countries.cm',
  sn: 'countries.sn',
  za: 'countries.za',
  ma: 'countries.ma',
  dz: 'countries.dz',
  ke: 'countries.ke',
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

// Mirrors backend/common.FreeAgentSigningResult - same "player judges on the spot" shape as
// ContractRenewalOffer, plus the transfer premium (if any) once accepted.
export interface FreeAgentSigningResult {
  accepted: boolean
  reason?: string
  expectedWage: number
  maxAcceptableYears: number
  contractYearsRemaining?: number
  wage?: number
  premium?: number
}

// Mirrors backend/common.TransferOfferResult - the player judges the personal terms (wage/years)
// before the offer ever reaches the selling club; offerId is only set once actually created.
export interface TransferOfferResult {
  accepted: boolean
  reason?: string
  expectedWage: number
  maxAcceptableYears: number
  offerId?: number
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
}

export const AI_LEVEL_CARETAKER = 1
export const AI_LEVEL_MANAGER = 2
export const AI_LEVEL_DIRECTOR = 3

// StadiumLastMatch is the headline facts of the most recent completed league match played at a
// team's stadium - mirrors backend common.StadiumLastMatch.
export interface StadiumLastMatch {
  matchId: number
  matchDate: string
  opponentId: number
  opponentName: string
  homeGoals: number
  awayGoals: number
  attendance: number
}

// StadiumSummary is the public-facing slice of a team's stadium - name, capacity, and its last
// played home match - mirrors backend common.StadiumSummary. Unlike the finances page's full
// Stadium (ticket price, pending upgrade), this is shown on any team's page.
export interface StadiumSummary {
  name: string
  capacity: number
  lastHomeMatch?: StadiumLastMatch
}

export interface TeamDetails {
  team: Team
  players: Player[]
  lastMatches: WeekMatch[] | null
  nextMatches: WeekMatch[] | null
  // Only sent for the caller's own team (/v1/user/team) - see backend common.TeamDetails.
  unavailability?: PlayerUnavailability[]
  // The squad's own players currently listed for transfer - only sent for the caller's own team,
  // same as unavailability. See backend common.TeamDetails.TransferListedPlayerIds.
  transferListedPlayerIds?: number[]
  stadium?: StadiumSummary
  // This team's current 0-99 supporter-base index - see backend common.GetFanbase. Public, same
  // visibility tier as stadium's own public fields.
  fanbase: number
}

// One player's current suspension or injury - reason is UNAVAILABLE_SUSPENSION or
// UNAVAILABLE_INJURY. Mirrors backend common.PlayerUnavailability.
export interface PlayerUnavailability {
  playerId: number
  reason: number
  matchesRemaining: number
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
export const EVENT_TYPE_MATCH_ABANDONED = 6

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
  // The home team's stadium at the time this match was fetched - see the backend's
  // common.WeekMatch.HomeStadiumName.
  homeStadiumName: string
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
  // Match-report flavor (issue: Hattrick-style match report) - absent for a match not yet
  // simulated, or one simulated before this existed.
  weather?: string
  attendance?: number
  refereeName?: string
  assistant1Name?: string
  assistant2Name?: string
  // Per-player match rating (1-10), keyed by player id as a string (JSON object keys are always
  // strings) - only present for a played match with lineup data (see backend
  // common.ComputeMatchRatings).
  ratings?: Record<string, number>
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

// Short (2-3 letter) labels for just the four core positions bestCorePosition (see
// utils/playerRating) ever returns - used wherever a player's suggested position needs to fit
// next to their name in a compact chip (transfer market rows, lineup editor cards) rather than
// the full POSITION_LABEL_KEYS word.
export const POSITION_SHORT_LABEL_KEYS: Record<number, string> = {
  [POSITION_GOALKEEPER]: 'positions.short.goalkeeper',
  [POSITION_DEFENSE]: 'positions.short.defense',
  [POSITION_MIDFIELD]: 'positions.short.midfield',
  [POSITION_STRIKER]: 'positions.short.striker',
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
export const FORMATION_THREE_FIVE_TWO = 5

// Values are i18n keys under "formations.*" - look them up with t(FORMATION_LABEL_KEYS[formation]).
export const FORMATION_LABEL_KEYS: Record<number, string> = {
  [FORMATION_BALANCED]: 'formations.balanced',
  [FORMATION_DEFENSIVE]: 'formations.defensive',
  [FORMATION_ATTACKING]: 'formations.attacking',
  [FORMATION_FOUR_THREE_THREE]: 'formations.fourThreeThree',
  [FORMATION_FOUR_FIVE_ONE]: 'formations.fourFiveOne',
  [FORMATION_THREE_FIVE_TWO]: 'formations.threeFiveTwo',
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
  [FORMATION_THREE_FIVE_TWO]: {
    [POSITION_GOALKEEPER]: 1,
    [POSITION_DEFENSE]: 1,
    [POSITION_LATERAL_DEFENSE_LEFT]: 1,
    [POSITION_LATERAL_DEFENSE_RIGHT]: 1,
    [POSITION_MIDFIELD]: 3,
    [POSITION_LATERAL_MIDFIELD_LEFT]: 1,
    [POSITION_LATERAL_MIDFIELD_RIGHT]: 1,
    [POSITION_STRIKER]: 2,
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
export const TRANSACTION_SEASON_TICKETS = 10
export const TRANSACTION_MERCHANDISING = 11

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
  [TRANSACTION_SEASON_TICKETS]: 'finances.reasons.seasonTickets',
  [TRANSACTION_MERCHANDISING]: 'finances.reasons.merchandising',
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
  name: string
  capacity: number
  ticketPrice: number
  // Set while a paid-for capacity expansion is still under construction (see the backend's
  // common.UpgradeStadium) - both undefined when there's no upgrade in progress.
  pendingCapacity?: number
  upgradeCompletesAt?: string
}

// Bounds for how many seats a stadium upgrade can add in one go - mirrors the backend's
// common.MinStadiumUpgradeSeats/MaxStadiumUpgradeSeats. Duplicated here rather than fetched,
// the same way MinTicketPrice/MaxTicketPrice aren't - the backend is the source of truth and
// rejects anything outside range regardless of what the frontend sends.
export const MIN_STADIUM_UPGRADE_SEATS = 100
export const MAX_STADIUM_UPGRADE_SEATS = 5000
export const DEFAULT_STADIUM_UPGRADE_SEATS = 500

// Cost formula for a stadium upgrade - mirrors the backend's common.UpgradeStadium: cost scales
// with both the stadium's current capacity and the number of seats requested (relative to this
// baseline), so a bigger stadium or a bigger expansion both cost more in absolute terms. Used to
// show the total price of the work before the user confirms - the backend remains the source of
// truth and recomputes/enforces this same formula server-side.
export const STADIUM_UPGRADE_COST_PER_CURRENT_SEAT = 10
export const STADIUM_UPGRADE_SEATS_BASELINE = 500

export function calculateStadiumUpgradeCost(currentCapacity: number, seats: number): number {
  return Math.floor((currentCapacity * STADIUM_UPGRADE_COST_PER_CURRENT_SEAT * seats) / STADIUM_UPGRADE_SEATS_BASELINE)
}

export interface FinancesResponse {
  budget: number
  transactions: TeamTransaction[]
  stadium: Stadium | null
  // True once the budget has crossed the backend's debt ceiling - buying, signing, and borrowing
  // on loan are all blocked while this is true (see backend common.ErrTeamInsolvent).
  insolvent: boolean
  // How far budget may go negative before insolvent flips true (see backend common.MaxDebt) -
  // used by the cash-flow forecast to project weeks until that point on the current trend.
  maxDebt: number
  // One of SPONSOR_DEAL_* - see SPONSOR_DEAL_LABEL_KEYS.
  sponsorDeal: number
  // What each of the three sponsor deals actually pays per match at this team's current tier -
  // lets the finance page show real numbers instead of just a label (issue #70).
  sponsorDealOptions: SponsorDealOption[]
  // True once this team has already chosen a deal for the current season - see
  // backend common.SetSponsorDeal, which then rejects any further change until next season.
  sponsorDealLocked: boolean
  // This team's current 0-99 supporter-base index - see backend common.GetFanbase. Feeds into
  // matchday attendance, sponsorDealOptions above, and season-ticket/merchandising income.
  fanbase: number
}

// One row of FinancesResponse.sponsorDealOptions - see backend engine.SponsorDealPreview.
export interface SponsorDealOption {
  deal: number
  win: number
  draw: number
  loss: number
}

// teams.sponsor_deal values from backend/common/sponsordeal.go.
export const SPONSOR_DEAL_STEADY = 0
export const SPONSOR_DEAL_BALANCED = 1
export const SPONSOR_DEAL_PERFORMANCE = 2

// Values are i18n keys under "finances.sponsorDeal.options.*".
export const SPONSOR_DEAL_LABEL_KEYS: Record<number, string> = {
  [SPONSOR_DEAL_STEADY]: 'finances.sponsorDeal.options.steady',
  [SPONSOR_DEAL_BALANCED]: 'finances.sponsorDeal.options.balanced',
  [SPONSOR_DEAL_PERFORMANCE]: 'finances.sponsorDeal.options.performance',
}

export const SPONSOR_DEAL_DESCRIPTION_KEYS: Record<number, string> = {
  [SPONSOR_DEAL_STEADY]: 'finances.sponsorDeal.options.steadyDescription',
  [SPONSOR_DEAL_BALANCED]: 'finances.sponsorDeal.options.balancedDescription',
  [SPONSOR_DEAL_PERFORMANCE]: 'finances.sponsorDeal.options.performanceDescription',
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
  // A single 0-99 aggregate of the seven core skills - unlike those underlying fields (zeroed by
  // the backend for someone else's rostered, unlisted player), this stays populated, so the
  // player page can show a rough scouting figure without the full breakdown. See backend
  // common.AverageSkill.
  averageSkill: number
  goals: number
  yellowCards: number
  redCards: number
  goalsForTeam: number
  unavailableReason?: number
  unavailableMatchesRemaining?: number
  onLoanFromTeamId?: number
  onLoanFromTeamName?: string
  onLoanReturnSeason?: number
  transferListingAskingPrice?: number
  loanListingDurationSeasons?: number
  history: TeamTransaction[]
  latestEvolution?: PlayerEvolution
  // Every club this player has permanently belonged to, most recent first - unlike `history`,
  // never redacted for a non-owner viewer (which clubs a player played for is public, the same
  // way a transfer-market listing is).
  clubHistory: PlayerClubHistoryEntry[]
  // This player's current team's most recent played league matches - see
  // backend/common/playermatches.go's GetPlayerRecentMatches.
  recentMatches: PlayerMatchSummary[]
}

export interface PlayerClubHistoryEntry {
  teamId: number
  teamName: string
  joinedAt: string
}

// Mirrors backend common.PlayerMatchSummary - one row in a player's "recent matches" list.
// `rating` is absent when the match has no lineup data to derive one from.
export interface PlayerMatchSummary {
  matchId: number
  matchDate: string
  opponentId: number
  opponentName: string
  isHome: boolean
  homeGoals: number
  awayGoals: number
  goals: number
  yellowCards: number
  redCards: number
  rating?: number
}

// One attribute's value before and after a season-end progression run - see backend
// common.SkillDelta. after === before means that attribute held steady this season, which is
// itself meaningful to show (not every skill drifts every season - see engine.driftSkillsForPosition).
export interface SkillDelta {
  before: number
  after: number
}

// Mirrors backend common.PlayerEvolution - one player's captured skill/age change from a single
// season-end progression run (see engine.ProgressPlayers). Shown on the player's own details page
// and, as TeamPlayerEvolution, in the squad-wide list a NOTIFICATION_PROGRESSION notification
// links to.
export interface PlayerEvolution {
  playerId: number
  season: number
  age: SkillDelta
  goalKeeper: SkillDelta
  defense: SkillDelta
  lateral: SkillDelta
  pass: SkillDelta
  midfield: SkillDelta
  dribbling: SkillDelta
  shot: SkillDelta
  createdAt: string
}

export interface TeamPlayerEvolution extends PlayerEvolution {
  firstName: string
  lastName: string
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
  offeredWage: number
  offeredYears: number
  createdAt: string
  // When this offer's admin-configurable negotiation window elapses and it becomes acceptable
  // (see backend common.GetOfferNegotiationWindowMinutes) - lets the receiving team keep
  // comparing offers instead of the first one in winning by default. Rejecting/withdrawing is
  // never gated by this, only accepting.
  acceptableAt: string
}

// Mirrors backend/common.MostViewedPlayer - the currently-listed player whose profile has been
// viewed the most (see PlayerDetailsHandler, which increments this on every fetch).
export interface MostViewedPlayer {
  player: Player
  viewCount: number
}

// Mirrors backend/common.TopTransferFee - the biggest fee ever paid for one player, and both
// clubs involved.
export interface TopTransferFee {
  player: Player
  amount: number
  buyerTeamId: number
  buyerTeamName: string
  sellerTeamId: number
  sellerTeamName: string
  occurredAt: string
}

// Mirrors backend/common.TransferMarketStats - the transfer market page's activity panel (issue
// #72). mostViewedPlayer/topTransferFee are absent when there's nothing to report yet.
export interface TransferMarketStats {
  listedPlayersCount: number
  offersLastHour: number
  offersLastDay: number
  mostViewedPlayer?: MostViewedPlayer
  topTransferFee?: TopTransferFee
}

// Mirrors backend/common.LeagueTransfer - one row of a league's recent-transfers panel (league
// details page and the dashboard).
export interface LeagueTransfer {
  player: Player
  amount: number
  buyerTeamId: number
  buyerTeamName: string
  sellerTeamId: number
  sellerTeamName: string
  occurredAt: string
}

export interface FriendlyChallenge {
  challengeId: number
  challengerTeamId: number
  challengerTeamName: string
  opponentTeamId: number
  opponentTeamName: string
  createdAt: string
  // The kickoff the challenger optionally suggested when sending the challenge - a suggested
  // default for the opponent's own accept-time kickoff picker, never enforced. Absent if they
  // left it up to the opponent.
  proposedKickoff?: string
  // Which side of the fixture the challenger chose to take when sending the challenge - true
  // puts the challenger at home once accepted, false puts the opponent at home instead.
  challengerIsHome: boolean
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
  // Only present once the friendly has been played (see backend common.EnrichPlayedFriendly) -
  // the same match-detail breakdown a league match's WeekMatch carries, so a played friendly can
  // show the same report tabs (events, zone-by-zone comparison, ratings, lineups).
  events?: MatchEvent[]
  matchInstants?: MatchInstant[]
  homeLineup?: Lineup
  awayLineup?: Lineup
  homePlayers?: Record<string, Player>
  awayPlayers?: Record<string, Player>
  ratings?: Record<string, number>
}

// The admin-configured weekdays a friendly may kick off on (see backend
// common.GetFriendlyScheduleWeekdays) - same bitmask convention as MatchSchedule.weekdaysMask.
export interface FriendlySchedule {
  weekdaysMask: number
}

// Either a domestic cup (country set, region empty) or an international one (region set,
// country empty, issue #64) - never both.
export interface Cup {
  cupId: number
  country: string
  region: string
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

// CupTieDetails is a single cup tie's full profile - the tie itself plus its event timeline and
// both teams' lineups/rosters, the cup equivalent of WeekMatch. Only populated (lineups/players)
// once the tie has two real teams - see backend/common.GetCupTieDetails.
export interface CupTieDetails extends CupTie {
  events: MatchEvent[] | null
  homeLineup: Lineup | null
  awayLineup: Lineup | null
  homePlayers: Record<string, Player> | null
  awayPlayers: Record<string, Player> | null
  // Only present once the tie has been played - the same per-player rating breakdown a league
  // match's WeekMatch or a played FriendlyMatch carries (see backend common.GetCupTieDetails).
  ratings?: Record<string, number>
}

// Match kind discriminators for TeamMatchSummary.kind - see backend/common/teammatches.go.
export const MATCH_KIND_LEAGUE = 'league'
export const MATCH_KIND_CUP = 'cup'
export const MATCH_KIND_FRIENDLY = 'friendly'

// TeamMatchSummary is one row in a team's combined fixture/results list (see useTeamMatches) -
// enough to render a list entry and route a click to the right detail/lineup page, since
// matchId's meaning depends on kind (a league match, cup tie, and friendly each draw their id
// from an independent sequence).
export interface TeamMatchSummary {
  kind: 'league' | 'cup' | 'friendly'
  matchId: number
  opponentId: number
  opponentName: string
  isHome: boolean
  matchDate: string
  played: boolean
  homeGoals: number
  awayGoals: number
  // Only meaningful for a cup tie - the round number within that cup's bracket.
  round?: number
}

// LiveMatchSummary is one match currently inside its live-broadcast window, anywhere in the
// world - see backend common.GetLiveMatches. Country/competitionName/tier are only set for a
// league match (kind === 'league'); a cup tie or friendly has no division to report. minute is
// only set on the dashboard's suggested pick (see backend common.enrichSuggestedLiveMatch), in
// which case homeGoals/awayGoals are the score as of that minute rather than the eventual final
// result.
export interface LiveMatchSummary {
  kind: 'league' | 'cup' | 'friendly'
  matchId: number
  homeTeamName: string
  awayTeamName: string
  matchDate: string
  homeGoals: number
  awayGoals: number
  country?: string
  competitionName?: string
  tier?: number
  minute?: number
}

// LiveMatchesOverview is the dashboard's "matches in play right now" summary - see backend
// common.GetLiveMatchesOverview. suggested is undefined when count is 0.
export interface LiveMatchesOverview {
  count: number
  suggested?: LiveMatchSummary
}

// notifications.notification_type values from backend/common/fmconstants.go
export const NOTIFICATION_MATCH_RESULT = 0
export const NOTIFICATION_PROMOTION = 1
export const NOTIFICATION_RELEGATION = 2
export const NOTIFICATION_TRANSFER = 3
export const NOTIFICATION_CUP = 4
export const NOTIFICATION_TRAINER = 5
export const NOTIFICATION_FRIENDLY = 6
export const NOTIFICATION_ACADEMY_PROMOTION = 7
export const NOTIFICATION_PROGRESSION = 8
export const NOTIFICATION_AWARD = 9
export const NOTIFICATION_FANBASE = 11

export interface Notification {
  notificationId: number
  notificationType: number
  // English text - used as the fallback when messageKey is absent (a notification created
  // before it existed). Prefer messageKey/messageParams (rendered via t()) when present, so the
  // notification shows in whatever language the viewer has chosen instead of always English.
  message: string
  messageKey?: string
  messageParams?: Record<string, unknown>
  // In-app route to navigate to on click (e.g. "/players/evolutions") - absent for every
  // notification type whose message alone already says everything relevant.
  link?: string
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
  stuckMatches: number
  activeCups: number
  transferWindowOpen: boolean
  matchScheduleWeekdaysMask: number
  matchScheduleHour: number
  matchScheduleMinute: number
  maxFriendliesPerWeek: number
  friendlyScheduleWeekdays: number
  promotionSlotsPerGroup: number
  relegationSlotsPerGroup: number
  freeAgentInjectionsPerWeek: number
  offerNegotiationWindowMinutes: number
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

// Mirrors backend/common.SearchResults - the global search page's response, grouped by what kind
// of thing matched. A coach or stadium result has no page of its own, so it links into the team
// that employs/owns it (same as TeamDetailsPage's CoachCard/StadiumCard).
export interface TeamSearchResult {
  id: number
  name: string
  country: string
  divisionName: string
}

export interface PlayerSearchResult {
  id: number
  firstName: string
  lastName: string
  teamId: number
  teamName: string
}

export interface CoachSearchResult {
  id: number
  firstName: string
  lastName: string
  teamId: number
  teamName: string
}

export interface StadiumSearchResult {
  teamId: number
  teamName: string
  stadiumName: string
}

export interface SearchResults {
  teams: TeamSearchResult[]
  players: PlayerSearchResult[]
  coaches: CoachSearchResult[]
  stadiums: StadiumSearchResult[]
}

export interface AdminLeagueInfo {
  leagueId: number
  country: string
  name: string
  tier: number
  schedule: MatchSchedule
  hasOverride: boolean
}
