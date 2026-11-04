import { useState, type ReactNode } from 'react'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
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
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import IconButton from '@mui/material/IconButton'
import MenuItem from '@mui/material/MenuItem'
import TextField from '@mui/material/TextField'
import { Link as RouterLink, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useDivisionDetails,
  useLeagueSeasons,
  useLeagueSeasonTable,
  useLeagueWeekMatches,
  useTopScorers,
  useUserTeam,
} from '../../api/hooks'
import type { ClassificationPosition, LeagueSeasonTable, ScorerStanding, WeekMatch } from '../../api/types'
import { EmptyState } from '../../components/EmptyState'
import { PageHeader } from '../../components/PageHeader'
import { TeamBadge } from '../../components/TeamBadge'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { isPast, mightStillBeLive } from '../../api/useLiveMatch'
import { formatDivisionName } from '../../utils/divisionName'
import { formatDate, formatDateTime } from '../../utils/formatDate'
import { RecentLeagueTransfers } from '../leagues/RecentLeagueTransfers'
import { CountryFlag } from '../../components/CountryFlag'

const RANK_COLORS = ['#D4AF37', '#A8A9AD', '#B08D57'] as const
const TOP_SCORERS_PREVIEW = 5

type DivisionTab = 'table' | 'fixtures' | 'scorers' | 'transfers'

export function DivisionDetailsPage() {
  const { t } = useTranslation()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: divisionDetails, isLoading, dataUpdatedAt: divisionUpdatedAt } = useDivisionDetails(id)
  const { data: seasons, isLoading: seasonsLoading } = useLeagueSeasons(id)
  const { data: userTeam } = useUserTeam(true)
  const [tab, setTab] = useState<DivisionTab>('table')

  // A `season` that is not one of the league's past seasons shows the current season.
  const pastSeasons = Array.isArray(seasons?.past) ? seasons.past : []
  const requested = requestedSeason(searchParams.get('season'))
  const pastSeason = requested !== undefined && pastSeasons.includes(requested) ? requested : undefined
  const { data: seasonTable, isLoading: seasonTableLoading } = useLeagueSeasonTable(id, pastSeason)

  const waitingForSeason = requested !== undefined && (seasonsLoading || (pastSeason !== undefined && seasonTableLoading))
  if (isLoading || !divisionDetails || waitingForSeason) {
    return <Skeleton variant="rectangular" height={400} sx={{ borderRadius: 2 }} />
  }

  const selectSeason = (season: number) => {
    // A new season starts on its default matchday pair, so the `week` of the other one is dropped.
    setSearchParams(season === seasons?.current ? {} : { season: String(season) }, { replace: true })
  }
  const picker =
    pastSeasons.length > 0 && seasons ? (
      <SeasonPicker current={seasons.current} past={pastSeasons} value={pastSeason ?? seasons.current} onSelect={selectSeason} />
    ) : null

  if (pastSeason !== undefined && seasonTable) {
    return (
      <PastSeasonView
        leagueId={id}
        table={seasonTable}
        picker={picker}
        myTeamId={userTeam?.team.id}
        isDesktop={isDesktop}
        tab={tab === 'fixtures' ? 'fixtures' : 'table'}
        onTab={setTab}
      />
    )
  }

  // Reverted once and reused below for both the round counter and the table itself - positions[0]'s
  // own `matches` count otherwise ticks over the instant a match is simulated too, up to 15
  // minutes before its official kickoff (see hideStillLiveResults).
  const positions = hideStillLiveResults(divisionDetails.classification.positions, divisionDetails.previousWeekMatches)
  const currentWeek = positions[0]?.matches ?? 0
  const teamCount = positions.filter((row) => row.teamId !== 0).length
  const totalWeeks = Math.max(0, (teamCount - 1) * 2)
  const myTeamId = userTeam?.team.id

  const promotionSlots = divisionDetails.promotionSlots ?? 0
  const relegationSlots = divisionDetails.relegationSlots ?? 0
  const rowCount = positions.filter((row) => row.teamId !== 0).length
  const table = (
    <Stack spacing={1}>
      {picker}
      <ClassificationTable
        positions={positions}
        myTeamId={myTeamId}
        zoneOf={(row) => zoneFor(row.position, rowCount, promotionSlots, relegationSlots)}
        showPromotionLegend={promotionSlots > 0}
        showRelegationLegend={relegationSlots > 0}
      />
    </Stack>
  )
  const fixtures = (
    <MatchdayFixtures
      leagueId={id}
      currentWeek={currentWeek}
      totalWeeks={totalWeeks}
      // The seeds are the backend's own W (before hideStillLiveResults reverts it): that is the
      // week previousWeekMatches belongs to.
      backendWeek={divisionDetails.classification.positions[0]?.matches ?? 0}
      previousWeekMatches={divisionDetails.previousWeekMatches}
      nextWeekMatches={divisionDetails.nextWeekMatches}
      updatedAt={divisionUpdatedAt}
      myTeamId={myTeamId}
    />
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
            {/* Four tabs share 328px at 360px wide: drop MUI's 90px minWidth and trim the padding. */}
            <Tab value="table" label={t('division.tabs.table')} sx={{ minWidth: 0, px: 1 }} />
            <Tab value="fixtures" label={t('division.tabs.fixtures')} sx={{ minWidth: 0, px: 1 }} />
            <Tab value="scorers" label={t('division.tabs.scorers')} sx={{ minWidth: 0, px: 1 }} />
            <Tab value="transfers" label={t('division.tabs.transfers')} sx={{ minWidth: 0, px: 1 }} />
          </Tabs>
          {tab === 'table' && table}
          {tab === 'fixtures' && fixtures}
          {tab === 'scorers' && <TopScorers leagueId={id} limit={TOP_SCORERS_PREVIEW} />}
          {tab === 'transfers' && <RecentLeagueTransfers leagueId={id} />}
        </>
      )}
    </Box>
  )
}

