import { useEffect, useState, type ReactNode } from 'react'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import AccountBalanceWalletOutlinedIcon from '@mui/icons-material/AccountBalanceWalletOutlined'
import CalendarTodayTwoToneIcon from '@mui/icons-material/CalendarTodayTwoTone'
import EditIcon from '@mui/icons-material/Edit'
import EmojiEventsOutlinedIcon from '@mui/icons-material/EmojiEventsOutlined'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined'
import HandshakeTwoToneIcon from '@mui/icons-material/HandshakeTwoTone'
import LeaderboardOutlinedIcon from '@mui/icons-material/LeaderboardOutlined'
import LeaderboardTwoToneIcon from '@mui/icons-material/LeaderboardTwoTone'
import LiveTvTwoToneIcon from '@mui/icons-material/LiveTvTwoTone'
import OutboundTwoToneIcon from '@mui/icons-material/OutboundTwoTone'
import ShuffleTwoToneIcon from '@mui/icons-material/ShuffleTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import Accordion from '@mui/material/Accordion'
import AccordionDetails from '@mui/material/AccordionDetails'
import AccordionSummary from '@mui/material/AccordionSummary'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardActions from '@mui/material/CardActions'
import CardContent from '@mui/material/CardContent'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useQueryClient } from '@tanstack/react-query'
import useMediaQuery from '@mui/material/useMediaQuery'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { StatTile } from '../../components/m3/StatTile'
import { RivalryChip } from '../../components/RivalryChip'
import {
  queryKeys,
  useDivisionDetails,
  useDivisionLiveScores,
  useFinances,
  useGroupDetails,
  useHeadToHead,
  useLiveMatchesOverview,
  useMe,
  useMyGroups,
  useRenameTeam,
  useTeamMatches,
  useTopScorers,
  useUserTeam,
} from '../../api/hooks'
import {
  SPONSOR_DEAL_LABEL_KEYS,
  type ClassificationPosition,
  type DivisionDetails,
  type FinancesResponse,
  type LiveMatchSummary,
  type LoanedOutPlayer,
  type Team,
  type TeamMatchSummary,
} from '../../api/types'
import { isPast, mightStillBeLive, useLiveMatch, type LiveMatchState } from '../../api/useLiveMatch'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { DivisionLiveScores } from '../../components/DivisionLiveScores'
import { TeamBadge } from '../../components/TeamBadge'
import { TeamNameLink } from '../../components/TeamNameLink'
import { CountryFlag } from '../../components/CountryFlag'
import { countdownParts, countdownUntil } from '../../utils/countdown'
import { useNow } from '../../utils/useNow'
import { formatDivisionName } from '../../utils/divisionName'
import { ratingTierColor } from '../../utils/playerRating'
import { HomeAwayIcon } from '../../components/HomeAwayIcon'
import { localizedCountryName } from '../../utils/countryOrder'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { PULSING_LIVE_SX } from '../../utils/liveChip'
import { liveMatchPath, matchKindColor, matchKindLabelKey, matchSummaryLink } from '../../utils/matchSummary'
import { hideStillLiveResults } from '../division/DivisionDetailsPage'
import { RecentLeagueTransfers } from '../leagues/RecentLeagueTransfers'
import { PendingPanel } from './PendingPanel'
import { SeasonRecap } from './SeasonRecap'

// currentStanding reads the team's classification row with the same "don't spoil a match that's
// still within its live-broadcast window" reversion DivisionDetailsPage's own table applies (see
// hideStillLiveResults) - without it, this dashboard's league-position chip would jump to its
// post-match value the instant a match is simulated, well before the live broadcast the rest of
// the app still shows the viewer actually finishes.
export function currentStanding(
  division: DivisionDetails | undefined,
  teamId: number,
): ClassificationPosition | undefined {
  if (!division) {
    return undefined
  }
  return hideStillLiveResults(division.classification.positions, division.previousWeekMatches).find((p) => p.teamId === teamId)
}

// selectRecentForm picks the last `limit` played matches (most recent last, for FormStrip's
// left-to-right oldest-to-newest strip) from an already-played, most-recent-first list (see
// DashboardPage's own playedMatches). Unlike LastResultCard (which masks a still-live lastMatch
// behind a LIVE chip itself), FormStrip has no per-chip live state to fall back to - it always
// renders a W/L/D result - so a still-live match is dropped here entirely rather than spoiling it
// the instant kickoff arrives, well before its broadcast (shown elsewhere on this very dashboard)
// finishes.
/** The loans the dashboard's "Loaned out" card lists: the `limit` that come back soonest (return
 * season, then name), so the manager sees which players return next. */
