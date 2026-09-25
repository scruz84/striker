import { describe, expect, it } from 'vitest'
import type { ClassificationPosition, WeekMatch } from '../../api/types'
import { hideStillLiveResults } from './DivisionDetailsPage'

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

describe('hideStillLiveResults', () => {
  it('leaves classification untouched when no match is still live', () => {
    const positions = [position({ teamId: 1, points: 10 })]
    const matches = [match({ matchDate: '2020-01-01T00:00:00Z' })]

    expect(hideStillLiveResults(positions, matches)).toBe(positions)
  })

  it('reverts a home win still within its broadcast window', () => {
    const positions = [
      position({ teamId: 1, position: 1, points: 13, matches: 5, wins: 4, draws: 1, losses: 0, goalsScored: 10, goalsReceived: 3 }),
      position({ teamId: 2, position: 2, points: 10, matches: 5, wins: 3, draws: 1, losses: 1, goalsScored: 8, goalsReceived: 5 }),
    ]
    const matches = [match({ home: 1, away: 2, homeGoals: 2, awayGoals: 0 })]

    const result = hideStillLiveResults(positions, matches)

    const team1 = result.find((r) => r.teamId === 1)!
    expect(team1.points).toBe(10)
    expect(team1.matches).toBe(4)
    expect(team1.wins).toBe(3)
    expect(team1.goalsScored).toBe(8)
    expect(team1.goalsReceived).toBe(3)
  })

  it('reorders positions once reverting changes the standings', () => {
    const positions = [
      position({ teamId: 1, position: 1, points: 13, matches: 5, wins: 4, goalsScored: 10, goalsReceived: 3 }),
      position({ teamId: 2, position: 2, points: 12, matches: 5, wins: 4, goalsScored: 9, goalsReceived: 4 }),
    ]
    // Team 1's still-live win is the only thing keeping it ahead of team 2 - reverting it should
    // drop team 1 below team 2.
    const matches = [match({ home: 1, away: 3, homeGoals: 1, awayGoals: 0 })]

    const result = hideStillLiveResults(positions, matches)

    expect(result[0].teamId).toBe(2)
    expect(result[0].position).toBe(1)
    expect(result[1].teamId).toBe(1)
    expect(result[1].position).toBe(2)
  })

  it('ignores an unplayed match', () => {
    const positions = [position({ teamId: 1, points: 10 })]
    const matches = [match({ played: false, homeGoals: 0, awayGoals: 0 })]

    expect(hideStillLiveResults(positions, matches)).toEqual(positions)
  })
})
