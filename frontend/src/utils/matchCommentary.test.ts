import { isValidElement, type ReactNode } from 'react'
import { describe, expect, it } from 'vitest'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_YELLOW_CARD,
  SET_PIECE_DIRECT_FREE_KICK,
  SET_PIECE_PENALTY,
  type MatchEvent,
} from '../api/types'
import {
  MISS_COLOUR_PERCENT,
  YELLOW_COLOUR_PERCENT,
  colourRoll,
  commentaryFor,
  fullTimeCase,
  fullTimeLine,
  goalSituation,
  halfTimeCase,
  kickoffLine,
  narrateMatch,
} from './matchCommentary'

function event(eventType: number, setPiece?: number, keeper: string | null = 'Keeper'): MatchEvent {
  return {
    eventType,
    minute: 16,
    teamOwner: 1,
    teamOwnerPlayers: [7],
    teamOwnerPlayerNames: ['Berlana'],
    otherTeam: 2,
    otherTeamPlayers: keeper ? [99] : null,
    otherTeamPlayerNames: keeper ? [keeper] : null,
    homeGoals: 1,
    awayGoals: 0,
    setPiece,
  }
}

// A fake t that records which key was asked for and answers with a single plain-text variant, so
// the rendered text can be read back without a router for the player links.
function fakeT(asked: string[]) {
  return (key: string): unknown => {
    asked.push(key)
    return key === 'match.setPiece.theKeeper' ? 'the goalkeeper' : [`${key}|{{keeper}}`]
  }
}

function keyOf(asked: string[]): string | undefined {
  return asked.find((key) => key.startsWith('match.commentary.'))
}

describe('commentaryFor set pieces', () => {
  it('picks its own phrasing for each set-piece goal and miss', () => {
    const cases: Array<[number, number, string]> = [
      [EVENT_TYPE_GOAL, SET_PIECE_PENALTY, 'match.commentary.penaltyGoal'],
      [EVENT_TYPE_NO_GOAL, SET_PIECE_PENALTY, 'match.commentary.penaltyMiss'],
      [EVENT_TYPE_GOAL, SET_PIECE_DIRECT_FREE_KICK, 'match.commentary.freeKickGoal'],
      [EVENT_TYPE_NO_GOAL, SET_PIECE_DIRECT_FREE_KICK, 'match.commentary.freeKickMiss'],
    ]
    for (const [eventType, setPiece, expectedKey] of cases) {
      const asked: string[] = []
      expect(commentaryFor(event(eventType, setPiece), fakeT(asked), 1, 'Home', 2, 'Away')).toBeDefined()
      expect(keyOf(asked)).toBe(expectedKey)
    }
  })

  it('falls back to open-play phrasing when setPiece is missing or 0', () => {
    for (const setPiece of [undefined, 0]) {
      const goal: string[] = []
      commentaryFor(event(EVENT_TYPE_GOAL, setPiece), fakeT(goal), 1, 'Home', 2, 'Away')
      expect(keyOf(goal)).toBe('match.commentary.goal')

      const miss: string[] = []
      commentaryFor(event(EVENT_TYPE_NO_GOAL, setPiece), fakeT(miss), 1, 'Home', 2, 'Away')
      expect(keyOf(miss)).toBe('match.commentary.noGoal')
    }
  })

  it('names the goalkeeper, or "the goalkeeper" when the event does not store one', () => {
    const withKeeper = commentaryFor(event(EVENT_TYPE_GOAL, SET_PIECE_PENALTY), fakeT([]), 1, 'Home', 2, 'Away')
    expect(withKeeper?.filter((node) => typeof node !== 'string' && node !== '')).toHaveLength(1) // the keeper link

    const withoutKeeper = commentaryFor(event(EVENT_TYPE_GOAL, SET_PIECE_PENALTY, null), fakeT([]), 1, 'Home', 2, 'Away')
    expect(withoutKeeper).toContain('the goalkeeper')
  })
})

// ---------------------------------------------------------------------------------------------
// narrateMatch (openspec match-commentary)
// ---------------------------------------------------------------------------------------------

