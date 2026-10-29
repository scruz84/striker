import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_MATCH_ABANDONED,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_YELLOW_CARD,
  type Lineup,
  type MatchEvent,
  type MatchInstant,
  type PlayerContribution,
  type PlayerRating,
} from '../api/types'

// A line-by-line port of backend common.ComputePlayerRatings (matchratings.go) - the formula the
// engine stores each match's final ratings with. This copy rebuilds them as of any earlier minute
// (the scrubber) and while a match is still being broadcast, so it must give the very same numbers:
// both are pinned by the shared fixture frontend/src/utils/testdata/match_ratings_parity.json (see
// liveMatchRatings.test.ts). Change one, change the other and regenerate the fixture.
const RATING_BASE = 6.0
const RATING_MIN = 1.0
const RATING_MAX = 10.0
const RATING_GOAL = 1.4
const RATING_YELLOW_CARD = -0.4
const RATING_RED_CARD = -1.5
const RATING_INJURY = -0.2
const RATING_RESULT_BONUS = 0.3
const RATING_ZONE_WEIGHT = 3.0
const RATING_ZONE_CAP = 1.5
// Fewer minutes than this and a player is "s.c." (sin calificar) - see PlayerRating.rated.
export const MATCH_RATING_MIN_MINUTES = 10
const RATING_FULL_MATCH = 90

/** Rounds half away from zero to one decimal, exactly like the backend's roundTenth (Math.round
 * alone rounds negative halves towards zero), and never returns -0. */
export function roundTenth(value: number): number {
  const scaled = value * 10
  const rounded = (Math.sign(scaled) * Math.round(Math.abs(scaled))) / 10
  return rounded === 0 ? 0 : rounded
}

interface Participant {
  teamId: number
  positionMinutes: Map<number, number>
  positionOrder: number[]
  firstPosition: number
  minutes: number
  edgeSum: number
  ownSum: number
  rivalSum: number
  hasContribution: boolean
  cameOn?: number
  wentOff?: number
  goals: number
  yellows: number
  reds: number
  injuries: number
}

/** Rates every player who featured in a match as of `upToMinute` - see the backend's
 * ComputePlayerRatings doc comment for the rules. Returns null when nothing can be rated (no
 * contributions and a missing lineup), otherwise the ratings sorted by player id. */
