import Box from '@mui/material/Box'
import Card from '@mui/material/Card'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { Player } from '../api/types'
import { overallRating, SKILL_FIELDS } from '../utils/playerRating'
import { CartoonAvatar } from './avatar/CartoonAvatar'
import { SkillBar } from './SkillBar'

export function PlayerCard({ player }: { player: Player }) {
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
    <Card variant="outlined" sx={{ p: 2, height: '100%', position: 'relative', overflow: 'visible' }}>
      <Box
        sx={{
          position: 'absolute',
          top: 8,
          right: 8,
          width: 34,
          height: 34,
          borderRadius: '50%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: ratingColor,
          color: theme.palette.getContrastText(ratingColor),
          fontWeight: 700,
          fontSize: '0.85rem',
        }}
        title={t('playerCard.overall')}
      >
        {overall}
      </Box>
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, sm: 3 }}>
          <Stack spacing={1} sx={{ alignItems: { xs: 'flex-start', sm: 'center' } }}>
            <Box sx={{ flexShrink: 0 }}>
              <CartoonAvatar seed={player.id} size={72} />
            </Box>
            <Box sx={{ textAlign: { xs: 'left', sm: 'center' } }}>
              <Typography variant="subtitle1" sx={{ fontWeight: 600, lineHeight: 1.2 }}>
                {player.firstName} {player.lastName}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {t('playerCard.age', { age: player.age })}
              </Typography>
            </Box>
            {traits.length > 0 && (
              <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', justifyContent: { xs: 'flex-start', sm: 'center' } }}>
                {traits.map((trait) => (
                  <Chip key={trait.key} label={trait.label} size="small" variant="outlined" />
                ))}
              </Stack>
            )}
          </Stack>
        </Grid>
        <Grid size={{ xs: 12, sm: 9 }}>
          <Grid container columnSpacing={3}>
            {SKILL_FIELDS.map((field) => (
              <Grid key={field} size={{ xs: 12, sm: 6 }}>
                <SkillBar label={t(`playerCard.skills.${field === 'goalKeeper' ? 'goalkeeper' : field}`)} value={player[field]} />
              </Grid>
            ))}
          </Grid>
        </Grid>
      </Grid>
    </Card>
  )
}