const TEAMS = { homeId: 1, homeName: 'Home', awayId: 2, awayName: 'Away' }

// Every pool has `size` lines reading "<key>#<index> <placeholders>", so a test can tell which
// pool and line each event got.
function poolT(size = 12) {
  return (key: string): unknown => {
    if (key === 'match.setPiece.theKeeper') return 'the goalkeeper'
    return Array.from({ length: size }, (_, i) => `${key}#${i} [{{player}}|{{keeper}}|{{score}}|{{goals}}]`)
  }
}

interface EventSpec {
  type: number
  minute: number
  side: 'home' | 'away'
  player?: number
  score?: [number, number]
  addedMinute?: number
  keeper?: string
}

function ev({ type, minute, side, player = 7, score = [0, 0], addedMinute, keeper }: EventSpec): MatchEvent {
  return {
    eventType: type,
    minute,
    addedMinute,
    teamOwner: side === 'home' ? 1 : 2,
    teamOwnerPlayers: [player],
    teamOwnerPlayerNames: [`P${player}`],
    otherTeam: side === 'home' ? 2 : 1,
    otherTeamPlayers: keeper ? [900] : null,
    otherTeamPlayerNames: keeper ? [keeper] : null,
    homeGoals: score[0],
    awayGoals: score[1],
  }
}

// Plain text of a narrated line: links become their label.
function plain(nodes: ReactNode[] | undefined): string {
  return (nodes ?? [])
    .map((node) => (isValidElement<{ children?: ReactNode }>(node) ? String(node.props.children) : String(node ?? '')))
    .join('')
}

function lineOf(entry: { text?: ReactNode[]; tail?: ReactNode[] }): string {
  return plain(entry.text) + (entry.tail ? ' ' + plain(entry.tail) : '')
}

function situationOf(entry: { tailKey?: string }): string | undefined {
  return entry.tailKey?.replace('match.commentary.situation.', '')
}

describe('narrateMatch goal situations', () => {
  it('tags a late equaliser rather than a plain equaliser', () => {
    const events = [
      ev({ type: EVENT_TYPE_GOAL, minute: 10, side: 'home', player: 1, score: [1, 0] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 20, side: 'home', player: 2, score: [2, 0] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 60, side: 'away', player: 11, score: [2, 1] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 88, side: 'away', player: 12, score: [2, 2] }),
    ]
    expect(narrateMatch(events, TEAMS, poolT()).map(situationOf)).toEqual(['opener', 'extendLead', 'pullsOneBack', 'lateEqualiser'])
  })

  it('puts a hat-trick above everything else and fills {{goals}}', () => {
    const events = [
      ev({ type: EVENT_TYPE_GOAL, minute: 5, side: 'away', player: 9, score: [0, 1] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 30, side: 'home', player: 3, score: [1, 1] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 40, side: 'home', player: 4, score: [2, 1] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 60, side: 'away', player: 9, score: [2, 2] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 70, side: 'home', player: 5, score: [3, 2] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 90, addedMinute: 2, side: 'away', player: 9, score: [3, 3] }),
    ]
    const narrated = narrateMatch(events, TEAMS, poolT())
    expect(situationOf(narrated[5])).toBe('hatTrick')
    expect(plain(narrated[5].tail)).toContain('|3]')
    expect(situationOf(narrated[3])).toBe('brace')
  })

  it('calls a lead taken after trailing a comeback', () => {
    const events = [
      ev({ type: EVENT_TYPE_GOAL, minute: 10, side: 'away', player: 11, score: [0, 1] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 20, side: 'away', player: 12, score: [0, 2] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 50, side: 'home', player: 1, score: [1, 2] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 60, side: 'home', player: 2, score: [2, 2] }),
      ev({ type: EVENT_TYPE_GOAL, minute: 70, side: 'home', player: 3, score: [3, 2] }),
    ]
    expect(narrateMatch(events, TEAMS, poolT()).map(situationOf)).toEqual(['opener', 'extendLead', 'pullsOneBack', 'equaliser', 'comeback'])
  })

  it('covers the score-based cases', () => {
    expect(goalSituation(1, 0, 1, 10, false)).toBe('opener')
    expect(goalSituation(4, 0, 1, 50, false)).toBe('rout')
    expect(goalSituation(3, 0, 1, 50, false)).toBe('extendLead')
    expect(goalSituation(2, 3, 1, 50, false)).toBe('pullsOneBack')
    expect(goalSituation(1, 3, 1, 50, false)).toBe('consolation')
    expect(goalSituation(2, 1, 1, 50, false)).toBe('goAhead')
    expect(goalSituation(2, 1, 1, 85, false)).toBe('lateWinner')
    expect(goalSituation(1, 0, 1, 88, false)).toBe('lateWinner')
    expect(goalSituation(1, 0, 1, 45, false)).toBe('opener') // first-half added time is not late
  })
})

