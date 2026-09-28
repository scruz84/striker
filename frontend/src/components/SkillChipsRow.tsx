import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { Player } from '../api/types'
import { ratingTierColor, SKILL_FIELDS, SKILL_SHORT_LABEL_KEYS } from '../utils/playerRating'

// A compact one-line strip of all seven skills, abbreviated to three letters and colored by tier
// (see ratingTierColor) - lets a listing card (transfer market) or lineup card show a player's
// actual strengths and weaknesses without opening their profile, in far less vertical space than
// PlayerStatsPopover's full-width SkillBar rows.
export function SkillChipsRow({ player }: { player: Player }) {
  const { t } = useTranslation()
  const theme = useTheme()
  return (
    <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', rowGap: 0.5, mt: 0.5 }}>
      {SKILL_FIELDS.map((field) => {
        const value = player[field]
        return (
          <Box
            key={field}
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
            <Typography component="span" sx={{ fontSize: '0.62rem', fontWeight: 600, color: 'text.secondary', lineHeight: 1.4 }}>
              {t(SKILL_SHORT_LABEL_KEYS[field])}
            </Typography>
            <Typography component="span" sx={{ fontSize: '0.68rem', fontWeight: 700, lineHeight: 1.4, color: ratingTierColor(value, theme.palette) }}>
              {value}
            </Typography>
          </Box>
        )
      })}
    </Stack>
  )
}
