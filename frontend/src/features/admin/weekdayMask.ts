import { WEEKDAY_OPTIONS } from '../../api/types'

// maskFromBits turns the ToggleButtonGroup's list of selected weekday bits into a single mask.
export function maskFromBits(bits: number[]): number {
  return bits.reduce((mask, bit) => mask | bit, 0)
}

// bitsFromMask is the inverse: the list of WEEKDAY_OPTIONS bits set in a weekday mask.
export function bitsFromMask(mask: number): number[] {
  return WEEKDAY_OPTIONS.filter((d) => (mask & d.bit) !== 0).map((d) => d.bit)
}
