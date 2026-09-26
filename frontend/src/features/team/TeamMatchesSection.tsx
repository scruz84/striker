import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useTeamMatches } from '../../api/hooks'
import type { TeamMatchSummary } from '../../api/types'
import { mightStillBeLive } from '../../api/useLiveMatch'
import { NestedTeamNameLink } from '../../components/TeamNameLink'
import { formatDayTime } from '../../utils/formatDate'
import { matchKindColor, matchKindLabelKey, matchSummaryLink } from '../../utils/matchSummary'

// TeamMatchesSection lists every league match, cup tie, and friendly a team has played or has
// scheduled - split into upcoming (soonest first) and played (most recent first), each row marked
// with its competition kind and, for an unplayed match, clickable through to set its lineup (see
// matchLink/kindLabel below) - the "see all my matches" view requested alongside per-match
// lineups for league/cup matches (see backend/common.GetTeamMatches).
export function TeamMatchesSection({ teamId }: { teamId: string }) {
  const { t } = useTranslation()
  const { data: matches, isLoading } = useTeamMatches(teamId)

  if (isLoading || !matches) {
    return <Skeleton variant="rectangular" height={200} />
  }

  const upcoming = matches.filter((m) => !m.played).sort((a, b) => a.matchDate.localeCompare(b.matchDate))
  const played = matches
    .filter((m) => m.played)
    .sort((a, b) => b.matchDate.localeCompare(a.matchDate))

  return (
    <>
      <Typography variant="subtitle1" sx={{ mt: 1.5 }}>
        {t('team.matches.upcoming')}
      </Typography>
      <MatchesList matches={upcoming} emptyKey="team.matches.noUpcoming" />

      <Typography variant="subtitle1" sx={{ mt: 1.5 }}>
        {t('team.matches.played')}
      </Typography>
      <MatchesList matches={played} emptyKey="team.matches.noPlayed" />
    </>
  )
}

// A compact one-line-per-match list rather than a table - a table's fixed columns (competition
// chip, opponent, date, result) forced the opponent's name to wrap onto two lines at phone width
// and pushed the result column off-screen entirely. This wraps its pieces as a group instead, so
// a narrow viewport only ever grows a row's height by one extra line, never four.
function MatchesList({ matches, emptyKey }: { matches: TeamMatchSummary[]; emptyKey: string }) {
  const { t } = useTranslation()

  if (matches.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t(emptyKey)}
      </Typography>
    )
  }

  return (
    <Paper variant="outlined" sx={{ mb: 1 }}>
      <Stack divider={<Box sx={{ borderBottom: '1px solid', borderColor: 'divider' }} />}>
        {matches.map((match) => (
          <MatchRow key={`${match.kind}-${match.matchId}`} match={match} />
        ))}
      </Stack>
    </Paper>
  )
}

function MatchRow({ match }: { match: TeamMatchSummary }) {
  const { t } = useTranslation()
  // See DivisionDetailsPage's identical mightStillBeLive check - played flips true well before
  // the match's result finishes broadcasting live.
  const stillLive = match.played && mightStillBeLive(match.matchDate)

  return (
    <Stack
      component={RouterLink}
      to={matchSummaryLink(match)}
      direction="row"
      spacing={1}
      sx={{
        alignItems: 'center',
        flexWrap: 'wrap',
        py: 0.75,
        px: 1,
        textDecoration: 'none',
        color: 'inherit',
        '&:hover': { bgcolor: 'action.hover' },
      }}
    >
      <Chip size="small" color={matchKindColor(match.kind)} label={t(matchKindLabelKey(match.kind))} sx={{ height: 20, fontSize: '0.65rem', flexShrink: 0 }} />
      <Chip
        size="small"
        variant="outlined"
        label={t(match.isHome ? 'team.matches.homeTag' : 'team.matches.awayTag')}
        sx={{ height: 20, width: 24, fontSize: '0.65rem', flexShrink: 0, '& .MuiChip-label': { px: 0 } }}
      />
      <NestedTeamNameLink teamId={match.opponentId} noWrap sx={{ flex: '1 1 60px', minWidth: 0, fontWeight: 500 }}>
        {match.opponentName}
      </NestedTeamNameLink>
      {stillLive ? (
        <Chip label={t('match.live')} color="error" size="small" sx={{ height: 20, fontSize: '0.65rem', flexShrink: 0, fontWeight: 700 }} />
      ) : (
        <>
          <Typography variant="caption" color="text.secondary" noWrap sx={{ flexShrink: 0 }}>
            {formatDayTime(match.matchDate)}
          </Typography>
          <Typography variant="body2" noWrap sx={{ flexShrink: 0, minWidth: 32, textAlign: 'right', fontWeight: 700 }}>
            {match.played ? `${match.homeGoals} - ${match.awayGoals}` : '-'}
          </Typography>
        </>
      )}
    </Stack>
  )
}
