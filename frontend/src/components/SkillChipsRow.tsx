import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { Player } from '../api/types'
import { CONDITION_FIELDS, CONDITION_SHORT_LABEL_KEYS, ratingTierColor, SKILL_FIELDS, SKILL_SHORT_LABEL_KEYS } from '../utils/playerRating'

function StatChip({ labelKey, value }: { labelKey: string; value: number }) {
  const { t } = useTranslation()
  const theme = useTheme()
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'baseline',
        gap: 0.375,
        px: 0.625,
        py: 0.125,
        borderRadius: 999,
        bgcolor: 'action.hover',
      }}
    >
      <Typography component="span" sx={{ fontSize: '0.6875rem', fontWeight: 600, color: 'text.secondary', lineHeight: 1.4 }}>
        {t(labelKey)}
      </Typography>
      <Typography component="span" sx={{ fontSize: '0.6875rem', fontWeight: 700, lineHeight: 1.4, color: ratingTierColor(value, theme.palette) }}>
        {value}
      </Typography>
    </Box>
  )
}

// A compact one-line strip of all seven skills plus Stamina/Form, abbreviated to three letters and
// colored by tier (see ratingTierColor) - lets a listing card (transfer market, squad, lineup
// bench) show a player's actual strengths, weaknesses, and current condition without opening their
// profile, in far less vertical space than PlayerStatsPopover's full-width SkillBar rows. Stamina/
// Form are appended after the seven skills rather than mixed in among them - they're conceptually
// different (see CONDITION_FIELDS) even though they share the same scale and coloring.
export function SkillChipsRow({ player }: { player: Player }) {
  return (
    <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', rowGap: 0.5, mt: 0.5 }}>
      {SKILL_FIELDS.map((field) => (
        <StatChip key={field} labelKey={SKILL_SHORT_LABEL_KEYS[field]} value={player[field]} />
      ))}
      {CONDITION_FIELDS.map((field) => (
        <StatChip key={field} labelKey={CONDITION_SHORT_LABEL_KEYS[field]} value={player[field]} />
      ))}
    </Stack>
  )
}
