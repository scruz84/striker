import { useState } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import ToggleButton from '@mui/material/ToggleButton'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import CloseIcon from '@mui/icons-material/Close'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { POSITION_SHORT_LABEL_KEYS, type LineupRoles, type Player } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerStatsPopover } from '../../components/PlayerStatsPopover'
import { bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'
import { CountryFlag } from '../../components/CountryFlag'

export type LineupRole = keyof LineupRoles

/** The three lineup roles in the order the panel shows them, with their i18n key suffix. */
export const LINEUP_ROLES: { role: LineupRole; key: string; taker: boolean }[] = [
  { role: 'captainId', key: 'captain', taker: false },
  { role: 'penaltyTakerId', key: 'penalties', taker: true },
  { role: 'freeKickTakerId', key: 'freeKicks', taker: true },
]

/** The card shown under the pitch while a player is picked up: who they are, their rating, and the
 * two ways to learn more (the skills popover, the full profile). Moving a player and inspecting
 * one used to be two tiny tap targets glued together; now tapping only ever moves, and the details
 * live here, one deliberate tap away. With `roles` it also carries the three lineup-role toggles
 * (captain, penalties, free kicks - one holder each); a goalkeeper can't take set pieces. */
export function SelectedPlayerPanel({
  player,
  onClear,
  roles,
  onToggleRole,
  isGoalkeeper = false,
}: {
  player: Player
  onClear: () => void
  roles?: LineupRoles
  onToggleRole?: (role: LineupRole) => void
  isGoalkeeper?: boolean
}) {
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
            <CountryFlag code={player.birthCountry} height={14} marginRight={0.5} />
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
      {roles && onToggleRole && (
        <Stack direction="row" spacing={0.75} sx={{ mt: 1, flexWrap: 'wrap', rowGap: 0.75 }} role="group" aria-label={t('lineup.roles.title')}>
          {LINEUP_ROLES.map(({ role, key, taker }) => (
            <ToggleButton
              key={role}
              value={role}
              size="small"
              selected={roles[role] === player.id}
              disabled={taker && isGoalkeeper && roles[role] !== player.id}
              onChange={() => onToggleRole(role)}
              sx={{ minHeight: 36, px: 1.25, textTransform: 'none' }}
            >
              {t(`lineup.roles.${key}`)}
            </ToggleButton>
          ))}
        </Stack>
      )}
      <Link component={RouterLink} to={`/player/${player.id}`} underline="hover" variant="body2" sx={{ display: 'inline-flex', alignItems: 'center', minHeight: 44, ml: 0.5 }}>
        {t('playerCard.viewProfile')}
      </Link>
      <PlayerStatsPopover player={player} anchorEl={anchor} onClose={() => setAnchor(null)} />
    </Paper>
  )
}
