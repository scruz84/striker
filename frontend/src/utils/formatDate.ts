import i18n from '../i18n'

// Full Intl locale tags for each supported UI language (src/i18n) - kept separate from the
// short i18next language codes since Intl.DateTimeFormat wants a region for consistent
// day/month ordering and separators.
const DATE_LOCALES: Record<string, string> = {
  en: 'en-GB',
  es: 'es-ES',
}

export function formatDateTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  const locale = DATE_LOCALES[i18n.resolvedLanguage ?? i18n.language] ?? DATE_LOCALES.en
  return new Intl.DateTimeFormat(locale, {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

// Date only, no time - for compact contexts (e.g. HeadToHeadCard's past-meetings list) where the
// full formatDateTime's extra ", HH:mm" just adds width without adding anything a viewer skimming
// a list of results needs.
export function formatDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  const locale = DATE_LOCALES[i18n.resolvedLanguage ?? i18n.language] ?? DATE_LOCALES.en
  return new Intl.DateTimeFormat(locale, { day: '2-digit', month: '2-digit', year: '2-digit' }).format(date)
}

// Day/month plus kickoff time, no year - for a list scoped to the current season (e.g.
// TeamMatchesSection's row) where the year is implied and formatDateTime's full "DD/MM/YYYY,
// HH:mm" is wider than a narrow row has room for.
export function formatDayTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  const locale = DATE_LOCALES[i18n.resolvedLanguage ?? i18n.language] ?? DATE_LOCALES.en
  return new Intl.DateTimeFormat(locale, { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).format(date)
}
