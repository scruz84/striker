import { describe, expect, it } from 'vitest'
import { EVENT_TYPE_GOAL, EVENT_TYPE_RED_CARD, type Lineup, type MatchEvent } from '../api/types'
import { computeLiveMatchRatings } from './liveMatchRatings'

function lineup(playerIds: number[]): Lineup {
  return {
    matchId: 1,
    teamId: 1,
    formation: 442,
    positions: playerIds.map((playerId) => ({ playerId, position: 1 })),
    substitutions: [],
  }
}

function goalEvent(minute: number, teamOwner: number, playerId: number): MatchEvent {
  return {
    eventType: EVENT_TYPE_GOAL,
    minute,
    teamOwner,
    teamOwnerPlayers: [playerId],
    teamOwnerPlayerNames: ['Scorer'],
    otherTeam: teamOwner === 1 ? 2 : 1,
    otherTeamPlayers: null,
    otherTeamPlayerNames: null,
    homeGoals: 0,
    awayGoals: 0,
  }
}

describe('computeLiveMatchRatings', () => {
  const home = lineup([1, 2])
  const away = lineup([10, 11])

  it('rates every named player at the base 6.0 with no events and a level score', () => {
    const ratings = computeLiveMatchRatings(home, away, [], 0, 0)
    expect(ratings).toEqual({ '1': 6, '2': 6, '10': 6, '11': 6 })
  })

  it('boosts the scorer and their team, and dents the trailing team', () => {
    const events = [goalEvent(10, 1, 1)]
    const ratings = computeLiveMatchRatings(home, away, events, 1, 0)
    expect(ratings['1']).toBeCloseTo(6 + 1.4 + 0.3)
    expect(ratings['2']).toBeCloseTo(6 + 0.3)
    expect(ratings['10']).toBeCloseTo(6 - 0.3)
    expect(ratings['11']).toBeCloseTo(6 - 0.3)
  })

  it('only applies an event delta to a player actually named in the lineups', () => {
    const events = [goalEvent(10, 1, 999)]
    const ratings = computeLiveMatchRatings(home, away, events, 1, 0)
    expect(ratings['1']).toBeCloseTo(6 + 0.3)
    expect(Object.keys(ratings)).not.toContain('999')
  })

  it('clamps a heavily penalized player to the 1.0 floor', () => {
    const events = Array.from({ length: 5 }, (_, i) => ({ ...goalEvent(i + 1, 1, 1), eventType: EVENT_TYPE_RED_CARD }))
    const ratings = computeLiveMatchRatings(home, away, events, 0, 0)
    expect(ratings['1']).toBe(1.0)
  })
})
