import { LegalDocument } from './LegalDocument'

/** Public Terms of Use (/terms), readable without signing in. */
export function TermsPage() {
  return <LegalDocument kind="terms" />
}
