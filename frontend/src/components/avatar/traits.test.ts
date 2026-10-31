import { describe, expect, it } from 'vitest'
import { COUNTRY_NAME_KEYS } from '../../api/types'
import { APPEARANCE_PROFILES, appearanceForCountry, derivePlayerFace, regionForCountry } from './traits'

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

describe('appearance by country', () => {
  const share = (country: string, pick: (face: ReturnType<typeof derivePlayerFace>) => boolean, samples = 1500) => {
    let n = 0
    for (let seed = 0; seed < samples; seed++) if (pick(derivePlayerFace(seed, country))) n++
    return n / samples
  }
  const DARK = new Set(['#7A4F30', '#A66F42'])
  const LIGHT = new Set(['#F0C49A', '#E3AD7C', '#F7DAB6'])
  const BLONDE = '#8C5A1E'

  it('gives every known country an appearance group', () => {
    for (const code of Object.keys(COUNTRY_NAME_KEYS)) {
      expect(APPEARANCE_PROFILES[appearanceForCountry(code)], code).toBeDefined()
    }
    // Not just the central-Europe fallback: only a missing or unknown code gets that.
    const fallback = appearanceForCountry(undefined)
    expect(appearanceForCountry('zz')).toBe(fallback)
    expect(appearanceForCountry('jm')).not.toBe(fallback)
  })

  it('keeps every option reachable in every group', () => {
    for (const [group, profile] of Object.entries(APPEARANCE_PROFILES)) {
      for (const weights of [profile.skin, profile.hairColor, profile.hairStyle]) {
        expect(Math.min(...weights), group).toBeGreaterThan(0)
      }
      expect(profile.skin).toHaveLength(6)
      expect(profile.hairColor).toHaveLength(6)
      expect(profile.hairStyle).toHaveLength(5)
    }
  })

  it('tells neighbours on the same continent apart', () => {
    // Skin: Jamaica and Nigeria skew dark, Iceland and Denmark light, Fiji dark though "asia" for cups.
    expect(share('jm', (f) => DARK.has(f.skin))).toBeGreaterThan(0.6)
    expect(share('ng', (f) => DARK.has(f.skin))).toBeGreaterThan(0.7)
    expect(share('fj', (f) => DARK.has(f.skin))).toBeGreaterThan(0.6)
    expect(share('is', (f) => LIGHT.has(f.skin))).toBeGreaterThan(0.85)
    expect(share('jp', (f) => DARK.has(f.skin))).toBeLessThan(0.15)
    // Hair colour: blond is common in Scandinavia, rare in Japan and the Gulf.
    expect(share('se', (f) => f.hairColor === BLONDE)).toBeGreaterThan(0.25)
    expect(share('jp', (f) => f.hairColor === BLONDE)).toBeLessThan(0.05)
    expect(share('sa', (f) => f.hairColor === BLONDE)).toBeLessThan(0.05)
  })

  it('skews hair style and facial hair by country', () => {
    expect(share('ng', (f) => f.hairStyle === 'curly')).toBeGreaterThan(share('jp', (f) => f.hairStyle === 'curly') + 0.2)
    expect(share('sa', (f) => f.facialHair !== 'none')).toBeGreaterThan(share('jp', (f) => f.facialHair !== 'none') + 0.3)
  })
})
