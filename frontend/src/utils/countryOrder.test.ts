import { describe, expect, it } from 'vitest'
import i18n from '../i18n'
import { compareCountryNames, localizedCountryName, sortByCountryName } from './countryOrder'

const t = (key: string) => i18n.t(key)

async function inLanguage<T>(language: string, run: () => T): Promise<T> {
  const previous = i18n.language
  await i18n.changeLanguage(language)
  try {
    return run()
  } finally {
    await i18n.changeLanguage(previous)
  }
}

const order = (codes: string[], language: string) =>
  inLanguage(language, () => sortByCountryName(codes, (code) => code, (key) => i18n.t(key), language).map((code) => i18n.t(`countries.${code}`)))

describe('country ordering', () => {
  it('sorts Spanish names alphabetically, not by code', async () => {
    expect(await order(['de', 'en', 'es', 'fr'], 'es')).toEqual(['Alemania', 'España', 'Francia', 'Inglaterra'])
  })

  it('sorts English names alphabetically', async () => {
    expect(await order(['de', 'en', 'es', 'fr'], 'en')).toEqual(['England', 'France', 'Germany', 'Spain'])
  })

  it('does not push accented names to the end', async () => {
    expect(await order(['be', 'az', 'at'], 'es')).toEqual(['Austria', 'Azerbaiyán', 'Bélgica'])
  })

  it('ignores case and accents when comparing', async () => {
    // Accents never move a name: the order is the plain-letter order of the Spanish names.
    const sorted = await order(['il', 'fo', 'nir', 'ie'], 'es')
    expect(sorted).toEqual([...sorted].sort((a, b) => a.localeCompare(b, 'es', { sensitivity: 'base' })))
  })

  it('orders every known country without throwing, in either language', async () => {
    const { COUNTRY_NAME_KEYS } = await import('../api/types')
    const codes = Object.keys(COUNTRY_NAME_KEYS)
    for (const language of ['en', 'es']) {
      const sorted = await order(codes, language)
      expect(sorted).toHaveLength(211)
      expect(sorted).toEqual([...sorted].sort(new Intl.Collator(language, { sensitivity: 'base' }).compare))
    }
  })

  it('does not reorder its input', () => {
    const input = ['fr', 'de']
    sortByCountryName(input, (code) => code, t, 'en')
    expect(input).toEqual(['fr', 'de'])
  })

  it('compares codes by their localized names', () => {
    expect(compareCountryNames('de', 'es', t, 'en')).toBeLessThan(0)
    expect(compareCountryNames('es', 'de', t, 'en')).toBeGreaterThan(0)
    expect(compareCountryNames('es', 'es', t, 'en')).toBe(0)
  })

  it('shows an unknown code as is', () => {
    expect(localizedCountryName(t, 'zz')).toBe('zz')
  })
})
