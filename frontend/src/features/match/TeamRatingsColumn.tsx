import Chip from '@mui/material/Chip'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import type { Lineup, Player } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { matchRatingColor } from '../../utils/matchRating'

// Shared by MatchRatingsTab (a finished match's report) and MatchMinuteRatings (live, or scrubbed
// back to an earlier minute) - both end up with the same shape of ratings (keyed by playerId as a
// string), just computed differently, so they render through this one column.
export function TeamRatingsColumn({
  teamId,
  title,
  lineup,
  players,
  ratings,
  sortByRating = false,
}: {
  teamId: number
  title: string
  lineup: Lineup
  players: Record<string, Player>
  ratings: Record<string, number>
  // Best rating first instead of the lineup's own positional order.
  sortByRating?: boolean
}) {
  const positions = sortByRating
    ? [...lineup.positions].sort((a, b) => (ratings[String(b.playerId)] ?? 0) - (ratings[String(a.playerId)] ?? 0))
    : lineup.positions
  return (
    <Stack spacing={1.25}>
      <Typography variant="subtitle2">
        <TeamNameLink teamId={teamId}>{title}</TeamNameLink>
      </Typography>
      {positions.map((pos) => {
        const player = players[String(pos.playerId)]
        const rating = ratings[String(pos.playerId)]
        if (!player || rating === undefined) {
          return null
        }
        return (
          <Stack key={pos.playerId} direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
            <CartoonAvatar seed={pos.playerId} birthCountry={player.birthCountry} size={32} />
            <Typography variant="body2" noWrap sx={{ flex: 1, minWidth: 0 }}>
              <PlayerNameLink playerId={pos.playerId}>
                {player.firstName} {player.lastName}
              </PlayerNameLink>
            </Typography>
            <Chip size="small" color={matchRatingColor(rating)} label={rating.toFixed(1)} sx={{ fontWeight: 700 }} />
          </Stack>
        )
      })}
    </Stack>
  )
}
