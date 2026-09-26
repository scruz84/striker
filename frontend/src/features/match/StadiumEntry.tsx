import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'

// Shown once, right as the lineups themselves appear (the same moment the match actually kicks
// off - see useLiveMatch's "lineups" message, published right before the first minute), so a
// viewer who opens the page ahead of kickoff sees the teams take to the pitch rather than the
// scoreboard silently switching from "Scheduled" to a live minute with no transition. Omitted
// when there's no stadium to name (a friendly - see WeekMatch.homeStadiumName's own comment).
export function StadiumEntry({ stadiumName, homeName, awayName }: { stadiumName: string; homeName: string; awayName: string }) {
  const { t } = useTranslation()
  if (!stadiumName) {
    return null
  }
  return (
    <Paper variant="outlined" sx={{ p: 2, mb: 2, textAlign: 'center' }}>
      <Stack spacing={0.5} sx={{ alignItems: 'center' }}>
        <Typography sx={{ fontSize: '1.5rem', lineHeight: 1 }} aria-hidden="true">
          🏟️
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {t('match.stadiumEntry', { home: homeName, away: awayName, stadium: stadiumName })}
        </Typography>
      </Stack>
    </Paper>
  )
}
