import { useTranslation } from 'react-i18next'
import { getLegalConfig, type LegalConfig } from '../../config'

export interface LegalValues {
  config: LegalConfig
  /** Interpolation values for the legal texts: the configured value, or a visible placeholder for
   * a required one that is missing (the page also shows a warning then). */
  values: { controllerName: string; contactEmail: string }
  /** Names of the required properties that are not set (empty when the deployment is complete). */
  missingRequired: string[]
}

/** Reads the deployment's legal identity (LEGAL_* properties, see config.ts). */
export function useLegalValues(): LegalValues {
  const { t } = useTranslation()
  const config = getLegalConfig()
  const placeholder = t('legal.notConfiguredValue')
  const missingRequired: string[] = []
  if (!config.controllerName) missingRequired.push('LEGAL_CONTROLLER_NAME')
  if (!config.contactEmail) missingRequired.push('LEGAL_CONTACT_EMAIL')
  return {
    config,
    values: {
      controllerName: config.controllerName || placeholder,
      contactEmail: config.contactEmail || placeholder,
    },
    missingRequired,
  }
}
