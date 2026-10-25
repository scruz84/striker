import en from '../i18n/locales/en.json'
import { COUNTRY_NAME_KEYS } from '../api/types'

// A national team's stored name (backend common.nationalTeamName) is just its country's English
// name ("Spain", not "Spain National Team") - the frontend is what actually localizes it, via the
// same country-code lookup already used for a player's birth country or a league's country (see
// COUNTRY_NAME_KEYS). Built from en.json rather than a second hardcoded country list, since
// en.json's own "countries" section is already that list in English. A legacy row created before
// the backend dropped the " National Team" suffix still carries it, so it's stripped first.
const CODE_BY_ENGLISH_NAME: Record<string, string> = Object.fromEntries(
  Object.entries((en as { countries: Record<string, string> }).countries).map(([code, name]) => [name, code]),
)

const LEGACY_SUFFIX = ' National Team'

// The country code of rawName when it names a national team (or, for a legacy row, did), otherwise
// undefined - lets a caller show that country's flag next to the name.
export function nationalTeamCountry(rawName: string): string | undefined {
  const bare = rawName.endsWith(LEGACY_SUFFIX) ? rawName.slice(0, -LEGACY_SUFFIX.length) : rawName
  return CODE_BY_ENGLISH_NAME[bare]
}

// Translates rawName into the current UI language when it's (or, for a legacy row, was) a known
// country's name - i.e. rawName names a national team. Any other name (an ordinary club's) is
// returned unchanged.
export function localizedTeamName(t: (key: string) => string, rawName: string): string {
  const code = nationalTeamCountry(rawName)
  return code ? t(COUNTRY_NAME_KEYS[code]) : rawName
}
