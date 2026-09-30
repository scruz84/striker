import { describe, expect, it } from 'vitest'
import type { Lineup, WeekMatch } from '../../api/types'
import { resolveLoadOutcome } from './LoadPreviousLineupControl'

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
