import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { COUNTRY_NAME_KEYS } from '../api/types'

type Translate = (key: string) => string

// One collator per UI language: building an Intl.Collator is far costlier than a comparison, and a
// country list sorts hundreds of times per render pass. `sensitivity: 'base'` ignores case and
// accents, so "Árabe" sorts with the As and "Azerbaiyán" lands between "Austria" and "Bélgica".
const collators = new Map<string, Intl.Collator>()

function collatorFor(language: string): Intl.Collator {
  let collator = collators.get(language)
  if (!collator) {
    collator = new Intl.Collator(language || undefined, { sensitivity: 'base' })
    collators.set(language, collator)
  }
  return collator
}

/** The country's name in the current UI language; an unknown code is shown as is. */
export function localizedCountryName(t: Translate, code: string): string {
  const key = COUNTRY_NAME_KEYS[code]
  return key ? t(key) : code
}

/** Orders two country codes by their localized names, using the UI language's collation. */
export function compareCountryNames(a: string, b: string, t: Translate, language: string): number {
  return collatorFor(language).compare(localizedCountryName(t, a), localizedCountryName(t, b))
}

/** A copy of items sorted alphabetically by the localized name of the country getCode returns. */
export function sortByCountryName<T>(items: readonly T[], getCode: (item: T) => string, t: Translate, language: string): T[] {
  return [...items].sort((x, y) => compareCountryNames(getCode(x), getCode(y), t, language))
}

/** Every UI list of countries goes through this: items sorted by localized name, re-sorted when the
 * UI language changes. Pair it with localizedCountryName (also returned) to print the names. */
export function useSortedCountries<T>(items: readonly T[] | undefined, getCode: (item: T) => string) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? i18n.language
  // getCode is a plain accessor that callers pass inline, so it is deliberately not a dependency.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const sorted = useMemo(() => (items ? sortByCountryName(items, getCode, t, language) : undefined), [items, t, language])
  return { sorted, countryName: (code: string) => localizedCountryName(t, code) }
}
