import { describe, expect, it } from 'vitest'
import { MATCH_KIND_CUP, MATCH_KIND_FRIENDLY, MATCH_KIND_LEAGUE } from '../api/types'
import { liveMatchPath } from './matchSummary'

describe('liveMatchPath', () => {
  it('points each competition kind at its own live broadcast', () => {
    expect(liveMatchPath({ kind: MATCH_KIND_LEAGUE, matchId: 7 })).toBe('/v1/teams/match/7/live')
    expect(liveMatchPath({ kind: MATCH_KIND_CUP, matchId: 8 })).toBe('/v1/cups/ties/8/live')
    expect(liveMatchPath({ kind: MATCH_KIND_FRIENDLY, matchId: 9 })).toBe('/v1/friendlies/9/live')
  })
})
