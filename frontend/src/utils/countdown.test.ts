import { describe, expect, it } from 'vitest'
import { countdownParts, countdownUntil } from './countdown'

const NOW = new Date('2026-10-20T12:00:00Z').getTime()

describe('countdownUntil', () => {
  it('breaks the remaining time into days, hours and minutes', () => {
    expect(countdownUntil('2026-10-22T15:20:00Z', NOW)).toEqual({ days: 2, hours: 3, minutes: 20 })
  })
  it('is undefined once the date has passed or is invalid', () => {
    expect(countdownUntil('2026-10-20T11:59:00Z', NOW)).toBeUndefined()
    expect(countdownUntil('nope', NOW)).toBeUndefined()
  })
})

describe('countdownParts', () => {
  it('picks the most compact unit pair', () => {
    expect(countdownParts({ days: 2, hours: 3, minutes: 20 }).key).toBe('days')
    expect(countdownParts({ days: 0, hours: 3, minutes: 20 }).key).toBe('hours')
    expect(countdownParts({ days: 0, hours: 0, minutes: 0 })).toEqual({ key: 'minutes', params: { minutes: 1 } })
  })
})
