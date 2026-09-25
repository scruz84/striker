import { describe, expect, it } from 'vitest'
import { daysUntilNextBirthday } from './playerBirthday'

describe('daysUntilNextBirthday', () => {
  const anchor = '2026-01-01T00:00:00Z'

  it('is 0 right at the anchor (a brand-new player)', () => {
    expect(daysUntilNextBirthday(anchor, new Date(anchor))).toBe(0)
  })

  it('counts down within the first 20-day cycle', () => {
    expect(daysUntilNextBirthday(anchor, new Date('2026-01-08T00:00:00Z'))).toBe(13)
  })

  it('is 0 again once a full cycle has elapsed', () => {
    expect(daysUntilNextBirthday(anchor, new Date('2026-01-21T00:00:00Z'))).toBe(0)
  })

  it('counts down within the second cycle', () => {
    expect(daysUntilNextBirthday(anchor, new Date('2026-01-30T00:00:00Z'))).toBe(11)
  })
})
