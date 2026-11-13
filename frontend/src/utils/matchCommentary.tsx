import type { ReactNode } from 'react'
import {
  EVENT_TYPE_ADDED_TIME,
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_MATCH_ABANDONED,
  EVENT_TYPE_MATCH_DEFAULT_RESULT,
  EVENT_TYPE_MENTALITY_CHANGE,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_YELLOW_CARD,
  SET_PIECE_DIRECT_FREE_KICK,
  SET_PIECE_PENALTY,
  MENTALITY_KEYS,
  type MatchEvent,
} from '../api/types'
import { PlayerNameLink } from '../components/PlayerNameLink'
import { TeamNameLink } from '../components/TeamNameLink'
import { compareMatchMinute } from './matchMinute'

const COMMENTARY_KEYS: Partial<Record<number, string>> = {
  [EVENT_TYPE_GOAL]: 'match.commentary.goal',
  [EVENT_TYPE_NO_GOAL]: 'match.commentary.noGoal',
  [EVENT_TYPE_YELLOW_CARD]: 'match.commentary.yellowCard',
  [EVENT_TYPE_RED_CARD]: 'match.commentary.redCard',
  [EVENT_TYPE_INJURY]: 'match.commentary.injury',
  [EVENT_TYPE_SUBSTITUTION]: 'match.commentary.substitution',
  [EVENT_TYPE_MATCH_ABANDONED]: 'match.commentary.abandoned',
  [EVENT_TYPE_MATCH_DEFAULT_RESULT]: 'match.commentary.defaultResult',
}

// A penalty or direct free kick narrates with its own phrasing; a missing setPiece (old or cached
// payload) falls through to the open-play key.
const SET_PIECE_COMMENTARY_KEYS: Partial<Record<number, Partial<Record<number, string>>>> = {
  [EVENT_TYPE_GOAL]: {
    [SET_PIECE_PENALTY]: 'match.commentary.penaltyGoal',
    [SET_PIECE_DIRECT_FREE_KICK]: 'match.commentary.freeKickGoal',
  },
  [EVENT_TYPE_NO_GOAL]: {
    [SET_PIECE_PENALTY]: 'match.commentary.penaltyMiss',
    [SET_PIECE_DIRECT_FREE_KICK]: 'match.commentary.freeKickMiss',
  },
}

function commentaryKey(event: MatchEvent): string | undefined {
  // A mentality change narrates the side's new approach (match.commentary.mentalityChange.*).
  if (event.eventType === EVENT_TYPE_MENTALITY_CHANGE) {
    const mentality = event.mentality === undefined ? undefined : MENTALITY_KEYS[event.mentality]
    return mentality ? `match.commentary.mentalityChange.${mentality}` : undefined
  }
  return SET_PIECE_COMMENTARY_KEYS[event.eventType]?.[event.setPiece ?? 0] ?? COMMENTARY_KEYS[event.eventType]
}

// Picks a stable index into a list of phrasing variants from a few small integers - the same
// event (same minute/type/player) always narrates the same way across re-renders, but different
// events land on different variants, without needing to store which one was picked anywhere.
function pickVariantIndex(count: number, seedParts: number[]): number {
  let hash = 0
  for (const part of seedParts) {
    hash = (hash * 31 + part) % 1000003
  }
  return Math.abs(hash) % count
}

// A {{key}} placeholder's fill-in text, plus the player or team it should link to when it names
// one - a placeholder with neither renders as plain text.
export interface CommentaryVar {
  text: string
  playerId?: number
  teamId?: number
}

// Splits template on its {{...}} placeholders and fills each one in from vars, rendering a
// placeholder with a playerId/teamId as a clickable PlayerNameLink/TeamNameLink (issue: clickable
// player and team names in the match report) and everything else - including a placeholder with
// no matching var - as plain text, same as interpolate's old `?? ''` fallback.
export function interpolateNodes(template: string, vars: Record<string, CommentaryVar>): ReactNode[] {
  return template.split(/(\{\{\w+\}\})/g).map((part, index) => {
    const match = /^\{\{(\w+)\}\}$/.exec(part)
    if (!match) {
      return part
    }
    const v = vars[match[1]]
    if (!v) {
      return ''
    }
    if (v.playerId !== undefined) {
      return (
        <PlayerNameLink key={index} playerId={v.playerId}>
          {v.text}
        </PlayerNameLink>
      )
    }
    if (v.teamId !== undefined) {
      return (
        <TeamNameLink key={index} teamId={v.teamId}>
          {v.text}
        </TeamNameLink>
      )
    }
    return v.text
  })
}

