import Chip from '@mui/material/Chip'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useTeamMatches } from '../../api/hooks'
import { MATCH_KIND_CUP, MATCH_KIND_LEAGUE, type TeamMatchSummary } from '../../api/types'
import { formatDateTime } from '../../utils/formatDate'

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
      <Typography variant="h6" sx={{ mt: 2 }}>
        {t('team.matches.upcoming')}
      </Typography>
      <MatchesTable matches={upcoming} emptyKey="team.matches.noUpcoming" />

      <Typography variant="h6" sx={{ mt: 2 }}>
        {t('team.matches.played')}
      </Typography>
      <MatchesTable matches={played} emptyKey="team.matches.noPlayed" />
    </>
  )
}

function MatchesTable({ matches, emptyKey }: { matches: TeamMatchSummary[]; emptyKey: string }) {
  const { t } = useTranslation()

  if (matches.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        {t(emptyKey)}
      </Typography>
    )
  }

  return (
    <TableContainer component={Paper}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t('team.matches.kind')}</TableCell>
            <TableCell>{t('team.matches.opponent')}</TableCell>
            <TableCell>{t('team.matches.date')}</TableCell>
            <TableCell align="right">{t('team.matches.result')}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {matches.map((match) => (
            <MatchRow key={`${match.kind}-${match.matchId}`} match={match} />
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

function MatchRow({ match }: { match: TeamMatchSummary }) {
  const { t } = useTranslation()
  const opponentLabel = match.isHome
    ? t('team.matches.vsHome', { name: match.opponentName })
    : t('team.matches.vsAway', { name: match.opponentName })

  return (
    <TableRow>
      <TableCell>
        <Chip size="small" color={kindColor(match.kind)} label={t(kindLabelKey(match.kind))} />
      </TableCell>
      <TableCell>
        <Link component={RouterLink} to={matchLink(match)} underline="hover" color="inherit">
          {opponentLabel}
        </Link>
      </TableCell>
      <TableCell>{formatDateTime(match.matchDate)}</TableCell>
      <TableCell align="right">
        {match.played ? `${match.homeGoals} - ${match.awayGoals}` : '-'}
      </TableCell>
    </TableRow>
  )
}

function kindColor(kind: TeamMatchSummary['kind']): 'primary' | 'secondary' | 'default' {
  switch (kind) {
    case MATCH_KIND_LEAGUE:
      return 'primary'
    case MATCH_KIND_CUP:
      return 'secondary'
    default:
      return 'default'
  }
}

function kindLabelKey(kind: TeamMatchSummary['kind']): string {
  switch (kind) {
    case MATCH_KIND_LEAGUE:
      return 'team.matches.kindLeague'
    case MATCH_KIND_CUP:
      return 'team.matches.kindCup'
    default:
      return 'team.matches.kindFriendly'
  }
}

function matchLink(match: TeamMatchSummary): string {
  switch (match.kind) {
    case MATCH_KIND_LEAGUE:
      return `/match/${match.matchId}`
    case MATCH_KIND_CUP:
      return `/cup/ties/${match.matchId}`
    default:
      return `/friendlies/${match.matchId}`
  }
}
