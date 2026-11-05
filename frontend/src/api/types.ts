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
  // False until the user accepted the current Privacy Policy and Terms (and confirmed the minimum
  // age) - Layout gates every route behind /accept-terms while this is false. termsVersion is the
  // version they must accept (backend common.CurrentTermsVersion).
  termsAccepted: boolean
  termsVersion: number
  // Set once after their club was released for inactivity; onboarding tells them.
  releasedTeamName?: string
}

// GET /v1/user/account - the Account page's data. The two thresholds are the backend's own, so
// the page states exactly what the engine applies.
export interface AccountInfo {
  name: string
  email: string
  provider: string
  country: string
  createdAt: string
  lastSeenAt: string
  termsVersion: number | null
  termsAcceptedAt: string | null
  teamName: string
  releaseAfterDays: number
  erasureAfterDays: number
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
  // Form (0-99) moves match-by-match with the player's match ratings (unlike Stamina, not
  // minute-by-minute) - also excluded from SKILL_FIELDS for the same reason. It doesn't feed the
  // rating itself. 50 is neutral (backend common.DefaultForm). See backend Player.Form.
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
  // This player's international caps count - see backend common.GetPlayerCapsBatch.
  caps: number
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
  be: 'countries.be',
  tr: 'countries.tr',
  at: 'countries.at',
  sco: 'countries.sco',
  wal: 'countries.wal',
  ie: 'countries.ie',
  ru: 'countries.ru',
  rs: 'countries.rs',
  hr: 'countries.hr',
  ba: 'countries.ba',
  si: 'countries.si',
  mk: 'countries.mk',
  me: 'countries.me',
  xk: 'countries.xk',
  ve: 'countries.ve',
  py: 'countries.py',
  bo: 'countries.bo',
  qa: 'countries.qa',
  ae: 'countries.ae',
  uz: 'countries.uz',
  tn: 'countries.tn',
  tz: 'countries.tz',
  ml: 'countries.ml',
  al: 'countries.al',
  ad: 'countries.ad',
  am: 'countries.am',
  az: 'countries.az',
  by: 'countries.by',
  bg: 'countries.bg',
  cy: 'countries.cy',
  cz: 'countries.cz',
  dk: 'countries.dk',
  ee: 'countries.ee',
  fo: 'countries.fo',
  fi: 'countries.fi',
  ge: 'countries.ge',
  gi: 'countries.gi',
  hu: 'countries.hu',
  is: 'countries.is',
  il: 'countries.il',
  kz: 'countries.kz',
  lv: 'countries.lv',
  li: 'countries.li',
  lt: 'countries.lt',
  lu: 'countries.lu',
  mt: 'countries.mt',
  md: 'countries.md',
  nir: 'countries.nir',
  no: 'countries.no',
  ro: 'countries.ro',
  sm: 'countries.sm',
  sk: 'countries.sk',
  ch: 'countries.ch',
  ua: 'countries.ua',
  ai: 'countries.ai',
  ag: 'countries.ag',
  aw: 'countries.aw',
  bs: 'countries.bs',
  bb: 'countries.bb',
  bz: 'countries.bz',
  bm: 'countries.bm',
  vg: 'countries.vg',
  ky: 'countries.ky',
  cr: 'countries.cr',
  cu: 'countries.cu',
  cw: 'countries.cw',
  dm: 'countries.dm',
  do: 'countries.do',
  sv: 'countries.sv',
  gd: 'countries.gd',
  gt: 'countries.gt',
  gy: 'countries.gy',
  ht: 'countries.ht',
  hn: 'countries.hn',
  jm: 'countries.jm',
  ms: 'countries.ms',
  ni: 'countries.ni',
  pa: 'countries.pa',
  pr: 'countries.pr',
  kn: 'countries.kn',
  lc: 'countries.lc',
  vc: 'countries.vc',
  sr: 'countries.sr',
  tt: 'countries.tt',
  tc: 'countries.tc',
  vi: 'countries.vi',
  af: 'countries.af',
  au: 'countries.au',
  bh: 'countries.bh',
  bd: 'countries.bd',
  bt: 'countries.bt',
  bn: 'countries.bn',
  kh: 'countries.kh',
  tw: 'countries.tw',
  gu: 'countries.gu',
  hk: 'countries.hk',
  iq: 'countries.iq',
  jo: 'countries.jo',
  kw: 'countries.kw',
  kg: 'countries.kg',
  la: 'countries.la',
  lb: 'countries.lb',
  mo: 'countries.mo',
  my: 'countries.my',
  mv: 'countries.mv',
  mn: 'countries.mn',
  mm: 'countries.mm',
  np: 'countries.np',
  kp: 'countries.kp',
  om: 'countries.om',
  pk: 'countries.pk',
  ps: 'countries.ps',
  sg: 'countries.sg',
  lk: 'countries.lk',
  sy: 'countries.sy',
  tj: 'countries.tj',
  tl: 'countries.tl',
  tm: 'countries.tm',
  ye: 'countries.ye',
  ao: 'countries.ao',
  bj: 'countries.bj',
  bw: 'countries.bw',
  bf: 'countries.bf',
  bi: 'countries.bi',
  cv: 'countries.cv',
  cf: 'countries.cf',
  td: 'countries.td',
  km: 'countries.km',
  cg: 'countries.cg',
  cd: 'countries.cd',
  dj: 'countries.dj',
  gq: 'countries.gq',
  er: 'countries.er',
  sz: 'countries.sz',
  et: 'countries.et',
  ga: 'countries.ga',
  gm: 'countries.gm',
  gn: 'countries.gn',
  gw: 'countries.gw',
  ls: 'countries.ls',
  lr: 'countries.lr',
  ly: 'countries.ly',
  mg: 'countries.mg',
  mw: 'countries.mw',
  mr: 'countries.mr',
  mu: 'countries.mu',
  mz: 'countries.mz',
  na: 'countries.na',
  ne: 'countries.ne',
  rw: 'countries.rw',
  st: 'countries.st',
  sc: 'countries.sc',
  sl: 'countries.sl',
  so: 'countries.so',
  ss: 'countries.ss',
  sd: 'countries.sd',
  tg: 'countries.tg',
  ug: 'countries.ug',
  zm: 'countries.zm',
  zw: 'countries.zw',
  as: 'countries.as',
  ck: 'countries.ck',
  fj: 'countries.fj',
  nc: 'countries.nc',
  nz: 'countries.nz',
  pg: 'countries.pg',
  ws: 'countries.ws',
  sb: 'countries.sb',
  pf: 'countries.pf',
  to: 'countries.to',
  vu: 'countries.vu',
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
  country: string
  name: string
  divisionId: number
  divisionName: string
  classificationPosition: number
  // Only meaningfully populated on the caller's own team (GET /v1/user/team) - see
  // useRenameTeam. Elsewhere it's just the zero value, false.
  nameChanged: boolean
  // True for a country's unowned national team (see backend common.CreateNationalTeam) - no
  // division, stadium, or manager, and its "players" below are the country's current pool of
  // eligible players rather than a real roster. Only ever true coming back from
  // GET /v1/teams/team/:id/, since that's the only lookup that can resolve to a national team.
  isNationalTeam?: boolean
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
  // This team's current head coach, or absent if it hasn't hired one (or is a national team,
  // which never has one) - see backend common.TeamDetails.Trainer. Public, same visibility tier
  // as stadium/fanbase; only the hire/fire actions themselves stay owner-only.
  trainer?: Trainer
  // Players this club has out on loan, and the players of `players` who are only there on loan -
  // see backend common.TeamDetails. Public, always arrays (empty for a national team).
  loanedOut: LoanedOutPlayer[]
  loanedIn: LoanedInPlayer[]
}

