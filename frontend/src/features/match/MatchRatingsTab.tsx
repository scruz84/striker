import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
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
      currentMinute={finalMatchMinute(match)}
      finalRatings={match.playerRatings?.length ? match.playerRatings : undefined}
    />
  )
}

// finalMatchMinute is the highest minute a scrubber needs to reach for a finished match - normally
// the last recorded MatchInstant's minute, but events can (rarely) carry a slightly later minute
// than the last instant (e.g. a stoppage-time card), and a friendly/cup tie without instants has
// none at all, so this takes whichever of the two is higher, floored at a full 90 minutes - the
// same rule as the backend's MatchFinalMinute, the minute its stored ratings were computed at.
function finalMatchMinute(match: MatchView): number {
  const instants = match.matchInstants ?? []
  const events = match.events ?? []
  const lastInstantMinute = instants.length > 0 ? instants[instants.length - 1].minute : 0
  const lastEventMinute = events.reduce((max, event) => Math.max(max, event.minute), 0)
  return Math.max(lastInstantMinute, lastEventMinute, 90)
}