export type Translate = (key: string, options?: { returnObjects: true }) => unknown

/** Turns one match event into a single radio-commentator-style line, picking one of several
 * i18n-translated phrasings for its event type (see match.commentary.* in the locale files) -
 * with any player or team name in that phrasing rendered as a clickable link to that player's/
 * team's page rather than plain text. Returns undefined for event types with no commentary (or
 * missing player data to fill one in) - callers should fall back to their own rendering in that
 * case. */
export function commentaryFor(
  event: MatchEvent,
  t: Translate,
  teamId: number,
  teamName: string,
  opponentId: number,
  opponentName: string,
): ReactNode[] | undefined {
  const key = commentaryKey(event)
  if (!key) {
    return undefined
  }

  const variants = t(key, { returnObjects: true })
  if (!Array.isArray(variants) || variants.length === 0) {
    return undefined
  }

  const playerIds = event.teamOwnerPlayers ?? []
  const playerNames = event.teamOwnerPlayerNames ?? []
  const template = variants[pickVariantIndex(variants.length, [event.minute, event.eventType, event.setPiece ?? 0, playerIds[0] ?? 0])] as string

  if (event.eventType === EVENT_TYPE_SUBSTITUTION) {
    if (playerNames.length < 2) {
      return undefined
    }
    return interpolateNodes(template, {
      team: { text: teamName, teamId },
      opponent: { text: opponentName, teamId: opponentId },
      outPlayer: { text: playerNames[0], playerId: playerIds[0] },
      inPlayer: { text: playerNames[1], playerId: playerIds[1] },
    })
  }

  // Unlike every other event type, an abandoned match names no player - just the short-handed
  // team (event.teamOwner, resolved to team/teamName by the caller) and the side awarded the win.
  // A default result likewise names no player: just the two sides (team is the home side).
  if (
    event.eventType === EVENT_TYPE_MATCH_ABANDONED ||
    event.eventType === EVENT_TYPE_MATCH_DEFAULT_RESULT ||
    event.eventType === EVENT_TYPE_MENTALITY_CHANGE
  ) {
    return interpolateNodes(template, {
      team: { text: teamName, teamId },
      opponent: { text: opponentName, teamId: opponentId },
    })
  }

  const player = playerNames[0]
  if (!player) {
    return undefined
  }
  // The defending goalkeeper of a chance (open play or set piece). An event stored before the
  // keeper was recorded has none, so a missing name reads as "the goalkeeper" instead of leaving a
  // hole in the sentence.
  const keeperId = event.otherTeamPlayers?.[0]
  const keeperName = event.otherTeamPlayerNames?.[0]
  const keeper: CommentaryVar = keeperName
    ? { text: keeperName, playerId: keeperId }
    : { text: String(t('match.setPiece.theKeeper')) }
  return interpolateNodes(template, {
    player: { text: player, playerId: playerIds[0] },
    team: { text: teamName, teamId },
    opponent: { text: opponentName, teamId: opponentId },
    keeper,
  })
}

// ---------------------------------------------------------------------------------------------
// Whole-match narration (openspec match-commentary). commentaryFor above narrates one event in
// isolation; narrateMatch narrates a whole event list in match order, so a line can read the story
// so far (an equaliser, a hat-trick, a second yellow) and a pool never repeats a line until it has
// used them all.
// ---------------------------------------------------------------------------------------------

export interface NarrationTeams {
  homeId: number
  homeName: string
  awayId: number
  awayName: string
  // Tells two fixtures between the same sides apart (the kickoff time), so a rematch doesn't read
  // like the first match. Known before kickoff, so it is the same live and after full time.
  matchKey?: string
}

/** One event's narration: the main line and its pool key, plus an optional second sentence (a
 * goal's situation, a late miss or a colour line), each already filled in. `text` is undefined for
 * an event with no commentary (the added-time board, or one missing its player). */
export interface NarratedEvent {
  event: MatchEvent
  key?: string
  text?: ReactNode[]
  tailKey?: string
  tail?: ReactNode[]
}

// The goal situations, in priority order (spec "Goal situation sentence").
export const GOAL_SITUATIONS = [
  'hatTrick',
  'lateWinner',
  'lateEqualiser',
  'comeback',
  'brace',
  'opener',
  'equaliser',
  'goAhead',
  'rout',
  'extendLead',
  'pullsOneBack',
  'consolation',
] as const
export type GoalSituation = (typeof GOAL_SITUATIONS)[number]

