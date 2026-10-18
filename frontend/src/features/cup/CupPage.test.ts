import { describe, expect, it } from 'vitest'
import type { CupTie } from '../../api/types'
import { cupTiePenaltyWinnerName, expectedTotalRounds, roundName } from './CupPage'

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

// A fake translate function good enough for roundName's own keys (cup.final/semifinal/
// quarterfinal/round) without pulling in react-i18next.
const t = (key: string, options?: Record<string, unknown>) => (options ? `${key}:${JSON.stringify(options)}` : key)

describe('expectedTotalRounds', () => {
  it('returns 0 for an empty bracket', () => {
    expect(expectedTotalRounds([])).toBe(0)
  })

  it('derives the total from round 1 entrants for a power-of-two field', () => {
    const ties = [
      tie({ round: 1, slotNumber: 1 }),
      tie({ round: 1, slotNumber: 2 }),
      tie({ round: 1, slotNumber: 3 }),
      tie({ round: 1, slotNumber: 4 }),
    ]
    expect(expectedTotalRounds(ties)).toBe(3) // 8 entrants -> QF, SF, then final
  })

  it('counts a bye slot as a single entrant', () => {
    const ties = [tie({ round: 1, slotNumber: 1, awayTeamId: null, awayTeamName: '' })]
    expect(expectedTotalRounds(ties)).toBe(0)
  })

  // Reproduces the European Cup bracket from the bug report: 9 entrants (one bye in round 1)
  // needs four rounds (9 -> 5 -> 3 -> 2 -> 1) before a real, two-team final is reached - not two,
  // which is what counting "rounds generated so far" would wrongly conclude once only round 1 and
  // round 2 exist.
  it('accounts for the extra rounds an odd field needs due to repeated byes', () => {
    const round1 = [
      tie({ round: 1, slotNumber: 1 }),
      tie({ round: 1, slotNumber: 2 }),
      tie({ round: 1, slotNumber: 3 }),
      tie({ round: 1, slotNumber: 4 }),
      tie({ round: 1, slotNumber: 5, awayTeamId: null, awayTeamName: '' }),
    ]
    expect(expectedTotalRounds(round1)).toBe(4)
  })

  it('ignores later rounds and always derives the total from round 1', () => {
    const ties = [
      tie({ round: 1, slotNumber: 1 }),
      tie({ round: 1, slotNumber: 2 }),
      tie({ round: 1, slotNumber: 3 }),
      tie({ round: 1, slotNumber: 4 }),
      tie({ round: 1, slotNumber: 5, awayTeamId: null, awayTeamName: '' }),
      tie({ round: 2, slotNumber: 1 }),
      tie({ round: 2, slotNumber: 2 }),
      tie({ round: 2, slotNumber: 3, awayTeamId: null, awayTeamName: '' }),
    ]
    expect(expectedTotalRounds(ties)).toBe(4)
  })
})

describe('roundName', () => {
  it('never calls an in-progress round "Final" just because it is the last one generated', () => {
    // With the old totalRounds=rounds.length behavior this round (2 of 2 generated so far) would
    // have been mislabeled "Final" even though it still has a bye advancing alongside two real
    // ties - see expectedTotalRounds' "odd field" test above for the matching bracket shape.
    expect(roundName(t, 2, 4)).toBe('cup.quarterfinal')
  })

  it('labels the true final only once the bracket has actually narrowed to one round left', () => {
    expect(roundName(t, 4, 4)).toBe('cup.final')
  })
})
