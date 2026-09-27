import Alert from '@mui/material/Alert'
import { useTranslation } from 'react-i18next'
import type { Player } from '../../api/types'

/** Shown while a suspended/injured player is still on the pitch (e.g. left over from a lineup
 * saved before they got hurt or sent off). On a per-match lineup (blocking, the default) saving
 * stays disabled until they're replaced; on a team's standing default lineup (blocking=false) this
 * is purely informational, since the default isn't tied to any one matchday's squad and can still
 * be saved as-is. */
export function UnavailableStartersWarning({ players, blocking = true }: { players: Player[]; blocking?: boolean }) {
  const { t } = useTranslation()
  return (
    <Alert severity="warning" sx={{ mb: { xs: 1, sm: 2 } }}>
      {t(blocking ? 'lineup.unavailableStarters' : 'lineup.unavailableStartersInfo', {
        count: players.length,
        names: players.map((p) => `${p.firstName} ${p.lastName}`).join(', '),
      })}
    </Alert>
  )
}
