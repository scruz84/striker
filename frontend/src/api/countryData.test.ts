import { describe, expect, it } from 'vitest'
import { COUNTRY_REGIONS } from '../components/avatar/traits'
import en from '../i18n/locales/en.json'
import es from '../i18n/locales/es.json'
import { COUNTRY_NAME_KEYS } from './types'

// Every country the game can generate (mirrors backend common.KnownCountries - the FIFA
// membership). A code missing any of its data must fail here, naming the code.
const CODES = Object.keys(COUNTRY_NAME_KEYS)
const names = (locale: unknown) => (locale as { countries: Record<string, string> }).countries

describe('country data', () => {
  it('covers the 211 FIFA members', () => {
    expect(CODES).toHaveLength(211)
    expect(new Set(CODES).size).toBe(211)
  })

  it('maps every code to its own countries.<code> key', () => {
    for (const code of CODES) expect(COUNTRY_NAME_KEYS[code], code).toBe(`countries.${code}`)
  })

  it('has a region for every code', () => {
    for (const code of CODES) {
      expect(['europe', 'americas', 'asia', 'africa'], `COUNTRY_REGIONS.${code}`).toContain(COUNTRY_REGIONS[code])
    }
    expect(Object.keys(COUNTRY_REGIONS).sort()).toEqual([...CODES].sort())
  })

  it('has an English and a Spanish name for every code, and no extras', () => {
    for (const locale of [en, es]) {
      for (const code of CODES) expect(names(locale)[code], `countries.${code}`).toBeTruthy()
      expect(Object.keys(names(locale)).sort()).toEqual([...CODES].sort())
    }
  })

  it('gives two countries the same English name never', () => {
    const englishNames = Object.values(names(en))
    expect(new Set(englishNames).size).toBe(englishNames.length)
  })
})