export function soonestLoanReturns(loanedOut: LoanedOutPlayer[], limit: number): LoanedOutPlayer[] {
  return [...loanedOut]
    .sort((a, b) => a.returnSeason - b.returnSeason || a.player.lastName.localeCompare(b.player.lastName))
    .slice(0, limit)
}

export function selectRecentForm(playedMatches: TeamMatchSummary[], limit: number): TeamMatchSummary[] {
  return playedMatches
    .filter((m) => !mightStillBeLive(m.matchDate))
    .slice(0, limit)
    .reverse()
}

const liveKey = (m: Pick<TeamMatchSummary, 'kind' | 'matchId'>) => `${m.kind}:${m.matchId}`

// selectLiveMatch picks the team's match that is on right now: kickoff reached and still inside the
// live broadcast window, the latest kickoff winning when two overlap. `played` is deliberately not
// required - a match whose simulation lags is still "on" for its owner - while a match simulated up
// to 15 minutes before kickoff is not live until kickoff arrives (isPast).
export function selectLiveMatch(teamMatches: TeamMatchSummary[] | undefined): TeamMatchSummary | undefined {
  return teamMatches
    ?.filter((m) => isPast(m.matchDate) && mightStillBeLive(m.matchDate))
    .sort((a, b) => b.matchDate.localeCompare(a.matchDate))[0]
}

// selectLastResult picks the latest match whose broadcast is over from a most-recent-first list of
// played matches - never one still being shown live (or any other match inside its live window),
// whose stored score would spoil it. A match whose broadcast just completed (`finished`) qualifies.
export function selectLastResult(playedMatches: TeamMatchSummary[], finished: ReadonlySet<string>): TeamMatchSummary | undefined {
  return playedMatches.find((m) => !mightStillBeLive(m.matchDate) || finished.has(liveKey(m)))
}

// useOwnLiveMatch follows the broadcast of the owner's live match (see selectLiveMatch), opening the
// socket only while one exists. Once the broadcast reports completion the match is remembered as
// finished, so the hero goes back to the next fixture and the "Last result" card may show its score.
export function useOwnLiveMatch(
  teamMatches: TeamMatchSummary[] | undefined,
  teamId: string | undefined,
): { match: TeamMatchSummary | undefined; live: LiveMatchState; finished: ReadonlySet<string> } {
  // Re-evaluates the live window as time passes, even if nothing else re-renders the page.
  useNow()
  const queryClient = useQueryClient()
  const [finished, setFinished] = useState<ReadonlySet<string>>(() => new Set())
  const candidate = selectLiveMatch(teamMatches?.filter((m) => !finished.has(liveKey(m))))
  const live = useLiveMatch(candidate ? liveMatchPath(candidate) : undefined, !!candidate)
  const completedKey = candidate && live.complete ? liveKey(candidate) : undefined

  // Remembered during render (not in an effect) so the hero never flashes back to live for a frame.
  if (completedKey && !finished.has(completedKey)) {
    setFinished(new Set(finished).add(completedKey))
  }

  useEffect(() => {
    if (completedKey && teamId) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.teamMatches(teamId) })
    }
  }, [completedKey, teamId, queryClient])

  return { match: completedKey ? undefined : candidate, live, finished }
}

