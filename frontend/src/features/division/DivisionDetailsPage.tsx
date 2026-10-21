import { useState } from 'react'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Switch from '@mui/material/Switch'
import FormControlLabel from '@mui/material/FormControlLabel'
import Tab from '@mui/material/Tab'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { alpha, useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useDivisionDetails, useTopScorers, useUserTeam } from '../../api/hooks'
import type { ClassificationPosition, ScorerStanding, WeekMatch } from '../../api/types'
import { EmptyState } from '../../components/EmptyState'
import { PageHeader } from '../../components/PageHeader'
import { TeamBadge } from '../../components/TeamBadge'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { isPast, mightStillBeLive } from '../../api/useLiveMatch'
import { formatDivisionName } from '../../utils/divisionName'
import { formatDate, formatDateTime } from '../../utils/formatDate'
import { RecentLeagueTransfers } from '../leagues/RecentLeagueTransfers'

const RANK_COLORS = ['#D4AF37', '#A8A9AD', '#B08D57'] as const
const TOP_SCORERS_PREVIEW = 5

type DivisionTab = 'table' | 'fixtures' | 'scorers'

export function DivisionDetailsPage() {
  const { t } = useTranslation()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const { id } = useParams<{ id: string }>()
  const { data: divisionDetails, isLoading } = useDivisionDetails(id)
  const { data: userTeam } = useUserTeam(true)
  const [tab, setTab] = useState<DivisionTab>('table')

  if (isLoading || !divisionDetails) {
    return <Skeleton variant="rectangular" height={400} sx={{ borderRadius: 2 }} />
  }

  // Reverted once and reused below for both the round counter and the table itself - positions[0]'s
  // own `matches` count otherwise ticks over the instant a match is simulated too, up to 15
  // minutes before its official kickoff (see hideStillLiveResults).
  const positions = hideStillLiveResults(divisionDetails.classification.positions, divisionDetails.previousWeekMatches)
  const currentWeek = positions[0]?.matches ?? 0
  const teamCount = positions.filter((row) => row.teamId !== 0).length
  const totalWeeks = Math.max(0, (teamCount - 1) * 2)
  const myTeamId = userTeam?.team.id

  const table = (
    <ClassificationTable
      positions={positions}
      myTeamId={myTeamId}
      promotionSlots={divisionDetails.promotionSlots ?? 0}
      relegationSlots={divisionDetails.relegationSlots ?? 0}
    />
  )
  const fixtures = (
    <Grid container spacing={2}>
      <Grid size={{ xs: 12, md: 6 }}>
        <FixturesCard title={t('division.lastWeek')} matches={divisionDetails.previousWeekMatches} myTeamId={myTeamId} />
      </Grid>
      <Grid size={{ xs: 12, md: 6 }}>
        <FixturesCard title={t('division.nextWeek')} matches={divisionDetails.nextWeekMatches} myTeamId={myTeamId} />
      </Grid>
    </Grid>
  )

  return (
    <Box>
      <PageHeader
        title={formatDivisionName(t, divisionDetails.name)}
        titleText={divisionDetails.name}
        subtitle={
          totalWeeks > 0
            ? t('division.subtitle', { season: divisionDetails.season, week: currentWeek, total: totalWeeks })
            : t('division.season', { season: divisionDetails.season })
        }
      />

      {isDesktop ? (
        <>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid size={{ md: 8 }}>{table}</Grid>
            <Grid size={{ md: 4 }}>
              <Stack spacing={2}>
                <TopScorers leagueId={id} limit={TOP_SCORERS_PREVIEW} />
                <RecentLeagueTransfers leagueId={id} />
              </Stack>
            </Grid>
          </Grid>
          {fixtures}
        </>
      ) : (
        <>
          <Tabs
            value={tab}
            onChange={(_event, value: DivisionTab) => setTab(value)}
            variant="fullWidth"
            sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}
          >
            <Tab value="table" label={t('division.tabs.table')} />
            <Tab value="fixtures" label={t('division.tabs.fixtures')} />
            <Tab value="scorers" label={t('division.tabs.scorers')} />
          </Tabs>
          {tab === 'table' && table}
          {tab === 'fixtures' && fixtures}
          {tab === 'scorers' && (
            <Stack spacing={2}>
              <TopScorers leagueId={id} limit={TOP_SCORERS_PREVIEW} />
              <RecentLeagueTransfers leagueId={id} />
            </Stack>
          )}
        </>
      )}
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

type Zone = 'promotion' | 'relegation' | undefined

function zoneFor(position: number, total: number, promotionSlots: number, relegationSlots: number): Zone {
  if (promotionSlots > 0 && position <= promotionSlots) return 'promotion'
  if (relegationSlots > 0 && position > total - relegationSlots) return 'relegation'
  return undefined
}

// Compact (phone) shows # Team PJ DG Pts; the full table adds G/E/P/GF/GC and scrolls
// horizontally with the first two columns pinned.
function ClassificationTable({
  positions,
  myTeamId,
  promotionSlots,
  relegationSlots,
}: {
  positions: ClassificationPosition[]
  myTeamId: number | undefined
  promotionSlots: number
  relegationSlots: number
}) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const [showFull, setShowFull] = useState(false)
  const full = isDesktop || showFull
  const rows = positions.filter((row) => row.teamId !== 0)
  const zoneColor = { promotion: theme.palette.success.main, relegation: theme.palette.error.main }

  return (
    <Box>
      {!isDesktop && (
        <FormControlLabel
          sx={{ ml: 0, mb: 0.5 }}
          control={<Switch size="small" checked={showFull} onChange={(event) => setShowFull(event.target.checked)} />}
          label={<Typography variant="body2">{t('division.fullTable')}</Typography>}
        />
      )}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small" aria-label={t('division.tabs.table')} sx={{ '& td, & th': { px: { xs: 0.75, sm: 2 } } }}>
          <TableHead>
            <TableRow sx={{ '& th': { fontWeight: 700, bgcolor: 'action.hover' } }}>
              <TableCell sx={{ width: 36 }}>{t('division.table.position')}</TableCell>
              <TableCell>{t('division.table.team')}</TableCell>
              <TableCell align="center" title={t('division.table.playedLong')}>{t('division.table.played')}</TableCell>
              {full && <TableCell align="center" title={t('division.table.winsLong')}>{t('division.table.wins')}</TableCell>}
              {full && <TableCell align="center" title={t('division.table.drawsLong')}>{t('division.table.draws')}</TableCell>}
              {full && <TableCell align="center" title={t('division.table.lossesLong')}>{t('division.table.losses')}</TableCell>}
              {full && <TableCell align="center" title={t('division.table.goalsForLong')}>{t('division.table.goalsFor')}</TableCell>}
              {full && <TableCell align="center" title={t('division.table.goalsAgainstLong')}>{t('division.table.goalsAgainst')}</TableCell>}
              <TableCell align="center" title={t('division.table.goalDifferenceLong')}>{t('division.table.goalDifference')}</TableCell>
              <TableCell align="center" title={t('division.table.pointsLong')}>{t('division.table.points')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row) => {
              const isMyTeam = row.teamId === myTeamId
              const goalDiff = row.goalsScored - row.goalsReceived
              const rankColor = RANK_COLORS[row.position - 1]
              const zone = zoneFor(row.position, rows.length, promotionSlots, relegationSlots)

              return (
                <TableRow
                  key={row.position}
                  hover
                  sx={{
                    ...(isMyTeam ? { bgcolor: alpha(theme.palette.primary.main, 0.08) } : undefined),
                    // Zone marker: a coloured bar on the row's leading edge.
                    '& > :first-of-type': { boxShadow: zone ? `inset 4px 0 0 ${zoneColor[zone]}` : undefined },
                  }}
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
                        typography: 'chipLabel',
                        fontWeight: 700,
                        bgcolor: rankColor ?? 'transparent',
                        color: rankColor ? '#fff' : 'text.primary',
                      }}
                    >
                      {row.position}
                    </Box>
                  </TableCell>
                  <TableCell sx={{ maxWidth: { xs: 150, sm: 'none' } }}>
                    <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                      <TeamBadge teamId={row.teamId} teamName={row.teamName} size={24} />
                      <RouterLink to={`/team/${row.teamId}`} style={{ color: 'inherit', textDecoration: 'none', minWidth: 0 }}>
                        <Typography
                          variant="body2"
                          noWrap
                          sx={{ fontWeight: isMyTeam ? 700 : 400, '&:hover': { textDecoration: 'underline' } }}
                        >
                          {row.teamName}
                        </Typography>
                      </RouterLink>
                      {isMyTeam && <Chip label={t('division.table.you')} size="small" color="primary" variant="outlined" sx={{ height: 20 }} />}
                    </Stack>
                  </TableCell>
                  <TableCell align="center">{row.matches}</TableCell>
                  {full && <TableCell align="center">{row.wins}</TableCell>}
                  {full && <TableCell align="center">{row.draws}</TableCell>}
                  {full && <TableCell align="center">{row.losses}</TableCell>}
                  {full && <TableCell align="center">{row.goalsScored}</TableCell>}
                  {full && <TableCell align="center">{row.goalsReceived}</TableCell>}
                  <TableCell
                    align="center"
                    sx={{
                      color: goalDiff > 0 ? 'success.main' : goalDiff < 0 ? 'error.main' : 'text.secondary',
                      fontWeight: 600,
                      fontVariantNumeric: 'tabular-nums',
                    }}
                  >
                    {goalDiff > 0 ? `+${goalDiff}` : goalDiff}
                  </TableCell>
                  <TableCell align="center" sx={{ fontWeight: 700 }}>
                    {row.points}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </TableContainer>
      <Stack direction="row" spacing={2} sx={{ mt: 1, flexWrap: 'wrap', rowGap: 0.5 }} aria-label={t('division.legend.title')}>
        {promotionSlots > 0 && <ZoneLegend color={zoneColor.promotion} label={t('division.legend.promotion')} />}
        {relegationSlots > 0 && <ZoneLegend color={zoneColor.relegation} label={t('division.legend.relegation')} />}
        <Typography variant="caption" color="text.secondary">
          {t('division.legend.columns')}
        </Typography>
      </Stack>
    </Box>
  )
}

