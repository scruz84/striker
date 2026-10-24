import { formatDateTime } from './formatDate'

const WEATHER_EMOJI: Record<string, string> = {
  sunny: '☀️',
  cloudy: '☁️',
  rain: '🌧️',
  windy: '🌬️',
}

// "Jornada 3 de 38 · 21/10/2026 20:00 · ☀️" - the competition label (matchday, cup round, "Amistoso"...),
// kickoff date and weather that sit inside the scoreboard card, the same for every competition. Any
// part that is unknown (no label while the cup details load, no date, no stored weather) is left
// out instead of shown blank.
export function matchSubtitle(label: string | undefined, matchDate: string | undefined, weather: string | undefined): string {
  const emoji = weather ? WEATHER_EMOJI[weather] : undefined
  return [label, matchDate ? formatDateTime(matchDate) : undefined, emoji].filter(Boolean).join(' · ')
}
