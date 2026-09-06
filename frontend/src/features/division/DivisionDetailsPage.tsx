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
import { formatDateTime } from '../../utils/formatDate'

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

  const currentWeek = divisionDetails.classification.positions[0]?.matches ?? 0
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
          <ClassificationTable positions={divisionDetails.classification.positions} myTeamId={myTeamId} />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <TopScorers leagueId={id} />
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
                    {scorer.firstName} {scorer.lastName}
                  </Typography>
                  <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                    {scorer.teamName}
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
          {matches.map((match) => (
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
                    <Typography variant="body2" align="right" noWrap sx={{ fontWeight: 500 }}>
                      {match.homeName}
                    </Typography>
                    <TeamBadge teamId={match.home} teamName={match.homeName} size={22} />
                  </Stack>
                </Grid>
                <Grid size={2}>
                  <Typography variant="body2" align="center" sx={{ fontWeight: 700 }}>
                    {match.played ? `${match.homeGoals} - ${match.awayGoals}` : t('division.vs')}
                  </Typography>
                </Grid>
                <Grid size={5}>
                  <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                    <TeamBadge teamId={match.away} teamName={match.awayName} size={22} />
                    <Typography variant="body2" noWrap sx={{ fontWeight: 500 }}>
                      {match.awayName}
                    </Typography>
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
          ))}
        </Stack>
      )}
    </Paper>
  )
}
