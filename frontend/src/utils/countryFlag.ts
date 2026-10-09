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
  nl: '🇳🇱',
  se: '🇸🇪',
  pl: '🇵🇱',
  gr: '🇬🇷',
  br: '🇧🇷',
  ar: '🇦🇷',
  mx: '🇲🇽',
  co: '🇨🇴',
  cl: '🇨🇱',
  pe: '🇵🇪',
  uy: '🇺🇾',
  us: '🇺🇸',
  ca: '🇨🇦',
  ec: '🇪🇨',
  jp: '🇯🇵',
  kr: '🇰🇷',
  cn: '🇨🇳',
  in: '🇮🇳',
  th: '🇹🇭',
  vn: '🇻🇳',
  id: '🇮🇩',
  ph: '🇵🇭',
  sa: '🇸🇦',
  ir: '🇮🇷',
  ng: '🇳🇬',
  eg: '🇪🇬',
  gh: '🇬🇭',
  ci: '🇨🇮',
  cm: '🇨🇲',
  sn: '🇸🇳',
  za: '🇿🇦',
  ma: '🇲🇦',
  dz: '🇩🇿',
  ke: '🇰🇪',
  be: '🇧🇪',
  tr: '🇹🇷',
  at: '🇦🇹',
  // Scotland/Wales have no ISO 3166-1 code and thus no regional-indicator flag emoji - only
  // multi-codepoint "tag sequence" subdivision flags exist for them, which hit the same
  // blank-box rendering problem noted above for England, so they reuse the UK flag too.
  sco: '🇬🇧',
  wal: '🇬🇧',
  ie: '🇮🇪',
  ru: '🇷🇺',
  rs: '🇷🇸',
  hr: '🇭🇷',
  ba: '🇧🇦',
  si: '🇸🇮',
  mk: '🇲🇰',
  me: '🇲🇪',
  // Kosovo has no ISO 3166-1 code either, but 🇽🇰 (built from the commonly-used user-assigned
  // "XK" code) is the de facto flag widely supported on modern platforms, unlike a tag sequence.
  xk: '🇽🇰',
  ve: '🇻🇪',
  py: '🇵🇾',
  bo: '🇧🇴',
  qa: '🇶🇦',
  ae: '🇦🇪',
  uz: '🇺🇿',
  tn: '🇹🇳',
  tz: '🇹🇿',
  ml: '🇲🇱',
}

export function countryFlag(code?: string): string | undefined {
  return code ? COUNTRY_FLAGS[code] : undefined
}
