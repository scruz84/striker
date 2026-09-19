// Maps a player's birthCountry code (see api/types.ts's COUNTRY_NAME_KEYS - the name pool a
// player was generated from, not a strict real-world nationality/passport) to a flag emoji for a
// quick visual cue next to their name. England uses the UK flag rather than the England
// subdivision flag (a multi-codepoint "tag sequence" emoji) since the latter renders as blank
// boxes on many fonts/platforms that don't support tag sequences, while the UK flag is a plain
// two-codepoint regional-indicator flag supported everywhere else in this map already is.
const COUNTRY_FLAGS: Record<string, string> = {
  es: '🇪🇸',
  en: '🇬🇧',
  fr: '🇫🇷',
  de: '🇩🇪',
  it: '🇮🇹',
  pt: '🇵🇹',
}

export function countryFlag(code?: string): string | undefined {
  return code ? COUNTRY_FLAGS[code] : undefined
}
