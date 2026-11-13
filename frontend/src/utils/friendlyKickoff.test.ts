import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defaultKickoffValue, isQuarterHour } from './friendlyKickoff'

describe('defaultKickoffValue', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('rounds up to the next quarter hour', () => {
    vi.setSystemTime(new Date(2026, 10, 13, 18, 7, 0))
    expect(defaultKickoffValue(5)).toBe('2026-11-13T18:15')
  })

  it('keeps a value already on a slot', () => {
    vi.setSystemTime(new Date(2026, 10, 13, 18, 25, 0))
    expect(defaultKickoffValue(5)).toBe('2026-11-13T18:30')
  })

  it('rolls over to the next day', () => {
    vi.setSystemTime(new Date(2026, 10, 13, 23, 52, 0))
    expect(defaultKickoffValue(5)).toBe('2026-11-14T00:00')
  })
})

describe('isQuarterHour', () => {
  it.each(['2026-11-13T18:00', '2026-11-13T18:15', '2026-11-13T18:30', '2026-11-13T18:45', ''])('accepts %s', (value) => {
    expect(isQuarterHour(value)).toBe(true)
  })

  it.each(['2026-11-13T18:05', '2026-11-13T18:50', '2026-11-13T18:15:30'])('rejects %s', (value) => {
    expect(isQuarterHour(value)).toBe(false)
  })
})
