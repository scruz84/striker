import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Link from '@mui/material/Link'
import Popover from '@mui/material/Popover'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { Player } from '../api/types'
import { overallRating, SKILL_FIELDS } from '../utils/playerRating'
import { CartoonAvatar } from './avatar/CartoonAvatar'
import { SkillBar } from './SkillBar'

/** A player's full skill breakdown in a popover - anchored to whatever was clicked to open it
 * (a pitch slot or bench chip in the lineup editor), so a manager can check what a player is
 * actually good at before deciding where to drop them, without leaving the pitch. */
export function PlayerStatsPopover({
  player,
  anchorEl,
  onClose,
}: {
  player: Player
  anchorEl: HTMLElement | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const theme = useTheme()
  const overall = overallRating(player)
  const ratingColor =
    overall >= 65 ? theme.palette.success.main : overall >= 45 ? theme.palette.warning.main : theme.palette.error.main

  const traits: Array<{ key: string; label: string }> = [
    player.header && { key: 'header', label: t('playerCard.traits.aerial') },
    player.quick && { key: 'quick', label: t('playerCard.traits.pace') },
    player.creative && { key: 'creative', label: t('playerCard.traits.creative') },
    player.technical && { key: 'technical', label: t('playerCard.traits.technical') },
  ].filter((trait): trait is { key: string; label: string } => !!trait)

  return (
    <Popover
      open={Boolean(anchorEl)}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorOrigin={{ vertical: 'center', horizontal: 'right' }}
      transformOrigin={{ vertical: 'center', horizontal: 'left' }}
      slotProps={{ paper: { sx: { p: 2, width: 260 } } }}
    >
      <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', mb: 1.5 }}>
        <CartoonAvatar seed={player.id} size={48} />
        <Box sx={{ minWidth: 0, flexGrow: 1 }}>
          <Typography variant="subtitle2" noWrap sx={{ fontWeight: 600 }}>
            {player.firstName} {player.lastName}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {t('playerCard.age', { age: player.age })}
          </Typography>
        </Box>
        <Box
          sx={{
            width: 32,
            height: 32,
            flexShrink: 0,
            borderRadius: '50%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            bgcolor: ratingColor,
            color: theme.palette.getContrastText(ratingColor),
            fontWeight: 700,
            fontSize: '0.8rem',
          }}
          title={t('playerCard.overall')}
        >
          {overall}
        </Box>
      </Stack>
      {traits.length > 0 && (
        <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', mb: 1.5 }}>
          {traits.map((trait) => (
            <Chip key={trait.key} label={trait.label} size="small" variant="outlined" />
          ))}
        </Stack>
      )}
      {SKILL_FIELDS.map((field) => (
        <SkillBar
          key={field}
          label={t(`playerCard.skills.${field === 'goalKeeper' ? 'goalkeeper' : field}`)}
          value={player[field]}
        />
      ))}
      <Link component={RouterLink} to={`/player/${player.id}`} underline="hover" variant="body2" sx={{ display: 'block', mt: 1.5 }}>
        {t('playerCard.viewProfile')}
      </Link>
    </Popover>
  )
}
