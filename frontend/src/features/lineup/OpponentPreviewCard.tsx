import { useState } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import useMediaQuery from '@mui/material/useMediaQuery'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useOpponentPreview } from '../../api/hooks'
import {
  FORMATION_LABEL_KEYS,
  type OpponentPreview,
  type TeamMatchSummary,
} from '../../api/types'
import { matchSummaryLink } from '../../utils/matchSummary'
import { ZoneComparisonRows } from '../match/ZoneComparisonRows'
import { likelyLines } from './opponentPreviewLines'

export interface OpponentPreviewCardProps {
  // The fixture to preview (kind + id as the lineup endpoints use them).
  kind: TeamMatchSummary['kind']
  matchId: number | string
  // The owner's own club, shown on the left of the projected zone duel.
  teamId: number
  teamName: string
}

/** What the owner can learn about the club he is about to play, from public data only (see
 * backend common.GetOpponentPreview): its form, the XI it has been using, its in-form players and
 * where his currently saved lineup is projected to win or lose each zone. Collapsed by default on
 * phones. Renders nothing while the fixture is locked or not his (the endpoint refuses). */
export function OpponentPreviewCard({ kind, matchId, teamId, teamName }: OpponentPreviewCardProps) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isPhone = useMediaQuery(theme.breakpoints.down('sm'))
  const [expanded, setExpanded] = useState<boolean | null>(null)
  const { data: preview, isLoading } = useOpponentPreview(kind, String(matchId))

  if (isLoading) {
    return <Skeleton variant="rectangular" height={64} sx={{ mb: 2 }} />
  }
  if (!preview) {
    return null
  }
  const open = expanded ?? !isPhone

  return (
    <Paper variant="outlined" sx={{ p: { xs: 1.5, sm: 2 }, mb: 2 }} aria-label={t('preview.title', { opponent: preview.opponent.name })}>
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', gap: 1 }}>
        <Box sx={{ minWidth: 0 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 700 }} noWrap>
            {t('preview.title', { opponent: preview.opponent.name })}
          </Typography>
          {preview.opponent.position > 0 && (
            <Typography variant="caption" color="text.secondary">
              {t('preview.position', { count: preview.opponent.position, ordinal: true })}
            </Typography>
          )}
        </Box>
        {isPhone && (
          <Button size="small" onClick={() => setExpanded(!open)} aria-expanded={open}>
            {t(open ? 'preview.hide' : 'preview.show')}
          </Button>
        )}
      </Stack>
      {open && <PreviewBody preview={preview} teamId={teamId} teamName={teamName} />}
    </Paper>
  )
}

function PreviewBody({ preview, teamId, teamName }: { preview: OpponentPreview; teamId: number; teamName: string }) {
  const { t } = useTranslation()
  const { opponent, likely, inForm, projection } = preview

  if (opponent.form.length === 0 && !likely) {
    return (
      <Typography color="text.secondary" sx={{ mt: 1 }}>
        {t('preview.noHistory')}
      </Typography>
    )
  }

  return (
    <Stack spacing={2} sx={{ mt: 1.5 }}>
      <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5 }}>
        <Typography variant="caption" color="text.secondary">
          {t('preview.form')}
        </Typography>
        {opponent.form.map((match) => {
          const ownGoals = match.isHome ? match.homeGoals : match.awayGoals
          const rivalGoals = match.isHome ? match.awayGoals : match.homeGoals
          const outcome = ownGoals > rivalGoals ? 'win' : ownGoals < rivalGoals ? 'loss' : 'draw'
          const description = `${match.opponentName} ${ownGoals}-${rivalGoals}`
          return (
            <Chip
              key={`${match.kind}-${match.matchId}`}
              size="small"
              color={outcome === 'win' ? 'success' : outcome === 'loss' ? 'error' : 'default'}
              component={RouterLink}
              to={matchSummaryLink(match)}
              clickable
              aria-label={`${t(`common.resultLong.${outcome}`)}: ${description}`}
              label={t(`dashboard.result.${outcome}`)}
              sx={{ height: 24, width: 28, typography: 'chipLabel', fontWeight: 700, '& .MuiChip-label': { px: 0 } }}
            />
          )
        })}
        <Typography variant="caption" color="text.secondary">
          {t('preview.formLatestFirst')}
        </Typography>
      </Stack>

      {likely && (
        <Box>
          <Typography variant="overline" color="text.secondary" sx={{ lineHeight: 1.5 }}>
            {t('preview.likelyXi', { formation: t(FORMATION_LABEL_KEYS[likely.formation] ?? 'formations.balanced') })}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
            <Link component={RouterLink} to={matchSummaryLink(likely)}>
              {t('preview.likelyXiSource')}
            </Link>
          </Typography>
          {likelyLines(likely.players).map((line) => (
            <Typography key={line.key} variant="body2" sx={{ mb: 0.25 }}>
              {line.players.map((p) => p.name).join(' · ')}
            </Typography>
          ))}
        </Box>
      )}

      {inForm.length > 0 && (
        <Box>
          <Typography variant="overline" color="text.secondary" sx={{ lineHeight: 1.5 }}>
            {t('preview.inForm')}
          </Typography>
          {inForm.map((p) => (
            <Typography key={p.playerId} variant="body2">
              {p.name} · {p.averageRating.toFixed(1)}{' '}
              <Typography component="span" variant="caption" color="text.secondary">
                ({t('preview.appearances', { count: p.appearances })})
              </Typography>
            </Typography>
          ))}
        </Box>
      )}

      {projection && (
        <Box>
          <Typography variant="overline" color="text.secondary" sx={{ lineHeight: 1.5 }}>
            {t('preview.projection')}
          </Typography>
          <ZoneComparisonRows
            homeQua={projection.own}
            awayQua={projection.opponent}
            homeGoalkeeperRating={projection.ownGoalkeeper}
            awayGoalkeeperRating={projection.opponentGoalkeeper}
            homeId={teamId}
            homeName={teamName}
            awayId={opponent.id}
            awayName={opponent.name}
          />
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.75 }}>
            {t('preview.projectionNote')}
          </Typography>
        </Box>
      )}
    </Stack>
  )
}