describe('narrateMatch cards and misses', () => {
  it('narrates a red card after a booking from the second-yellow pool', () => {
    const events = [
      ev({ type: EVENT_TYPE_YELLOW_CARD, minute: 30, side: 'home', player: 4 }),
      ev({ type: EVENT_TYPE_RED_CARD, minute: 61, side: 'home', player: 4 }),
      ev({ type: EVENT_TYPE_RED_CARD, minute: 70, side: 'away', player: 14 }),
    ]
    const narrated = narrateMatch(events, TEAMS, poolT())
    expect(narrated[1].key).toBe('match.commentary.secondYellow')
    expect(narrated[2].key).toBe('match.commentary.redCard')
  })

  it('adds a late-miss sentence only when the side is level or one behind', () => {
    const missedEqualiser = narrateMatch([ev({ type: EVENT_TYPE_NO_GOAL, minute: 89, side: 'home', score: [1, 2] })], TEAMS, poolT())
    expect(missedEqualiser[0].tailKey).toBe('match.commentary.lateMiss')
    const comfortable = narrateMatch([ev({ type: EVENT_TYPE_NO_GOAL, minute: 88, side: 'home', score: [3, 0] })], TEAMS, poolT())
    expect(comfortable[0].tailKey).not.toBe('match.commentary.lateMiss')
    const early = narrateMatch([ev({ type: EVENT_TYPE_NO_GOAL, minute: 84, side: 'home', score: [1, 2] })], TEAMS, poolT())
    expect(early[0].tailKey).not.toBe('match.commentary.lateMiss')
  })

  it('gives about 30% of misses and 25% of bookings a colour sentence', () => {
    const share = (type: number) => {
      let hits = 0
      for (let i = 0; i < 10000; i++) {
        const event = ev({ type, minute: 1 + (i % 84), side: i % 2 ? 'home' : 'away', player: 100 + Math.floor(i / 84) })
        if (colourRoll({ homeId: 1 + (i % 7), awayId: 50 + (i % 13) }, event, type === EVENT_TYPE_NO_GOAL ? MISS_COLOUR_PERCENT : YELLOW_COLOUR_PERCENT)) hits++
      }
      return hits / 10000
    }
    const misses = share(EVENT_TYPE_NO_GOAL)
    expect(misses).toBeGreaterThanOrEqual(0.27)
    expect(misses).toBeLessThanOrEqual(0.33)
    const yellows = share(EVENT_TYPE_YELLOW_CARD)
    expect(yellows).toBeGreaterThanOrEqual(0.22)
    expect(yellows).toBeLessThanOrEqual(0.28)
  })

  it('names the open-play keeper, or "the goalkeeper" for an old event', () => {
    const [withKeeper, withoutKeeper] = narrateMatch(
      [ev({ type: EVENT_TYPE_NO_GOAL, minute: 10, side: 'home', keeper: 'Ruiz' }), ev({ type: EVENT_TYPE_NO_GOAL, minute: 20, side: 'home' })],
      TEAMS,
      poolT(),
    )
    expect(plain(withKeeper.text)).toContain('|Ruiz|')
    expect(plain(withoutKeeper.text)).toContain('|the goalkeeper|')
  })
})