// PastSeasonView is a finished season, read-only: its final table (zones from the recorded
// promotions and relegations, a trophy on the champion) and its matchdays, opening on the last pair
// with no chips. Top scorers and recent transfers belong to the current season and are not shown.
function PastSeasonView({
  leagueId,
  table,
  picker,
  myTeamId,
  isDesktop,
  tab,
  onTab,
}: {
  leagueId: string | undefined
  table: LeagueSeasonTable
  picker: ReactNode
  myTeamId: number | undefined
  isDesktop: boolean
  tab: 'table' | 'fixtures'
  onTab: (tab: DivisionTab) => void
}) {
  const { t } = useTranslation()
  const positions = pastSeasonPositions(table)
  const byTeam = new Map(table.positions.map((row) => [row.teamId, row]))
  const standings = (
    <Stack spacing={1}>
      {picker}
      <ClassificationTable
        positions={positions}
        myTeamId={myTeamId}
        zoneOf={(row) => pastSeasonZone(byTeam.get(row.teamId))}
        champion
        showPromotionLegend={table.positions.some((row) => row.promoted)}
        showRelegationLegend={table.positions.some((row) => row.relegated)}
      />
    </Stack>
  )
  const fixtures = (
    <MatchdayFixtures
      leagueId={leagueId}
      season={table.season}
      currentWeek={table.totalWeeks}
      totalWeeks={table.totalWeeks}
      backendWeek={0}
      previousWeekMatches={[]}
      nextWeekMatches={[]}
      updatedAt={0}
      myTeamId={myTeamId}
    />
  )

  return (
    <Box>
      <PageHeader
        title={formatDivisionName(t, table.name)}
        titleText={table.name}
        subtitle={t('division.seasonFinal', { season: table.season })}
      />
      {isDesktop ? (
        <Stack spacing={2}>
          <Grid container spacing={2}>
            <Grid size={{ md: 8 }}>{standings}</Grid>
          </Grid>
          {fixtures}
        </Stack>
      ) : (
        <>
          <Tabs
            value={tab}
            onChange={(_event, value: DivisionTab) => onTab(value)}
            variant="fullWidth"
            sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}
          >
            <Tab value="table" label={t('division.tabs.table')} sx={{ minWidth: 0, px: 1 }} />
            <Tab value="fixtures" label={t('division.tabs.fixtures')} sx={{ minWidth: 0, px: 1 }} />
          </Tabs>
          {tab === 'table' && standings}
          {tab === 'fixtures' && fixtures}
        </>
      )}
    </Box>
  )
}

// pastSeasonPositions shapes a past season's final table like the live classification.
function pastSeasonPositions(table: LeagueSeasonTable): ClassificationPosition[] {
  return table.positions.map((row) => ({
    position: row.position,
    teamId: row.teamId,
    teamName: row.name,
    points: row.points,
    matches: row.matches,
    wins: row.wins,
    draws: row.draws,
    losses: row.losses,
    goalsScored: row.goalsScored,
    goalsReceived: row.goalsReceived,
  }))
}

// A past season's zones are what really happened, not the current slot counts.
function pastSeasonZone(row: { promoted: boolean; relegated: boolean } | undefined): Zone {
  if (row?.promoted) return 'promotion'
  if (row?.relegated) return 'relegation'
  return undefined
}

// requestedSeason reads the `season` query parameter: anything that isn't a positive integer is ignored.
function requestedSeason(param: string | null): number | undefined {
  if (param === null || !/^\d+$/.test(param)) return undefined
  const season = Number(param)
  return season >= 1 ? season : undefined
}