// A player a club has out on loan, with the borrower and the loan's terms. Mirrors backend
// common.LoanedOutPlayer. wageContributionPercent is the share of the wage the lender keeps
// paying; buyOptionFee is 0 when the loan has no buy option.
export interface LoanedOutPlayer {
  player: Player
  borrowingTeamId: number
  borrowingTeamName: string
  borrowingDivisionId: number
  borrowingDivisionName: string
  // The borrowing club's country and that country's current season: returnSeason counts in that
  // country's own seasons, which need not match the lender's.
  borrowingCountry: string
  borrowingCountrySeason: number
  returnSeason: number
  wageContributionPercent: number
  buyOptionFee: number
  goalsForBorrower: number
}

// A squad player who is only on loan to the club. Mirrors backend common.LoanedInPlayer.
export interface LoanedInPlayer {
  playerId: number
  parentTeamId: number
  parentTeamName: string
  returnSeason: number
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

// ProjectedReplacement says who the pre-kickoff refill would put in an unavailable starter's slot
// if the match kicked off now. Only the lineup GET endpoints an owner opens fill it, and only when a
// starter is unavailable; the saved lineup itself is never changed.
export interface ProjectedReplacement {
  playerOutId: number
  playerInId: number
  position: number
}

export interface Lineup {
  matchId: number
  teamId: number
  formation: number
  positions: LineupPosition[]
  substitutions: LineupSubstitution[]
  projectedReplacements?: ProjectedReplacement[]
  // How the side plays (see backend common/mentality.go): MENTALITY_DEFENSIVE, _BALANCED or
  // _ATTACKING. A played match's lineup carries the mentality it kicked off with.
  mentality?: number
  // The lineup's conditional match orders (see backend common/lineuporders.go) - only in the
  // owner's own lineup editor payloads; nobody else ever sees a team's orders.
  orders?: LineupOrder[]
  // The lineup's optional roles (see backend common/lineuproles.go); missing = automatic. A played
  // match's lineups only carry the captain - the set-piece takers are the owner's business.
  captainId?: number
  penaltyTakerId?: number
  freeKickTakerId?: number
}

// The three optional lineup roles, as the lineup PUT endpoints take them.
export interface LineupRoles {
  captainId?: number
  penaltyTakerId?: number
  freeKickTakerId?: number
}

// Mirrors backend common.LineupOrder: armed from fromMinute (1-89), fired once at the first minute
// whose starting score meets condition, doing action.
export interface LineupOrder {
  fromMinute: number
  condition: number
  action: number
  mentality: number
  playerOutId?: number
  playerInId?: number
}

export const ORDER_CONDITION_LOSING = 0
export const ORDER_CONDITION_LOSING_BY_2 = 1
export const ORDER_CONDITION_DRAWING = 2
export const ORDER_CONDITION_WINNING = 3
export const ORDER_CONDITION_WINNING_BY_2 = 4
export const ORDER_CONDITION_KEYS: Record<number, string> = {
  [ORDER_CONDITION_LOSING]: 'losing',
  [ORDER_CONDITION_LOSING_BY_2]: 'losingBy2',
  [ORDER_CONDITION_DRAWING]: 'drawing',
  [ORDER_CONDITION_WINNING]: 'winning',
  [ORDER_CONDITION_WINNING_BY_2]: 'winningBy2',
}
export const ORDER_ACTION_MENTALITY = 0
export const ORDER_ACTION_SUBSTITUTION = 1
export const MAX_CONDITIONAL_ORDERS = 3
export const ORDER_MIN_FROM_MINUTE = 1
export const ORDER_MAX_FROM_MINUTE = 89

export const MENTALITY_DEFENSIVE = -1
export const MENTALITY_BALANCED = 0
export const MENTALITY_ATTACKING = 1

export const MENTALITY_KEYS: Record<number, 'defensive' | 'balanced' | 'attacking'> = {
  [MENTALITY_DEFENSIVE]: 'defensive',
  [MENTALITY_BALANCED]: 'balanced',
  [MENTALITY_ATTACKING]: 'attacking',
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
  // How a GOAL / NO_GOAL came about (SET_PIECE_*); optional so an old or cached payload without it
  // reads as open play. A set-piece event lists the defending goalkeeper in otherTeamPlayers.
  setPiece?: number
  // The side's new mentality on an EVENT_TYPE_MENTALITY_CHANGE event (MENTALITY_*).
  mentality?: number
}

// SetPiece values from backend/common/fmconstants.go
export const SET_PIECE_NONE = 0
export const SET_PIECE_PENALTY = 1
export const SET_PIECE_DIRECT_FREE_KICK = 2

// EventType values from backend/common/fmconstants.go
export const EVENT_TYPE_GOAL = 0
export const EVENT_TYPE_NO_GOAL = 1
export const EVENT_TYPE_YELLOW_CARD = 2
export const EVENT_TYPE_RED_CARD = 3
export const EVENT_TYPE_INJURY = 4
export const EVENT_TYPE_SUBSTITUTION = 5
export const EVENT_TYPE_MATCH_ABANDONED = 6
// A fixture settled 0-0 after repeated simulation failures (backend common.MATCH_DEFAULT_RESULT) -
// teamOwner is the home side, otherTeam the away side; it is the only event such a match stores.
export const EVENT_TYPE_MATCH_DEFAULT_RESULT = 7
// A conditional order switched teamOwner's mentality (backend common.MENTALITY_CHANGE) - its new
// mentality is in `mentality`.
export const EVENT_TYPE_MENTALITY_CHANGE = 8

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
  // This minute's zone contribution of every player on the pitch, per side (h = home, a = away),
  // keyed by player id - absent for an instant saved before stored ratings. See backend
  // common.InstantContributions and utils/liveMatchRatings.ts, which builds the ratings from them.
  playerContributions?: InstantContributions
}

