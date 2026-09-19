import { describe, expect, it } from 'vitest'
import { derivePlayerFace, regionForCountry } from './traits'

describe('regionForCountry', () => {
  it('maps every known country to europe', () => {
    for (const code of ['es', 'en', 'fr', 'de', 'it', 'pt']) {
      expect(regionForCountry(code)).toBe('europe')
    }
  })

  it('falls back to europe for missing or unrecognized codes', () => {
    expect(regionForCountry(undefined)).toBe('europe')
    expect(regionForCountry('')).toBe('europe')
    expect(regionForCountry('xx')).toBe('europe')
  })
})

describe('derivePlayerFace', () => {
  it('is deterministic for the same seed and birth country', () => {
    expect(derivePlayerFace(42, 'es')).toEqual(derivePlayerFace(42, 'es'))
  })

  it('skews skin tone toward the lighter end of the palette for european players', () => {
    const lightTones = new Set(['#F0C49A', '#E3AD7C', '#F7DAB6'])
    let lightCount = 0
    const samples = 500
    for (let seed = 0; seed < samples; seed++) {
      const { skin } = derivePlayerFace(seed, 'es')
      if (lightTones.has(skin)) lightCount++
    }
    // Weighted table gives light tones ~82% odds - well above the ~50% a uniform pick over 6
    // tones (3 of which are "light") would produce, so a wide margin avoids test flakiness.
    expect(lightCount / samples).toBeGreaterThan(0.65)
  })

  it('still reaches every skin tone for a european player, just rarely for the darkest one', () => {
    const seen = new Set<string>()
    for (let seed = 0; seed < 2000; seed++) {
      seen.add(derivePlayerFace(seed, 'es').skin)
    }
    expect(seen.size).toBe(6)
  })
})
