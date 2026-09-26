import { useState, type ReactNode } from 'react'
import AccountBalanceWalletTwoToneIcon from '@mui/icons-material/AccountBalanceWalletTwoTone'
import CalendarTodayTwoToneIcon from '@mui/icons-material/CalendarTodayTwoTone'
import CompareArrowsTwoToneIcon from '@mui/icons-material/CompareArrowsTwoTone'
import EditIcon from '@mui/icons-material/Edit'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import GroupsTwoToneIcon from '@mui/icons-material/GroupsTwoTone'
import HandshakeTwoToneIcon from '@mui/icons-material/HandshakeTwoTone'
import LeaderboardTwoToneIcon from '@mui/icons-material/LeaderboardTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
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
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useAcademy,
  useDivisionDetails,
  useFinances,
  useFriendlyChallengesReceived,
  useGroupDetails,
  useHeadToHead,
  useMe,
  useMyGroups,
  useOffersReceived,
  useRenameTeam,
  useTeamMatches,
  useTopScorers,
  useUserTeam,
} from '../../api/hooks'
import { SPONSOR_DEAL_LABEL_KEYS, type ClassificationPosition, type FinancesResponse, type Player, type Team, type TeamMatchSummary } from '../../api/types'
import { mightStillBeLive } from '../../api/useLiveMatch'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamBadge } from '../../components/TeamBadge'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { matchKindColor, matchKindLabelKey, matchSummaryLink } from '../../utils/matchSummary'
import { useSquadUnavailability } from '../lineup/useSquadUnavailability'
import { RecentLeagueTransfers } from '../leagues/RecentLeagueTransfers'
import { SeasonRecap } from './SeasonRecap'

export function DashboardPage() {
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
  const playedMatches = teamMatches?.filter((m) => m.played).sort((a, b) => b.matchDate.localeCompare(a.matchDate))
  const nextMatch = teamMatches?.filter((m) => !m.played).sort((a, b) => a.matchDate.localeCompare(b.matchDate))[0]
  const lastMatch = playedMatches?.[0]
  const recentForm = playedMatches?.slice(0, 5).reverse()
  const standing = division?.classification.positions.find((p) => p.teamId === team.id)

  return (
    <Box>
      <SeasonRecap enabled={!!loggedUser} />
      <LiveMatchBanner match={lastMatch} />
      <TeamHeaderCard team={team} standing={standing} recentForm={recentForm} />
      <AttentionAlerts teamId={team.id} players={players} finances={finances} />
      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <NextMatchCard match={nextMatch} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <LastResultCard match={lastMatch} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <BudgetCard budget={finances?.budget} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <SquadCard teamId={team.id} squadSize={players.length} />
        </Grid>
      </Grid>
      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <TopScorerCard leagueId={String(team.divisionId)} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <HeadToHeadCard teamId={team.id} nextMatch={nextMatch} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <GroupSnapshotCard teamId={team.id} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <SponsorDealCard finances={finances} />
        </Grid>
      </Grid>
      <RecentLeagueTransfers leagueId={String(team.divisionId)} />
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
    <Alert
      severity="info"
      icon={<SportsSoccerTwoToneIcon fontSize="inherit" />}
      action={
        <Button color="inherit" size="small" component={RouterLink} to={matchSummaryLink(match)}>
          {t('dashboard.viewMatch')}
        </Button>
      }
      sx={{ mb: 2 }}
    >
      {t('dashboard.liveMatchBanner', { opponent: match.opponentName })}
    </Alert>
  )
}

// CONTRACT_EXPIRY_WARNING_YEARS mirrors the last value a rostered player's contractYearsRemaining
// ever shows before AdvancePlayerContract releases them to free agency at season end (see
// backend/common.Player.ContractYearsRemaining) - by the time it reads 1, there's exactly one
// season left to renew or lose them for nothing.
const CONTRACT_EXPIRY_WARNING_YEARS = 1

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
    <Stack spacing={1} sx={{ mb: 2 }}>
      {alerts}
    </Stack>
  )
}

function TeamHeaderCard({
  team,
  standing,
  recentForm,
}: {
  team: Team
  standing: ClassificationPosition | undefined
  recentForm: TeamMatchSummary[] | undefined
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
            <TeamBadge teamId={team.id} teamName={team.name} size={56} />
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
                  {team.divisionName}
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
          <Button component={RouterLink} to="/dashboard/lineup" variant="contained" size="small">
            {t('dashboard.editLineup')}
          </Button>
        </Stack>
      </CardContent>
    </Card>
  )
}

