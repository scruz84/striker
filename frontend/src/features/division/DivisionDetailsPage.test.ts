import { describe, expect, it } from 'vitest'
import type { ClassificationPosition, WeekMatch } from '../../api/types'
import { hideStillLiveResults, matchdayChip, matchdayPair } from './DivisionDetailsPage'

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
  it('breaks an exact tie by the starting order, then the team id', () => {
    // A still-live 1-1 draw between teams 1 and 2 reverts both to an untouched team 3's line.
    const drawn = { points: 1, matches: 1, draws: 1, goalsScored: 1, goalsReceived: 1 }
    const positions = [
      position({ teamId: 1, position: 1, seed: null, ...drawn }),
      position({ teamId: 2, position: 2, seed: 2, ...drawn }),
      position({ teamId: 3, position: 3, seed: 1 }),
      position({ teamId: 4, position: 4 }),
    ]
    const matches = [match({ home: 1, away: 2, homeGoals: 1, awayGoals: 1 })]

    const result = hideStillLiveResults(positions, matches)
    expect(result.map((row) => row.teamId)).toEqual([3, 2, 1, 4])
    expect(result.map((row) => row.position)).toEqual([1, 2, 3, 4])
  })

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

  // Reproduces the reported bug: a match is simulated (and flips `played` true) up to 15 minutes
  // before its own scheduled matchDate, so a manager checking the table in that window must still
  // see it reverted, exactly as if the match hadn't been simulated yet.
  it('reverts a match simulated minutes before its own kickoff', () => {
    const positions = [
      position({ teamId: 1, position: 1, points: 13, matches: 5, wins: 4, draws: 1, losses: 0, goalsScored: 10, goalsReceived: 3 }),
      position({ teamId: 2, position: 2, points: 10, matches: 5, wins: 3, draws: 1, losses: 1, goalsScored: 8, goalsReceived: 5 }),
    ]
    const matches = [
      match({ home: 1, away: 2, homeGoals: 2, awayGoals: 0, matchDate: new Date(Date.now() + 10 * 60 * 1000).toISOString() }),
    ]

    const result = hideStillLiveResults(positions, matches)

    const team1 = result.find((r) => r.teamId === 1)!
    expect(team1.points).toBe(10)
    expect(team1.matches).toBe(4)
    expect(team1.wins).toBe(3)
  })
})

describe('matchdayPair', () => {
  const pair = (currentWeek: number, requested?: number, totalWeeks = 30) => matchdayPair({ currentWeek, totalWeeks, requested })

  it('opens on the current week and its next one', () => {
    expect(pair(12)).toEqual({ left: 12, isDefault: true })
  })

  it('opens on matchdays 1 and 2 before the first matchday', () => {
    expect(pair(0)).toEqual({ left: 1, isDefault: true })
  })

  it('opens on the last two matchdays once the season is finished', () => {
    expect(pair(30)).toEqual({ left: 29, isDefault: true })
  })

  it('clamps an out-of-range requested week into the season', () => {
    expect(pair(12, 99)).toEqual({ left: 29, isDefault: false })
    expect(pair(12, 0)).toEqual({ left: 1, isDefault: false })
  })

  it('treats no request as the default pair', () => {
    expect(pair(12, undefined)).toEqual({ left: 12, isDefault: true })
  })

  it('is the default pair when the request equals it', () => {
    expect(pair(12, 12).isDefault).toBe(true)
  })

  it('shows matchdays 1 and 2 for a season too short to have two', () => {
    expect(matchdayPair({ currentWeek: 0, totalWeeks: 0 })).toEqual({ left: 1, isDefault: true })
  })
})

describe('matchdayChip', () => {
  it('tags the matchday just played and the one up next', () => {
    expect(matchdayChip(12, 12)).toBe('lastWeek')
    expect(matchdayChip(13, 12)).toBe('nextWeek')
    expect(matchdayChip(11, 12)).toBeUndefined()
  })

  it('tags only the upcoming matchday before the first one', () => {
    expect(matchdayChip(1, 0)).toBe('nextWeek')
    expect(matchdayChip(2, 0)).toBeUndefined()
  })

  it('tags only the last matchday once the season is finished', () => {
    expect(matchdayChip(29, 30)).toBeUndefined()
    expect(matchdayChip(30, 30)).toBe('lastWeek')
  })
})
