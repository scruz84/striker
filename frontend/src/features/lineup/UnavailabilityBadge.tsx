import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import LocalHospitalIcon from '@mui/icons-material/LocalHospital'
import { useTranslation } from 'react-i18next'
import { UNAVAILABLE_INJURY, type PlayerUnavailability } from '../../api/types'

interface UnavailabilityBadgeProps {
  unavailability: PlayerUnavailability
  size?: number
}

/** Flags a suspended (red card) or injured (medical cross) player in a lineup editor, with the
 * same "out for N more matches" wording as the player's own page as its tooltip. */
export function UnavailabilityBadge({ unavailability, size = 18 }: UnavailabilityBadgeProps) {
  const { t } = useTranslation()
  const injured = unavailability.reason === UNAVAILABLE_INJURY
  const label = t(injured ? 'playerPage.status.injured' : 'playerPage.status.suspended', {
    count: unavailability.matchesRemaining,
  })

  return (
    <Tooltip title={label}>
      <Box
        role="img"
        aria-label={label}
        sx={{
          width: size,
          height: size,
          borderRadius: '50%',
          bgcolor: '#fff',
          border: '1px solid rgba(0,0,0,0.25)',
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
        }}
      >
        {injured ? (
          <LocalHospitalIcon sx={{ fontSize: size * 0.8, color: 'error.main' }} />
        ) : (
          // A red card - there's no stock icon for one, and it reads more clearly as "sent off /
          // suspended" than any generic "blocked" glyph would.
          <Box sx={{ width: size * 0.42, height: size * 0.6, borderRadius: '2px', bgcolor: 'error.main' }} />
        )}
      </Box>
    </Tooltip>
  )
}