export function DashboardPage() {
  const { t } = useTranslation()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading } = useUserTeam(!!loggedUser)
  const { data: finances } = useFinances(!!userTeam)
  const { data: division } = useDivisionDetails(userTeam ? String(userTeam.team.divisionId) : undefined)
  // Sourced from every league match, cup tie, and friendly the team has, rather than just the
  // division's league fixtures - a league-only "next match" quietly skips a friendly kicking off
  // sooner, which is exactly the bug this dashboard used to have.
  const { data: teamMatches } = useTeamMatches(userTeam ? String(userTeam.team.id) : undefined)
  const { match: liveMatch, live, finished } = useOwnLiveMatch(teamMatches, userTeam ? String(userTeam.team.id) : undefined)

  if (isLoading || !userTeam) {
    return <Skeleton variant="rectangular" height={320} sx={{ borderRadius: 2 }} />
  }

  const { team, players } = userTeam
  // A league match or friendly is read into simulation up to 15 minutes before its scheduled
  // matchDate (see backend GetSimulableMatchesStmtSql/GetSimulableFriendliesStmtSql), so `played`
  // can flip true while matchDate is still in the future. Gating on isPast too keeps such a match
  // out of the recent-results/streak list, and in the upcoming list, until its advertised kickoff
  // has actually arrived - otherwise the dashboard would announce it as live, or already won or
  // lost, before the viewer ever saw it start.
  const playedMatches = teamMatches
    ?.filter((m) => m.played && isPast(m.matchDate))
    .sort((a, b) => b.matchDate.localeCompare(a.matchDate))
  const nextMatch = teamMatches?.filter((m) => !isPast(m.matchDate)).sort((a, b) => a.matchDate.localeCompare(b.matchDate))[0]
  const lastMatch = playedMatches && selectLastResult(playedMatches, finished)
  const recentForm = playedMatches && selectRecentForm(playedMatches, 5)
  const standing = currentStanding(division, team.id)

  const secondary = (
    <Grid container spacing={2}>
      <Grid size={{ xs: 12, sm: 6, md: 4 }}>
        <TopScorerCard leagueId={String(team.divisionId)} />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 4 }}>
        <GroupSnapshotCard teamId={team.id} />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 4 }}>
        <SponsorDealCard finances={finances} />
      </Grid>
      <Grid size={12}>
        <RecentLeagueTransfers leagueId={String(team.divisionId)} />
      </Grid>
    </Grid>
  )

  // Phone order is by urgency: next match (countdown, rival record, lineup CTA) -> things that need
  // an answer -> last result -> the four KPIs -> everything else folded away. Desktop keeps the
  // identity header first and a roomier grid.
  return (
    <Box>
      <SeasonRecap enabled={!!loggedUser} />
      {isDesktop && <TeamHeaderCard team={team} standing={standing} recentForm={recentForm} />}
      {liveMatch ? <LiveMatchHero match={liveMatch} live={live} team={team} divisionId={team.divisionId} /> : <NextMatchHero match={nextMatch} teamId={team.id} />}
      <PendingPanel teamId={team.id} players={players} finances={finances} nextMatch={nextMatch} />
      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <LastResultCard match={lastMatch} recentForm={isDesktop ? undefined : recentForm} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <KpiGrid
            budget={finances?.budget}
            squadSize={players.length}
            teamId={team.id}
            standing={standing}
            divisionId={team.divisionId}
            fanbase={userTeam.fanbase}
          />
        </Grid>
      </Grid>
      <LoanedOutCard teamId={team.id} loanedOut={userTeam.loanedOut ?? []} />
      <LiveMatchesCard />
      {isDesktop ? (
        secondary
      ) : (
        <>
          <Accordion variant="outlined" disableGutters sx={{ mb: 2 }}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                {t('dashboard.moreSection')}
              </Typography>
            </AccordionSummary>
            <AccordionDetails>{secondary}</AccordionDetails>
          </Accordion>
          <TeamHeaderCard team={team} standing={standing} recentForm={undefined} compact />
        </>
      )}
    </Box>
  )
}

// A dashboard-wide (not just the owner's own team) glance at how much of the world is currently
// mid-broadcast, with one suggested match - prioritized toward the top divisions of any country
// (see backend common.PickRandomLiveMatch) - to jump straight into watching. Refreshes itself
// periodically (see useLiveMatchesOverview) since kickoffs and final whistles happen on their own
// schedule; the shuffle button re-asks the backend for a fresh random pick on demand.
function LiveMatchesCard() {
  const { t } = useTranslation()
  const { data, isLoading, isFetching, refetch } = useLiveMatchesOverview(true)
  const suggested = data?.suggested

  // Nothing to watch -> nothing to show (a card announcing an empty world is just scrolling).
  if (isLoading || !data || data.count === 0) {
    return null
  }

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          spacing={2}
          sx={{ alignItems: { sm: 'center' }, justifyContent: 'space-between' }}
        >
          <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
            <LiveTvTwoToneIcon color="error" />
            <Box>
              <Typography variant="subtitle1" sx={{ fontWeight: 700, lineHeight: 1.3 }}>
                {t('dashboard.liveWorldwide')}
              </Typography>
              {isLoading ? (
                <Skeleton width={160} height={20} />
              ) : (
                <Typography variant="body2" color="text.secondary">
                  {data && data.count > 0
                    ? t('dashboard.liveMatchesCount', { count: data.count })
                    : t('dashboard.noLiveMatches')}
                </Typography>
              )}
            </Box>
          </Stack>
          {suggested && <SuggestedLiveMatch match={suggested} isFetching={isFetching} onShuffle={refetch} />}
        </Stack>
      </CardContent>
    </Card>
  )
}

