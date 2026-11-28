import { useState } from 'react'
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import StarOutlineIcon from '@mui/icons-material/StarOutlineOutlined'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink } from 'react-router-dom'
import { useBestXi } from '../../api/hooks'
import type { BestXiPlayer } from '../../api/types'
import { EmptyState } from '../../components/EmptyState'
import { PitchField, PitchSlotPosition } from '../../components/pitch/PitchField'
import { PITCH_HEIGHT_EDITOR } from '../../components/pitch/pitchLayout'
import { placeBestXi } from './bestXiLayout'

// A league's team of the week, or of the season (openspec best-xi), with arrows through the
// rounds of the season that have one. `initialWeek` opens a given round (the notification's link),
// `season` undefined is the current season.
export function BestXiCard({ leagueId, season, initialWeek }: { leagueId: string | undefined; season?: number; initialWeek?: number }) {
  const { t } = useTranslation()
  const [week, setWeek] = useState<number | undefined>(initialWeek)
  const { data: response, isLoading } = useBestXi(leagueId, season, week)
  // Anything but a best XI payload (a failed or unexpected response) reads as none stored.
  const data = response && Array.isArray(response.players) && Array.isArray(response.weeks) ? response : undefined
  const weeks = data?.weeks ?? []
  const shown = data?.week
  const index = shown !== undefined ? weeks.indexOf(shown) : -1
  // Matchdays first, the team of the season (week 0) last.
  const ordered = [...weeks.filter((w) => w > 0), ...weeks.filter((w) => w === 0)]
  const position = shown !== undefined ? ordered.indexOf(shown) : -1
  const label = shown === 0 ? t('stats.bestXi.season') : shown ? t('stats.bestXi.week', { week: shown }) : ''

  return (
    <Paper variant="outlined">
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', p: 2, pb: 1 }}>
        <StarOutlineIcon color="primary" fontSize="small" />
        <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 700, flexGrow: 1 }}>
          {t('stats.bestXi.title')}
        </Typography>
        {ordered.length > 0 && index >= 0 && (
          <Stack direction="row" sx={{ alignItems: 'center' }}>
            <IconButton
              size="small"
              aria-label={t('stats.bestXi.previous')}
              disabled={position <= 0}
              onClick={() => setWeek(ordered[position - 1])}
            >
              <ChevronLeftIcon />
            </IconButton>
            <Typography variant="body2" sx={{ minWidth: 88, textAlign: 'center', fontWeight: 600 }}>
              {label}
            </Typography>
            <IconButton
              size="small"
              aria-label={t('stats.bestXi.next')}
              disabled={position < 0 || position >= ordered.length - 1}
              onClick={() => setWeek(ordered[position + 1])}
            >
              <ChevronRightIcon />
            </IconButton>
          </Stack>
        )}
      </Stack>
      {isLoading ? (
        <Skeleton variant="rectangular" height={200} sx={{ m: 2, borderRadius: 1 }} />
      ) : !data || data.players.length === 0 ? (
        <EmptyState compact title={t('stats.bestXi.empty')} />
      ) : (
        <Box sx={{ px: 1, pb: 2 }}>
          <PitchField half height={PITCH_HEIGHT_EDITOR}>
            {placeBestXi(data.players).map(({ player, x, y }) => (
              <PitchSlotPosition key={player.slot} x={x} y={y} pitchHeight={PITCH_HEIGHT_EDITOR}>
                <BestXiChip player={player} seasonXi={data.week === 0} />
              </PitchSlotPosition>
            ))}
          </PitchField>
        </Box>
      )}
    </Paper>
  )
}

function BestXiChip({ player, seasonXi }: { player: BestXiPlayer; seasonXi: boolean }) {
  const { t } = useTranslation()
  return (
    <Box
      component={RouterLink}
      to={`/player/${player.playerId}`}
      title={`${player.firstName} ${player.lastName} · ${player.teamName}`}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        width: 64,
        textDecoration: 'none',
        color: '#fff',
      }}
    >
      <Box
        sx={{
          px: 0.75,
          borderRadius: 1,
          bgcolor: 'rgba(0,0,0,0.55)',
          typography: 'chipLabel',
          fontWeight: 700,
        }}
      >
        {player.rating.toFixed(1)}
      </Box>
      <Typography variant="caption" noWrap sx={{ maxWidth: '100%', fontWeight: 700, lineHeight: 1.2, textShadow: '0 1px 2px rgba(0,0,0,0.8)' }}>
        {player.lastName}
      </Typography>
      <Typography variant="caption" noWrap sx={{ maxWidth: '100%', fontSize: '0.625rem', lineHeight: 1.1, opacity: 0.9, textShadow: '0 1px 2px rgba(0,0,0,0.8)' }}>
        {seasonXi ? t('stats.bestXi.matches', { count: player.matches }) : player.teamName}
      </Typography>
    </Box>
  )
}
