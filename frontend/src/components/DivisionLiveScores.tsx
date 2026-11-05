import ExpandLessIcon from '@mui/icons-material/ExpandLess'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Collapse from '@mui/material/Collapse'
import Stack from '@mui/material/Stack'
import { useTheme } from '@mui/material/styles'
import Typography from '@mui/material/Typography'
import useMediaQuery from '@mui/material/useMediaQuery'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink } from 'react-router-dom'
import type { LeagueLiveScore } from '../api/types'
import { PULSING_LIVE_SX } from '../utils/liveChip'
import { liveScoreLabel } from '../utils/liveScoreLabel'
import { TeamBadge } from './TeamBadge'

// The "N matches · G goals" line a collapsed list reads.
function useLiveScoresSummary(entries: LeagueLiveScore[]): string {
  const { t } = useTranslation()
  const goals = entries.reduce((sum, entry) => sum + entry.homeGoals + entry.awayGoals, 0)
  return t('liveScores.summary', {
    matches: t('liveScores.matchCount', { count: entries.length }),
    goals: t('liveScores.goalCount', { count: goals }),
  })
}

function TeamLine({ id, name, goals, showGoals }: { id: number; name: string; goals: number; showGoals: boolean }) {
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', minWidth: 0 }}>
      <TeamBadge teamId={id} teamName={name} size={22} />
      <Typography variant="body2" sx={{ flex: 1, minWidth: 0, overflowWrap: 'anywhere', lineHeight: 1.25 }}>
        {name}
      </Typography>
      {showGoals && (
        <Typography variant="body2" component="span" sx={{ fontWeight: 800, minWidth: 16, textAlign: 'right' }}>
          {goals}
        </Typography>
      )}
    </Stack>
  )
}

function LiveScoreRow({ entry }: { entry: LeagueLiveScore }) {
  const { t } = useTranslation()
  const label = liveScoreLabel(entry, t)
  // A pending match has no score yet: show the kickoff time alone, never a 0 - 0.
  const showGoals = entry.status !== 'pending'
  return (
    <Box
      component={RouterLink}
      to={`/match/${entry.matchId}`}
      sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.75, color: 'inherit', textDecoration: 'none', '&:hover': { opacity: 0.85 } }}
    >
      <Stack spacing={0.5} sx={{ flex: 1, minWidth: 0 }}>
        <TeamLine id={entry.home} name={entry.homeName} goals={entry.homeGoals} showGoals={showGoals} />
        <TeamLine id={entry.away} name={entry.awayName} goals={entry.awayGoals} showGoals={showGoals} />
      </Stack>
      <Chip
        size="small"
        variant={label.color === 'error' ? 'filled' : 'outlined'}
        label={label.text}
        sx={{
          flexShrink: 0,
          fontWeight: 700,
          color: 'inherit',
          borderColor: 'currentColor',
          ...(label.pulsing ? { bgcolor: 'onErrorContainer', color: 'errorContainer', ...PULSING_LIVE_SX } : {}),
        }}
      />
    </Box>
  )
}

// The other matches of a matchday with their broadcast score and state, one compact row each
// (teams stacked, chip at the end so it fits a 360 px phone). Colours inherit, so the list reads on
// the dashboard hero's error-container surface. It starts collapsed on phones behind a toggle
// reading "N matches · G goals", and expanded on desktop.
export function DivisionLiveScores({ entries }: { entries: LeagueLiveScore[] }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isPhone = useMediaQuery(theme.breakpoints.down('md'))
  // null until the user taps the toggle, so the initial state follows the screen size.
  const [userExpanded, setUserExpanded] = useState<boolean | null>(null)
  const expanded = userExpanded ?? !isPhone
  const summary = useLiveScoresSummary(entries)

  return (
    <Box sx={{ mt: 2, pt: 1.5, borderTop: 1, borderColor: 'divider' }}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', gap: 1 }}>
        <Typography variant="overline" sx={{ lineHeight: 1.4 }}>
          {t('liveScores.restOfMatchday')}
        </Typography>
        <Button
          size="small"
          color="inherit"
          aria-expanded={expanded}
          onClick={() => setUserExpanded(!expanded)}
          endIcon={expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          sx={{ textTransform: 'none', minWidth: 0, textAlign: 'right' }}
        >
          {summary}
        </Button>
      </Stack>
      <Collapse in={expanded} unmountOnExit>
        <Stack divider={<Box sx={{ borderTop: 1, borderColor: 'divider', opacity: 0.5 }} />}>
          {entries.map((entry) => (
            <LiveScoreRow key={entry.matchId} entry={entry} />
          ))}
        </Stack>
      </Collapse>
    </Box>
  )
}
