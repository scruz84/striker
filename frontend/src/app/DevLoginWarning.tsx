import Alert from '@mui/material/Alert'
import { useTranslation } from 'react-i18next'
import { useDevLoginUsers } from '../api/hooks'

// A thin strip across the top of the app whenever the backend has dev login on (it lists a
// non-empty roster only then). Dev login signs anyone in as any dev user, admin included, with
// no credentials - easy to leave on by mistake, so it is made impossible to miss.
export function DevLoginWarning() {
  const { t } = useTranslation()
  const { data: devUsers } = useDevLoginUsers()
  if (!devUsers || devUsers.length === 0) return null
  return (
    <Alert severity="warning" variant="filled" square role="alert" sx={{ py: 0, borderRadius: 0 }}>
      {t('devLogin.warningStrip')}
    </Alert>
  )
}