// A goal or miss is late from minute 85 on; second-half added time is stored as minute 90, so it
// counts, while first-half added time (minute 45) does not.
export const LATE_MINUTE = 85
// The share (out of 100) of missed chances and yellow cards that get a colour sentence.
export const MISS_COLOUR_PERCENT = 30
export const YELLOW_COLOUR_PERCENT = 25

/** Picks the situation of a goal from the scorers' point of view: `scored` and `conceded` are the
 * score after the goal, `goals` the scorer's goals so far this match (this one included),
 * `trailed` whether the scoring side was behind at any earlier point. */
export function goalSituation(scored: number, conceded: number, goals: number, minute: number, trailed: boolean): GoalSituation {
  const late = minute >= LATE_MINUTE
  if (goals >= 3) return 'hatTrick'
  if (late && scored === conceded + 1) return 'lateWinner'
  if (late && scored === conceded) return 'lateEqualiser'
  if (scored === conceded + 1 && trailed) return 'comeback'
  if (goals === 2) return 'brace'
  if (scored + conceded === 1) return 'opener'
  if (scored === conceded) return 'equaliser'
  if (scored === conceded + 1) return 'goAhead'
  if (scored - conceded >= 4) return 'rout'
  if (scored - conceded >= 2) return 'extendLead'
  if (conceded - scored === 1) return 'pullsOneBack'
  return 'consolation'
}

// A small, well-mixed 32-bit hash of a few integers and a string, so seeds and rolls spread evenly
// (the plain *31 hash above clusters for neighbouring minutes).
export function mixHash(parts: number[], text = ''): number {
  let h = 0x811c9dc5
  const feed = (value: number) => {
    h ^= value & 0xffffffff
    h = Math.imul(h, 0x01000193)
    h ^= h >>> 15
  }
  for (const part of parts) feed(part)
  for (let i = 0; i < text.length; i++) feed(text.charCodeAt(i))
  h = Math.imul(h ^ (h >>> 16), 0x85ebca6b)
  h = Math.imul(h ^ (h >>> 13), 0xc2b2ae35)
  return (h ^ (h >>> 16)) >>> 0
}

// mulberry32: a tiny seeded PRNG, enough to shuffle a pool of a few dozen lines.
function seededRandom(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let x = Math.imul(a ^ (a >>> 15), 1 | a)
    x = (x + Math.imul(x ^ (x >>> 7), 61 | x)) ^ x
    return ((x ^ (x >>> 14)) >>> 0) / 4294967296
  }
}

function shuffledIndexes(count: number, seed: number): number[] {
  const order = Array.from({ length: count }, (_, i) => i)
  const random = seededRandom(seed)
  for (let i = count - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1))
    ;[order[i], order[j]] = [order[j], order[i]]
  }
  return order
}

/** Hands out a pool's lines for one match: the k-th use of a pool gets the k-th entry of a seeded
 * shuffle of that pool, so every line is used once before any repeats (spec "No repeated lines
 * within a match"). The seed is the match (its teams and kickoff) plus the pool key; nothing else. */
export class LinePicker {
  private readonly uses = new Map<string, number>()
  private readonly orders = new Map<string, number[]>()

  constructor(private readonly teams: Pick<NarrationTeams, 'homeId' | 'awayId' | 'matchKey'>) {}

  next(key: string, count: number): number {
    let order = this.orders.get(key)
    if (!order || order.length !== count) {
      order = shuffledIndexes(count, matchSeed(this.teams, key))
      this.orders.set(key, order)
    }
    const used = this.uses.get(key) ?? 0
    this.uses.set(key, used + 1)
    return order[used % count]
  }
}

// The seed of one match's picks for one pool or roll: its two teams, its kickoff and a label.
function matchSeed(teams: Pick<NarrationTeams, 'homeId' | 'awayId' | 'matchKey'>, label: string, parts: number[] = []): number {
  return mixHash([teams.homeId, teams.awayId, ...parts], `${teams.matchKey ?? ''}|${label}`)
}

/** Whether an event gets a colour sentence: a fixed roll from the match and the event itself, so
 * the same event always gets the same answer (spec "Colour sentences"). */
export function colourRoll(teams: Pick<NarrationTeams, 'homeId' | 'awayId' | 'matchKey'>, event: MatchEvent, percent: number): boolean {
  const roll = matchSeed(teams, 'colour', [
    event.minute,
    event.addedMinute ?? 0,
    event.eventType,
    event.teamOwnerPlayers?.[0] ?? 0,
    event.teamOwner,
  ])
  return roll % 100 < percent
}

function poolOf(t: Translate, key: string): string[] | undefined {
  const lines = t(key, { returnObjects: true })
  return Array.isArray(lines) && lines.length > 0 ? (lines as string[]) : undefined
}

