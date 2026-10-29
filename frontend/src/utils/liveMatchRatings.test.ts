import { describe, expect, it } from 'vitest'
import { EVENT_TYPE_SUBSTITUTION, type Lineup, type MatchEvent, type MatchInstant, type PlayerRating } from '../api/types'
import parityFixture from './testdata/match_ratings_parity.json'
import { computePlayerRatings, roundTenth, startingLineup } from './liveMatchRatings'

// The fixture the backend's TestComputePlayerRatings_ParityFixture also asserts: the same
// matches must give the same ratings, record for record, in Go and here.
interface ParityCase {
  name: string
  homeTeamId: number
  awayTeamId: number
  homeLineup: Lineup | null
  awayLineup: Lineup | null
  instants: MatchInstant[] | null
  events: MatchEvent[]
  expected: { minute: number; ratings: PlayerRating[] | null }[]
}

const cases = (parityFixture as unknown as { cases: ParityCase[] }).cases

describe('computePlayerRatings', () => {
  it('has fixture cases to check', () => {
    expect(cases.length).toBeGreaterThan(0)
  })

  for (const c of cases) {
    for (const expected of c.expected) {
      it(`matches the backend for "${c.name}" at minute ${expected.minute}`, () => {
        const got = computePlayerRatings(c.homeTeamId, c.awayTeamId, c.homeLineup, c.awayLineup, c.instants ?? [], c.events, expected.minute)
        expect(got).toEqual(expected.ratings)
      })
    }
  }

  it('returns null without contributions or lineups', () => {
    expect(computePlayerRatings(1, 2, null, null, [], [], 90)).toBeNull()
  })
})

describe('roundTenth', () => {
  it('rounds half away from zero, like the backend', () => {
    expect(roundTenth(0.45)).toBe(0.5)
    expect(roundTenth(-0.45)).toBe(-0.5)
    expect(roundTenth(-0.15)).toBe(-0.2)
    expect(roundTenth(2.8000000000000003)).toBe(2.8)
  })

  it('never returns negative zero', () => {
    expect(Object.is(roundTenth(-0.04), 0)).toBe(true)
  })
})

describe('startingLineup', () => {
  const sub = (minute: number, teamOwner: number, out: number, into: number): MatchEvent => ({
    eventType: EVENT_TYPE_SUBSTITUTION,
    minute,
    teamOwner,
    teamOwnerPlayers: [out, into],
    teamOwnerPlayerNames: ['Out', 'In'],
    otherTeam: teamOwner === 1 ? 2 : 1,
    otherTeamPlayers: null,
    otherTeamPlayerNames: null,
    homeGoals: 0,
    awayGoals: 0,
  })
  const lineup = (playerIds: number[]): Lineup => ({
    matchId: 1,
    teamId: 1,
    formation: 442,
    positions: playerIds.map((playerId, i) => ({ playerId, position: i })),
    substitutions: [],
  })

  it('undoes substitutions already replayed onto a lineup, last first', () => {
    // 7 -> 17 at 60', then 17 -> 27 at 80': the lineup now shows 27 in 7's slot.
    const events = [sub(60, 1, 7, 17), sub(80, 1, 17, 27), sub(70, 2, 107, 117)]
    expect(startingLineup(lineup([1, 27, 3]), events)?.positions.map((p) => p.playerId)).toEqual([1, 7, 3])
  })

  it('leaves a starting XI as it is', () => {
    expect(startingLineup(lineup([1, 7, 3]), [sub(60, 1, 7, 17)])?.positions.map((p) => p.playerId)).toEqual([1, 7, 3])
  })
})
