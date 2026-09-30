import { describe, expect, it } from 'vitest'
import type { ClassificationPosition, DivisionDetails, WeekMatch } from '../../api/types'
import { positionOf } from './MatchDetailsPage'

function position(overrides: Partial<ClassificationPosition>): ClassificationPosition {
  return {
    position: 1,
    teamId: 1,
    teamName: 'Team',
    points: 0,
    matches: 0,
    wins: 0,
    draws: 0,
    losses: 0,
    goalsScored: 0,
    goalsReceived: 0,
    ...overrides,
  }
}

function match(overrides: Partial<WeekMatch>): WeekMatch {
  return {
    matchId: 1,
    divisionId: 1,
    season: 1,
    weekNumber: 1,
    matchDate: new Date().toISOString(),
    played: true,
    inSimulation: false,
    home: 1,
    homeName: 'Home',
    homeStadiumName: '',
    away: 2,
    awayName: 'Away',
    homeGoals: 0,
    awayGoals: 0,
    events: null,
    matchInstants: null,
    homeLineup: null,
    awayLineup: null,
    homePlayers: null,
    awayPlayers: null,
    ...overrides,
  }
}

function division(overrides: Partial<DivisionDetails>): DivisionDetails {
  return {
    id: 1,
    season: 1,
    name: 'Division',
    classification: { positions: [] },
    previousWeekMatches: [],
    nextWeekMatches: [],
    ...overrides,
  }
}

describe('positionOf', () => {
  it('returns undefined while the division has not loaded', () => {
    expect(positionOf(undefined, 1)).toBeUndefined()
  })

  it('reports the settled position once the match is no longer still-live', () => {
    const div = division({
      classification: {
        positions: [
          position({ teamId: 1, position: 1, points: 13, matches: 5, wins: 4, goalsScored: 10, goalsReceived: 3 }),
          position({ teamId: 2, position: 2, points: 10 }),
        ],
      },
      previousWeekMatches: [match({ home: 1, away: 2, matchDate: '2020-01-01T00:00:00Z' })],
    })

    expect(positionOf(div, 1)).toBe(1)
  })

  it('hides this very match own effect on the standings while it is still within the live-broadcast window', () => {
    // Team 1's win over team 2, just simulated, is the only thing separating them - viewing that
    // match's own page (this very still-live match) should not already reveal it moved team 1
    // ahead of team 2, well before the live broadcast shown on the same page finishes.
    const div = division({
      classification: {
        positions: [
          position({ teamId: 1, position: 1, points: 13, matches: 5, wins: 4, goalsScored: 10, goalsReceived: 3 }),
          position({ teamId: 2, position: 2, points: 12, matches: 5, wins: 4, goalsScored: 9, goalsReceived: 4 }),
        ],
      },
      previousWeekMatches: [match({ home: 1, away: 2, homeGoals: 1, awayGoals: 0 })],
    })

    expect(positionOf(div, 1)).toBe(2)
    expect(positionOf(div, 2)).toBe(1)
  })
})
