import { useState } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import CloseIcon from '@mui/icons-material/Close'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { POSITION_SHORT_LABEL_KEYS, type Player } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerStatsPopover } from '../../components/PlayerStatsPopover'
import { bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'

/** The card shown under the pitch while a player is picked up: who they are, their rating, and the
 * two ways to learn more (the skills popover, the full profile). Moving a player and inspecting
 * one used to be two tiny tap targets glued together; now tapping only ever moves, and the details
 * live here, one deliberate tap away. */
export function SelectedPlayerPanel({ player, onClear }: { player: Player; onClear: () => void }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const rating = overallRating(player)

  return (
    <Paper variant="outlined" sx={{ p: 1, mt: 1, borderColor: 'warning.main', borderWidth: 2 }} aria-live="polite">
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center' }}>
        <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={40} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
            {player.firstName} {player.lastName}
          </Typography>
          <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
            <Chip size="small" variant="outlined" label={t(POSITION_SHORT_LABEL_KEYS[bestCorePosition(player)])} sx={{ height: 20 }} />
            <Typography variant="caption" color="text.secondary">
              {t('lineup.ageShort', { age: player.age })}
            </Typography>
            <Typography variant="caption" sx={{ fontWeight: 700, color: ratingTierColor(rating, theme.palette) }}>
              {rating}
            </Typography>
          </Stack>
        </Box>
        <Button size="small" onClick={(event) => setAnchor(event.currentTarget)}>
          {t('lineup.selected.skills')}
        </Button>
        <IconButton onClick={onClear} aria-label={t('lineup.selected.clear')}>
          <CloseIcon />
        </IconButton>
      </Stack>
      <Link component={RouterLink} to={`/player/${player.id}`} underline="hover" variant="body2" sx={{ display: 'inline-flex', alignItems: 'center', minHeight: 44, ml: 0.5 }}>
        {t('playerCard.viewProfile')}
      </Link>
      <PlayerStatsPopover player={player} anchorEl={anchor} onClose={() => setAnchor(null)} />
    </Paper>
  )
}