function SuggestedLiveMatch({
  match,
  isFetching,
  onShuffle,
}: {
  match: LiveMatchSummary
  isFetching: boolean
  onShuffle: () => void
}) {
  const { t } = useTranslation()

  return (
    <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} sx={{ alignItems: { sm: 'center' } }}>
      <Stack spacing={0.5} sx={{ minWidth: 0 }}>
        <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
          <MatchKindChip kind={match.kind} />
          {match.kind === 'league' && match.tier !== undefined && (
            <Chip
              size="small"
              variant="outlined"
              label={t('dashboard.tierLabel', { tier: match.tier })}
              sx={{ height: 20, fontSize: '0.6875rem' }}
            />
          )}
          <CountryFlag code={match.country} height={14} />
          {match.minute !== undefined && (
            <Chip
              size="small"
              color="error"
              label={t('dashboard.liveMatchMinute', { minute: match.minute })}
              sx={{ height: 20, fontSize: '0.6875rem', fontWeight: 700 }}
            />
          )}
        </Stack>
        <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
          {match.homeTeamName} {match.homeGoals} - {match.awayGoals} {match.awayTeamName}
        </Typography>
      </Stack>
      <Stack direction="row" spacing={1}>
        <Tooltip title={t('dashboard.shuffleLiveMatch')}>
          <span>
            <IconButton size="small" onClick={() => onShuffle()} disabled={isFetching}>
              <ShuffleTwoToneIcon fontSize="small" />
            </IconButton>
          </span>
        </Tooltip>
        <Button size="small" variant="contained" component={RouterLink} to={matchSummaryLink(match)}>
          {t('dashboard.watchLive')}
        </Button>
      </Stack>
    </Stack>
  )
}

function TeamHeaderCard({
  team,
  standing,
  recentForm,
  compact = false,
}: {
  team: Team
  standing: ClassificationPosition | undefined
  recentForm: TeamMatchSummary[] | undefined
  /** Phone variant: no lineup button (the next-match card carries it) and a smaller crest. */
  compact?: boolean
}) {
  const { t } = useTranslation()
  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          spacing={2}
          sx={{ alignItems: { sm: 'center' }, justifyContent: 'space-between' }}
        >
          <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
            <TeamBadge teamId={team.id} teamName={team.name} size={compact ? 40 : 56} />
            <Box>
              <TeamNameHeader team={team} />
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mt: 0.5 }}>
                <Link
                  component={RouterLink}
                  to={`/division/${team.divisionId}`}
                  underline="hover"
                  variant="body2"
                  color="text.secondary"
                >
                  {formatDivisionName(t, team.divisionName)}
                </Link>
                {standing && (
                  <Chip
                    size="small"
                    variant="outlined"
                    label={t('dashboard.leaguePosition', { position: standing.position })}
                  />
                )}
              </Stack>
              {recentForm && recentForm.length > 0 && <FormStrip matches={recentForm} />}
            </Box>
          </Stack>
          {!compact && (
            <Button component={RouterLink} to="/dashboard/lineup" variant="contained" size="small">
              {t('dashboard.editLineup')}
            </Button>
          )}
        </Stack>
      </CardContent>
    </Card>
  )
}

// The past-5-results-at-a-glance strip, oldest first (left) to most recent (right) - reuses the
// same league/cup/friendly match data the next/last match cards derive from, so it costs no extra
// request. Each chip is a link to that match (a tap target, with the rival and score as its
// accessible name - hover tooltips do not exist on a phone), and a caption says which end is the
// latest.
function FormStrip({ matches }: { matches: TeamMatchSummary[] }) {
  const { t } = useTranslation()
  return (
    <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', mt: 0.75, flexWrap: 'wrap', rowGap: 0.5 }}>
      <Typography variant="caption" color="text.secondary">
        {t('dashboard.form')}
      </Typography>
      {matches.map((match) => {
        const ownGoals = match.isHome ? match.homeGoals : match.awayGoals
        const oppGoals = match.isHome ? match.awayGoals : match.homeGoals
        const outcome = ownGoals > oppGoals ? 'win' : ownGoals < oppGoals ? 'loss' : 'draw'
        const color = outcome === 'win' ? 'success' : outcome === 'loss' ? 'error' : 'default'
        const description = `${match.opponentName} ${ownGoals}-${oppGoals}`
        return (
          <Tooltip key={`${match.kind}-${match.matchId}`} title={description}>
            <Chip
              size="small"
              color={color}
              component={RouterLink}
              to={matchSummaryLink(match)}
              clickable
              aria-label={`${t(`common.resultLong.${outcome}`)}: ${description}`}
              label={t(`dashboard.result.${outcome}`)}
              sx={{ height: 24, width: 28, typography: 'chipLabel', fontWeight: 700, '& .MuiChip-label': { px: 0 } }}
            />
          </Tooltip>
        )
      })}
      <Typography variant="caption" color="text.secondary">
        {t('dashboard.formLatest')}
      </Typography>
    </Stack>
  )
}

