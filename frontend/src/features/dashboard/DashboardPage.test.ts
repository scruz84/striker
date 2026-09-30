import { describe, expect, it } from 'vitest'
import type { ClassificationPosition, DivisionDetails, WeekMatch } from '../../api/types'
import { currentStanding } from './DashboardPage'

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

describe('currentStanding', () => {
  it('returns undefined while the division has not loaded', () => {
    expect(currentStanding(undefined, 1)).toBeUndefined()
  })

  it('returns undefined for a team not in the division', () => {
    const div = division({ classification: { positions: [position({ teamId: 2 })] } })
    expect(currentStanding(div, 1)).toBeUndefined()
  })

  it('reverts the team row while its match is still within the live-broadcast window', () => {
    const div = division({
      classification: {
        positions: [
          position({ teamId: 1, points: 13, matches: 5, wins: 4, draws: 1, losses: 0, goalsScored: 10, goalsReceived: 3 }),
        ],
      },
      previousWeekMatches: [match({ home: 1, away: 2, homeGoals: 2, awayGoals: 0 })],
    })

    const standing = currentStanding(div, 1)

    expect(standing?.points).toBe(10)
    expect(standing?.matches).toBe(4)
    expect(standing?.wins).toBe(3)
  })

  it('leaves the team row untouched once its match is no longer still-live', () => {
    const div = division({
      classification: { positions: [position({ teamId: 1, points: 13 })] },
      previousWeekMatches: [match({ home: 1, away: 2, matchDate: '2020-01-01T00:00:00Z' })],
    })

    expect(currentStanding(div, 1)?.points).toBe(13)
  })
})