// The season picker above the standings: the current season, then the past ones, newest first.
function SeasonPicker({
  current,
  past,
  value,
  onSelect,
}: {
  current: number
  past: number[]
  value: number
  onSelect: (season: number) => void
}) {
  const { t } = useTranslation()
  return (
    <TextField
      select
      size="small"
      value={value}
      onChange={(event) => onSelect(Number(event.target.value))}
      slotProps={{ htmlInput: { 'aria-label': t('division.seasonPicker') } }}
      sx={{ alignSelf: 'flex-start', minWidth: 200, maxWidth: '100%', '& .MuiInputBase-root': { minHeight: TAP_TARGET } }}
    >
      <MenuItem value={current} sx={{ minHeight: TAP_TARGET }}>
        {t('division.seasonCurrent', { season: current })}
      </MenuItem>
      {past.map((season) => (
        <MenuItem key={season} value={season} sx={{ minHeight: TAP_TARGET }}>
          {t('division.seasonOption', { season })}
        </MenuItem>
      ))}
    </TextField>
  )
}

// matchdayPair resolves which two matchdays the fixtures section shows: `left` and left + 1. It
// is the requested week (the `week` query parameter), or the current week by default, clamped so
// both fit in the season (1..totalWeeks - 1, and 1 for a season too short to have two). isDefault
// tells whether that is the pair the page opens on - the navigator then drops the parameter.
export function matchdayPair({ currentWeek, totalWeeks, requested }: { currentWeek: number; totalWeeks: number; requested?: number }): {
  left: number
  isDefault: boolean
} {
  const max = Math.max(1, totalWeeks - 1)
  const clamp = (week: number) => Math.min(Math.max(week, 1), max)
  const left = clamp(requested ?? currentWeek)
  return { left, isDefault: left === clamp(currentWeek) }
}

// matchdayChip labels the matchday that was just played or is next up - whichever pair is shown.
export function matchdayChip(week: number, currentWeek: number): 'lastWeek' | 'nextWeek' | undefined {
  if (week === currentWeek) return 'lastWeek'
  if (week === currentWeek + 1) return 'nextWeek'
  return undefined
}

// requestedWeek reads the `week` query parameter: anything that isn't an integer is ignored.
function requestedWeek(param: string | null): number | undefined {
  if (param === null || !/^-?\d+$/.test(param)) return undefined
  return Number(param)
}

function MatchdayFixtures({
  leagueId,
  season,
  currentWeek,
  totalWeeks,
  backendWeek,
  previousWeekMatches,
  nextWeekMatches,
  updatedAt,
  myTeamId,
}: {
  leagueId: string | undefined
  // A past season: no seeds, no chips, and every matchday is requested for that season.
  season?: number
  currentWeek: number
  totalWeeks: number
  backendWeek: number
  previousWeekMatches: WeekMatch[]
  nextWeekMatches: WeekMatch[]
  updatedAt: number
  myTeamId: number | undefined
}) {
  const [searchParams, setSearchParams] = useSearchParams()
  const { left, isDefault } = matchdayPair({ currentWeek, totalWeeks, requested: requestedWeek(searchParams.get('week')) })
  const right = left + 1

  const seedFor = (week: number) => {
    if (season !== undefined) return undefined
    if (week === backendWeek) return { matches: previousWeekMatches, updatedAt }
    if (week === backendWeek + 1) return { matches: nextWeekMatches, updatedAt }
    return undefined
  }
  const leftQuery = useLeagueWeekMatches(leagueId, left, seedFor(left), season)
  const rightQuery = useLeagueWeekMatches(leagueId, right, seedFor(right), season)
  const chipFor = (week: number) => (season === undefined ? matchdayChip(week, currentWeek) : undefined)

  const goTo = (week: number) => {
    const { left: target, isDefault: toDefault } = matchdayPair({ currentWeek, totalWeeks, requested: week })
    // Only `week` changes: a chosen `season` stays.
    const next = new URLSearchParams(searchParams)
    if (toDefault) next.delete('week')
    else next.set('week', String(target))
    setSearchParams(next, { replace: true })
  }

  return (
    <Stack spacing={2}>
      <MatchdayNavigator left={left} totalWeeks={totalWeeks} isDefault={isDefault} onSelect={goTo} onReset={() => goTo(currentWeek)} />
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <FixturesCard
            week={left}
            chip={chipFor(left)}
            matches={leftQuery.data}
            failed={leftQuery.isError}
            myTeamId={myTeamId}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <FixturesCard
            week={right}
            chip={chipFor(right)}
            matches={rightQuery.data}
            failed={rightQuery.isError}
            myTeamId={myTeamId}
          />
        </Grid>
      </Grid>
    </Stack>
  )
}

