import { useMemo } from 'react'
import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid'
import IconButton from '@mui/material/IconButton'
import Paper from '@mui/material/Paper'
import Slider from '@mui/material/Slider'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import RestoreIcon from '@mui/icons-material/Restore'
import { useTranslation } from 'react-i18next'
import type { Lineup, MatchEvent, Player } from '../../api/types'
import { computeRatingsAsOfMinute } from '../../utils/liveMatchRatings'
import { useMinuteScrub, type MinuteScrub } from '../../utils/useMinuteScrub'
import { TeamRatingsColumn } from './TeamRatingsColumn'

export interface MatchMinuteRatingsProps {
  homeName: string
  awayName: string
  homeLineup: Lineup
  awayLineup: Lineup
  homePlayers: Record<string, Player>
  awayPlayers: Record<string, Player>
  events: MatchEvent[]
  // The latest minute a viewer can scrub to - the broadcast's current minute for a live match, or
  // the final whistle's minute for a played one.
  currentMinute: number
  // True for a still-broadcasting match - only affects the title/copy shown at the current minute.
  live?: boolean
  // The backend's own final ratings (with its per-player jitter - see ComputeMatchRatings), shown
  // once the scrubber is at `currentMinute` instead of this component's own live-style
  // approximation. Omitted for a still-live match, which has no "final" yet.
  finalRatings?: Record<string, number>
  // Lets a caller share this panel's scrub position with other panels on the same page (e.g. the
  // lineup pitch's per-player ratings). Omitted, this owns its own scrub position exactly as
  // before.
  scrub?: MinuteScrub
}

/** Player ratings for one match minute, with a slider (plus step buttons) to scrub back through
 * every earlier minute - shared by both the live view (MatchDetailsPage) and a finished match's
 * "Calificaciones" tab (MatchRatingsTab), the two moments a viewer might want ratings "as of an
 * exact minute" rather than only ever the latest/final ones. */
export function MatchMinuteRatings({
  homeName,
  awayName,
  homeLineup,
  awayLineup,
  homePlayers,
  awayPlayers,
  events,
  currentMinute,
  live = false,
  finalRatings,
  scrub,
}: MatchMinuteRatingsProps) {
  const { t } = useTranslation()
  const ownScrub = useMinuteScrub(currentMinute)
  const { minute, atNow, goToMinute } = scrub ?? ownScrub

  const ratings = useMemo(() => {
    if (atNow && finalRatings) {
      return finalRatings
    }
    return computeRatingsAsOfMinute(homeLineup, awayLineup, events, minute)
  }, [atNow, finalRatings, homeLineup, awayLineup, events, minute])

  const title = atNow ? t(live ? 'match.ratings.liveTitle' : 'match.ratings.title') : t('match.ratings.atMinute', { minute })

  return (
    <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
        <Typography variant="subtitle2">{title}</Typography>
        {!atNow && (
          <IconButton size="small" aria-label={t('match.ratings.jumpToNow')} title={t('match.ratings.jumpToNow')} onClick={() => goToMinute(currentMinute)}>
            <RestoreIcon fontSize="small" />
          </IconButton>
        )}
      </Stack>
      {currentMinute > 0 && (
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 2 }}>
          <IconButton size="small" aria-label={t('match.ratings.previousMinute')} disabled={minute <= 0} onClick={() => goToMinute(minute - 1)}>
            <ChevronLeftIcon fontSize="small" />
          </IconButton>
          <Slider
            size="small"
            value={minute}
            min={0}
            max={currentMinute}
            step={1}
            valueLabelDisplay="auto"
            valueLabelFormat={(value) => `${value}'`}
            onChange={(_event, value) => goToMinute(Array.isArray(value) ? value[0] : value)}
            sx={{ flex: 1 }}
          />
          <IconButton
            size="small"
            aria-label={t('match.ratings.nextMinute')}
            disabled={minute >= currentMinute}
            onClick={() => goToMinute(minute + 1)}
          >
            <ChevronRightIcon fontSize="small" />
          </IconButton>
          <Box sx={{ minWidth: 32, textAlign: 'right' }}>
            <Typography variant="body2" color="text.secondary">
              {minute}&apos;
            </Typography>
          </Box>
        </Stack>
      )}
      <Grid container spacing={3}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TeamRatingsColumn title={homeName} lineup={homeLineup} players={homePlayers} ratings={ratings} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <TeamRatingsColumn title={awayName} lineup={awayLineup} players={awayPlayers} ratings={ratings} />
        </Grid>
      </Grid>
    </Paper>
  )
}