describe('narrateMatch line choice', () => {
  const twelveGoals = Array.from({ length: 12 }, (_, i) =>
    ev({ type: EVENT_TYPE_GOAL, minute: 5 + i * 7, side: i % 2 ? 'away' : 'home', player: 20 + i, score: [Math.ceil((i + 1) / 2), Math.floor((i + 1) / 2)] }),
  )

  it('uses every line of a pool before repeating one', () => {
    const lines = narrateMatch(twelveGoals, TEAMS, poolT(12)).map((entry) => plain(entry.text).split(' ')[0])
    expect(new Set(lines).size).toBe(12)
  })

  it('narrates the same match the same way every time, in any input order', () => {
    const first = narrateMatch(twelveGoals, TEAMS, poolT()).map(lineOf)
    const again = narrateMatch([...twelveGoals].reverse(), TEAMS, poolT()).map(lineOf)
    expect(again).toEqual(first)
  })

  it('narrates a rematch on another date differently', () => {
    const first = narrateMatch(twelveGoals, { ...TEAMS, matchKey: '2026-10-01T18:00:00Z' }, poolT()).map(lineOf)
    const rematch = narrateMatch(twelveGoals, { ...TEAMS, matchKey: '2027-03-01T18:00:00Z' }, poolT()).map(lineOf)
    expect(rematch).not.toEqual(first)
  })

  it('reads the same with or without the red cards and injuries hidden while live', () => {
    const visible = [
      ev({ type: EVENT_TYPE_YELLOW_CARD, minute: 12, side: 'home', player: 4 }),
      ev({ type: EVENT_TYPE_GOAL, minute: 20, side: 'away', player: 9, score: [0, 1] }),
      ev({ type: EVENT_TYPE_NO_GOAL, minute: 25, side: 'home', player: 5 }),
      ev({ type: EVENT_TYPE_SUBSTITUTION, minute: 31, side: 'home', player: 6 }),
      ev({ type: EVENT_TYPE_GOAL, minute: 70, side: 'home', player: 5, score: [1, 1] }),
    ]
    visible[3].teamOwnerPlayers = [6, 16]
    visible[3].teamOwnerPlayerNames = ['P6', 'P16']
    const hidden = [
      ev({ type: EVENT_TYPE_INJURY, minute: 30, side: 'home', player: 6 }),
      ev({ type: EVENT_TYPE_RED_CARD, minute: 50, side: 'home', player: 4 }),
      ev({ type: EVENT_TYPE_RED_CARD, minute: 60, side: 'away', player: 9 }),
    ]
    const live = narrateMatch(visible, TEAMS, poolT(3)).map(lineOf)
    const finished = narrateMatch([...visible, ...hidden], TEAMS, poolT(3))
      .filter((entry) => entry.event.eventType !== EVENT_TYPE_INJURY && entry.event.eventType !== EVENT_TYPE_RED_CARD)
      .map(lineOf)
    expect(finished).toEqual(live)
  })
})

describe('bookend lines', () => {
  it('picks the half-time and full-time case from the score', () => {
    expect(halfTimeCase(0, 0)).toBe('goalless')
    expect(halfTimeCase(1, 1)).toBe('level')
    expect(halfTimeCase(1, 0)).toBe('homeLeads')
    expect(halfTimeCase(0, 2)).toBe('awayLeads')
    expect(fullTimeCase(0, 0)).toBe('goallessDraw')
    expect(fullTimeCase(2, 2)).toBe('scoringDraw')
    expect(fullTimeCase(2, 1)).toBe('homeWin')
    expect(fullTimeCase(1, 3)).toBe('awayWin')
    expect(fullTimeCase(0, 4)).toBe('bigWin')
  })

  it('names the winner of a big win and keeps the kickoff line stable', () => {
    const t = (key: string): unknown => [`${key} {{winner}} {{score}}`]
    expect(plain(fullTimeLine(t, TEAMS, 0, 4))).toBe('match.report.fullTimeResult.bigWin Away 0-4')
    const kickoffT = (key: string): unknown => Array.from({ length: 6 }, (_, i) => `${key}#${i}`)
    expect(plain(kickoffLine(kickoffT, TEAMS))).toBe(plain(kickoffLine(kickoffT, TEAMS)))
  })
})
