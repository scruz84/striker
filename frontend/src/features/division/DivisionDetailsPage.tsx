import CalendarTodayTwoToneIcon from '@mui/icons-material/CalendarTodayTwoTone'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { alpha, useTheme } from '@mui/material/styles'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useDivisionDetails, useTopScorers, useUserTeam } from '../../api/hooks'
import type { ClassificationPosition, ScorerStanding, WeekMatch } from '../../api/types'
import { TeamBadge } from '../../components/TeamBadge'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { NestedTeamNameLink, TeamNameLink } from '../../components/TeamNameLink'
import { isPast, mightStillBeLive } from '../../api/useLiveMatch'
import { formatDateTime } from '../../utils/formatDate'
import { RecentLeagueTransfers } from '../leagues/RecentLeagueTransfers'

const RANK_COLORS = ['#D4AF37', '#A8A9AD', '#B08D57'] as const

export function DivisionDetailsPage() {
  const { t } = useTranslation()
  const theme = useTheme()
  const { id } = useParams<{ id: string }>()
  const { data: divisionDetails, isLoading } = useDivisionDetails(id)
  const { data: userTeam } = useUserTeam(true)

  if (isLoading || !divisionDetails) {
    return <Skeleton variant="rectangular" height={400} sx={{ borderRadius: 2 }} />
  }

  // Reverted once and reused below for both the week chip and the table itself - positions[0]'s
  // own `matches` count otherwise ticks over the instant a match is simulated too, up to 15
  // minutes before its official kickoff (see hideStillLiveResults).
  const positions = hideStillLiveResults(divisionDetails.classification.positions, divisionDetails.previousWeekMatches)
  const currentWeek = positions[0]?.matches ?? 0
  const myTeamId = userTeam?.team.id

  return (
    <Box>
      <Paper variant="outlined" sx={{ overflow: 'hidden', mb: 2 }}>
        <Box
          sx={{
            height: 4,
            background: `linear-gradient(90deg, ${theme.palette.primary.main}, ${theme.palette.primary.light})`,
          }}
        />
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          spacing={1}
          sx={{ p: { xs: 2, sm: 3 }, alignItems: { sm: 'center' }, justifyContent: 'space-between' }}
        >
          <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
            <EmojiEventsTwoToneIcon color="primary" fontSize="large" />
            <Box>
              <Typography variant="h5" sx={{ fontWeight: 700 }}>
                {divisionDetails.name}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {t('division.season', { season: divisionDetails.season })}
              </Typography>
            </Box>
          </Stack>
          <Chip label={t('division.weekChip', { week: currentWeek })} variant="outlined" sx={{ fontWeight: 600 }} />
        </Stack>
      </Paper>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 8 }}>
          <ClassificationTable positions={positions} myTeamId={myTeamId} />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <Stack spacing={2}>
            <TopScorers leagueId={id} />
            <RecentLeagueTransfers leagueId={id} />
          </Stack>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <FixturesCard title={t('division.lastWeek')} matches={divisionDetails.previousWeekMatches} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <FixturesCard title={t('division.nextWeek')} matches={divisionDetails.nextWeekMatches} />
        </Grid>
      </Grid>
    </Box>
  )
}

// hideStillLiveResults reverts each team's classification row to how it looked just before any
// match of theirs that's still within its live-broadcast window (see mightStillBeLive) - the
// same "don't spoil a match that still looks live everywhere else" reasoning FixturesCard's own
// stillLive check already applies to the score column (issue #37), extended here to the
// classification table itself (issue #82): points/wins/draws/losses/goals otherwise update the
// instant a match is simulated, well before its result finishes broadcasting to viewers, which
// leaks the outcome indirectly even with the fixture's own score hidden. previousWeekMatches is
// the only week recent enough to still be within that window - anything older has certainly
// finished broadcasting.
export function hideStillLiveResults(positions: ClassificationPosition[], previousWeekMatches: WeekMatch[]): ClassificationPosition[] {
  const stillLiveMatchByTeam = new Map<number, WeekMatch>()
  for (const match of previousWeekMatches) {
    if (match.played && mightStillBeLive(match.matchDate)) {
      stillLiveMatchByTeam.set(match.home, match)
      stillLiveMatchByTeam.set(match.away, match)
    }
  }
  if (stillLiveMatchByTeam.size === 0) {
    return positions
  }

  const reverted = positions.map((row) => {
    const match = stillLiveMatchByTeam.get(row.teamId)
    if (!match) {
      return row
    }
    const isHome = match.home === row.teamId
    const goalsFor = isHome ? match.homeGoals : match.awayGoals
    const goalsAgainst = isHome ? match.awayGoals : match.homeGoals
    const won = goalsFor > goalsAgainst
    const drawn = goalsFor === goalsAgainst
    return {
      ...row,
      points: row.points - (won ? 3 : drawn ? 1 : 0),
      matches: row.matches - 1,
      wins: row.wins - (won ? 1 : 0),
      draws: row.draws - (drawn ? 1 : 0),
      losses: row.losses - (!won && !drawn ? 1 : 0),
      goalsScored: row.goalsScored - goalsFor,
      goalsReceived: row.goalsReceived - goalsAgainst,
    }
  })

  // Re-sort by the same ordering the backend itself uses (see GetDivisionDetailsStmtSql), then
  // reassign positions - a team whose still-live match briefly overstated its points would
  // otherwise keep its post-match rank until the broadcast catches up.
  reverted.sort((a, b) => {
    if (b.points !== a.points) return b.points - a.points
    const diffA = a.goalsScored - a.goalsReceived
    const diffB = b.goalsScored - b.goalsReceived
    if (diffB !== diffA) return diffB - diffA
    if (b.goalsScored !== a.goalsScored) return b.goalsScored - a.goalsScored
    if (b.wins !== a.wins) return b.wins - a.wins
    return b.draws - a.draws
  })
  return reverted.map((row, index) => ({ ...row, position: index + 1 }))
}

