import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  POSITION_DEFENSE,
  POSITION_GOALKEEPER,
  POSITION_LATERAL_DEFENSE_LEFT,
  POSITION_LATERAL_DEFENSE_RIGHT,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_LATERAL_MIDFIELD_RIGHT,
  POSITION_LATERAL_STRIKER_LEFT,
  POSITION_LATERAL_STRIKER_RIGHT,
  POSITION_MIDFIELD,
  POSITION_SHORT_LABEL_KEYS,
  POSITION_STRIKER,
  type Lineup,
  type Player,
  type PlayerRating,
} from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { MATCH_RATING_MIN_MINUTES } from '../../utils/liveMatchRatings'
import { matchRatingColor } from '../../utils/matchRating'
import { CountryFlag } from '../../components/CountryFlag'

// Shared by MatchRatingsTab (a finished match's report) and MatchMinuteRatings (live, or scrubbed
// back to an earlier minute) - both end up with the same PlayerRating records (stored by the
// backend, or rebuilt by utils/liveMatchRatings for an earlier minute), so they render through
// this one column: one row per player who featured, ordered by position on the pitch (goalkeeper,
// defenders, midfielders, strikers - each line left to right), or by rating.
export function TeamRatingsColumn({
  teamId,
  title,
  startingLineup,
  players,
  ratings,
  sortByRating = false,
}: {
  teamId: number
  title: string
  // The XI this side kicked off with - only its order is used, to list the starters.
  startingLineup: Lineup | null
  players: Record<string, Player>
  // This side's ratings only.
  ratings: PlayerRating[]
  // Best rating first (unrated last) instead of the positional order.
  sortByRating?: boolean
}) {
  const startingIndex = new Map((startingLineup?.positions ?? []).map((pos, index) => [pos.playerId, index]))
  // Within one position: starters first in the XI's own order, then substitutes in the order they
  // came on.
  const byLineup = (a: PlayerRating, b: PlayerRating) => {
    const aStarter = a.cameOn == null
    const bStarter = b.cameOn == null
    if (aStarter !== bStarter) {
      return aStarter ? -1 : 1
    }
    if (aStarter) {
      return (
        (startingIndex.get(a.playerId) ?? Number.MAX_SAFE_INTEGER) - (startingIndex.get(b.playerId) ?? Number.MAX_SAFE_INTEGER) ||
        a.playerId - b.playerId
      )
    }
    return (a.cameOn ?? 0) - (b.cameOn ?? 0) || a.playerId - b.playerId
  }
  const byPosition = (a: PlayerRating, b: PlayerRating) => positionRank(a.position) - positionRank(b.position) || byLineup(a, b)
  const byRating = (a: PlayerRating, b: PlayerRating) => {
    if (a.rated !== b.rated) {
      return a.rated ? -1 : 1
    }
    return b.rating - a.rating || byPosition(a, b)
  }
  const rows = [...ratings].sort(sortByRating ? byRating : byPosition)

  return (
    <Stack spacing={1.25}>
      <Typography variant="subtitle2">
        <TeamNameLink teamId={teamId}>{title}</TeamNameLink>
      </Typography>
      {rows.map((rating) => {
        const player = players[String(rating.playerId)]
        if (!player) {
          return null
        }
        return <RatingRow key={rating.playerId} rating={rating} player={player} />
      })}
    </Stack>
  )
}

// Goalkeeper, then defenders, midfielders and strikers, each line left to right - the position ids
// themselves list the centre before the flanks.
const POSITION_ORDER = [
  POSITION_GOALKEEPER,
  POSITION_LATERAL_DEFENSE_LEFT,
  POSITION_DEFENSE,
  POSITION_LATERAL_DEFENSE_RIGHT,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_MIDFIELD,
  POSITION_LATERAL_MIDFIELD_RIGHT,
  POSITION_LATERAL_STRIKER_LEFT,
  POSITION_STRIKER,
  POSITION_LATERAL_STRIKER_RIGHT,
]

function positionRank(position: number): number {
  const rank = POSITION_ORDER.indexOf(position)
  return rank < 0 ? POSITION_ORDER.length : rank
}