const TAP_TARGET = 44

// ‹ [Jornada picker] › plus a "current matchday" button that only shows off the default pair.
function MatchdayNavigator({
  left,
  totalWeeks,
  isDefault,
  onSelect,
  onReset,
}: {
  left: number
  totalWeeks: number
  isDefault: boolean
  onSelect: (week: number) => void
  onReset: () => void
}) {
  const { t } = useTranslation()
  const weeks = Array.from({ length: Math.max(totalWeeks, left + 1) }, (_unused, index) => index + 1)
  const lastLeft = Math.max(1, totalWeeks - 1)

  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 1 }}>
      <IconButton
        aria-label={t('division.previousMatchday')}
        disabled={left <= 1}
        onClick={() => onSelect(left - 1)}
        sx={{ width: TAP_TARGET, height: TAP_TARGET }}
      >
        <ChevronLeftIcon />
      </IconButton>
      <TextField
        select
        size="small"
        value={left}
        onChange={(event) => onSelect(Number(event.target.value))}
        slotProps={{ htmlInput: { 'aria-label': t('division.matchdayPicker') } }}
        sx={{ minWidth: 150, '& .MuiInputBase-root': { minHeight: TAP_TARGET } }}
      >
        {weeks.map((week) => (
          <MenuItem key={week} value={week} sx={{ minHeight: TAP_TARGET }}>
            {t('division.matchday', { week })}
          </MenuItem>
        ))}
      </TextField>
      <IconButton
        aria-label={t('division.nextMatchday')}
        disabled={left >= lastLeft}
        onClick={() => onSelect(left + 1)}
        sx={{ width: TAP_TARGET, height: TAP_TARGET }}
      >
        <ChevronRightIcon />
      </IconButton>
      {!isDefault && (
        <Button onClick={onReset} sx={{ minHeight: TAP_TARGET }}>
          {t('division.backToCurrent')}
        </Button>
      )}
    </Stack>
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
    if (b.draws !== a.draws) return b.draws - a.draws
    // Then the season's starting order (no seed last) and the team id, as the backend does.
    const seedA = a.seed ?? Number.MAX_SAFE_INTEGER
    const seedB = b.seed ?? Number.MAX_SAFE_INTEGER
    if (seedA !== seedB) return seedA - seedB
    return a.teamId - b.teamId
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
  zoneOf,
  champion = false,
  showPromotionLegend,
  showRelegationLegend,
}: {
  positions: ClassificationPosition[]
  myTeamId: number | undefined
  zoneOf: (row: ClassificationPosition) => Zone
  // A finished season: a trophy marks the team in first place.
  champion?: boolean
  showPromotionLegend: boolean
  showRelegationLegend: boolean
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
              const zone = zoneOf(row)

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
                      <TeamNameLink teamId={row.teamId} variant="body2" noWrap sx={{ minWidth: 0, fontWeight: isMyTeam ? 700 : 400 }}>
                        {row.teamName}
                      </TeamNameLink>
                      {champion && row.position === 1 && (
                        <EmojiEventsTwoToneIcon
                          titleAccess={t('division.champion')}
                          fontSize="small"
                          sx={{ color: RANK_COLORS[0], flexShrink: 0 }}
                        />
                      )}
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
        {showPromotionLegend && <ZoneLegend color={zoneColor.promotion} label={t('division.legend.promotion')} />}
        {showRelegationLegend && <ZoneLegend color={zoneColor.relegation} label={t('division.legend.relegation')} />}
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
                      <CountryFlag code={scorer.birthCountry} height={14} marginRight={0.5} />
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
function FixturesCard({
  week,
  chip,
  matches,
  failed,
  myTeamId,
}: {
  week: number
  chip: 'lastWeek' | 'nextWeek' | undefined
  // undefined while the matchday is loading
  matches: WeekMatch[] | undefined
  failed?: boolean
  myTeamId: number | undefined
}) {
  const { t } = useTranslation()
  const theme = useTheme()

  const groups: { date: string; matches: WeekMatch[] }[] = []
  for (const match of matches ?? []) {
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
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', px: 2, pt: 2, pb: matches?.length === 0 ? 2 : 1 }}>
        <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 700 }}>
          {t('division.matchday', { week })}
        </Typography>
        {chip && <Chip label={t(`division.${chip}`)} size="small" color="primary" variant="outlined" />}
      </Stack>
      {matches === undefined ? (
        failed ? (
          <EmptyState compact title={t('division.matchdayError')} />
        ) : (
          <Skeleton variant="rectangular" height={120} sx={{ m: 2, borderRadius: 1 }} />
        )
      ) : matches.length === 0 ? (
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