export interface InstantContributions {
  h: Record<string, PlayerContribution>
  a: Record<string, PlayerContribution>
}

// One player's share of the zone duels his position feeds during one minute (backend
// common.PlayerContribution): p = position, o = his own value, r = the rival's per-player value in
// the mirrored duel(s).
export interface PlayerContribution {
  p: number
  o: number
  r: number
}

// One player's rating for one match with its breakdown (backend common.PlayerRating): rating is
// 6.0 + zone + goals + cards + injury + result, clamped to [1, 10]. rated is false for an
// appearance under 10 minutes ("s.c.") - the breakdown is still there, but no rating is shown.
export interface PlayerRating {
  playerId: number
  teamId: number
  rated: boolean
  rating: number
  minutes: number
  position: number
  cameOn?: number
  wentOff?: number
  // Mean zone contribution against the rival's - absent for a match simulated before stored ratings.
  own?: number
  rival?: number
  zone: number
  goals: number
  cards: number
  injury: number
  result: number
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
  // Per-player match rating (1-10) of every rated player, keyed by player id as a string (JSON
  // object keys are always strings) - only present for a played match with lineup data.
  ratings?: Record<string, number>
  // Every player who featured, with his rating breakdown (see PlayerRating).
  playerRatings?: PlayerRating[]
  // The match's rivalry level (RIVALRY_*): the one it was played at, or the pair's current one.
  rivalry?: number
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
  // Place in the season's starting order, the last tie-break before the team id; null when the team has none.
  seed?: number | null
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
  // How many of the table's top / bottom finishers move tier at season rollover (0 when there is
  // no tier above / below) - drives the promotion and relegation zones. Absent on responses cached
  // before the field existed.
  promotionSlots?: number
  relegationSlots?: number
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

// Short (2-3 letter) labels for every position - used wherever a position needs to fit next to a
// player's name in a compact chip or line (transfer market rows, lineup editor cards, the match
// ratings list) rather than the full POSITION_LABEL_KEYS word.
export const POSITION_SHORT_LABEL_KEYS: Record<number, string> = {
  [POSITION_GOALKEEPER]: 'positions.short.goalkeeper',
  [POSITION_DEFENSE]: 'positions.short.defense',
  [POSITION_LATERAL_DEFENSE_LEFT]: 'positions.short.leftBack',
  [POSITION_LATERAL_DEFENSE_RIGHT]: 'positions.short.rightBack',
  [POSITION_MIDFIELD]: 'positions.short.midfield',
  [POSITION_LATERAL_MIDFIELD_LEFT]: 'positions.short.leftMidfield',
  [POSITION_LATERAL_MIDFIELD_RIGHT]: 'positions.short.rightMidfield',
  [POSITION_STRIKER]: 'positions.short.striker',
  [POSITION_LATERAL_STRIKER_LEFT]: 'positions.short.leftStriker',
  [POSITION_LATERAL_STRIKER_RIGHT]: 'positions.short.rightStriker',
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

// player_transfers.kind values from backend/common/fmconstants.go
export const PLAYER_TRANSFER_LISTING = 0
export const PLAYER_TRANSFER_OFFER = 1
export const PLAYER_TRANSFER_FREE_AGENT = 2
export const PLAYER_TRANSFER_LOAN = 3
export const PLAYER_TRANSFER_BUY_OPTION = 4
export const PLAYER_TRANSFER_RELEASE = 5
export const PLAYER_TRANSFER_PERMANENT = 6

// Values are i18n keys under "playerPage.history.kinds.*".
export const PLAYER_TRANSFER_KIND_KEYS: Record<number, string> = {
  [PLAYER_TRANSFER_LISTING]: 'playerPage.history.kinds.listing',
  [PLAYER_TRANSFER_OFFER]: 'playerPage.history.kinds.offer',
  [PLAYER_TRANSFER_FREE_AGENT]: 'playerPage.history.kinds.freeAgent',
  [PLAYER_TRANSFER_LOAN]: 'playerPage.history.kinds.loan',
  [PLAYER_TRANSFER_BUY_OPTION]: 'playerPage.history.kinds.buyOption',
  [PLAYER_TRANSFER_RELEASE]: 'playerPage.history.kinds.release',
  [PLAYER_TRANSFER_PERMANENT]: 'playerPage.history.kinds.permanent',
}

// One row of a player's transfer history. A missing fromTeamId means the player came from free
// agency; a missing toTeamId means they were released.
export interface PlayerTransfer {
  id: number
  fromTeamId?: number
  fromTeamName?: string
  toTeamId?: number
  toTeamName?: string
  fee: number
  kind: number
  createdAt: string
}

// One recent sale of a player similar to a transfer target - backend common.TransferComparable.
// Only public figures: position, age and average skill are the snapshot taken when he was sold.
export interface TransferComparable {
  transferId: number
  playerId: number
  playerName: string
  position: number
  age: number
  averageSkill: number
  fee: number
  kind: number
  createdAt: string
  fromTeamId?: number
  fromTeamName?: string
  toTeamId?: number
  toTeamName?: string
}

// GET /v1/teams/player/:id/transfer-comparables - backend common.TransferComparablesResult. The
// fee figures are absent when there are no comparables; broadened means the wide band was used.
export interface TransferComparablesResult {
  position: number
  age: number
  averageSkill: number
  broadened: boolean
  count: number
  medianFee?: number
  minFee?: number
  maxFee?: number
  comparables: TransferComparable[]
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
export const TRANSACTION_ADMIN_MASS_INJECTION = 12
export const TRANSACTION_LOAN_FEE = 13
export const TRANSACTION_LOAN_WAGE_CONTRIBUTION = 14
export const TRANSACTION_STADIUM_MAINTENANCE = 15
export const TRANSACTION_TEAM_RESET = 16

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
  [TRANSACTION_ADMIN_MASS_INJECTION]: 'finances.reasons.adminMassInjection',
  [TRANSACTION_LOAN_FEE]: 'finances.reasons.loanFee',
  [TRANSACTION_LOAN_WAGE_CONTRIBUTION]: 'finances.reasons.loanWageContribution',
  [TRANSACTION_STADIUM_MAINTENANCE]: 'finances.reasons.stadiumMaintenance',
  [TRANSACTION_TEAM_RESET]: 'finances.reasons.teamReset',
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
// Lowest ticket price an owner may set (backend common.MinTicketPrice); the highest comes from
// FinancesResponse.maxTicketPrice.
export const MIN_TICKET_PRICE = 1
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

// Stadium maintenance cost per round - mirrors the backend's engine.StadiumMaintenanceCost:
// linear in capacity, so a bigger stadium costs more to run every round, not just more to build.
// Used to show the ongoing cost on the stadium card - the backend remains the source of truth.
export const STADIUM_MAINTENANCE_COST_PER_SEAT = 1
export const STADIUM_MAINTENANCE_COST_DIVISOR = 10

export function calculateStadiumMaintenanceCost(capacity: number): number {
  return Math.floor((capacity * STADIUM_MAINTENANCE_COST_PER_SEAT) / STADIUM_MAINTENANCE_COST_DIVISOR)
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
  // This team's club-member count - see backend common.GetSocios. The paying core of its home
  // crowd: attendance is socios x price x form x rival x weather, capped at stadium capacity.
  socios: number
  // Multiplier this team's division applies to every wage it pays - see backend
  // common.DivisionWageFactor. Contract wages are the base figures; this is what they are
  // multiplied by when actually charged each round.
  wageFactor: number
  // The price that maximises this team's expected gate revenue - the one a CPU Manager team
  // charges (backend engine.RecommendedTicketPrice) - and the highest price an owner may set
  // (backend common.MaxTicketPrice). The lowest is MIN_TICKET_PRICE.
  recommendedTicketPrice: number
  maxTicketPrice: number
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
  // This listing's player's international caps count - see backend common.GetPlayerCapsBatch.
  caps: number
}

// One followed player in the club's watchlist (see backend common.WatchlistEntry): his current
// club (none for a free agent) and, when he is on the market, his open transfer or loan listing.
export interface WatchlistEntry {
  player: Player
  teamName?: string
  followedAt: string
  transferListing?: TransferListing
  loanListing?: LoanListing
}

// The most players a club can follow - mirrors backend common.MaxWatchlistSize.
export const MAX_WATCHLIST_SIZE = 30

export interface LoanListing {
  listingId: number
  player: Player
  lendingTeamId: number
  lendingTeamName: string
  durationSeasons: number
  // All three are optional loan terms (zero/off by default) - see backend common.CreateLoanListing.
  loanFee: number
  wageContributionPercent: number
  buyOptionFee: number
  listedAt: string
  // This listing's player's international caps count - see backend common.GetPlayerCapsBatch.
  caps: number
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
  // The league the team played in that season; null when it was not recorded.
  leagueId: number | null
  leagueName: string | null
  finalPosition: number | null
  promoted: boolean
  relegated: boolean
  cupResult: string | null
}

// A league's season picker: the current season and every past one, newest first.
export interface LeagueSeasons {
  current: number
  past: number[]
}

// One line of a past season's final table. `name` is the club's current name.
export interface LeagueSeasonPosition {
  position: number
  teamId: number
  name: string
  matches: number
  points: number
  wins: number
  draws: number
  losses: number
  goalsScored: number
  goalsReceived: number
  promoted: boolean
  relegated: boolean
}

// A league's final table for a past season. `name` is the league name recorded that season.
export interface LeagueSeasonTable {
  id: number
  name: string
  season: number
  totalWeeks: number
  positions: LeagueSeasonPosition[]
}

export interface ScorerStanding {
  playerId: number
  firstName: string
  lastName: string
  // See Player.birthCountry - empty for a legacy player without one.
  birthCountry?: string
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
  // How many played national-team friendlies this player was selected to start - see backend
  // common.SetNationalTeamLineup/GetPlayerCareerStats.
  caps: number
  unavailableReason?: number
  unavailableMatchesRemaining?: number
  onLoanFromTeamId?: number
  onLoanFromTeamName?: string
  onLoanReturnSeason?: number
  // Only present when the loan actually carries that term - see backend common.PlayerStatus.
  onLoanWageContributionPercent?: number
  onLoanBuyOptionFee?: number
  transferListingAskingPrice?: number
  loanListingDurationSeasons?: number
  // How many times this player has been their league's Golden Boot / Player of the Season - see
  // backend common.GetPlayerSeasonAwards.
  seasonAwards: { goldenBoots: number; playerOfSeasons: number }
  history: TeamTransaction[]
  // This player's transfers, newest first - unlike `history` (the owning team's accounting
  // ledger), never redacted: it is market information. See backend common.GetPlayerTransfers.
  transfers: PlayerTransfer[]
  latestEvolution?: PlayerEvolution
  // Every club this player has permanently belonged to, most recent first - unlike `history`,
  // never redacted for a non-owner viewer (which clubs a player played for is public, the same
  // way a transfer-market listing is).
  clubHistory: PlayerClubHistoryEntry[]
  // This player's current team's most recent played league matches - see
  // backend/common/playermatches.go's GetPlayerRecentMatches.
  recentMatches: PlayerMatchSummary[]
  // This player's most recent national-team call-ups (friendly or World Cup knockout tie) - see
  // backend common.GetRecentNationalTeamCallUps.
  recentCallUps: NationalTeamCallUp[]
}

// Mirrors backend common.NationalTeamCallUp - one row in a player's "recent call-ups" list.
// `kind` picks which detail page to link to: "friendly" -> /friendlies/:matchId, "cupTie" ->
// /cup/ties/:matchId.
export interface NationalTeamCallUp {
  kind: 'friendly' | 'cupTie'
  matchId: number
  teamName: string
  opponentName: string
  matchDate: string
  played: boolean
  goalsFor: number
  goalsAgainst: number
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
  birthCountry?: string
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
  // The selling club's counter-offer (see backend common.CounterOffer): a higher fee that now
  // waits for the buyer to accept or decline until counterExpiresAt. Absent when not countered.
  counterAmount?: number
  counteredAt?: string
  counterExpiresAt?: string
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
  // The fee paid (>= 0) - the loan fee for a loan, 0 when it was free.
  amount: number
  // One of the PLAYER_TRANSFER_* values; PLAYER_TRANSFER_LOAN rows get a "Loan" chip.
  kind: number
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
  playerRatings?: PlayerRating[]
  // The home team's stadium (absent for a national team, which has none) and the stored report
  // flavor of a played friendly - absent for one played before the match viewer stored them.
  homeStadiumName?: string
  weather?: string
  attendance?: number
  refereeName?: string
  assistant1Name?: string
  assistant2Name?: string
  // Set only when this friendly is a World Cup group fixture (see backend common.FriendlyMatch).
  worldCupGroupName?: string
  worldCupRound?: number
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
// Mirrors backend common.WorldCup - one edition's top-level record. knockoutCupId is null while
// stage is "groups"; once it's set, that cup's bracket is fetched through the existing
// useCupDetails(cupId) hook, same as any other cup.
export interface WorldCup {
  worldCupId: number
  season: number
  name: string
  stage: 'groups' | 'knockout'
  completed: boolean
  championTeamId: number | null
  championTeamName: string
  knockoutCupId: number | null
}

// Mirrors backend common.WorldCupGroupStanding - one team's running record in a group, already
// ranked (points, then goal difference, then goals for) by the backend.
export interface WorldCupGroupStanding {
  teamId: number
  teamName: string
  played: number
  won: number
  drawn: number
  lost: number
  goalsFor: number
  goalsAgainst: number
}

// Broadcast-safe status every national-team feed reports (see backend
// common.applyNationalMatchStatus): "scheduled" carries no score, "live" the minute and the score
// as of that minute, "finished" the final score.
export type NationalMatchStatus = 'scheduled' | 'live' | 'finished'

// Mirrors backend common.WorldCupGroupMatch - one group fixture, broadcast-safe.
export interface WorldCupGroupMatch {
  matchId: number
  round: number
  homeTeamId: number
  homeTeamName: string
  awayTeamId: number
  awayTeamName: string
  matchDate: string
  status: NationalMatchStatus
  minute?: number
  homeGoals?: number
  awayGoals?: number
}

// Mirrors backend common.WorldCupGroup.
export interface WorldCupGroup {
  groupId: number
  groupName: string
  standings: WorldCupGroupStanding[]
  matches: WorldCupGroupMatch[]
}

// Response shape of GET /v1/national-teams/world-cup/current. worldCup is null when none has
// ever been played; nextWorldCupEstimate is set only while none is running and enough countries
// are active for one to start.
export interface CurrentWorldCup {
  worldCup: WorldCup | null
  groups: WorldCupGroup[]
  nextWorldCupEstimate?: string | null
}

export type NationalMatchKind = 'friendly' | 'worldCupGroup' | 'worldCupKnockout'

// Mirrors backend common.NationalMatch - one national-team fixture of any kind, broadcast-safe
// (see NationalMatchStatus). id is a friendly match id, or a cup tie id for a knockout tie.
// groupName/round label a World Cup group match (round = matchday); round/totalRounds a knockout
// tie. myPlayers counts the viewer's own players in its saved call-up.
export interface NationalMatch {
  kind: NationalMatchKind
  id: number
  homeTeamId: number
  homeTeamName: string
  homeCountry: string
  awayTeamId: number
  awayTeamName: string
  awayCountry: string
  kickoff: string
  status: NationalMatchStatus
  minute?: number
  homeGoals?: number
  awayGoals?: number
  groupName?: string
  round?: number
  totalRounds?: number
  myPlayers?: number
}

// One page of GET /v1/national-teams/results.
export interface NationalTeamResultsPage {
  matches: NationalMatch[]
  hasMore: boolean
}

export interface CupTieDetails extends CupTie {
  events: MatchEvent[] | null
  homeLineup: Lineup | null
  awayLineup: Lineup | null
  homePlayers: Record<string, Player> | null
  awayPlayers: Record<string, Player> | null
  // Only present once the tie has been played - the same per-player rating breakdown a league
  // match's WeekMatch or a played FriendlyMatch carries (see backend common.GetCupTieDetails).
  ratings?: Record<string, number>
  playerRatings?: PlayerRating[]
  // Same additions as FriendlyMatch: per-minute instants (for the zone duel), the home stadium and
  // the stored report flavor - only for a played tie simulated after the match viewer stored them.
  matchInstants?: MatchInstant[]
  homeStadiumName?: string
  weather?: string
  attendance?: number
  refereeName?: string
  assistant1Name?: string
  assistant2Name?: string
  // The match's rivalry level (RIVALRY_*): the one it was played at, or the pair's current one.
  rivalry?: number
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
  // The fixture's rivalry level (RIVALRY_*): the one it was played at, or the pair's current one
  // for a fixture still to come. 0 for a friendly.
  rivalry?: number
}

// Rivalry levels (backend common.RIVALRY_*, see openspec rivalries).
export const RIVALRY_NONE = 0
export const RIVALRY_RIVAL = 1
export const RIVALRY_CLASICO = 2
// The highest rivalry score a pair can reach (backend common.RivalryMaxScore).
export const RIVALRY_MAX_SCORE = 40

// One of a club's rivals as its team page lists it (backend common.TeamRivalry).
export interface TeamRivalry {
  teamId: number
  teamName: string
  score: number
  level: number
}

// OpponentPreview is the owner's preview of an upcoming fixture - see backend
// common.GetOpponentPreview and handlers.OpponentPreviewHandler. Everything about the opponent is
// public data (finished results, kickoff lineups, stored ratings); projection rates the owner's
// saved lineup against the opponent's likely XI with the simulation's own minute-1 zone ratings
// (null when the opponent has no history to project against).
export interface OpponentPreviewPlayer {
  playerId: number
  name: string
  position: number
}

export interface OpponentPreviewLikelyLineup {
  kind: TeamMatchSummary['kind']
  matchId: number
  formation: number
  players: OpponentPreviewPlayer[]
}

export interface OpponentPreviewInFormPlayer extends OpponentPreviewPlayer {
  averageRating: number
  appearances: number
}

export interface OpponentPreviewProjection {
  own: TeamQualification
  opponent: TeamQualification
  ownGoalkeeper: number
  opponentGoalkeeper: number
}

export interface OpponentPreview {
  opponent: { id: number; name: string; position: number; form: TeamMatchSummary[] }
  likely: OpponentPreviewLikelyLineup | null
  inForm: OpponentPreviewInFormPlayer[]
  projection: OpponentPreviewProjection | null
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

// LeagueLiveScore is one league match inside its live-broadcast window, as its broadcast shows it
// right now (never the stored result) - see backend common.GetLeagueLiveScores. A pending entry is
// a match that has not kicked off or is not simulated yet (minute 0, 0-0).
export type LeagueLiveStatus = 'pending' | 'live' | 'halftime' | 'finished'

export interface LeagueLiveScore {
  matchId: number
  weekNumber: number
  matchDate: string
  home: number
  homeName: string
  away: number
  awayName: string
  status: LeagueLiveStatus
  minute: number
  homeGoals: number
  awayGoals: number
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
export const NOTIFICATION_ACCOUNT = 13
export const NOTIFICATION_RIVALRY = 14
export const NOTIFICATION_COUNTER_OFFER = 15
export const NOTIFICATION_WATCHLIST = 16

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

// CountrySeason is one country's own current season and match progress - current_season now
// advances per country rather than globally, so there's no single season number for the whole
// game to show at once (see AdminOverview.seasons).
export interface CountrySeason {
  country: string
  season: number
  matchesTotal: number
  matchesUnplayed: number
}

// One count per kind of fixture a simulation can fail on (backend common.FixtureKindCounts).
export interface FixtureKindCounts {
  leagueMatches: number
  friendlies: number
  cupTies: number
}

// An unplayed fixture whose simulation failed at least once - kind is 'league', 'friendly' or
// 'cup', id the match, friendly or tie id.
export interface FailingFixture {
  kind: 'league' | 'friendly' | 'cup'
  id: number
  homeName: string
  awayName: string
  matchDate: string
  attempts: number
  lastError: string
}

// What the admin overview shows about simulations that did not go through (backend
// common.SimulationHealth): rows claimed long ago and still unplayed, fixtures whose simulation
// failed (the list is capped at 50, failingCounts has the totals) and classification rows out of
// step with the matches played.
export interface SimulationHealth {
  stuck: number
  failingCounts: FixtureKindCounts
  failing: FailingFixture[]
  standingsMismatches: number
}

// What the "finish stuck matches" action did, per kind of fixture: re-simulated to a result,
// released for their later kickoff, settled 0-0 after their attempts ran out, and still failing
// afterwards.
export interface FinishStuckMatchesResult {
  resimulated: FixtureKindCounts
  // Fixtures that had used all their attempts but were re-dated to a later kickoff: they get their
  // attempts back and are simulated normally at that kickoff.
  released: FixtureKindCounts
  defaulted: FixtureKindCounts
  stillFailing: FixtureKindCounts
}

export interface AdminOverview {
  seasons: CountrySeason[]
  userCount: number
  teamCount: number
  leagueCount: number
  simulation: SimulationHealth
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
  // One row per AI level (backend common.AiLevelStats), human = owned clubs.
  aiLevels?: AiLevelStats[]
}

// One AI level's clubs (backend common.AiLevelStats): how many, their average budget, how many
// are insolvent and the league points per match they have taken this season.
export interface AiLevelStats {
  level: 'human' | 'caretaker' | 'manager' | 'director'
  teams: number
  averageBudget: number
  insolvent: number
  pointsPerMatch: number
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
  // When the user last did anything - what the inactivity rules (club released after 60 days,
  // account erased after 365) count from.
  lastSeenAt: string
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
  birthCountry?: string
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
