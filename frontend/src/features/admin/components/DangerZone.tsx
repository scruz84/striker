import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { AdminSection } from './AdminSection'

// The irreversible actions, kept apart from the everyday settings so they can't be hit by accident
// while scrolling. Each action inside still asks for its own confirmation.
export function DangerZone({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <AdminSection tone="danger" title={t('admin.dangerZone.title')} description={t('admin.dangerZone.description')}>
      {children}
    </AdminSection>
  )
}
