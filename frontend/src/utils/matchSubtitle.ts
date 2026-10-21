import type { WeekMatch } from '../api/types'
import { formatDateTime } from './formatDate'

const WEATHER_EMOJI: Record<string, string> = {
  sunny: '☀️',
  cloudy: '☁️',
  rain: '🌧️',
  windy: '🌬️',
}

// "Jornada 3 de 38 · 21/10/2026 20:00 · ☀️" - the matchday, kickoff date and weather that used to
// sit in a separate strip above the scoreboard, now inside its card. weekTotal is the division's
// total number of matchdays (derived from its team count), omitted - falling back to just the week
// number - while the division details haven't loaded or if the division no longer exists.
export function matchSubtitle(match: WeekMatch, weekTotal: number | undefined, t: (key: string, options?: Record<string, unknown>) => string): string {
  const matchday = weekTotal ? t('match.matchday', { week: match.weekNumber, total: weekTotal }) : t('division.weekChip', { week: match.weekNumber })
  const weather = match.weather ? WEATHER_EMOJI[match.weather] : undefined
  return [matchday, formatDateTime(match.matchDate), weather].filter(Boolean).join(' · ')
}
