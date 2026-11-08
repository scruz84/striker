import { describe, expect, it } from 'vitest'
import { clauseValue, isLastSeason, renewalWageFloor } from './contractTerms'

// team-finances "Contract renewal and release": "No free wage cut" and "Last-season veteran".
describe('renewalWageFloor', () => {
  it('keeps the whole wage with 2 or more seasons left', () => {
    expect(renewalWageFloor(2000, 3)).toBe(2000)
    expect(renewalWageFloor(2000, 2)).toBe(2000)
  })

  it('accepts 85% in the last season, rounded up', () => {
    expect(renewalWageFloor(2000, 1)).toBe(1700)
    expect(renewalWageFloor(1001, 1)).toBe(851)
  })
})

// pre-contracts "Last-season marker": "Rival's player" and "Two seasons left".
describe('isLastSeason', () => {
  it('reads the public marker of another club player', () => {
    expect(isLastSeason({ teamId: 4, contractExpiring: true, contractYearsRemaining: 0 })).toBe(true)
    expect(isLastSeason({ teamId: 4, contractYearsRemaining: 0 })).toBe(false)
  })

  it('reads the contract of an own player', () => {
    expect(isLastSeason({ teamId: 4, contractYearsRemaining: 1 })).toBe(true)
    expect(isLastSeason({ teamId: 4, contractYearsRemaining: 2 })).toBe(false)
  })

  it('is false for a free agent', () => {
    expect(isLastSeason({ teamId: 0, contractExpiring: true, contractYearsRemaining: 1 })).toBe(false)
  })
})

// release-clauses "Setting a clause": an empty field sends no clause.
describe('clauseValue', () => {
  it('sends the typed clause, or none', () => {
    expect(clauseValue('900000')).toBe(900000)
    expect(clauseValue('')).toBeUndefined()
    expect(clauseValue('0')).toBeUndefined()
  })
})
