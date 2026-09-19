import Alert from '@mui/material/Alert'
import { useTranslation } from 'react-i18next'

// Shown in place of MatchLineupForm once useIsLineupLocked flips true - saving would be rejected
// server-side at this point anyway (see common.IsLineupLocked/ErrLineupLocked), so there's no
// point still offering the editor.
export function LineupLockedNotice() {
  const { t } = useTranslation()
  return <Alert severity="info">{t('lineup.locked')}</Alert>
}
