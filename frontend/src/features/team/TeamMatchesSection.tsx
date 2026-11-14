import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { RivalryChip } from '../../components/RivalryChip'
import { useTeamMatches } from '../../api/hooks'
import type { TeamMatchSummary } from '../../api/types'
import { isPast, mightStillBeLive } from '../../api/useLiveMatch'
import { HomeAwayIcon } from '../../components/HomeAwayIcon'
import { NestedTeamNameLink } from '../../components/TeamNameLink'
import { formatDayTime } from '../../utils/formatDate'
import { matchKindColor, matchKindLabelKey, matchSummaryLink } from '../../utils/matchSummary'
import { TeamNameWithFlag } from '../../components/TeamNameWithFlag'

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

  // A match is read into simulation up to 15 minutes before its scheduled matchDate (see backend
  // GetSimulableMatchesStmtSql/GetSimulableFriendliesStmtSql), so `played` can flip true while
  // matchDate is still ahead - bucket by isPast instead so such a match still shows as upcoming
  // until its advertised kickoff actually arrives.
  const upcoming = matches.filter((m) => !isPast(m.matchDate)).sort((a, b) => a.matchDate.localeCompare(b.matchDate))
  const played = matches
    .filter((m) => m.played && isPast(m.matchDate))
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
// Exported so TeamDetailsPage can reuse the same row rendering for its recent-matches summary.
export function MatchesList({ matches, emptyKey }: { matches: TeamMatchSummary[]; emptyKey: string }) {
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

type Outcome = 'win' | 'draw' | 'loss'

function outcomeOf(match: TeamMatchSummary): Outcome {
  const mine = match.isHome ? match.homeGoals : match.awayGoals
  const theirs = match.isHome ? match.awayGoals : match.homeGoals
  return mine > theirs ? 'win' : mine < theirs ? 'loss' : 'draw'
}

// Two lines on a phone: the opponent (full name, wrapping if need be) with the result on the
// right, and below it the competition, home/away marker and kickoff - instead of five items
// fighting for one line and truncating the name.
function MatchRow({ match }: { match: TeamMatchSummary }) {
  const { t } = useTranslation()
  // See DivisionDetailsPage's identical mightStillBeLive check - played flips true well before
  // the match's result finishes broadcasting live.
  const stillLive = match.played && mightStillBeLive(match.matchDate)
  const showResult = match.played && isPast(match.matchDate)
  const outcome = showResult ? outcomeOf(match) : undefined

  return (
    <Stack
      component={RouterLink}
      to={matchSummaryLink(match)}
      direction="row"
      spacing={1.5}
      sx={{
        alignItems: 'center',
        py: 1,
        px: 1.5,
        minHeight: 56,
        textDecoration: 'none',
        color: 'inherit',
        '&:hover': { bgcolor: 'action.hover' },
      }}
    >
      <Box sx={{ flexGrow: 1, minWidth: 0 }}>
        <NestedTeamNameLink teamId={match.opponentId} sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
          <TeamNameWithFlag rawName={match.opponentName} />
        </NestedTeamNameLink>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5, mt: 0.25 }}>
          <Chip size="small" color={matchKindColor(match.kind)} label={t(matchKindLabelKey(match.kind, match.competition))} sx={{ height: 20 }} />
          <RivalryChip level={match.rivalry} />
          <HomeAwayIcon isHome={match.isHome} />
          <Typography variant="caption" color="text.secondary">
            {formatDayTime(match.matchDate)}
          </Typography>
        </Stack>
      </Box>
      {stillLive ? (
        <Chip label={t('match.live')} color="error" size="small" sx={{ flexShrink: 0, fontWeight: 700 }} />
      ) : (
        <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', flexShrink: 0 }}>
          {outcome && (
            <Chip
              size="small"
              label={t(`common.result.${outcome}`)}
              color={outcome === 'win' ? 'success' : outcome === 'loss' ? 'error' : 'default'}
              variant={outcome === 'draw' ? 'outlined' : 'filled'}
              sx={{ height: 24, minWidth: 24, '& .MuiChip-label': { px: 0.75 } }}
            />
          )}
          <Typography variant="body2" sx={{ fontWeight: 700, minWidth: 36, textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>
            {showResult ? `${match.homeGoals} - ${match.awayGoals}` : '-'}
          </Typography>
        </Stack>
      )}
    </Stack>
  )
}
