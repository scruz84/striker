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
