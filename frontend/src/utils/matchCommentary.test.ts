import { describe, expect, it } from 'vitest'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_NO_GOAL,
  SET_PIECE_DIRECT_FREE_KICK,
  SET_PIECE_PENALTY,
  type MatchEvent,
} from '../api/types'
import { commentaryFor } from './matchCommentary'

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
