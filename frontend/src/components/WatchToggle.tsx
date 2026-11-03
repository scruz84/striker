import IconButton from '@mui/material/IconButton'
import Tooltip from '@mui/material/Tooltip'
import StarIcon from '@mui/icons-material/Star'
import StarBorderIcon from '@mui/icons-material/StarBorder'
import { useTranslation } from 'react-i18next'
import { useMe, useUnwatchPlayer, useUserTeam, useWatchlist, useWatchPlayer } from '../api/hooks'
import { MAX_WATCHLIST_SIZE } from '../api/types'

/** The star that follows / unfollows a player (openspec player-watchlist "Following players").
 * Shown only to a signed-in manager, and never on one of his own players; every star on screen
 * reads the same cached watchlist, so toggling one updates them all. */
export function WatchToggle({
  player,
  size = 'medium',
}: {
  player: { id: number; teamId: number; firstName: string; lastName: string }
  size?: 'small' | 'medium'
}) {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { data: userTeam } = useUserTeam(!!me)
  const ownTeamId = userTeam?.team.id
  const { data: watchlist } = useWatchlist(ownTeamId !== undefined)
  const watch = useWatchPlayer()
  const unwatch = useUnwatchPlayer()

  if (ownTeamId === undefined || player.teamId === ownTeamId || !watchlist) {
    return null
  }

  const name = `${player.firstName} ${player.lastName}`
  const watched = watchlist.some((entry) => entry.player.id === player.id)
  const full = !watched && watchlist.length >= MAX_WATCHLIST_SIZE
  const busy = watch.isPending || unwatch.isPending
  const label = watched ? t('watchlist.unfollow', { name }) : t('watchlist.follow', { name })

  return (
    <Tooltip title={full ? t('watchlist.full', { count: MAX_WATCHLIST_SIZE }) : label}>
      <span>
        <IconButton
          size={size}
          aria-label={label}
          aria-pressed={watched}
          disabled={full || busy}
          color={watched ? 'warning' : 'default'}
          onClick={(event) => {
            // The star often sits inside a clickable card or row; following must not open it.
            event.stopPropagation()
            if (watched) {
              unwatch.mutate(player.id)
            } else {
              watch.mutate(player.id)
            }
          }}
        >
          {watched ? <StarIcon fontSize={size === 'small' ? 'small' : 'medium'} /> : <StarBorderIcon fontSize={size === 'small' ? 'small' : 'medium'} />}
        </IconButton>
      </span>
    </Tooltip>
  )
}