function StatCard({
  icon,
  title,
  action,
  children,
}: {
  icon: ReactNode
  title: string
  action?: ReactNode
  children: ReactNode
}) {
  return (
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flexGrow: 1 }}>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1.5 }}>
          {icon}
          <Typography variant="overline" color="text.secondary" sx={{ lineHeight: 1 }}>
            {title}
          </Typography>
        </Stack>
        {children}
      </CardContent>
      {action && <CardActions sx={{ pt: 0 }}>{action}</CardActions>}
    </Card>
  )
}

function OpponentLine({ match }: { match: TeamMatchSummary }) {
  const { t } = useTranslation()

  return (
    <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
      <TeamBadge teamId={match.opponentId} teamName={match.opponentName} size={32} />
      <Box sx={{ minWidth: 0 }}>
        <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
          <TeamNameLink teamId={match.opponentId}>{match.opponentName}</TeamNameLink>
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {match.isHome ? t('dashboard.home') : t('dashboard.away')}
        </Typography>
      </Box>
    </Stack>
  )
}

function MatchKindChip({ kind }: { kind: TeamMatchSummary['kind'] }) {
  const { t } = useTranslation()
  return (
    <Chip
      size="small"
      variant="outlined"
      color={matchKindColor(kind)}
      label={t(matchKindLabelKey(kind))}
      sx={{ height: 20, alignSelf: 'flex-start' }}
    />
  )
}

// The first thing on the page: who and when, with a live countdown, the record against that rival
// (so the history sits next to the match it is about) and the one action that matters before a
// match - editing the lineup.
function NextMatchHero({ match, teamId }: { match: TeamMatchSummary | undefined; teamId: number }) {
  const { t } = useTranslation()
  const now = useNow()
  const { data: h2h } = useHeadToHead(teamId, match?.opponentId)

  if (!match) {
    return (
      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Typography variant="overline" color="text.secondary">
            {t('dashboard.nextMatch')}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {t('dashboard.noUpcomingMatch')}
          </Typography>
        </CardContent>
      </Card>
    )
  }

  const countdown = countdownUntil(match.matchDate, now)
  const parts = countdown && countdownParts(countdown)

  return (
    <Card sx={{ mb: 2, bgcolor: 'primaryContainer', color: 'onPrimaryContainer' }}>
      <CardContent>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1, flexWrap: 'wrap', rowGap: 0.5 }}>
          <CalendarTodayTwoToneIcon fontSize="small" />
          <Typography variant="overline" sx={{ lineHeight: 1 }}>
            {t('dashboard.nextMatch')}
          </Typography>
          <MatchKindChip kind={match.kind} />
          <RivalryChip level={match.rivalry} />
        </Stack>
        <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
          <TeamBadge teamId={match.opponentId} teamName={match.opponentName} size={48} />
          <Box sx={{ minWidth: 0 }}>
            <Typography variant="h6" component="div" sx={{ fontWeight: 700, overflowWrap: 'anywhere', lineHeight: 1.25 }}>
              <TeamNameLink teamId={match.opponentId}>{match.opponentName}</TeamNameLink>
            </Typography>
            <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
              <HomeAwayIcon isHome={match.isHome} />
              <Typography variant="body2">{match.isHome ? t('dashboard.home') : t('dashboard.away')}</Typography>
            </Stack>
          </Box>
        </Stack>
        <Typography variant="body2" sx={{ mt: 1.5 }}>
          {formatDateTime(match.matchDate)}
          {parts && ` · ${t(`dashboard.countdown.${parts.key}`, parts.params)}`}
        </Typography>
        {h2h && h2h.totalMatches > 0 && (
          <Typography variant="body2" sx={{ mt: 0.5, fontWeight: 600 }}>
            {t('dashboard.headToHeadRecord', { w: h2h.teamAWins, d: h2h.draws, l: h2h.teamBWins })}
          </Typography>
        )}
        {h2h && h2h.totalMatches === 0 && (
          <Typography variant="body2" sx={{ mt: 0.5 }}>
            {t('dashboard.headToHeadFirstMeeting', { opponent: match.opponentName })}
          </Typography>
        )}
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mt: 2 }}>
          <Button component={RouterLink} to="/dashboard/lineup" variant="contained" fullWidth sx={{ maxWidth: { sm: 260 } }}>
            {t('dashboard.editLineup')}
          </Button>
          <Button component={RouterLink} to={matchSummaryLink(match)} color="inherit" sx={{ maxWidth: { sm: 200 } }}>
            {t('dashboard.viewMatch')}
          </Button>
        </Stack>
      </CardContent>
    </Card>
  )
}