function scoreText(event: Pick<MatchEvent, 'homeGoals' | 'awayGoals'>): string {
  return `${event.homeGoals}-${event.awayGoals}`
}

// The pools red cards and injuries use. Those events are hidden while a match is live, so they
// must only ever touch these pools (and no shared state), or hiding them would change the lines of
// the events around them (spec "Commentary stays the same from live to finished").
const SPOILER_EVENT_TYPES = new Set<number>([EVENT_TYPE_RED_CARD, EVENT_TYPE_INJURY])

/** Narrates a whole match: `events` in any order (they are put in match order here) and every one
 * of them, including the ones the report collapses, so collapsing never changes a line. Returns one
 * entry per event, in match order. */
export function narrateMatch(events: MatchEvent[], teams: NarrationTeams, t: Translate): NarratedEvent[] {
  const ordered = [...events].sort(compareMatchMinute)
  const picker = new LinePicker(teams)
  const playerGoals = new Map<number, number>()
  const bookedPlayers = new Set<number>()
  const trailed = { home: false, away: false }

  return ordered.map((event): NarratedEvent => {
    if (event.eventType === EVENT_TYPE_ADDED_TIME) {
      return { event }
    }
    const isHome = event.teamOwner === teams.homeId
    const teamId = isHome ? teams.homeId : teams.awayId
    const teamName = isHome ? teams.homeName : teams.awayName
    const opponentId = isHome ? teams.awayId : teams.homeId
    const opponentName = isHome ? teams.awayName : teams.homeName
    const playerId = event.teamOwnerPlayers?.[0]
    const scored = isHome ? event.homeGoals : event.awayGoals
    const conceded = isHome ? event.awayGoals : event.homeGoals

    // Main line pool: the event's own, except a red card for a player already booked.
    let key = commentaryKey(event)
    if (event.eventType === EVENT_TYPE_RED_CARD && playerId !== undefined && bookedPlayers.has(playerId)) {
      key = 'match.commentary.secondYellow'
    }

    // Situation and tail pool (state updates happen after, so a goal only sees what came before).
    let tailKey: string | undefined
    let goals = 0
    if (event.eventType === EVENT_TYPE_GOAL) {
      goals = (playerId === undefined ? 0 : playerGoals.get(playerId) ?? 0) + 1
      const situation = goalSituation(scored, conceded, goals, event.minute, isHome ? trailed.home : trailed.away)
      tailKey = `match.commentary.situation.${situation}`
    } else if (event.eventType === EVENT_TYPE_NO_GOAL) {
      if (event.minute >= LATE_MINUTE && (conceded - scored === 0 || conceded - scored === 1)) {
        tailKey = 'match.commentary.lateMiss'
      } else if (colourRoll(teams, event, MISS_COLOUR_PERCENT)) {
        tailKey = 'match.commentary.colour.miss'
      }
    } else if (event.eventType === EVENT_TYPE_YELLOW_CARD && colourRoll(teams, event, YELLOW_COLOUR_PERCENT)) {
      tailKey = 'match.commentary.colour.yellowCard'
    }

    // Shared state - never touched by the events hidden while live.
    if (!SPOILER_EVENT_TYPES.has(event.eventType)) {
      if (event.eventType === EVENT_TYPE_GOAL) {
        if (playerId !== undefined) playerGoals.set(playerId, goals)
        if (event.homeGoals < event.awayGoals) trailed.home = true
        if (event.awayGoals < event.homeGoals) trailed.away = true
      }
      if (event.eventType === EVENT_TYPE_YELLOW_CARD && playerId !== undefined) {
        bookedPlayers.add(playerId)
      }
    }

    const narration = key
      ? renderNarration(event, key, tailKey, picker, t, { teamId, teamName, opponentId, opponentName, goals })
      : undefined
    if (!narration) {
      return { event, key }
    }
    return { event, key, text: narration.text, tailKey: narration.tail ? tailKey : undefined, tail: narration.tail }
  })
}

