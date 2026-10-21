import i18n from '../i18n'

// Full Intl locale tags for each supported UI language (src/i18n) - kept separate from the
// short i18next language codes since Intl wants a region for consistent day/month ordering,
// separators and currency placement.
const INTL_LOCALES: Record<string, string> = {
  en: 'en-GB',
  es: 'es-ES',
}

export function intlLocale(): string {
  return INTL_LOCALES[i18n.resolvedLanguage ?? i18n.language] ?? INTL_LOCALES.en
}
