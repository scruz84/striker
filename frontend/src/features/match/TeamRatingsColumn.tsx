import Chip from '@mui/material/Chip'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import type { Lineup, Player } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { matchRatingColor } from '../../utils/matchRating'

// Shared by MatchRatingsTab (a finished match's report) and MatchMinuteRatings (live, or scrubbed
// back to an earlier minute) - both end up with the same shape of ratings (keyed by playerId as a
// string), just computed differently, so they render through this one column.
export function TeamRatingsColumn({
  title,
  lineup,
  players,
  ratings,
}: {
  title: string
  lineup: Lineup
  players: Record<string, Player>
  ratings: Record<string, number>
}) {
  return (
    <Stack spacing={1.25}>
      <Typography variant="subtitle2">{title}</Typography>
      {lineup.positions.map((pos) => {
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