function renderNarration(
  event: MatchEvent,
  key: string,
  tailKey: string | undefined,
  picker: LinePicker,
  t: Translate,
  side: { teamId: number; teamName: string; opponentId: number; opponentName: string; goals: number },
): { text: ReactNode[]; tail?: ReactNode[] } | undefined {
  const pool = poolOf(t, key)
  if (!pool) return undefined

  const team: CommentaryVar = { text: side.teamName, teamId: side.teamId }
  const opponent: CommentaryVar = { text: side.opponentName, teamId: side.opponentId }
  const playerIds = event.teamOwnerPlayers ?? []
  const playerNames = event.teamOwnerPlayerNames ?? []
  let vars: Record<string, CommentaryVar>

  if (event.eventType === EVENT_TYPE_SUBSTITUTION) {
    if (playerNames.length < 2) return undefined
    vars = {
      team,
      opponent,
      outPlayer: { text: playerNames[0], playerId: playerIds[0] },
      inPlayer: { text: playerNames[1], playerId: playerIds[1] },
    }
  } else if (
    event.eventType === EVENT_TYPE_MATCH_ABANDONED ||
    event.eventType === EVENT_TYPE_MATCH_DEFAULT_RESULT ||
    event.eventType === EVENT_TYPE_MENTALITY_CHANGE
  ) {
    vars = { team, opponent }
  } else {
    if (!playerNames[0]) return undefined
    const keeperName = event.otherTeamPlayerNames?.[0]
    vars = {
      player: { text: playerNames[0], playerId: playerIds[0] },
      team,
      opponent,
      keeper: keeperName
        ? { text: keeperName, playerId: event.otherTeamPlayers?.[0] }
        : { text: String(t('match.setPiece.theKeeper')) },
      score: { text: scoreText(event) },
      goals: { text: String(side.goals) },
    }
  }

  const text = interpolateNodes(pool[picker.next(key, pool.length)], vars)
  const tailPool = tailKey ? poolOf(t, tailKey) : undefined
  const tail = tailKey && tailPool ? interpolateNodes(tailPool[picker.next(tailKey, tailPool.length)], vars) : undefined
  return { text, tail }
}

// ---------------------------------------------------------------------------------------------
// Opening, half-time and full-time lines (spec "Kickoff line", "Full-time line by result",
// match-viewer "Half-time line in the chronicle"). Each is used once per match, so its line is a
// plain seeded pick from the match.
// ---------------------------------------------------------------------------------------------

export type HalfTimeCase = 'goalless' | 'level' | 'homeLeads' | 'awayLeads'
export type FullTimeCase = 'goallessDraw' | 'scoringDraw' | 'homeWin' | 'awayWin' | 'bigWin'

export function halfTimeCase(homeGoals: number, awayGoals: number): HalfTimeCase {
  if (homeGoals === 0 && awayGoals === 0) return 'goalless'
  if (homeGoals === awayGoals) return 'level'
  return homeGoals > awayGoals ? 'homeLeads' : 'awayLeads'
}

export function fullTimeCase(homeGoals: number, awayGoals: number): FullTimeCase {
  if (homeGoals === awayGoals) return homeGoals === 0 ? 'goallessDraw' : 'scoringDraw'
  if (Math.abs(homeGoals - awayGoals) >= 3) return 'bigWin'
  return homeGoals > awayGoals ? 'homeWin' : 'awayWin'
}

function bookendLine(t: Translate, key: string, teams: NarrationTeams, vars: Record<string, CommentaryVar>): ReactNode[] | undefined {
  const pool = poolOf(t, key)
  if (!pool) return undefined
  return interpolateNodes(pool[matchSeed(teams, key) % pool.length], vars)
}

function teamVars(teams: NarrationTeams): Record<string, CommentaryVar> {
  return {
    home: { text: teams.homeName, teamId: teams.homeId },
    away: { text: teams.awayName, teamId: teams.awayId },
  }
}

export function kickoffLine(t: Translate, teams: NarrationTeams): ReactNode[] | undefined {
  return bookendLine(t, 'match.report.kickoff', teams, teamVars(teams))
}

export function halfTimeLine(t: Translate, teams: NarrationTeams, homeGoals: number, awayGoals: number): ReactNode[] | undefined {
  const key = `match.report.halfTime.${halfTimeCase(homeGoals, awayGoals)}`
  return bookendLine(t, key, teams, { ...teamVars(teams), score: { text: `${homeGoals}-${awayGoals}` } })
}

export function fullTimeLine(t: Translate, teams: NarrationTeams, homeGoals: number, awayGoals: number): ReactNode[] | undefined {
  const key = `match.report.fullTimeResult.${fullTimeCase(homeGoals, awayGoals)}`
  const homeWon = homeGoals > awayGoals
  const winner: CommentaryVar = homeWon ? { text: teams.homeName, teamId: teams.homeId } : { text: teams.awayName, teamId: teams.awayId }
  const loser: CommentaryVar = homeWon ? { text: teams.awayName, teamId: teams.awayId } : { text: teams.homeName, teamId: teams.homeId }
  return bookendLine(t, key, teams, { ...teamVars(teams), score: { text: `${homeGoals}-${awayGoals}` }, winner, loser })
}
