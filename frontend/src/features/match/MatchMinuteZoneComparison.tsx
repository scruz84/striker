import { useMemo, useState } from 'react'
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
import { ZoneComparisonRows } from './ZoneComparisonRows'

export interface MatchMinuteZoneComparisonProps {
  homeName: string
  awayName: string
  homeGoalkeeperRating: number
  awayGoalkeeperRating: number
  // Every minute recorded for this match so far (see backend common.MatchInstant), in order -
  // the full history a played match's report reads back, or whatever a live broadcast has
  // accumulated up to `currentMinute`.
  matchInstants: MatchInstant[]
  // The latest minute a viewer can scrub to - the broadcast's current minute for a live match, or
  // the final whistle's minute for a played one.
  currentMinute: number
  // True for a still-broadcasting match - only affects the title/copy shown at the current minute.
  live?: boolean
}

// instantAsOfMinute picks the latest recorded instant at or before `minute` - matchInstants is
// always ordered by minute (one row per simulated minute), so the last one not past the target
// minute is exactly the zone-battle snapshot as it stood then.
function instantAsOfMinute(instants: MatchInstant[], minute: number): MatchInstant | undefined {
  let result: MatchInstant | undefined
  for (const instant of instants) {
    if (instant.minute > minute) {
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
  homeName,
  awayName,
  homeGoalkeeperRating,
  awayGoalkeeperRating,
  matchInstants,
  currentMinute,
  live = false,
}: MatchMinuteZoneComparisonProps) {
  const { t } = useTranslation()
  // null means "pinned to now" - see MatchMinuteRatings' identical reasoning.
  const [scrubbedMinute, setScrubbedMinute] = useState<number | null>(null)
  const minute = scrubbedMinute ?? currentMinute
  const atNow = minute >= currentMinute

  const instant = useMemo(() => instantAsOfMinute(matchInstants, minute), [matchInstants, minute])

  const goToMinute = (value: number) => {
    const clamped = Math.max(0, Math.min(currentMinute, value))
    setScrubbedMinute(clamped >= currentMinute ? null : clamped)
  }

  if (!instant) {
    return null
  }

  const title = atNow ? t(live ? 'match.positionBattle.liveTitle' : 'match.positionBattle.title') : t('match.positionBattle.atMinute', { minute })

  return (
    <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
        <Typography variant="subtitle1">{title}</Typography>
        {!atNow && (
          <IconButton size="small" aria-label={t('match.ratings.jumpToNow')} title={t('match.ratings.jumpToNow')} onClick={() => goToMinute(currentMinute)}>
            <RestoreIcon fontSize="small" />
          </IconButton>
        )}
      </Stack>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('match.positionBattle.description')}
      </Typography>
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
      <ZoneComparisonRows
        homeQua={instant.homeQua}
        awayQua={instant.awayQua}
        homeGoalkeeperRating={homeGoalkeeperRating}
        awayGoalkeeperRating={awayGoalkeeperRating}
        homeName={homeName}
        awayName={awayName}
      />
    </Paper>
  )
}
