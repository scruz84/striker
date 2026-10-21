import { useTranslation } from 'react-i18next'
import { useMe, useSetUserLanguage } from '../api/hooks'
import type { SupportedLanguage } from '../i18n'

/** Switches the UI language and, for a signed-in user, saves it as their preference. */
export function useChangeLanguage() {
  const { i18n } = useTranslation()
  const { data: me } = useMe()
  const setUserLanguage = useSetUserLanguage()
  const currentLanguage = (i18n.resolvedLanguage ?? i18n.language) as SupportedLanguage

  const changeLanguage = (language: SupportedLanguage) => {
    void i18n.changeLanguage(language)
    if (me) {
      setUserLanguage.mutate(language)
    }
  }
  return { currentLanguage, changeLanguage }
}
