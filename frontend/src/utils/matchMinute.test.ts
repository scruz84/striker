import { describe, expect, it } from 'vitest'
import { compareMatchMinute, formatMatchMinute, isAtOrBefore } from './matchMinute'

describe('formatMatchMinute', () => {
  it('labels a regulation minute', () => {
    expect(formatMatchMinute(63)).toBe("63'")
    expect(formatMatchMinute(45, 0)).toBe("45'")
  })

  it('labels added time with the clock minute it hangs off', () => {
    expect(formatMatchMinute(45, 2)).toBe("45+2'")
    expect(formatMatchMinute(90, 3)).toBe("90+3'")
  })
})

describe('compareMatchMinute', () => {
  it('orders 44, 45+2, 46 in match order (stoppage-time "Ordering across halves")', () => {
    const minutes = [{ minute: 46 }, { minute: 45, addedMinute: 2 }, { minute: 44 }, { minute: 45 }]
    expect([...minutes].sort(compareMatchMinute)).toEqual([
      { minute: 44 },
      { minute: 45 },
      { minute: 45, addedMinute: 2 },
      { minute: 46 },
    ])
  })

  it('treats a missing added minute as regulation', () => {
    expect(compareMatchMinute({ minute: 90 }, { minute: 90, addedMinute: 0 })).toBe(0)
    expect(isAtOrBefore({ minute: 90, addedMinute: 1 }, { minute: 90 })).toBe(false)
    expect(isAtOrBefore({ minute: 45, addedMinute: 2 }, { minute: 46 })).toBe(true)
  })
})
