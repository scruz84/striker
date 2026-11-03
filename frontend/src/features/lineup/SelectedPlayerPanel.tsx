import Box from '@mui/material/Box'
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
import { SkillChipsRow } from '../../components/SkillChipsRow'
import { bestCorePosition, overallRating, ratingTierColor } from '../../utils/playerRating'
import { CountryFlag } from '../../components/CountryFlag'
import { TOUCH } from '../../touchMedia'

export type LineupRole = keyof LineupRoles

/** The three lineup roles in the order the panel shows them, with their i18n key suffix. */
export const LINEUP_ROLES: { role: LineupRole; key: string; taker: boolean }[] = [
  { role: 'captainId', key: 'captain', taker: false },
  { role: 'penaltyTakerId', key: 'penalties', taker: true },
  { role: 'freeKickTakerId', key: 'freeKicks', taker: true },
]

/** The compact card shown under the pitch while a player is picked up: who they are, their rating,
 * every skill at a glance (SkillChipsRow - always visible, since it's what a manager weighs when
 * placing a player or handing out a role) and a link to the full profile. Tapping a pitch slot only
 * ever moves; the details live here. With `roles` it also carries the three lineup-role toggles
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
  const rating = overallRating(player)

  return (
    <Paper variant="outlined" sx={{ px: 1, py: 0.75, mt: 1, borderColor: 'warning.main', borderWidth: 2 }} aria-live="polite">
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={32} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Typography variant="body2" noWrap sx={{ fontWeight: 600, lineHeight: 1.25 }}>
            <CountryFlag code={player.birthCountry} height={14} marginRight={0.5} />
            {player.firstName} {player.lastName}
          </Typography>
          <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
            <Chip
              size="small"
              variant="outlined"
              translate="no"
              className="notranslate"
              label={t(POSITION_SHORT_LABEL_KEYS[bestCorePosition(player)])}
              sx={{ height: 18, '& .MuiChip-label': { px: 0.75 } }}
            />
            <Typography variant="caption" color="text.secondary" noWrap>
              {t('lineup.ageShort', { age: player.age })}
            </Typography>
            <Typography variant="caption" sx={{ fontWeight: 700, color: ratingTierColor(rating, theme.palette) }}>
              {rating}
            </Typography>
          </Stack>
        </Box>
        <IconButton size="small" onClick={onClear} aria-label={t('lineup.selected.clear')}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </Stack>
      <SkillChipsRow player={player} />
      <Stack direction="row" spacing={0.5} sx={{ mt: 0.75, alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5 }}>
        {roles && onToggleRole && (
          <Stack direction="row" spacing={0.5} role="group" aria-label={t('lineup.roles.title')}>
            {LINEUP_ROLES.map(({ role, key, taker }) => (
              <ToggleButton
                key={role}
                value={role}
                size="small"
                color="primary"
                selected={roles[role] === player.id}
                disabled={taker && isGoalkeeper && roles[role] !== player.id}
                onChange={() => onToggleRole(role)}
                sx={{ minHeight: 32, [TOUCH]: { minHeight: 36 }, py: 0.25, px: 1, textTransform: 'none', lineHeight: 1.2 }}
              >
                {t(`lineup.roles.${key}`)}
              </ToggleButton>
            ))}
          </Stack>
        )}
        <Link
          component={RouterLink}
          to={`/player/${player.id}`}
          underline="hover"
          variant="body2"
          sx={{ display: 'inline-flex', alignItems: 'center', minHeight: 32, ml: 'auto !important', pl: 0.5, whiteSpace: 'nowrap' }}
        >
          {t('lineup.selected.profile')}
        </Link>
      </Stack>
    </Paper>
  )
}
