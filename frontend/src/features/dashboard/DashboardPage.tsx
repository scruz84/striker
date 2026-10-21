import { useState, type ReactNode } from 'react'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import AccountBalanceWalletTwoToneIcon from '@mui/icons-material/AccountBalanceWalletTwoTone'
import CalendarTodayTwoToneIcon from '@mui/icons-material/CalendarTodayTwoTone'
import EditIcon from '@mui/icons-material/Edit'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import GroupsTwoToneIcon from '@mui/icons-material/GroupsTwoTone'
import HandshakeTwoToneIcon from '@mui/icons-material/HandshakeTwoTone'
import LeaderboardTwoToneIcon from '@mui/icons-material/LeaderboardTwoTone'
import LiveTvTwoToneIcon from '@mui/icons-material/LiveTvTwoTone'
import ShuffleTwoToneIcon from '@mui/icons-material/ShuffleTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import Accordion from '@mui/material/Accordion'
import AccordionDetails from '@mui/material/AccordionDetails'
import AccordionSummary from '@mui/material/AccordionSummary'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardActionArea from '@mui/material/CardActionArea'
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
import useMediaQuery from '@mui/material/useMediaQuery'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useAcademy,
  useDivisionDetails,
  useFinances,
  useFriendlyChallengesReceived,
  useGroupDetails,
  useHeadToHead,
  useLiveMatchesOverview,
  useMe,
  useMyGroups,
  useOffersReceived,
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
  type Player,
  type Team,
  type TeamMatchSummary,
} from '../../api/types'
import { isPast, mightStillBeLive } from '../../api/useLiveMatch'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamBadge } from '../../components/TeamBadge'
import { TeamNameLink } from '../../components/TeamNameLink'
import { countryFlag } from '../../utils/countryFlag'
import { countdownParts, countdownUntil } from '../../utils/countdown'
import { useNow } from '../../utils/useNow'
import { formatDivisionName } from '../../utils/divisionName'
import { ratingTierColor } from '../../utils/playerRating'
import { HomeAwayIcon } from '../../components/HomeAwayIcon'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { matchKindColor, matchKindLabelKey, matchSummaryLink } from '../../utils/matchSummary'
import { useSquadUnavailability } from '../lineup/useSquadUnavailability'
import { hideStillLiveResults } from '../division/DivisionDetailsPage'
import { RecentLeagueTransfers } from '../leagues/RecentLeagueTransfers'
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
export function selectRecentForm(playedMatches: TeamMatchSummary[], limit: number): TeamMatchSummary[] {
  return playedMatches
    .filter((m) => !mightStillBeLive(m.matchDate))
    .slice(0, limit)
    .reverse()
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
  const lastMatch = playedMatches?.[0]
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
      <LiveMatchBanner match={lastMatch} />
      {isDesktop && <TeamHeaderCard team={team} standing={standing} recentForm={recentForm} />}
      <NextMatchHero match={nextMatch} teamId={team.id} />
      <AttentionAlerts teamId={team.id} players={players} finances={finances} />
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

// A prominent banner (rather than just the small LIVE chip LastResultCard already shows) for the
// common case of a team owner landing on their dashboard while their own match is broadcasting -
// `match.played` flips true in the DB well before its live broadcast finishes pacing out to
// viewers, so this is the same "still might be live" heuristic LastResultCard already uses, just
// surfaced somewhere a viewer can't miss it (issue: dashboard live match notice). Works for a
// friendly or cup tie too, not just a league match, since `match` now comes from useTeamMatches.
function LiveMatchBanner({ match }: { match: TeamMatchSummary | undefined }) {
  const { t } = useTranslation()

  if (!match || !match.played || !mightStillBeLive(match.matchDate)) {
    return null
  }

  return (
    <Box sx={{ mb: 2, ...STACKED_ALERT_ACTION_SX }}>
      <Alert
        severity="info"
        icon={<SportsSoccerTwoToneIcon fontSize="inherit" />}
        action={
          <Button color="inherit" size="small" component={RouterLink} to={matchSummaryLink(match)}>
            {t('dashboard.viewMatch')}
          </Button>
        }
      >
        {t('dashboard.liveMatchBanner', { opponent: match.opponentName })}
      </Alert>
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
  const flag = countryFlag(match.country)

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
          {flag && <span aria-hidden="true">{flag}</span>}
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

// CONTRACT_EXPIRY_WARNING_YEARS mirrors the last value a rostered player's contractYearsRemaining
// ever shows before AdvancePlayerContract releases them to free agency at season end (see
// backend/common.Player.ContractYearsRemaining) - by the time it reads 1, there's exactly one
// season left to renew or lose them for nothing.
const CONTRACT_EXPIRY_WARNING_YEARS = 1

// On a phone an Alert's action button squeezes into a narrow column beside the text and splits
// over two lines; here it drops under the message at full width instead.
const STACKED_ALERT_ACTION_SX = {
  '@media (max-width: 599.95px)': {
    '& .MuiAlert-root': { flexWrap: 'wrap' },
    '& .MuiAlert-message': { flex: 1 },
    '& .MuiAlert-action': { width: '100%', m: 0, p: 0, pl: '34px', justifyContent: 'flex-start' },
  },
} as const

// A row of dismissal-free notices for things that need the owner's action but have no other
// presence on the dashboard - insolvency, contracts about to lapse, unavailable players ahead of
// the next match, an academy prospect ready to promote, and unanswered transfer offers/friendly
// challenges. Renders nothing once there's nothing pending, so it never adds permanent clutter.
// Ordered roughly by how costly ignoring each one is.
function AttentionAlerts({
  teamId,
  players,
  finances,
}: {
  teamId: number
  players: Player[]
  finances: FinancesResponse | undefined
}) {
  const { t } = useTranslation()
  const unavailability = useSquadUnavailability()
  const { data: academy } = useAcademy()
  const { data: offersReceived } = useOffersReceived(true)
  const { data: challengesReceived } = useFriendlyChallengesReceived(true)
  const expiringContracts = players.filter((p) => p.contractYearsRemaining <= CONTRACT_EXPIRY_WARNING_YEARS)

  const alerts: ReactNode[] = []

  if (finances?.insolvent) {
    alerts.push(
      <Alert
        key="insolvent"
        severity="error"
        action={
          <Button color="inherit" size="small" component={RouterLink} to="/dashboard/finances">
            {t('dashboard.viewFinances')}
          </Button>
        }
      >
        {t('dashboard.insolvent')}
      </Alert>,
    )
  }

  if (expiringContracts.length > 0) {
    alerts.push(
      <Alert
        key="expiringContracts"
        severity="warning"
        action={
          <Button color="inherit" size="small" component={RouterLink} to={`/team/${teamId}`}>
            {t('dashboard.viewTeam')}
          </Button>
        }
      >
        {t('dashboard.contractsExpiring', { count: expiringContracts.length })}
      </Alert>,
    )
  }

  if (unavailability.size > 0) {
    alerts.push(
      <Alert
        key="unavailable"
        severity="warning"
        action={
          <Button color="inherit" size="small" component={RouterLink} to={`/team/${teamId}`}>
            {t('dashboard.viewTeam')}
          </Button>
        }
      >
        {t('dashboard.unavailablePlayers', { count: unavailability.size })}
      </Alert>,
    )
  }

  if (academy?.canPromote) {
    alerts.push(
      <Alert
        key="academy"
        severity="success"
        action={
          <Button color="inherit" size="small" component={RouterLink} to={`/team/${teamId}`}>
            {t('dashboard.viewTeam')}
          </Button>
        }
      >
        {t('team.academy.available')}
      </Alert>,
    )
  }

  if (offersReceived && offersReceived.length > 0) {
    alerts.push(
      <Alert
        key="offers"
        severity="info"
        action={
          <Button color="inherit" size="small" component={RouterLink} to="/transfers?tab=offers">
            {t('dashboard.viewOffers')}
          </Button>
        }
      >
        {t('dashboard.offersReceived', { count: offersReceived.length })}
      </Alert>,
    )
  }

  if (challengesReceived && challengesReceived.length > 0) {
    alerts.push(
      <Alert
        key="challenges"
        severity="info"
        action={
          <Button color="inherit" size="small" component={RouterLink} to="/friendlies">
            {t('dashboard.viewChallenges')}
          </Button>
        }
      >
        {t('dashboard.friendlyChallengesReceived', { count: challengesReceived.length })}
      </Alert>,
    )
  }

  if (alerts.length === 0) {
    return null
  }

  return (
    <Stack spacing={1} sx={{ mb: 2, ...STACKED_ALERT_ACTION_SX }}>
      {alerts}
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

  // played flips true in the DB the instant a match is simulated, well before its result
  // finishes broadcasting to viewers in real time (~90 minutes, see mightStillBeLive's comment)
  // - showing the final score/outcome here immediately would spoil a match that still looks
  // "live" everywhere else. Show a LIVE chip instead until that window has elapsed, same as
  // DivisionDetailsPage's FixturesCard.
  const stillLive = match.played && mightStillBeLive(match.matchDate)

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
        {stillLive ? (
          <Chip label={t('match.live')} color="error" size="small" sx={{ fontWeight: 700 }} />
        ) : (
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            <Chip label={t(`dashboard.result.${outcome}`)} color={color} size="small" sx={{ fontWeight: 700 }} />
            <Typography variant="body2" sx={{ fontWeight: 700 }}>
              {ownGoals} - {oppGoals}
            </Typography>
          </Stack>
        )}
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
      icon: <AccountBalanceWalletTwoToneIcon color="primary" fontSize="small" />,
      label: t('dashboard.kpi.budget'),
      to: '/dashboard/finances',
      value: budget === undefined ? <Skeleton width={80} height={32} /> : formatCurrency(budget, { compact: true }),
      color: budget !== undefined && budget < 0 ? theme.palette.error.main : undefined,
    },
    {
      key: 'squad',
      icon: <GroupsTwoToneIcon color="primary" fontSize="small" />,
      label: t('dashboard.kpi.squad'),
      to: `/team/${teamId}`,
      value: squadSize,
    },
    {
      key: 'position',
      icon: <LeaderboardTwoToneIcon color="primary" fontSize="small" />,
      label: t('dashboard.kpi.position'),
      to: `/division/${divisionId}`,
      value: standing ? t('dashboard.positionValue', { position: standing.position }) : '-',
    },
    {
      key: 'fanbase',
      icon: <EmojiEventsTwoToneIcon color="primary" fontSize="small" />,
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
          <Card variant="outlined" sx={{ height: '100%' }}>
            <CardActionArea component={RouterLink} to={tile.to} sx={{ height: '100%', p: 1.5, minHeight: 80 }}>
              <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', mb: 0.5 }}>
                {tile.icon}
                <Typography variant="overline" color="text.secondary" sx={{ lineHeight: 1 }}>
                  {tile.label}
                </Typography>
              </Stack>
              <Typography variant="stat" component="div" sx={{ fontSize: '1.5rem', color: tile.color }}>
                {tile.value}
              </Typography>
            </CardActionArea>
          </Card>
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
