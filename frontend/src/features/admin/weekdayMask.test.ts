import { describe, expect, it } from 'vitest'
import { ALL_WEEKDAYS_MASK, WEEKDAY_OPTIONS } from '../../api/types'
import { bitsFromMask, maskFromBits } from './weekdayMask'

describe('weekdayMask', () => {
  it('turns an empty selection into an empty mask', () => {
    expect(maskFromBits([])).toBe(0)
    expect(bitsFromMask(0)).toEqual([])
  })

  it('round-trips every single weekday', () => {
    for (const day of WEEKDAY_OPTIONS) {
      expect(maskFromBits(bitsFromMask(day.bit))).toBe(day.bit)
      expect(bitsFromMask(day.bit)).toEqual([day.bit])
    }
  })

  it('selects every weekday for the all-days mask', () => {
    expect(bitsFromMask(ALL_WEEKDAYS_MASK)).toHaveLength(WEEKDAY_OPTIONS.length)
    expect(maskFromBits(WEEKDAY_OPTIONS.map((d) => d.bit))).toBe(ALL_WEEKDAYS_MASK)
  })
})
