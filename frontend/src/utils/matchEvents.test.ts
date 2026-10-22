import { describe, expect, it } from 'vitest'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_NO_GOAL,
  SET_PIECE_DIRECT_FREE_KICK,
  SET_PIECE_PENALTY,
  type MatchEvent,
} from '../api/types'
import { isMinorMissedChance, isMissedPenalty } from './matchEvents'

function event(eventType: number, setPiece?: number): MatchEvent {
  return {
    eventType,
    minute: 10,
    teamOwner: 1,
    teamOwnerPlayers: [1],
    teamOwnerPlayerNames: ['Berlana'],
    otherTeam: 2,
    otherTeamPlayers: null,
    otherTeamPlayerNames: null,
    homeGoals: 0,
    awayGoals: 0,
    setPiece,
  }
}

describe('missed chances in the report', () => {
  it('always shows a missed penalty, even without the full commentary', () => {
    expect(isMissedPenalty(event(EVENT_TYPE_NO_GOAL, SET_PIECE_PENALTY))).toBe(true)
    expect(isMinorMissedChance(event(EVENT_TYPE_NO_GOAL, SET_PIECE_PENALTY))).toBe(false)
  })

  it('keeps missed free kicks and open-play misses for the full commentary', () => {
    expect(isMinorMissedChance(event(EVENT_TYPE_NO_GOAL, SET_PIECE_DIRECT_FREE_KICK))).toBe(true)
    expect(isMinorMissedChance(event(EVENT_TYPE_NO_GOAL))).toBe(true)
  })

  it('never treats a goal, penalty or not, as a minor missed chance', () => {
    expect(isMinorMissedChance(event(EVENT_TYPE_GOAL, SET_PIECE_PENALTY))).toBe(false)
    expect(isMissedPenalty(event(EVENT_TYPE_GOAL, SET_PIECE_PENALTY))).toBe(false)
  })
})
