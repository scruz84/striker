import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink } from 'react-router-dom'
import type { Lineup, Player } from '../../api/types'
import { matchRatingColor } from '../../utils/matchRating'

const CHIP_SIZE = 44
const RING_THICKNESS = 3

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

// A Hattrick-style rating badge: a single ring around the score, filled proportionally to the
// rating (out of matchRatingMax=10 - see backend common.ComputeMatchRatings) and colored by how
// good it is. Replaces the old face + shirt-number + rating trio, which crammed three separate
// pieces of information into one small pitch slot.
export function MatchPitchChip({ player, rating }: { player: Player; rating?: number }) {
  const theme = useTheme()
  const label =
    rating === undefined
      ? `${player.firstName} ${player.lastName} (${player.age})`
      : `${player.firstName} ${player.lastName} (${player.age}) - ${rating.toFixed(1)}`
  const ratingColor = rating === undefined ? undefined : matchRatingColor(rating)
  const ringColor =
    ratingColor === undefined ? 'rgba(255,255,255,0.5)' : ratingColor === 'default' ? theme.palette.grey[300] : theme.palette[ratingColor].main
  const fillPct = rating === undefined ? 0 : Math.max(0, Math.min(100, (rating / 10) * 100))

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
        <Box
          sx={{
            width: CHIP_SIZE,
            height: CHIP_SIZE,
            borderRadius: '50%',
            background:
              rating === undefined
                ? 'none'
                : `conic-gradient(${ringColor} ${fillPct}%, rgba(255,255,255,0.25) ${fillPct}%)`,
            border: rating === undefined ? '2px solid rgba(255,255,255,0.5)' : 'none',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Box
            sx={{
              width: CHIP_SIZE - RING_THICKNESS * 2,
              height: CHIP_SIZE - RING_THICKNESS * 2,
              borderRadius: '50%',
              bgcolor: 'rgba(20,20,20,0.82)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Typography sx={{ color: '#fff', fontSize: rating === undefined ? '0.85rem' : '0.75rem', fontWeight: 700, lineHeight: 1 }}>
              {rating === undefined ? player.shirtNumber ?? '' : rating.toFixed(1)}
            </Typography>
          </Box>
        </Box>
        <Box sx={{ mt: 0.4, px: 0.5, borderRadius: 1, bgcolor: 'rgba(0,0,0,0.6)', maxWidth: '100%' }}>
          <Typography noWrap sx={{ color: '#fff', fontSize: '0.6875rem', fontWeight: 600, lineHeight: 1.3, textAlign: 'center' }}>
            {player.lastName}
          </Typography>
        </Box>
      </Box>
    </Tooltip>
  )
}
