import { LegalDocument } from './LegalDocument'

/** Public Privacy Policy (/privacy), readable without signing in. */
export function PrivacyPage() {
  return <LegalDocument kind="privacy" />
}
