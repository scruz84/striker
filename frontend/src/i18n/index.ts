import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import es from './locales/es.json'

export const SUPPORTED_LANGUAGES = {
  en: 'English',
  es: 'Español',
} as const

export type SupportedLanguage = keyof typeof SUPPORTED_LANGUAGES

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      es: { translation: es },
    },
    fallbackLng: 'en',
    supportedLngs: Object.keys(SUPPORTED_LANGUAGES),
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      lookupLocalStorage: 'striker-language',
    },
  })

// Keep <html lang> in step with the UI language so the browser's native pickers, hyphenation
// and screen readers use it instead of the static "en" in index.html.
function syncDocumentLanguage() {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = i18n.resolvedLanguage ?? i18n.language ?? 'en'
  }
}
syncDocumentLanguage()
i18n.on('languageChanged', syncDocumentLanguage)

export default i18n
