import Alert from '@mui/material/Alert'
import { useTranslation } from 'react-i18next'
import type { Player } from '../../api/types'

/** Shown while a suspended/injured player is still on the pitch (e.g. left over from a lineup
 * saved before they got hurt or sent off) - saving stays blocked until they're replaced. */
export function UnavailableStartersWarning({ players }: { players: Player[] }) {
  const { t } = useTranslation()
  return (
    <Alert severity="warning" sx={{ mb: { xs: 1, sm: 2 } }}>
      {t('lineup.unavailableStarters', {
        count: players.length,
        names: players.map((p) => `${p.firstName} ${p.lastName}`).join(', '),
      })}
    </Alert>
  )
}