function RatingRow({ rating, player }: { rating: PlayerRating; player: Player }) {
  const { t } = useTranslation()
  const positionKey = POSITION_SHORT_LABEL_KEYS[rating.position]
  const details = [positionKey ? t(positionKey) : null, minutesLabel(rating, t)].filter(Boolean).join(' · ')

  return (
    <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
      <CartoonAvatar seed={rating.playerId} birthCountry={player.birthCountry} size={32} />
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography variant="body2" noWrap>
          <CountryFlag code={player.birthCountry} height={14} marginRight={0.5} />
          <PlayerNameLink playerId={rating.playerId}>
            {player.firstName} {player.lastName}
          </PlayerNameLink>
        </Typography>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', minWidth: 0 }}>
          <Typography variant="caption" color="text.secondary" noWrap>
            {details}
          </Typography>
          {rating.own !== undefined && rating.rival !== undefined && <ContributionLabel own={rating.own} rival={rating.rival} />}
        </Stack>
      </Box>
      <Tooltip title={<RatingBreakdown rating={rating} />} enterTouchDelay={0} leaveTouchDelay={4000} arrow>
        {rating.rated ? (
          <Chip size="small" color={matchRatingColor(rating.rating)} label={rating.rating.toFixed(1)} sx={{ fontWeight: 700, minWidth: 44 }} />
        ) : (
          <Chip size="small" variant="outlined" label={t('match.ratings.unrated')} sx={{ minWidth: 44 }} />
        )}
      </Tooltip>
    </Stack>
  )
}

function minutesLabel(rating: PlayerRating, t: (key: string, options?: Record<string, unknown>) => string): string {
  const parts = [t('match.ratings.minutes', { minutes: rating.minutes })]
  if (rating.cameOn !== undefined) {
    parts.push(t('match.ratings.cameOn', { minute: rating.cameOn }))
  }
  if (rating.wentOff !== undefined) {
    parts.push(t('match.ratings.wentOff', { minute: rating.wentOff }))
  }
  return parts.join(' ')
}

// "58 vs 45": the player's mean zone contribution against the rival's, the own figure tinted by
// who won the duel so the edge reads at a glance (the number is always shown, never color alone).
function ContributionLabel({ own, rival }: { own: number; rival: number }) {
  const { t } = useTranslation()
  const color = own > rival ? 'success.main' : own < rival ? 'error.main' : 'text.secondary'
  return (
    <Tooltip title={t('match.ratings.contributionHint')} enterTouchDelay={0} arrow>
      <Typography variant="caption" color="text.secondary" noWrap sx={{ flexShrink: 0 }}>
        <Box component="span" sx={{ color, fontWeight: 700 }}>
          {Math.round(own)}
        </Box>{' '}
        {t('match.ratings.versus', { rival: Math.round(rival) })}
      </Typography>
    </Tooltip>
  )
}

function signed(value: number): string {
  return `${value > 0 ? '+' : ''}${value.toFixed(1)}`
}

/** "Base 6.0 · Zona +0.5 · Goles +2.8 · Resultado +0.3" - every non-zero component, each to one
 * decimal, so it adds up to the rating (bar the [1, 10] clamp). */
export function RatingBreakdown({ rating }: { rating: PlayerRating }) {
  const { t } = useTranslation()
  const parts = [`${t('match.ratings.breakdown.base')} 6.0`]
  const components: [string, number][] = [
    ['zone', rating.zone],
    ['goals', rating.goals],
    ['cards', rating.cards],
    ['injury', rating.injury],
    ['result', rating.result],
  ]
  for (const [key, value] of components) {
    if (value !== 0) {
      parts.push(`${t(`match.ratings.breakdown.${key}`)} ${signed(value)}`)
    }
  }
  return (
    <Box>
      <Typography variant="caption" sx={{ display: 'block' }}>
        {parts.join(' · ')}
      </Typography>
      {!rating.rated && (
        <Typography variant="caption" sx={{ display: 'block', mt: 0.5 }}>
          {t('match.ratings.unratedHint', { minutes: MATCH_RATING_MIN_MINUTES })}
        </Typography>
      )}
    </Box>
  )
}