// Takes over the hero slot while the owner's own match is being broadcast. The score and minute are
// the broadcast's, never the stored result (which is final as soon as the match is simulated), so
// the hero cannot spoil what the viewer has not watched yet. When the live channel is unreachable
// it keeps the label and the button but drops the numbers rather than show a wrong score.
export function LiveMatchHero({
  match,
  live,
  team,
  divisionId,
}: {
  match: TeamMatchSummary
  live: LiveMatchState
  team: Pick<Team, 'id' | 'name'>
  divisionId?: number
}) {
  const { t } = useTranslation()
  const home = match.isHome ? { id: team.id, name: team.name } : { id: match.opponentId, name: match.opponentName }
  const away = match.isHome ? { id: match.opponentId, name: match.opponentName } : { id: team.id, name: team.name }
  const showNumbers = !live.giveUp
  const status = live.halftime ? t('match.halftime') : live.minute > 0 ? t('dashboard.liveMatchMinute', { minute: live.minute }) : undefined

  return (
    <Card sx={{ mb: 2, bgcolor: 'errorContainer', color: 'onErrorContainer' }}>
      <CardContent>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1.5, flexWrap: 'wrap', rowGap: 0.5 }}>
          <Chip size="small" label={t('match.live')} sx={{ ...PULSING_LIVE_SX, bgcolor: 'onErrorContainer', color: 'errorContainer' }} />
          <Typography variant="overline" sx={{ lineHeight: 1 }}>
            {t('dashboard.liveNow')}
          </Typography>
          <MatchKindChip kind={match.kind} />
        </Stack>
        <Stack direction="row" spacing={{ xs: 0.5, sm: 2 }} sx={{ alignItems: 'center' }}>
          <HeroTeam id={home.id} name={home.name} />
          <Box sx={{ textAlign: 'center', minWidth: 96 }}>
            {showNumbers && (
              <>
                <Typography variant="h4" component="div" sx={{ fontWeight: 800, whiteSpace: 'nowrap' }}>
                  {live.homeGoals} - {live.awayGoals}
                </Typography>
                {status && <Typography variant="body2">{status}</Typography>}
              </>
            )}
          </Box>
          <HeroTeam id={away.id} name={away.name} />
        </Stack>
        <Button component={RouterLink} to={matchSummaryLink(match)} variant="contained" color="error" fullWidth sx={{ mt: 2, maxWidth: { sm: 260 } }}>
          {t('dashboard.watchLive')}
        </Button>
        {match.kind === 'league' && divisionId !== undefined && <RestOfMatchday divisionId={divisionId} ownMatchId={match.matchId} />}
      </CardContent>
    </Card>
  )
}

// The other matches of the owner's league matchday with their broadcast score and minute (the
// division live scores, never the stored results). Hidden while loading, when the request fails,
// when the own match is not in the response (so its matchday is unknown) and when there is nothing
// else to list - the hero's own score, minute and button never depend on it.
function RestOfMatchday({ divisionId, ownMatchId }: { divisionId: number; ownMatchId: number }) {
  const { data } = useDivisionLiveScores(divisionId, true)
  const own = data?.find((entry) => entry.matchId === ownMatchId)
  if (!data || !own) return null
  const others = data.filter((entry) => entry.weekNumber === own.weekNumber && entry.matchId !== ownMatchId)
  if (others.length === 0) return null
  return <DivisionLiveScores entries={others} />
}

function HeroTeam({ id, name }: { id: number; name: string }) {
  return (
    <Stack spacing={0.5} sx={{ flex: 1, minWidth: 0, alignItems: 'center', textAlign: 'center' }}>
      <TeamBadge teamId={id} teamName={name} size={48} />
      <Typography variant="subtitle1" component="div" sx={{ fontWeight: 700, overflowWrap: 'anywhere', lineHeight: 1.25 }}>
        <TeamNameLink teamId={id}>{name}</TeamNameLink>
      </Typography>
    </Stack>
  )
}

