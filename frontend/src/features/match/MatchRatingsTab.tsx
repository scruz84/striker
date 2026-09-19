import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { Lineup, Player, WeekMatch } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerNameLink } from '../../components/PlayerNameLink'

// Most players cluster around the 6.0 base rating (see backend common.ComputeMatchRatings) -
// only a genuinely strong or poor showing should read as green/red, so "default" (a neutral
// gray) covers that whole unremarkable middle band rather than warning-orange swallowing it.
function ratingColor(rating: number): 'success' | 'error' | 'default' {
  if (rating >= 7.5) return 'success'
  if (rating < 4) return 'error'
  return 'default'
}

// A "Calificaciones" tab: every player who featured, grouped by team, with the 1-10 rating
// backend common.ComputeMatchRatings derived for them - see that function's own doc comment for
// how the rating is computed (it's a simple heuristic, not a real performance model).
export function MatchRatingsTab({ match }: { match: WeekMatch }) {
  const { t } = useTranslation()

  if (!match.ratings || !match.homeLineup || !match.awayLineup || !match.homePlayers || !match.awayPlayers) {
    return <Typography color="text.secondary">{t('match.noLineupData')}</Typography>
  }

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, sm: 6 }}>
        <TeamRatingsColumn title={match.homeName} lineup={match.homeLineup} players={match.homePlayers} ratings={match.ratings} />
      </Grid>
      <Grid size={{ xs: 12, sm: 6 }}>
        <TeamRatingsColumn title={match.awayName} lineup={match.awayLineup} players={match.awayPlayers} ratings={match.ratings} />
      </Grid>
    </Grid>
  )
}

function TeamRatingsColumn({
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
            <Chip size="small" color={ratingColor(rating)} label={rating.toFixed(1)} sx={{ fontWeight: 700 }} />
          </Stack>
        )
      })}
    </Stack>
  )
}
