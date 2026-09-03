import { describe, expect, it } from 'vitest'
import { formatDateTime } from './formatDate'

describe('formatDateTime', () => {
  it('formats an ISO date as DD/MM/YYYY, HH:mm', () => {
    const result = formatDateTime('2026-03-05T14:30:00Z')
    expect(result).toMatch(/^\d{2}\/\d{2}\/2026, \d{2}:\d{2}$/)
  })

  it('returns the input unchanged when it is not a valid date', () => {
    expect(formatDateTime('not-a-date')).toBe('not-a-date')
  })
})
