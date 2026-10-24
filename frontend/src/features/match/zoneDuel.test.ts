import { describe, expect, it } from 'vitest'
import { duelShares, duelTooltip, formatStrength } from './zoneDuel'

describe('zone duel formatting', () => {
  it('rounds the strength to a whole number and keeps one decimal in the tooltip', () => {
    expect(formatStrength(42.6)).toBe('43')
    expect(formatStrength(28.4)).toBe('28')
    expect(duelTooltip('vs ataque central del rival', 42.6, 28.4)).toBe('vs ataque central del rival: 42.6 - 28.4')
  })

  it('shows whole numbers with a decimal in the tooltip too', () => {
    expect(duelTooltip('x', 40, 12)).toBe('x: 40.0 - 12.0')
  })

  it('splits the shares of the central defensive duel 60% / 40%', () => {
    const shares = duelShares(42.6, 28.4)
    expect(Math.round(shares.home)).toBe(60)
    expect(Math.round(shares.away)).toBe(40)
  })

  it('is an even 50% / 50% with "0" when both sides are at zero', () => {
    expect(duelShares(0, 0)).toEqual({ home: 50, away: 50 })
    expect(formatStrength(0)).toBe('0')
  })
})
