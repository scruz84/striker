import { describe, expect, it } from 'vitest'
import type { ClassificationPosition, LeagueLiveScore, WeekMatch } from '../../api/types'
import { hideStillLiveResults } from './DivisionDetailsPage'
import { applyLiveScores, compareClassification } from './liveTable'

const row = (overrides: Partial<ClassificationPosition>): ClassificationPosition => ({
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
})

const minutesAgo = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString()

function weekMatch(overrides: Partial<WeekMatch>): WeekMatch {
  return {
    matchId: 1,
    divisionId: 7,
    season: 1,
    weekNumber: 12,
    matchDate: minutesAgo(40),
    played: true,
    inSimulation: false,
    home: 1,
    homeName: 'Home',
    homeStadiumName: '',
    away: 2,
    awayName: 'Away',
    homeGoals: 3,
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

function entry(match: WeekMatch, overrides: Partial<LeagueLiveScore> = {}): LeagueLiveScore {
  return {
    matchId: match.matchId,
    weekNumber: match.weekNumber,
    matchDate: match.matchDate,
    home: match.home,
    homeName: match.homeName,
    away: match.away,
    awayName: match.awayName,
    status: 'live',
    minute: 70,
    homeGoals: 0,
    awayGoals: 0,
    ...overrides,
  }
}

const byId = (...entries: LeagueLiveScore[]) => new Map(entries.map((e) => [e.matchId, e]))
const order = (positions: ClassificationPosition[]) => positions.map((r) => r.teamId)

// Before the matchday: team 4 leads, B (team 2) is 4th on 21 and A (team 1) 5th on 20.
function preMatchday(): ClassificationPosition[] {
  return [
    row({ teamId: 3, position: 1, points: 30, matches: 11, goalsScored: 20, goalsReceived: 5 }),
    row({ teamId: 4, position: 2, points: 28, matches: 11, goalsScored: 18, goalsReceived: 6 }),
    row({ teamId: 5, position: 3, points: 25, matches: 11, goalsScored: 15, goalsReceived: 8 }),
    row({ teamId: 2, position: 4, points: 21, matches: 11, goalsScored: 12, goalsReceived: 10 }),
    row({ teamId: 1, position: 5, points: 20, matches: 11, goalsScored: 10, goalsReceived: 10 }),
    row({ teamId: 6, position: 6, points: 10, matches: 11, goalsScored: 5, goalsReceived: 20 }),
  ]
}

describe('applyLiveScores', () => {
  // The backend's table already counts the stored results (A 3-0 over C, B 0-0 with D): the page
  // reverts them first, then the live table adds the broadcast scores.
  const aMatch = weekMatch({ matchId: 10, home: 1, away: 6, homeGoals: 3, awayGoals: 0 })
  const bMatch = weekMatch({ matchId: 11, home: 2, away: 5, homeGoals: 0, awayGoals: 0 })
  const stored = preMatchday().map((r) =>
    r.teamId === 1
      ? { ...r, points: 23, matches: 12, wins: 1, goalsScored: 13 }
      : r.teamId === 6
        ? { ...r, matches: 12, losses: 1, goalsReceived: 23 }
        : r.teamId === 2
          ? { ...r, points: 22, matches: 12, draws: 1 }
          : r.teamId === 5
            ? { ...r, points: 26, matches: 12, draws: 1 }
            : r,
  )
  const reverted = () => hideStillLiveResults(stored, [aMatch, bMatch])

  it('moves a team up on a live win (Live win moves a team up)', () => {
    // A leads 1-0 at minute 70 while B draws 0-0 with team 5.
    const live = byId(entry(aMatch, { homeGoals: 1, awayGoals: 0 }), entry(bMatch, { homeGoals: 0, awayGoals: 0 }))
    const result = applyLiveScores(reverted(), [aMatch, bMatch], live)

    const a = result.positions.find((r) => r.teamId === 1)!
    const b = result.positions.find((r) => r.teamId === 2)!
    expect(a).toMatchObject({ points: 23, position: 4, matches: 12, wins: 1, goalsScored: 11, goalsReceived: 10 })
    expect(b).toMatchObject({ points: 22, position: 5, matches: 12, draws: 1 })
    expect(result.deltas.get(1)).toBe(1)
    expect(result.deltas.get(2)).toBe(-1)
    expect(result.playing).toEqual(new Set([1, 6, 2, 5]))
  })

  it('counts a draw as one point each', () => {
    const live = byId(entry(aMatch, { homeGoals: 2, awayGoals: 2 }))
    const result = applyLiveScores(reverted(), [aMatch, bMatch], live)
    expect(result.positions.find((r) => r.teamId === 1)).toMatchObject({ points: 21, draws: 1, wins: 0 })
    expect(result.positions.find((r) => r.teamId === 6)).toMatchObject({ points: 11, draws: 1 })
  })

  it('adds nothing for a pending entry (Pending match adds nothing)', () => {
    const live = byId(entry(aMatch, { status: 'pending', minute: 0 }), entry(bMatch, { status: 'pending', minute: 0 }))
    const base = reverted()
    const result = applyLiveScores(base, [aMatch, bMatch], live)
    expect(result.positions).toEqual(base)
    expect(result.deltas.size).toBe(0)
    expect(result.playing.size).toBe(0)
  })

  it('ignores an entry of a match that is not among the reverted ones', () => {
    const stray = weekMatch({ matchId: 99, weekNumber: 3, home: 1, away: 2, matchDate: '2020-01-01T00:00:00Z' })
    const base = reverted()
    const result = applyLiveScores(base, [aMatch, bMatch], byId(entry(stray, { homeGoals: 5, awayGoals: 0 })))
    expect(result.positions).toEqual(base)
  })

  it('counts a finished match but only marks live and half-time teams as playing', () => {
    const live = byId(entry(aMatch, { status: 'finished', minute: 90, homeGoals: 1, awayGoals: 0 }), entry(bMatch, { status: 'halftime', minute: 45 }))
    const result = applyLiveScores(reverted(), [aMatch, bMatch], live)
    expect(result.positions.find((r) => r.teamId === 1)).toMatchObject({ points: 23 })
    expect(result.playing).toEqual(new Set([2, 5]))
  })

  it('breaks exact ties like hideStillLiveResults: starting order, then team id', () => {
    const tied = [
      row({ teamId: 9, position: 1, seed: null }),
      row({ teamId: 8, position: 2, seed: 2 }),
      row({ teamId: 7, position: 3, seed: 1 }),
      row({ teamId: 6, position: 4, seed: 1 }),
    ]
    const result = applyLiveScores(tied, [], new Map())
    expect(order(result.positions)).toEqual([6, 7, 8, 9])
    expect(order(result.positions)).toEqual(order([...tied].sort(compareClassification)))
  })
})
