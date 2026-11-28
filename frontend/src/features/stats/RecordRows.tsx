import type { ReactNode } from 'react'
import Chip from '@mui/material/Chip'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink } from 'react-router-dom'
import type { RecordMatch, RecordPlayer, RecordStreak } from '../../api/types'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { formatShortDate } from '../../utils/formatDate'

// One labelled record: the label on the left, its holder on the right.
export function RecordRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'baseline', gap: 1.5 }}>
      <Typography variant="body2" color="text.secondary" sx={{ flexShrink: 0 }}>
        {label}
      </Typography>
      <Typography variant="body2" component="div" sx={{ fontWeight: 600, textAlign: 'right', minWidth: 0, overflowWrap: 'anywhere' }}>
        {children}
      </Typography>
    </Stack>
  )
}

// A titled card of record rows.
export function RecordCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Typography variant="subtitle1" component="h3" sx={{ fontWeight: 700, mb: 1 }}>
        {title}
      </Typography>
      <Stack spacing={0.75}>{children}</Stack>
    </Paper>
  )
}

// A record fixture as its scoreline, linking to its report.
export function RecordMatchLink({ match }: { match: RecordMatch }) {
  const { t } = useTranslation()
  const to = match.kind === 'cup' ? `/cup/ties/${match.matchId}` : `/match/${match.matchId}`
  return (
    <Link component={RouterLink} to={to} underline="hover" color="inherit">
      {match.homeTeamName} {match.homeGoals}-{match.awayGoals} {match.awayTeamName}
      <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 0.75 }}>
        {t('stats.seasonShort', { season: match.season })}
      </Typography>
    </Link>
  )
}

export function RecordStreakValue({ streak }: { streak: RecordStreak }) {
  const { t } = useTranslation()
  return (
    <>
      {t('stats.records.matches', { count: streak.length })}
      <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 0.75 }}>
        {formatShortDate(streak.from)} – {formatShortDate(streak.to)}
      </Typography>
      {streak.ongoing && <Chip size="small" color="primary" sx={{ ml: 0.75 }} label={t('stats.records.ongoing')} />}
    </>
  )
}

// An all-time list of players: rank, name and the figure it ranks by.
export function RecordPlayerList({ title, players, figure }: { title: string; players: RecordPlayer[]; figure: 'goals' | 'appearances' }) {
  const { t } = useTranslation()
  if (players.length === 0) return null
  return (
    <RecordCard title={title}>
      {players.map((p, index) => (
        <RecordRow key={p.playerId} label={`${index + 1}.`}>
          <PlayerNameLink playerId={p.playerId}>
            {p.firstName} {p.lastName}
          </PlayerNameLink>
          {p.retired && (
            <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 0.5 }}>
              ({t('stats.team.retired')})
            </Typography>
          )}
          {' · '}
          {figure === 'goals' ? t('stats.goalsShort', { count: p.goals }) : t('stats.records.matches', { count: p.appearances })}
        </RecordRow>
      ))}
    </RecordCard>
  )
}
