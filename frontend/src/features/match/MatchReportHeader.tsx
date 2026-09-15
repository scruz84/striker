import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { WeekMatch } from '../../api/types'
import { formatDateTime } from '../../utils/formatDate'

const WEATHER_EMOJI: Record<string, string> = {
  sunny: '☀️',
  cloudy: '☁️',
  rain: '🌧️',
  windy: '🌬️',
}

// A small strip above the Scoreboard - competition/matchday on the left, kickoff date and
// weather on the right - the Hattrick-style "medal + jornada + weather icon" row above a match's
// score. weekTotal is the division's total number of matchdays this season (derived from its
// team count - see MatchDetailsPage), omitted (falls back to just the week number) when the
// division details fetch hasn't resolved yet or the division no longer exists.
export function MatchReportHeader({ match, weekTotal }: { match: WeekMatch; weekTotal: number | undefined }) {
  const { t } = useTranslation()

  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between', px: 0.5, mb: 1, flexWrap: 'wrap' }}>
      <Typography variant="body2" color="text.secondary">
        🏆{' '}
        {weekTotal ? t('match.matchday', { week: match.weekNumber, total: weekTotal }) : t('division.weekChip', { week: match.weekNumber })}
      </Typography>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <Typography variant="body2" color="text.secondary">
          {formatDateTime(match.matchDate)}
        </Typography>
        {match.weather && (
          <Typography component="span" aria-label={t(`match.weather.${match.weather}`)} title={t(`match.weather.${match.weather}`)}>
            {WEATHER_EMOJI[match.weather]}
          </Typography>
        )}
      </Stack>
    </Stack>
  )
}
