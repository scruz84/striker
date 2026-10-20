import { useTranslation } from 'react-i18next'
import { useNotify } from '../../components/notifyContext'

// Builds the { onSuccess, onError } options for a mutation so an admin action reports its result as
// a transient notification (see NotifyProvider) instead of a permanent Alert under the form. The
// error falls back to a generic message when a form has no specific one.
export function useAdminFeedback() {
  const { t } = useTranslation()
  const notify = useNotify()
  return (successKey: string, errorKey?: string) => ({
    onSuccess: () => notify.success(t(successKey)),
    onError: () => notify.error(t(errorKey ?? 'admin.genericError')),
  })
}
