import type { TFunction } from 'i18next'
import type { LeagueLiveScore } from '../api/types'
import { intlLocale } from './locale'
import { matchMinuteText } from './matchMinute'

export interface LiveScoreLabel {
  text: string
  // 'error' while the match is in play (pulsing), 'default' for half-time/FT/pending.
  color: 'error' | 'default'
  pulsing: boolean
}

function kickoffTime(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(intlLocale(), { hour: '2-digit', minute: '2-digit', hour12: false }).format(date)
}

// The state chip of a division live score, shared by the dashboard's "rest of the matchday" list and
// the league page's matchday cards: "63'" (or "90+2'" in added time) while live, "Half-time",
// "FT"/"Final" once over, and the kickoff time while the match has not started broadcasting.
export function liveScoreLabel(entry: Pick<LeagueLiveScore, 'status' | 'minute' | 'addedMinute' | 'matchDate'>, t: TFunction): LiveScoreLabel {
  switch (entry.status) {
    case 'live':
      return { text: t('dashboard.liveMatchMinute', { minute: matchMinuteText(entry.minute, entry.addedMinute) }), color: 'error', pulsing: true }
    case 'halftime':
      return { text: t('match.halftime'), color: 'default', pulsing: false }
    case 'finished':
      return { text: t('liveScores.fullTime'), color: 'default', pulsing: false }
    default:
      return { text: kickoffTime(entry.matchDate), color: 'default', pulsing: false }
  }
}