function LastResultCard({ match, recentForm }: { match: TeamMatchSummary | undefined; recentForm?: TeamMatchSummary[] }) {
  const { t } = useTranslation()

  if (!match) {
    return (
      <StatCard icon={<SportsSoccerTwoToneIcon color="primary" fontSize="small" />} title={t('dashboard.lastResult')}>
        <Typography variant="body2" color="text.secondary">
          {t('dashboard.noRecentMatch')}
        </Typography>
      </StatCard>
    )
  }

  // The caller (DashboardPage's lastMatch) only passes a match whose broadcast is over, so the
  // stored score is safe to show here.
  const ownGoals = match.isHome ? match.homeGoals : match.awayGoals
  const oppGoals = match.isHome ? match.awayGoals : match.homeGoals
  const outcome = ownGoals > oppGoals ? 'win' : ownGoals < oppGoals ? 'loss' : 'draw'
  const color = outcome === 'win' ? 'success' : outcome === 'loss' ? 'error' : 'default'

  return (
    <StatCard
      icon={<SportsSoccerTwoToneIcon color="primary" fontSize="small" />}
      title={t('dashboard.lastResult')}
      action={
        <Button size="small" component={RouterLink} to={matchSummaryLink(match)}>
          {t('dashboard.viewMatch')}
        </Button>
      }
    >
      <Stack spacing={0.5}>
        <MatchKindChip kind={match.kind} />
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
          <Chip label={t(`dashboard.result.${outcome}`)} color={color} size="small" sx={{ fontWeight: 700 }} />
          <Typography variant="body2" sx={{ fontWeight: 700 }}>
            {ownGoals} - {oppGoals}
          </Typography>
        </Stack>
        <OpponentLine match={match} />
        {recentForm && recentForm.length > 0 && <FormStrip matches={recentForm} />}
      </Stack>
    </StatCard>
  )
}

// The four numbers worth a glance, as a 2x2 grid of tiles that are links themselves (no separate
// "Ver ..." buttons whose only tap target is their text).
function KpiGrid({
  budget,
  squadSize,
  teamId,
  standing,
  divisionId,
  fanbase,
}: {
  budget: number | undefined
  squadSize: number
  teamId: number
  standing: ClassificationPosition | undefined
  divisionId: number
  fanbase: number
}) {
  const { t } = useTranslation()
  const theme = useTheme()
  const tiles = [
    {
      key: 'budget',
      icon: <AccountBalanceWalletOutlinedIcon />,
      label: t('dashboard.kpi.budget'),
      to: '/dashboard/finances',
      value: budget === undefined ? <Skeleton width={80} height={32} /> : formatCurrency(budget, { compact: true }),
      color: budget !== undefined && budget < 0 ? theme.palette.error.main : undefined,
    },
    {
      key: 'squad',
      icon: <GroupsOutlinedIcon />,
      label: t('dashboard.kpi.squad'),
      to: `/team/${teamId}`,
      value: squadSize,
    },
    {
      key: 'position',
      icon: <LeaderboardOutlinedIcon />,
      label: t('dashboard.kpi.position'),
      to: `/division/${divisionId}`,
      value: standing ? t('dashboard.positionValue', { position: standing.position }) : '-',
    },
    {
      key: 'fanbase',
      icon: <EmojiEventsOutlinedIcon />,
      label: t('dashboard.kpi.fanbase'),
      to: '/dashboard/finances',
      value: fanbase,
      color: ratingTierColor(fanbase, theme.palette),
    },
  ]

  return (
    <Grid container spacing={1.5}>
      {tiles.map((tile) => (
        <Grid key={tile.key} size={6}>
          <StatTile icon={tile.icon} label={tile.label} value={tile.value} valueColor={tile.color} to={tile.to} />
        </Grid>
      ))}
    </Grid>
  )
}

function TopScorerCard({ leagueId }: { leagueId: string }) {
  const { t } = useTranslation()
  const { data: scorers } = useTopScorers(leagueId)
  const leader = scorers?.[0]

  return (
    <StatCard
      icon={<EmojiEventsTwoToneIcon color="primary" fontSize="small" />}
      title={t('dashboard.topScorer')}
      action={
        <Button size="small" component={RouterLink} to={`/division/${leagueId}`}>
          {t('dashboard.viewLeague')}
        </Button>
      }
    >
      {!scorers ? (
        <Skeleton width={100} height={32} />
      ) : !leader ? (
        <Typography variant="body2" color="text.secondary">
          {t('division.topScorers.noGoals')}
        </Typography>
      ) : (
        <Stack spacing={0.5}>
          <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
            <CountryFlag code={leader.birthCountry} height={14} marginRight={0.5} />
            <PlayerNameLink playerId={leader.playerId}>
              {leader.firstName} {leader.lastName}
            </PlayerNameLink>
          </Typography>
          <Typography variant="caption" color="text.secondary" noWrap>
            <TeamNameLink teamId={leader.teamId}>{leader.teamName}</TeamNameLink>
          </Typography>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            {t('dashboard.goalsCount', { count: leader.goals })}
          </Typography>
        </Stack>
      )}
    </StatCard>
  )
}