export function computePlayerRatings(
  homeTeamId: number,
  awayTeamId: number,
  homeLineup: Lineup | null,
  awayLineup: Lineup | null,
  instants: MatchInstant[],
  events: MatchEvent[],
  upToMinute: number,
): PlayerRating[] | null {
  const participants = new Map<number, Participant>()
  const participant = (playerId: number, teamId: number): Participant => {
    let p = participants.get(playerId)
    if (!p) {
      p = {
        teamId,
        positionMinutes: new Map(),
        positionOrder: [],
        firstPosition: -1,
        minutes: 0,
        edgeSum: 0,
        ownSum: 0,
        rivalSum: 0,
        hasContribution: false,
        goals: 0,
        yellows: 0,
        reds: 0,
        injuries: 0,
      }
      participants.set(playerId, p)
    }
    return p
  }

  let elapsed = 0
  const withContributions = instants.some((instant) => instant.minute <= upToMinute && instant.playerContributions != null)

  if (withContributions) {
    for (const instant of instants) {
      if (instant.minute > upToMinute || instant.playerContributions == null) {
        continue
      }
      elapsed++
      const sides: [number, Record<string, PlayerContribution> | undefined][] = [
        [homeTeamId, instant.playerContributions.h],
        [awayTeamId, instant.playerContributions.a],
      ]
      for (const [teamId, contributions] of sides) {
        for (const [key, c] of Object.entries(contributions ?? {})) {
          const p = participant(Number(key), teamId)
          const own = roundTenth(c.o)
          const rival = roundTenth(c.r)
          const edge = own + rival > 0 ? (own - rival) / (own + rival) : 0
          p.minutes++
          p.edgeSum += edge
          p.ownSum += own
          p.rivalSum += rival
          p.hasContribution = true
          if (p.firstPosition < 0) {
            p.firstPosition = c.p
          }
          if (!p.positionMinutes.has(c.p)) {
            p.positionOrder.push(c.p)
          }
          p.positionMinutes.set(c.p, (p.positionMinutes.get(c.p) ?? 0) + 1)
        }
      }
    }
  } else {
    if (!homeLineup || !awayLineup) {
      return null
    }
    for (const lineup of [homeLineup, awayLineup]) {
      for (const pos of lineup.positions) {
        participant(pos.playerId, lineup.teamId).firstPosition = pos.position
      }
    }
    elapsed = upToMinute
    for (const event of events) {
      if (event.minute <= upToMinute && event.eventType === EVENT_TYPE_MATCH_ABANDONED && event.minute < elapsed) {
        elapsed = event.minute
      }
    }
    if (elapsed > RATING_FULL_MATCH) {
      elapsed = RATING_FULL_MATCH
    }
  }

  let homeGoals = 0
  let awayGoals = 0
  for (const event of events) {
    if (event.minute > upToMinute) {
      continue
    }
    const players = event.teamOwnerPlayers ?? []
    if (event.eventType === EVENT_TYPE_GOAL || event.eventType === EVENT_TYPE_MATCH_ABANDONED) {
      homeGoals = event.homeGoals
      awayGoals = event.awayGoals
    } else if (event.eventType === EVENT_TYPE_SUBSTITUTION) {
      if (players.length !== 2) {
        continue
      }
      const out = participant(players[0], event.teamOwner)
      const into = participant(players[1], event.teamOwner)
      if (out.wentOff === undefined) {
        out.wentOff = event.minute
      }
      if (into.cameOn === undefined) {
        into.cameOn = event.minute
      }
      if (into.firstPosition < 0) {
        into.firstPosition = out.firstPosition
      }
    }
    let credit: ((p: Participant) => void) | undefined
    switch (event.eventType) {
      case EVENT_TYPE_GOAL:
        credit = (p) => p.goals++
        break
      case EVENT_TYPE_YELLOW_CARD:
        credit = (p) => p.yellows++
        break
      case EVENT_TYPE_RED_CARD:
        credit = (p) => p.reds++
        break
      case EVENT_TYPE_INJURY:
        credit = (p) => p.injuries++
        break
    }
    if (!credit) {
      continue
    }
    for (const playerId of players) {
      const p = participant(playerId, event.teamOwner)
      credit(p)
      if ((event.eventType === EVENT_TYPE_RED_CARD || event.eventType === EVENT_TYPE_INJURY) && p.wentOff === undefined) {
        p.wentOff = event.minute
      }
    }
  }

  if (!withContributions) {
    for (const p of participants.values()) {
      const start = p.cameOn ?? 0
      let end = elapsed
      if (p.wentOff !== undefined && p.wentOff < end) {
        end = p.wentOff
      }
      if (end > start) {
        p.minutes = end - start
      }
    }
  }

  const threshold = Math.min(MATCH_RATING_MIN_MINUTES, elapsed)

  const ratings: PlayerRating[] = []
  for (const [playerId, p] of participants) {
    const r: PlayerRating = {
      playerId,
      teamId: p.teamId,
      rated: p.minutes >= threshold,
      rating: 0,
      minutes: p.minutes,
      position: mostPlayedPosition(p),
      zone: 0,
      goals: roundTenth(p.goals * RATING_GOAL),
      cards: roundTenth(p.yellows * RATING_YELLOW_CARD + p.reds * RATING_RED_CARD),
      injury: roundTenth(p.injuries * RATING_INJURY),
      result: 0,
    }
    if (p.cameOn !== undefined) {
      r.cameOn = p.cameOn
    }
    if (p.wentOff !== undefined) {
      r.wentOff = p.wentOff
    }
    if (p.hasContribution) {
      r.own = roundTenth(p.ownSum / p.minutes)
      r.rival = roundTenth(p.rivalSum / p.minutes)
      const zone = (RATING_ZONE_WEIGHT * p.edgeSum) / p.minutes
      r.zone = roundTenth(Math.max(-RATING_ZONE_CAP, Math.min(RATING_ZONE_CAP, zone)))
    }
    const minutes = Math.min(p.minutes, RATING_FULL_MATCH)
    const isHome = p.teamId === homeTeamId
    if ((isHome && homeGoals > awayGoals) || (!isHome && awayGoals > homeGoals)) {
      r.result = roundTenth((RATING_RESULT_BONUS * minutes) / RATING_FULL_MATCH)
    } else if ((isHome && awayGoals > homeGoals) || (!isHome && homeGoals > awayGoals)) {
      r.result = roundTenth((-RATING_RESULT_BONUS * minutes) / RATING_FULL_MATCH)
    }
    const rating = roundTenth(RATING_BASE + r.zone + r.goals + r.cards + r.injury + r.result)
    r.rating = Math.max(RATING_MIN, Math.min(RATING_MAX, rating))
    ratings.push(r)
  }
  ratings.sort((a, b) => a.playerId - b.playerId)
  return ratings
}

function mostPlayedPosition(p: Participant): number {
  let best = p.firstPosition
  let bestMinutes = 0
  for (const position of p.positionOrder) {
    const minutes = p.positionMinutes.get(position) ?? 0
    if (minutes > bestMinutes) {
      best = position
      bestMinutes = minutes
    }
  }
  return best < 0 ? 0 : best
}

/** The XI a side kicked off with: a finished league match's lineup comes back with every
 * substitution already replayed onto it (see the backend's WeekMatchById), so this undoes them,
 * last first. A lineup that already is the starting XI (a cup tie, a friendly, the live
 * "lineups" message) comes back unchanged, since none of its players came on. */
export function startingLineup(lineup: Lineup | null, events: MatchEvent[]): Lineup | null {
  if (!lineup) {
    return null
  }
  const positions = lineup.positions.map((pos) => ({ ...pos }))
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i]
    const players = event.teamOwnerPlayers ?? []
    if (event.eventType !== EVENT_TYPE_SUBSTITUTION || event.teamOwner !== lineup.teamId || players.length !== 2) {
      continue
    }
    const slot = positions.find((pos) => pos.playerId === players[1])
    if (slot) {
      slot.playerId = players[0]
    }
  }
  return { ...lineup, positions }
}