// The team header's own past-5-results-at-a-glance strip, oldest first (left) to most recent
// (right) - reuses the same league/cup/friendly match data the next/last match cards derive from,
// so it costs no extra request.
function FormStrip({ matches }: { matches: TeamMatchSummary[] }) {
  const { t } = useTranslation()
  return (
    <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center', mt: 0.75 }}>
      <Typography variant="caption" color="text.secondary">
        {t('dashboard.form')}
      </Typography>
      {matches.map((match) => {
        const ownGoals = match.isHome ? match.homeGoals : match.awayGoals
        const oppGoals = match.isHome ? match.awayGoals : match.homeGoals
        const outcome = ownGoals > oppGoals ? 'win' : ownGoals < oppGoals ? 'loss' : 'draw'
        const color = outcome === 'win' ? 'success' : outcome === 'loss' ? 'error' : 'default'
        return (
          <Tooltip key={`${match.kind}-${match.matchId}`} title={`${match.opponentName} ${ownGoals}-${oppGoals}`}>
            <Chip
              size="small"
              color={color}
              label={t(`dashboard.result.${outcome}`)}
              sx={{ height: 20, width: 24, fontSize: '0.65rem', fontWeight: 700, '& .MuiChip-label': { px: 0 } }}
            />
          </Tooltip>
        )
      })}
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
      sx={{ height: 18, fontSize: '0.65rem', alignSelf: 'flex-start' }}
    />
  )
}

function NextMatchCard({ match }: { match: TeamMatchSummary | undefined }) {
  const { t } = useTranslation()
  return (
    <StatCard
      icon={<CalendarTodayTwoToneIcon color="primary" fontSize="small" />}
      title={t('dashboard.nextMatch')}
      action={
        match && (
          <Button size="small" component={RouterLink} to={matchSummaryLink(match)}>
            {t('dashboard.viewMatch')}
          </Button>
        )
      }
    >
      {match ? (
        <Stack spacing={0.5}>
          <MatchKindChip kind={match.kind} />
          <OpponentLine match={match} />
          <Typography variant="caption" color="text.secondary">
            {formatDateTime(match.matchDate)}
          </Typography>
        </Stack>
      ) : (
        <Typography variant="body2" color="text.secondary">
          {t('dashboard.noUpcomingMatch')}
        </Typography>
      )}
    </StatCard>
  )
}

function LastResultCard({ match }: { match: TeamMatchSummary | undefined }) {
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
      </Stack>
    </StatCard>
  )
}

function BudgetCard({ budget }: { budget: number | undefined }) {
  const { t } = useTranslation()
  return (
    <StatCard
      icon={<AccountBalanceWalletTwoToneIcon color="primary" fontSize="small" />}
      title={t('finances.budget')}
      action={
        <Button size="small" component={RouterLink} to="/dashboard/finances">
          {t('dashboard.viewFinances')}
        </Button>
      }
    >
      {budget === undefined ? (
        <Skeleton width={100} height={32} />
      ) : (
        <Typography variant="h5" sx={{ fontWeight: 700 }}>
          {formatCurrency(budget)}
        </Typography>
      )}
    </StatCard>
  )
}

function SquadCard({ teamId, squadSize }: { teamId: number; squadSize: number }) {
  const { t } = useTranslation()
  return (
    <StatCard
      icon={<GroupsTwoToneIcon color="primary" fontSize="small" />}
      title={t('header.nav.squad')}
      action={
        <Button size="small" component={RouterLink} to={`/team/${teamId}`}>
          {t('dashboard.viewSquad')}
        </Button>
      }
    >
      <Typography variant="h5" sx={{ fontWeight: 700 }}>
        {squadSize}
      </Typography>
      <Typography variant="caption" color="text.secondary">
        {t('dashboard.players')}
      </Typography>
    </StatCard>
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

// The record against whoever's up next (across league, cup, and friendly meetings alike, since
// HeadToHead isn't scoped to one competition) - context for how that match has historically gone,
// right next to the card that says when it's happening.
function HeadToHeadCard({ teamId, nextMatch }: { teamId: number; nextMatch: TeamMatchSummary | undefined }) {
  const { t } = useTranslation()
  const { data: h2h } = useHeadToHead(teamId, nextMatch?.opponentId)

  return (
    <StatCard
      icon={<CompareArrowsTwoToneIcon color="primary" fontSize="small" />}
      title={t('dashboard.headToHead')}
      action={
        nextMatch && (
          <Button size="small" component={RouterLink} to={`/team/${nextMatch.opponentId}`}>
            {t('dashboard.viewTeam')}
          </Button>
        )
      }
    >
      {!nextMatch ? (
        <Typography variant="body2" color="text.secondary">
          {t('dashboard.noUpcomingMatch')}
        </Typography>
      ) : !h2h || h2h.totalMatches === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {t('dashboard.headToHeadFirstMeeting', { opponent: nextMatch.opponentName })}
        </Typography>
      ) : (
        <Stack spacing={0.5}>
          <Typography variant="caption" color="text.secondary" noWrap>
            {t('dashboard.vsOpponent', { opponent: nextMatch.opponentName })}
          </Typography>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            {h2h.teamAWins}W {h2h.draws}D {h2h.teamBWins}L
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