// A snapshot of the owner's own friend group (see GroupsPage) - just enough to notice a rank
// change without leaving the dashboard. Only the first group is shown if the owner belongs to
// several, same as this being a glance-only card rather than the full leaderboard.
function GroupSnapshotCard({ teamId }: { teamId: number }) {
  const { t } = useTranslation()
  const { data: groups } = useMyGroups(true)
  const firstGroup = groups?.[0]
  const { data: details } = useGroupDetails(firstGroup ? String(firstGroup.groupId) : undefined, !!firstGroup)
  const rank = details?.leaderboard.findIndex((m) => m.teamId === teamId)

  return (
    <StatCard
      icon={<LeaderboardTwoToneIcon color="primary" fontSize="small" />}
      title={t('groups.title')}
      action={
        <Button size="small" component={RouterLink} to="/groups">
          {t('dashboard.viewGroups')}
        </Button>
      }
    >
      {!groups ? (
        <Skeleton width={100} height={32} />
      ) : !firstGroup ? (
        <Typography variant="body2" color="text.secondary">
          {t('groups.noGroups')}
        </Typography>
      ) : !details || rank === undefined || rank < 0 ? (
        <Skeleton width={100} height={32} />
      ) : (
        <Stack spacing={0.5}>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            {t('dashboard.groupRank', { rank: rank + 1, total: details.leaderboard.length })}
          </Typography>
          <Typography variant="caption" color="text.secondary" noWrap>
            {firstGroup.name}
          </Typography>
        </Stack>
      )}
    </StatCard>
  )
}

// The manager's players currently out on loan (count, the three returning soonest) with a link to
// the full list on his squad tab. Not rendered at all when nobody is out.
function LoanedOutCard({ teamId, loanedOut }: { teamId: number; loanedOut: LoanedOutPlayer[] }) {
  const { t } = useTranslation()
  if (loanedOut.length === 0) return null
  return (
    <Box sx={{ mb: 2 }} data-testid="dashboard-loaned-out">
      <StatCard
        icon={<OutboundTwoToneIcon color="primary" fontSize="small" />}
        title={t('dashboard.loanedOut.title')}
        action={
          <Button size="small" component={RouterLink} to={`/team/${teamId}?tab=squad`}>
            {t('dashboard.loanedOut.seeAll')}
          </Button>
        }
      >
        <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
          {t('dashboard.loanedOut.count', { count: loanedOut.length })}
        </Typography>
        <Stack spacing={0.25}>
          {soonestLoanReturns(loanedOut, 3).map((loan) => (
            <Typography key={loan.player.id} variant="body2" color="text.secondary">
              {loan.player.firstName} {loan.player.lastName} · {loan.borrowingTeamName} · {t('dashboard.loanedOut.returns', { season: loan.returnSeason, country: localizedCountryName(t, loan.borrowingCountry) })}
            </Typography>
          ))}
        </Stack>
      </StatCard>
    </Box>
  )
}

function SponsorDealCard({ finances }: { finances: FinancesResponse | undefined }) {
  const { t } = useTranslation()
  return (
    <StatCard
      icon={<HandshakeTwoToneIcon color="primary" fontSize="small" />}
      title={t('finances.sponsorDeal.title')}
      action={
        <Button size="small" component={RouterLink} to="/dashboard/finances">
          {t('dashboard.viewFinances')}
        </Button>
      }
    >
      {!finances ? (
        <Skeleton width={100} height={32} />
      ) : (
        <Stack spacing={0.5}>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            {t(SPONSOR_DEAL_LABEL_KEYS[finances.sponsorDeal])}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {finances.sponsorDealLocked ? t('finances.sponsorDeal.locked') : t('dashboard.sponsorDealTip')}
          </Typography>
        </Stack>
      )}
    </StatCard>
  )
}

function TeamNameHeader({ team }: { team: Team }) {
  const { t } = useTranslation()
  const renameTeam = useRenameTeam()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(team.name)

  const handleSave = () => {
    const trimmed = name.trim()
    if (!trimmed || trimmed === team.name) {
      setEditing(false)
      return
    }
    renameTeam.mutate(trimmed, { onSuccess: () => setEditing(false) })
  }

  if (editing) {
    return (
      <Stack>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
          <TextField
            size="small"
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={renameTeam.isPending}
          />
          <Button size="small" variant="contained" onClick={handleSave} disabled={renameTeam.isPending}>
            {t('dashboard.renameSave')}
          </Button>
          <Button
            size="small"
            onClick={() => {
              setEditing(false)
              setName(team.name)
            }}
          >
            {t('common.cancel')}
          </Button>
        </Stack>
        {renameTeam.isError && (
          <Alert severity="error" sx={{ mt: 1 }}>
            {t('dashboard.renameError')}
          </Alert>
        )}
      </Stack>
    )
  }

  return (
    <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
      <Typography variant="h6">{team.name}</Typography>
      {!team.nameChanged && (
        <Tooltip title={t('dashboard.renameTeam')}>
          <IconButton size="small" onClick={() => setEditing(true)}>
            <EditIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      )}
    </Stack>
  )
}
