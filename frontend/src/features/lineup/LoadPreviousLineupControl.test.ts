import { describe, expect, it } from 'vitest'
import type { Lineup, TeamMatchSummary, WeekMatch } from '../../api/types'
import { resolveLoadOutcome, selectPastLeagueMatches } from './LoadPreviousLineupControl'

const TEAM_ID = 1
const OPPONENT_ID = 2

function lineup(teamId: number): Lineup {
  return { matchId: 10, teamId, formation: 1, positions: [], substitutions: [] }
}

function weekMatch(overrides: Partial<WeekMatch> = {}): WeekMatch {
  return {
    matchId: 10,
    divisionId: 1,
    season: 1,
    weekNumber: 1,
    matchDate: '2026-01-01T00:00:00Z',
    played: true,
    inSimulation: false,
    home: TEAM_ID,
    homeName: 'Home',
    homeStadiumName: 'Stadium',
    away: OPPONENT_ID,
    awayName: 'Away',
    homeGoals: 1,
    awayGoals: 0,
    events: null,
    matchInstants: null,
    homeLineup: lineup(TEAM_ID),
    awayLineup: lineup(OPPONENT_ID),
    homePlayers: null,
    awayPlayers: null,
    ...overrides,
  }
}

describe('resolveLoadOutcome', () => {
  it('waits while still fetching, even if an error or stale data is also present', () => {
    expect(resolveLoadOutcome(TEAM_ID, true, true, weekMatch())).toEqual({ action: 'wait' })
  })

  it('waits while nothing has come back yet', () => {
    expect(resolveLoadOutcome(TEAM_ID, false, false, undefined)).toEqual({ action: 'wait' })
  })

  it('errors on a failed fetch even when stale cached data is still present - the race this exists to prevent', () => {
    // React Query keeps the last-good `data` around through a failed *background* refetch of an
    // already-cached query: isFetching goes back to false, data stays truthy, but so does
    // isError. The stale data must not win, or the pitch/bench would silently change to it while
    // the error banner also shows, with no way to tell whether the load actually happened.
    const got = resolveLoadOutcome(TEAM_ID, false, true, weekMatch())
    expect(got).toEqual({ action: 'error' })
  })

  it('errors when the fetch failed and there is no cached data either', () => {
    expect(resolveLoadOutcome(TEAM_ID, false, true, undefined)).toEqual({ action: 'error' })
  })

  it('applies the home lineup when this team was the home side', () => {
    const match = weekMatch({ home: TEAM_ID, away: OPPONENT_ID })
    expect(resolveLoadOutcome(TEAM_ID, false, false, match)).toEqual({ action: 'apply', lineup: match.homeLineup })
  })

  it('applies the away lineup when this team was the away side', () => {
    const match = weekMatch({ home: OPPONENT_ID, away: TEAM_ID })
    expect(resolveLoadOutcome(TEAM_ID, false, false, match)).toEqual({ action: 'apply', lineup: match.awayLineup })
  })

  it('errors when the match has no lineup on our side (should not normally happen, but the field is nullable)', () => {
    const match = weekMatch({ home: TEAM_ID, away: OPPONENT_ID, homeLineup: null })
    expect(resolveLoadOutcome(TEAM_ID, false, false, match)).toEqual({ action: 'error' })
  })
})

function teamMatch(overrides: Partial<TeamMatchSummary>): TeamMatchSummary {
  return {
    kind: 'league',
    matchId: 1,
    opponentId: 2,
    opponentName: 'Opponent',
    isHome: true,
    matchDate: '2020-01-01T00:00:00Z',
    played: true,
    homeGoals: 0,
    awayGoals: 0,
    ...overrides,
  }
}

describe('selectPastLeagueMatches', () => {
  it('includes a genuinely finished league match', () => {
    const matches = [teamMatch({ matchDate: '2020-01-01T00:00:00Z' })]
    expect(selectPastLeagueMatches(matches, 15)).toHaveLength(1)
  })

  it('excludes a cup tie or friendly', () => {
    const matches = [
      teamMatch({ kind: 'cup', matchDate: '2020-01-01T00:00:00Z' }),
      teamMatch({ kind: 'friendly', matchDate: '2020-01-01T00:00:00Z' }),
    ]
    expect(selectPastLeagueMatches(matches, 15)).toHaveLength(0)
  })

  it('excludes an unplayed match', () => {
    const matches = [teamMatch({ played: false, matchDate: '2020-01-01T00:00:00Z' })]
    expect(selectPastLeagueMatches(matches, 15)).toHaveLength(0)
  })

  it('excludes a match simulated early whose kickoff has not arrived yet', () => {
    // played can flip true up to 15 minutes before matchDate - a manager should not be offered
    // (or shown the score of) a match they haven't seen kick off yet.
    const matches = [teamMatch({ played: true, matchDate: new Date(Date.now() + 5 * 60 * 1000).toISOString() })]
    expect(selectPastLeagueMatches(matches, 15)).toHaveLength(0)
  })

  it('excludes a match still within its live-broadcast window', () => {
    const matches = [teamMatch({ played: true, matchDate: new Date(Date.now() - 30 * 60 * 1000).toISOString() })]
    expect(selectPastLeagueMatches(matches, 15)).toHaveLength(0)
  })

  it('caps the result at the given limit, most recent first', () => {
    const matches = [
      teamMatch({ matchId: 1, matchDate: '2020-01-01T00:00:00Z' }),
      teamMatch({ matchId: 2, matchDate: '2020-01-03T00:00:00Z' }),
      teamMatch({ matchId: 3, matchDate: '2020-01-02T00:00:00Z' }),
    ]
    const result = selectPastLeagueMatches(matches, 2)
    expect(result.map((m) => m.matchId)).toEqual([2, 3])
  })
})
