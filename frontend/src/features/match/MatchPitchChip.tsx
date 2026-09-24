import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink } from 'react-router-dom'
import type { Lineup, Player } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { matchRatingColor } from '../../utils/matchRating'

// Lineup.positions is a flat list; a formation's central slots (e.g. two center-backs) share a
// position id and are told apart only by their order of appearance for that position.
export function slotIndexWithinPosition(lineup: Lineup, target: Lineup['positions'][number]): number {
  let index = 0
  for (const pos of lineup.positions) {
    if (pos === target) return index
    if (pos.position === target.position) index++
  }
  return index
}

export function MatchPitchChip({ player, rating }: { player: Player; rating?: number }) {
  const theme = useTheme()
  const label =
    rating === undefined
      ? `${player.firstName} ${player.lastName} (${player.age})`
      : `${player.firstName} ${player.lastName} (${player.age}) - ${rating.toFixed(1)}`
  const ratingColor = rating === undefined ? undefined : matchRatingColor(rating)
  const ratingBg =
    ratingColor === undefined ? undefined : ratingColor === 'default' ? theme.palette.grey[600] : theme.palette[ratingColor].main
  return (
    <Tooltip title={label}>
      <Box
        component={RouterLink}
        to={`/player/${player.id}`}
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          width: 68,
          textDecoration: 'none',
          color: 'inherit',
        }}
      >
        <Box sx={{ position: 'relative', lineHeight: 0 }}>
          <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={40} />
          {!!player.shirtNumber && (
            <Box
              sx={{
                position: 'absolute',
                top: -4,
                right: -4,
                width: 16,
                height: 16,
                borderRadius: '50%',
                bgcolor: 'rgba(20,20,20,0.85)',
                border: '1px solid rgba(255,255,255,0.85)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Typography sx={{ color: '#fff', fontSize: '0.55rem', fontWeight: 700, lineHeight: 1 }}>
                {player.shirtNumber}
              </Typography>
            </Box>
          )}
          {rating !== undefined && (
            <Box
              sx={{
                position: 'absolute',
                top: -4,
                left: -4,
                width: 16,
                height: 16,
                borderRadius: '50%',
                bgcolor: ratingBg,
                border: '1px solid rgba(255,255,255,0.85)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Typography sx={{ color: '#fff', fontSize: '0.5rem', fontWeight: 700, lineHeight: 1 }}>
                {rating.toFixed(1)}
              </Typography>
            </Box>
          )}
        </Box>
        <Box sx={{ mt: 0.25, px: 0.5, borderRadius: 1, bgcolor: 'rgba(0,0,0,0.6)', maxWidth: '100%' }}>
          <Typography noWrap sx={{ color: '#fff', fontSize: '0.6rem', fontWeight: 600, lineHeight: 1.3, textAlign: 'center' }}>
            {player.lastName}
          </Typography>
        </Box>
      </Box>
    </Tooltip>
  )
}
