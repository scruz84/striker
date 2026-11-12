import { useMemo } from 'react'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import Paper from '@mui/material/Paper'
import Slider from '@mui/material/Slider'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import RestoreIcon from '@mui/icons-material/Restore'
import { useTranslation } from 'react-i18next'
import type { MatchInstant } from '../../api/types'
import { formatMatchMinute, isAtOrBefore, matchMinuteText, timelineMinute, type MatchMinute } from '../../utils/matchMinute'
import { useMinuteScrub, type MinuteScrub } from '../../utils/useMinuteScrub'
import { ZoneComparisonRows } from './ZoneComparisonRows'

export interface MatchMinuteZoneComparisonProps {
  homeId: number
  homeName: string
  awayId: number
  awayName: string
  homeGoalkeeperRating: number
  awayGoalkeeperRating: number
  // Every minute recorded for this match so far (see backend common.MatchInstant), in order -
  // the full history a played match's report reads back, or whatever a live broadcast has
  // accumulated so far.
  matchInstants: MatchInstant[]
  // The minutes a viewer can scrub through, in match order, added minutes included (see
  // utils/matchMinute.matchTimeline): its last one is the broadcast's current minute for a live
  // match, or the final whistle's for a played one.
  timeline: MatchMinute[]
  // True for a still-broadcasting match - only affects the title/copy shown at the current minute.
  live?: boolean
  // Lets a caller share this panel's scrub position with other panels on the same page (e.g. the
  // lineup pitch's per-player ratings - see FriendlyMatchPage/MatchDetailsPage's live views).
  // Omitted, this owns its own scrub position exactly as before.
  scrub?: MinuteScrub
}

// instantAsOfMinute picks the latest recorded instant at or before `minute` - matchInstants is
// always in match order (one row per simulated minute, 45+1 between 45 and 46), so the last one
// not past the target minute is exactly the zone-battle snapshot as it stood then.
function instantAsOfMinute(instants: MatchInstant[], minute: MatchMinute): MatchInstant | undefined {
  let result: MatchInstant | undefined
  for (const instant of instants) {
    if (!isAtOrBefore(instant, minute)) {
      break
    }
    result = instant
  }
  return result ?? instants[0]
}

/** The zone-by-zone strength comparison (ZoneComparisonRows) for one match minute, with a slider
 * (plus step buttons) to scrub back through every earlier minute - shared by both the live view
 * and a finished match's "Jugadores" tab, the same way MatchMinuteRatings already lets a viewer
 * scrub player ratings instead of only ever showing the latest/final snapshot. */
export function MatchMinuteZoneComparison({
  homeId,
  homeName,
  awayId,
  awayName,
  homeGoalkeeperRating,
  awayGoalkeeperRating,
  matchInstants,
  timeline,
  live = false,
  scrub,
}: MatchMinuteZoneComparisonProps) {
  const { t } = useTranslation()
  // The scrubber moves in timeline steps (see useMinuteScrub): the last step is "now".
  const lastStep = timeline.length
  const ownScrub = useMinuteScrub(lastStep)
  const { minute: step, atNow, goToMinute } = scrub ?? ownScrub
  const at = timelineMinute(timeline, step)

  const instant = useMemo(() => instantAsOfMinute(matchInstants, at), [matchInstants, at])

  if (!instant) {
    return null
  }

  const title = atNow
    ? t(live ? 'match.positionBattle.liveTitle' : 'match.positionBattle.title')
    : t('match.positionBattle.atMinute', { minute: matchMinuteText(at.minute, at.addedMinute) })

  return (
    <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
        <Typography variant="subtitle1">{title}</Typography>
        {!atNow && (
          <IconButton size="small" aria-label={t('match.ratings.jumpToNow')} title={t('match.ratings.jumpToNow')} onClick={() => goToMinute(lastStep)}>
            <RestoreIcon fontSize="small" />
          </IconButton>
        )}
      </Stack>
      {lastStep > 0 && (
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 2 }}>
          <IconButton size="small" aria-label={t('match.ratings.previousMinute')} disabled={step <= 0} onClick={() => goToMinute(step - 1)}>
            <ChevronLeftIcon fontSize="small" />
          </IconButton>
          <Slider
            size="small"
            value={step}
            min={0}
            max={lastStep}
            step={1}
            valueLabelDisplay="auto"
            valueLabelFormat={(value) => {
              const label = timelineMinute(timeline, value)
              return formatMatchMinute(label.minute, label.addedMinute)
            }}
            onChange={(_event, value) => goToMinute(Array.isArray(value) ? value[0] : value)}
            sx={{ flex: 1 }}
          />
          <IconButton
            size="small"
            aria-label={t('match.ratings.nextMinute')}
            disabled={step >= lastStep}
            onClick={() => goToMinute(step + 1)}
          >
            <ChevronRightIcon fontSize="small" />
          </IconButton>
          <Box sx={{ minWidth: 40, textAlign: 'right' }}>
            <Typography variant="body2" color="text.secondary">
              {formatMatchMinute(at.minute, at.addedMinute)}
            </Typography>
          </Box>
        </Stack>
      )}
      <ZoneComparisonRows
        homeQua={instant.homeQua}
        awayQua={instant.awayQua}
        homeGoalkeeperRating={homeGoalkeeperRating}
        awayGoalkeeperRating={awayGoalkeeperRating}
        homeId={homeId}
        homeName={homeName}
        awayId={awayId}
        awayName={awayName}
      />
    </Paper>
  )
}