function ClassificationTable({
  positions,
  myTeamId,
}: {
  positions: ClassificationPosition[]
  myTeamId: number | undefined
}) {
  const { t } = useTranslation()
  const theme = useTheme()

  return (
    <TableContainer component={Paper} variant="outlined">
      <Table size="small" aria-label="Classification">
        <TableHead>
          <TableRow sx={{ '& th': { fontWeight: 700, bgcolor: 'action.hover' } }}>
            <TableCell>{t('division.table.position')}</TableCell>
            <TableCell>{t('division.table.team')}</TableCell>
            <TableCell align="center">{t('division.table.points')}</TableCell>
            <TableCell align="center">{t('division.table.average')}</TableCell>
            <TableCell align="center">{t('division.table.wins')}</TableCell>
            <TableCell align="center">{t('division.table.draws')}</TableCell>
            <TableCell align="center">{t('division.table.losses')}</TableCell>
            <TableCell align="center">{t('division.table.goalsFor')}</TableCell>
            <TableCell align="center">{t('division.table.goalsAgainst')}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {positions.map((row) => {
            const isMyTeam = row.teamId === myTeamId
            const goalDiff = row.goalsScored - row.goalsReceived
            const rankColor = RANK_COLORS[row.position - 1]

            return (
              <TableRow
                key={row.position}
                hover
                sx={isMyTeam ? { bgcolor: alpha(theme.palette.primary.main, 0.08) } : undefined}
              >
                <TableCell component="th" scope="row">
                  <Box
                    sx={{
                      width: 22,
                      height: 22,
                      borderRadius: '50%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 12,
                      fontWeight: 700,
                      bgcolor: rankColor ?? 'transparent',
                      color: rankColor ? '#fff' : 'text.primary',
                    }}
                  >
                    {row.position}
                  </Box>
                </TableCell>
                <TableCell>
                  <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                    <TeamBadge teamId={row.teamId} teamName={row.teamName} size={24} />
                    <RouterLink
                      to={`/team/${row.teamId}`}
                      style={{ color: 'inherit', textDecoration: 'none' }}
                    >
                      <Typography
                        variant="body2"
                        sx={{ fontWeight: isMyTeam ? 700 : 400, '&:hover': { textDecoration: 'underline' } }}
                      >
                        {row.teamName}
                      </Typography>
                    </RouterLink>
                    {isMyTeam && (
                      <Chip label={t('division.table.you')} size="small" color="primary" variant="outlined" sx={{ height: 18, fontSize: 10 }} />
                    )}
                  </Stack>
                </TableCell>
                <TableCell align="center" sx={{ fontWeight: 700 }}>
                  {row.points}
                </TableCell>
                <TableCell
                  align="center"
                  sx={{
                    color: goalDiff > 0 ? 'success.main' : goalDiff < 0 ? 'error.main' : 'text.secondary',
                    fontWeight: 600,
                  }}
                >
                  {goalDiff > 0 ? `+${goalDiff}` : goalDiff}
                </TableCell>
                <TableCell align="center">{row.wins}</TableCell>
                <TableCell align="center">{row.draws}</TableCell>
                <TableCell align="center">{row.losses}</TableCell>
                <TableCell align="center">{row.goalsScored}</TableCell>
                <TableCell align="center">{row.goalsReceived}</TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

function TopScorers({ leagueId }: { leagueId: string | undefined }) {
  const { t } = useTranslation()
  const { data: scorers, isLoading } = useTopScorers(leagueId)

  return (
    <Paper variant="outlined" sx={{ height: '100%' }}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', p: 2, pb: 1 }}>
        <SportsSoccerTwoToneIcon color="primary" fontSize="small" />
        <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
          {t('division.topScorers.title')}
        </Typography>
      </Stack>
      {isLoading || !scorers ? (
        <Skeleton variant="rectangular" height={100} sx={{ m: 2, borderRadius: 1 }} />
      ) : scorers.length === 0 ? (
        <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
          {t('division.topScorers.noGoals')}
        </Typography>
      ) : (
        <Stack divider={<Box sx={{ borderBottom: '1px solid', borderColor: 'divider' }} />}>
          {scorers.map((scorer: ScorerStanding, index: number) => {
            const rankColor = RANK_COLORS[index]
            return (
              <Stack
                key={scorer.playerId}
                direction="row"
                spacing={1.5}
                sx={{ alignItems: 'center', px: 2, py: 1 }}
              >
                <Box
                  sx={{
                    width: 22,
                    height: 22,
                    flexShrink: 0,
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 11,
                    fontWeight: 700,
                    bgcolor: rankColor ?? 'action.selected',
                    color: rankColor ? '#fff' : 'text.secondary',
                  }}
                >
                  {index + 1}
                </Box>
                <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                  <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
                    <PlayerNameLink playerId={scorer.playerId}>
                      {scorer.firstName} {scorer.lastName}
                    </PlayerNameLink>
                  </Typography>
                  <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                    <TeamNameLink teamId={scorer.teamId}>{scorer.teamName}</TeamNameLink>
                  </Typography>
                </Box>
                <Typography variant="body2" sx={{ fontWeight: 700 }}>
                  {scorer.goals}
                </Typography>
              </Stack>
            )
          })}
        </Stack>
      )}
    </Paper>
  )
}

function FixturesCard({ title, matches }: { title: string; matches: WeekMatch[] }) {
  const { t } = useTranslation()

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden', height: '100%' }}>
      <Typography variant="subtitle1" sx={{ fontWeight: 700, px: 2, pt: 2, pb: matches.length === 0 ? 2 : 1 }}>
        {title}
      </Typography>
      {matches.length === 0 ? (
        <Typography variant="body2" color="text.secondary" sx={{ px: 2, pb: 2 }}>
          {t('division.noMatches')}
        </Typography>
      ) : (
        <Stack divider={<Box sx={{ borderBottom: '1px solid', borderColor: 'divider' }} />}>
          {matches.map((match) => {
            // played flips true in the DB the instant a match is simulated, well before its
            // result finishes broadcasting to viewers in real time (~90 minutes, see
            // mightStillBeLive's comment) - showing the final score here immediately would spoil
            // a match that still looks "live" everywhere else (the match details page, the team
            // matches list). Show a LIVE chip instead until that window has elapsed.
            const stillLive = match.played && mightStillBeLive(match.matchDate)
            return (
              <Box
                key={match.matchId}
                component={RouterLink}
                to={`/match/${match.matchId}`}
                sx={{
                  display: 'block',
                  textDecoration: 'none',
                  color: 'inherit',
                  px: 2,
                  py: 1.25,
                  '&:hover': { bgcolor: 'action.hover' },
                }}
              >
                <Grid container sx={{ alignItems: 'center' }}>
                  <Grid size={5}>
                    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'flex-end' }}>
                      <NestedTeamNameLink teamId={match.home} variant="body2" align="right" noWrap sx={{ fontWeight: 500 }}>
                        {match.homeName}
                      </NestedTeamNameLink>
                      <TeamBadge teamId={match.home} teamName={match.homeName} size={22} />
                    </Stack>
                  </Grid>
                  <Grid size={2}>
                    {stillLive ? (
                      <Chip label={t('match.live')} color="error" size="small" sx={{ display: 'flex', mx: 'auto', width: 'fit-content', fontWeight: 700 }} />
                    ) : (
                      <Typography variant="body2" align="center" sx={{ fontWeight: 700 }}>
                        {match.played && isPast(match.matchDate) ? `${match.homeGoals} - ${match.awayGoals}` : t('division.vs')}
                      </Typography>
                    )}
                  </Grid>
                  <Grid size={5}>
                    <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                      <TeamBadge teamId={match.away} teamName={match.awayName} size={22} />
                      <NestedTeamNameLink teamId={match.away} variant="body2" noWrap sx={{ fontWeight: 500 }}>
                        {match.awayName}
                      </NestedTeamNameLink>
                    </Stack>
                  </Grid>
                </Grid>
                <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center', justifyContent: 'center', mt: 0.5 }}>
                  <CalendarTodayTwoToneIcon sx={{ fontSize: 12 }} color="disabled" />
                  <Typography variant="caption" color="text.secondary">
                    {formatDateTime(match.matchDate)}
                  </Typography>
                </Stack>
              </Box>
            )
          })}
        </Stack>
      )}
    </Paper>
  )
}