function ZoneLegend({ color, label }: { color: string; label: string }) {
  return (
    <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
      <Box sx={{ width: 4, height: 16, borderRadius: 1, bgcolor: color }} />
      <Typography variant="caption">{label}</Typography>
    </Stack>
  )
}

function TopScorers({ leagueId, limit }: { leagueId: string | undefined; limit: number }) {
  const { t } = useTranslation()
  const { data: scorers, isLoading } = useTopScorers(leagueId)
  const [showAll, setShowAll] = useState(false)
  const visible = scorers ? (showAll ? scorers : scorers.slice(0, limit)) : []

  return (
    <Paper variant="outlined" sx={{ height: '100%' }}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', p: 2, pb: 1 }}>
        <SportsSoccerTwoToneIcon color="primary" fontSize="small" />
        <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 700 }}>
          {t('division.topScorers.title')}
        </Typography>
      </Stack>
      {isLoading || !scorers ? (
        <Skeleton variant="rectangular" height={100} sx={{ m: 2, borderRadius: 1 }} />
      ) : scorers.length === 0 ? (
        <EmptyState compact title={t('division.topScorers.noGoals')} />
      ) : (
        <>
          <Stack divider={<Box sx={{ borderBottom: '1px solid', borderColor: 'divider' }} />}>
            {visible.map((scorer: ScorerStanding, index: number) => {
              const rankColor = RANK_COLORS[index]
              return (
                <Stack key={scorer.playerId} direction="row" spacing={1.5} sx={{ alignItems: 'center', px: 2, py: 1 }}>
                  <Box
                    sx={{
                      width: 22,
                      height: 22,
                      flexShrink: 0,
                      borderRadius: '50%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      typography: 'chipLabel',
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
          {scorers.length > limit && (
            <Box sx={{ p: 1, textAlign: 'center' }}>
              <Button onClick={() => setShowAll((current) => !current)}>
                {showAll ? t('common.viewLess') : t('division.topScorers.viewAll', { count: scorers.length })}
              </Button>
            </Box>
          )}
        </>
      )}
    </Paper>
  )
}

// A matchday's fixtures grouped under their date, each as two lines (home above away) with the
// score on the right - so full team names fit on a 360px screen. The user's own match is
// highlighted.
function FixturesCard({ title, matches, myTeamId }: { title: string; matches: WeekMatch[]; myTeamId: number | undefined }) {
  const { t } = useTranslation()
  const theme = useTheme()

  const groups: { date: string; matches: WeekMatch[] }[] = []
  for (const match of matches) {
    const key = formatDate(match.matchDate)
    const group = groups.find((g) => formatDate(g.matches[0].matchDate) === key)
    if (group) {
      group.matches.push(match)
    } else {
      groups.push({ date: match.matchDate, matches: [match] })
    }
  }

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden', height: '100%' }}>
      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 700, px: 2, pt: 2, pb: matches.length === 0 ? 2 : 1 }}>
        {title}
      </Typography>
      {matches.length === 0 ? (
        <EmptyState compact title={t('division.noMatches')} />
      ) : (
        groups.map((group) => (
          <Box key={group.date}>
            <Typography variant="overline" color="text.secondary" component="div" sx={{ px: 2, py: 0.5, bgcolor: 'action.hover' }}>
              {formatDateTime(group.date)}
            </Typography>
            <Stack divider={<Box sx={{ borderBottom: '1px solid', borderColor: 'divider' }} />}>
              {group.matches.map((match) => {
                // played flips true in the DB the instant a match is simulated, well before its
                // result finishes broadcasting to viewers in real time (~90 minutes, see
                // mightStillBeLive's comment) - showing the final score here immediately would
                // spoil a match that still looks "live" everywhere else (the match details page,
                // the team matches list). Show a LIVE chip instead until that window has elapsed.
                const stillLive = match.played && mightStillBeLive(match.matchDate)
                const showScore = !stillLive && match.played && isPast(match.matchDate)
                const mine = match.home === myTeamId || match.away === myTeamId
                return (
                  <Box
                    key={match.matchId}
                    component={RouterLink}
                    to={`/match/${match.matchId}`}
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 1.5,
                      textDecoration: 'none',
                      color: 'inherit',
                      px: 2,
                      py: 1,
                      minHeight: 56,
                      bgcolor: mine ? alpha(theme.palette.primary.main, 0.08) : undefined,
                      '&:hover': { bgcolor: 'action.hover' },
                    }}
                  >
                    <Stack spacing={0.5} sx={{ flexGrow: 1, minWidth: 0 }}>
                      {[
                        { id: match.home, name: match.homeName, goals: match.homeGoals },
                        { id: match.away, name: match.awayName, goals: match.awayGoals },
                      ].map((side) => (
                        <Stack key={side.id} direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                          <TeamBadge teamId={side.id} teamName={side.name} size={20} />
                          <Typography variant="body2" sx={{ fontWeight: side.id === myTeamId ? 700 : 500, overflowWrap: 'anywhere', flexGrow: 1 }}>
                            {side.name}
                          </Typography>
                          {showScore && (
                            <Typography variant="body2" sx={{ fontWeight: 700, fontVariantNumeric: 'tabular-nums' }}>
                              {side.goals}
                            </Typography>
                          )}
                        </Stack>
                      ))}
                    </Stack>
                    {stillLive && <Chip label={t('match.live')} color="error" size="small" sx={{ fontWeight: 700 }} />}
                  </Box>
                )
              })}
            </Stack>
          </Box>
        ))
      )}
    </Paper>
  )
}
