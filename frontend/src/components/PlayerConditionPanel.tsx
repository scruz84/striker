import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { Player } from '../api/types'
import { ratingTierColor } from '../utils/playerRating'

/** Stamina and Form, the player details page's second attribute panel - deliberately separate
 * from PlayerAttributesPanel's seven core skills (and never folded into SKILL_FIELDS/
 * overallRating, see playerRating.ts), since neither measures footballing quality: Stamina is
 * physical conditioning that fatigues a player's in-match skills minute by minute, Form is recent
 * performance that nudges match ratings up or down match by match. */
export function PlayerConditionPanel({ player }: { player: Player }) {
  const { t } = useTranslation()
  const theme = useTheme()

  return (
    <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2.5 } }}>
      <Typography variant="subtitle1" sx={{ fontWeight: 700, mb: { xs: 1, sm: 2 } }}>
        {t('playerPage.condition.title')}
      </Typography>
      <Stack spacing={{ xs: 0.75, sm: 1.25 }}>
        <ConditionBar
          label={t('playerPage.condition.stamina')}
          hint={t('playerPage.condition.staminaHint')}
          value={player.stamina}
          color={ratingTierColor(player.stamina, theme.palette)}
        />
        <ConditionBar
          label={t('playerPage.condition.form')}
          hint={t('playerPage.condition.formHint')}
          value={player.form}
          color={ratingTierColor(player.form, theme.palette)}
        />
      </Stack>
    </Paper>
  )
}

function ConditionBar({ label, hint, value, color }: { label: string; hint: string; value: number; color: string }) {
  return (
    <Stack direction="row" sx={{ alignItems: 'center', gap: { xs: 1, sm: 1.5 } }}>
      <Tooltip title={hint}>
        <Typography variant="body2" noWrap sx={{ width: { xs: 76, sm: 100 }, flexShrink: 0 }}>
          {label}
        </Typography>
      </Tooltip>
      <Box sx={{ flex: 1, height: 6, borderRadius: 1, bgcolor: 'action.hover', overflow: 'hidden' }}>
        <Box sx={{ display: 'block', width: `${Math.min(value, 100)}%`, height: '100%', bgcolor: color, borderRadius: 1 }} />
      </Box>
      <Typography variant="body2" color="text.secondary" sx={{ width: 24, textAlign: 'right', flexShrink: 0, fontWeight: 600 }}>
        {value}
      </Typography>
    </Stack>
  )
}
