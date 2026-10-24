import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { MatchMinuteRatings } from './MatchMinuteRatings'
import type { MatchView } from './matchViewModel'
import { TeamRatingsColumn } from './TeamRatingsColumn'

// A "Calificaciones" tab: every player who featured, grouped by team, with the 1-10 rating
// backend common.ComputeMatchRatings derived for them - see that function's own doc comment for
// how the rating is computed (it's a simple heuristic, not a real performance model). When the
// match's minute-by-minute events are available, this hands off to MatchMinuteRatings so a viewer
// can scrub back to see the ratings as they stood at any earlier minute, rather than only ever
// seeing the final whistle's numbers.
export function MatchRatingsTab({ match }: { match: MatchView }) {
  const { t } = useTranslation()

  if (!match.ratings || !match.homeLineup || !match.awayLineup || !match.homePlayers || !match.awayPlayers) {
    return <Typography color="text.secondary">{t('match.noLineupData')}</Typography>
  }

  if (match.events && match.events.length > 0) {
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
        events={match.events}
        currentMinute={finalMatchMinute(match)}
        finalRatings={match.ratings}
      />
    )
  }

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, sm: 6 }}>
        <TeamRatingsColumn teamId={match.homeId} title={match.homeName} lineup={match.homeLineup} players={match.homePlayers} ratings={match.ratings} />
      </Grid>
      <Grid size={{ xs: 12, sm: 6 }}>
        <TeamRatingsColumn teamId={match.awayId} title={match.awayName} lineup={match.awayLineup} players={match.awayPlayers} ratings={match.ratings} />
      </Grid>
    </Grid>
  )
}

// finalMatchMinute is the highest minute a scrubber needs to reach for a finished match - normally
// the last recorded MatchInstant's minute, but events can (rarely) carry a slightly later minute
// than the last instant (e.g. a stoppage-time card), and a friendly/cup tie without instants has
// none at all, so this takes whichever of the two is higher, floored at a full 90 minutes.
function finalMatchMinute(match: MatchView): number {
  const instants = match.matchInstants ?? []
  const events = match.events ?? []
  const lastInstantMinute = instants.length > 0 ? instants[instants.length - 1].minute : 0
  const lastEventMinute = events.reduce((max, event) => Math.max(max, event.minute), 0)
  return Math.max(lastInstantMinute, lastEventMinute, 90)
}
