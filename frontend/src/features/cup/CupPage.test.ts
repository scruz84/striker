import { describe, expect, it } from 'vitest'
import type { CupTie } from '../../api/types'
import { cupTiePenaltyWinnerName } from './CupPage'

function tie(overrides: Partial<CupTie>): CupTie {
  return {
    tieId: 1,
    cupId: 1,
    round: 1,
    slotNumber: 1,
    homeTeamId: 1,
    homeTeamName: 'Home',
    awayTeamId: 2,
    awayTeamName: 'Away',
    matchDate: new Date().toISOString(),
    played: true,
    homeGoals: 1,
    awayGoals: 1,
    wentToPenalties: false,
    winnerTeamId: null,
    ...overrides,
  }
}

describe('cupTiePenaltyWinnerName', () => {
  it('names the home team when it won the shootout', () => {
    expect(cupTiePenaltyWinnerName(tie({ wentToPenalties: true, winnerTeamId: 1 }))).toBe('Home')
  })

  it('names the away team when it won the shootout', () => {
    expect(cupTiePenaltyWinnerName(tie({ wentToPenalties: true, winnerTeamId: 2 }))).toBe('Away')
  })

  it('returns undefined for a tie decided in normal time', () => {
    expect(cupTiePenaltyWinnerName(tie({ wentToPenalties: false, winnerTeamId: 1, homeGoals: 2, awayGoals: 0 }))).toBeUndefined()
  })

  it('returns undefined when penalties happened but no winner has been recorded yet', () => {
    expect(cupTiePenaltyWinnerName(tie({ wentToPenalties: true, winnerTeamId: null }))).toBeUndefined()
  })

  it('returns undefined for an unplayed tie', () => {
    expect(cupTiePenaltyWinnerName(tie({ played: false, wentToPenalties: false, winnerTeamId: null }))).toBeUndefined()
  })
})
