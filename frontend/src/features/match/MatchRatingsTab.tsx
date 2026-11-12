import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { matchTimeline } from '../../utils/matchMinute'
import { MatchMinuteRatings } from './MatchMinuteRatings'
import type { MatchView } from './matchViewModel'

// A "Notas" tab: every player who featured, grouped by team, with the rating the backend stored
// for them at full time (see backend common.ComputePlayerRatings and openspec/specs/match-ratings),
// and a scrubber to see the ratings as they stood at any earlier minute.
export function MatchRatingsTab({ match }: { match: MatchView }) {
  const { t } = useTranslation()

  // Without stored ratings in the payload (a response cached before the backend started sending
  // them), the scrubber still rebuilds them from the lineups, instants and events - it just has no
  // stored record to show at the final minute.
  const hasRatings = (match.playerRatings?.length ?? 0) > 0 || (match.homeLineup !== null && match.awayLineup !== null)
  if (!hasRatings || !match.homePlayers || !match.awayPlayers) {
    return <Typography color="text.secondary">{t('match.noLineupData')}</Typography>
  }

  return (
    <MatchMinuteRatings
      homeId={match.homeId}
      homeName={match.homeName}
      awayId={match.awayId}
      awayName={match.awayName}
      homeLineup={match.homeLineup}
      awayLineup={match.awayLineup}
      homePlayers={match.homePlayers}
      awayPlayers={match.awayPlayers}
      events={match.events ?? []}
      instants={match.matchInstants ?? []}
      timeline={matchTimeline(match.matchInstants ?? [], match.events ?? [])}
      finalRatings={match.playerRatings?.length ? match.playerRatings : undefined}
    />
  )
}
