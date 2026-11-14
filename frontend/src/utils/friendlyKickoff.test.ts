import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defaultKickoffValue, isQuarterHour, isWeekdayAllowed, kickoffHelperKey } from './friendlyKickoff'

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

// The server judges a friendly's weekday in UTC (common.isAllowedFriendlyKickoff); the picker must
// agree for a kickoff whose local day and UTC day differ (issue #403). Run in Madrid (UTC+1 in
// November): Saturday 00:30 local is Friday 23:30 UTC, Sunday 00:30 local is Saturday 23:30 UTC.
describe('isWeekdayAllowed across midnight', () => {
  const saturdaysOnly = { weekdaysMask: 1 << 6 }
  beforeEach(() => {
    vi.stubEnv('TZ', 'Europe/Madrid')
  })
  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('runs in a zone ahead of UTC', () => {
    expect(new Date('2026-11-14T00:30').getUTCHours()).toBe(23)
  })

  it('refuses Saturday 00:30 local, which the server sees as Friday', () => {
    expect(isWeekdayAllowed(saturdaysOnly, '2026-11-14T00:30')).toBe(false)
  })

  it('accepts Sunday 00:30 local, which the server sees as Saturday', () => {
    expect(isWeekdayAllowed(saturdaysOnly, '2026-11-15T00:30')).toBe(true)
  })
})

describe('kickoffHelperKey', () => {
  it('reminds that restricted days are in UTC', () => {
    expect(kickoffHelperKey({ weekdaysMask: 1 << 6 }, '')).toBe('friendlies.received.kickoffDaysUtc')
  })

  it('says nothing when every day is open', () => {
    expect(kickoffHelperKey({ weekdaysMask: 0x7f }, '2026-11-14T18:00')).toBeNull()
  })

  it('shows the error first', () => {
    expect(kickoffHelperKey({ weekdaysMask: 0x7f }, '2026-11-14T18:05')).toBe('friendlies.received.kickoffNotQuarterHour')
  })
})
