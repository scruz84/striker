import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink } from 'react-router-dom'
import { mightStillBeLive } from '../../api/useLiveMatch'
import { NestedTeamNameLink } from '../../components/TeamNameLink'
import {
  useAcceptFriendlyChallenge,
  useChallengeToFriendly,
  useFriendlyChallengesReceived,
  useFriendlyChallengesSent,
  useFriendlySchedule,
  useMe,
  useRecentFriendlies,
  useRejectFriendlyChallenge,
  useUpcomingFriendlies,
  useUserTeam,
  useWithdrawFriendlyChallenge,
} from '../../api/hooks'
import type { FriendlyChallenge, FriendlyMatch } from '../../api/types'
import { defaultKickoffValue, isWeekdayAllowed, MIN_KICKOFF_LEAD_MINUTES, toDateTimeLocalValue } from '../../utils/friendlyKickoff'
import { formatDateTime } from '../../utils/formatDate'

export function FriendliesPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const enabled = !!loggedUser

  const { data: received, isLoading: receivedLoading } = useFriendlyChallengesReceived(enabled)
  const { data: sent, isLoading: sentLoading } = useFriendlyChallengesSent(enabled)
  const { data: upcoming, isLoading: upcomingLoading } = useUpcomingFriendlies(enabled)
  const { data: recent, isLoading: recentLoading } = useRecentFriendlies(enabled)

  const { data: schedule } = useFriendlySchedule(enabled)
  const challenge = useChallengeToFriendly()
  const [opponentTeamId, setOpponentTeamId] = useState('')
  const [proposedKickoff, setProposedKickoff] = useState('')
  const [challengerIsHome, setChallengerIsHome] = useState(true)

  const proposedKickoffAllowed = isWeekdayAllowed(schedule, proposedKickoff)

  const handleChallenge = (event: FormEvent) => {
    event.preventDefault()
    const id = Number(opponentTeamId)
    if (!id || id <= 0) return
    if (proposedKickoff && !proposedKickoffAllowed) return
    challenge.mutate(
      {
        opponentTeamId: id,
        proposedKickoff: proposedKickoff ? new Date(proposedKickoff).toISOString() : undefined,
        challengerIsHome,
      },
      {
        onSuccess: () => {
          setOpponentTeamId('')
          setProposedKickoff('')
          setChallengerIsHome(true)
        },
      },
    )
  }

  const myTeamId = userTeam?.team.id

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('friendlies.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('friendlies.subtitle')}
      </Typography>

      <Paper sx={{ p: 2, mb: 3 }} component="form" onSubmit={handleChallenge}>
        <Typography variant="subtitle1" gutterBottom>
          {t('friendlies.challenge.title')}
        </Typography>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start', flexWrap: 'wrap' }}>
          <TextField
            size="small"
            label={t('friendlies.challenge.opponentTeamId')}
            value={opponentTeamId}
            onChange={(e) => setOpponentTeamId(e.target.value)}
            sx={{ width: 160 }}
          />
          <TextField
            size="small"
            type="datetime-local"
            label={t('friendlies.challenge.proposedKickoff')}
            value={proposedKickoff}
            onChange={(e) => setProposedKickoff(e.target.value)}
            error={!proposedKickoffAllowed}
            helperText={!proposedKickoffAllowed ? t('friendlies.received.kickoffNotAllowed') : ' '}
            slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: defaultKickoffValue(MIN_KICKOFF_LEAD_MINUTES) } }}
          />
          <ToggleButtonGroup
            size="small"
            exclusive
            value={challengerIsHome}
            onChange={(_event, value: boolean | null) => {
              if (value !== null) setChallengerIsHome(value)
            }}
          >
            <ToggleButton value={true}>{t('friendlies.challenge.home')}</ToggleButton>
            <ToggleButton value={false}>{t('friendlies.challenge.away')}</ToggleButton>
          </ToggleButtonGroup>
          <Button type="submit" variant="contained" disabled={challenge.isPending || (!!proposedKickoff && !proposedKickoffAllowed)}>
            {t('friendlies.challenge.button')}
          </Button>
        </Stack>
        {challenge.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('friendlies.challenge.success')}</Alert>}
        {challenge.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('friendlies.challenge.error')}</Alert>}
      </Paper>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <ChallengesReceived challenges={received} isLoading={receivedLoading} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <ChallengesSent challenges={sent} isLoading={sentLoading} />
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <MatchList
            title={t('friendlies.upcoming.title')}
            emptyLabel={t('friendlies.upcoming.empty')}
            matches={upcoming}
            isLoading={upcomingLoading}
            myTeamId={myTeamId}
            showResult={false}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <MatchList
            title={t('friendlies.recent.title')}
            emptyLabel={t('friendlies.recent.empty')}
            matches={recent}
            isLoading={recentLoading}
            myTeamId={myTeamId}
            showResult
          />
        </Grid>
      </Grid>
    </Box>
  )
}

function ChallengesReceived({ challenges, isLoading }: { challenges?: FriendlyChallenge[]; isLoading: boolean }) {
  const { t } = useTranslation()
  const acceptChallenge = useAcceptFriendlyChallenge()
  const rejectChallenge = useRejectFriendlyChallenge()
  const { data: schedule } = useFriendlySchedule(true)
  const [pickingId, setPickingId] = useState<number | null>(null)
  const [kickoff, setKickoff] = useState('')

  const startPicking = (challenge: FriendlyChallenge) => {
    setPickingId(challenge.challengeId)
    setKickoff(
      challenge.proposedKickoff ? toDateTimeLocalValue(new Date(challenge.proposedKickoff)) : defaultKickoffValue(MIN_KICKOFF_LEAD_MINUTES * 2),
    )
  }

  const weekdayAllowed = (value: string) => isWeekdayAllowed(schedule, value)

  const handleConfirm = (challengeId: number) => {
    if (!kickoff || !weekdayAllowed(kickoff)) return
    acceptChallenge.mutate(
      { challengeId, kickoff: new Date(kickoff).toISOString() },
      { onSuccess: () => setPickingId(null) },
    )
  }

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('friendlies.received.title')}
      </Typography>
      {isLoading || !challenges ? (
        <Skeleton variant="rectangular" height={80} />
      ) : challenges.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {t('friendlies.received.empty')}
        </Typography>
      ) : (
        <Stack spacing={1.5}>
          {challenges.map((c) => (
            <Stack key={c.challengeId} spacing={1}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
                <Stack>
                  <Typography variant="body2" component={RouterLink} to={`/team/${c.challengerTeamId}`}>
                    {c.challengerTeamName}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {c.challengerIsHome ? t('friendlies.received.youPlayAway') : t('friendlies.received.youPlayHome')}
                  </Typography>
                  {c.proposedKickoff && pickingId !== c.challengeId && (
                    <Typography variant="caption" color="text.secondary">
                      {t('friendlies.received.proposedKickoff', { date: formatDateTime(c.proposedKickoff) })}
                    </Typography>
                  )}
                </Stack>
                {pickingId !== c.challengeId && (
                  <Stack direction="row" spacing={1}>
                    <Button size="small" variant="contained" onClick={() => startPicking(c)}>
                      {t('friendlies.received.accept')}
                    </Button>
                    <Button
                      size="small"
                      color="secondary"
                      onClick={() => rejectChallenge.mutate(c.challengeId)}
                      disabled={rejectChallenge.isPending}
                    >
                      {t('friendlies.received.reject')}
                    </Button>
                  </Stack>
                )}
              </Stack>
              {pickingId === c.challengeId && (
                <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start', flexWrap: 'wrap' }}>
                  <TextField
                    size="small"
                    type="datetime-local"
                    label={t('friendlies.received.kickoff')}
                    value={kickoff}
                    onChange={(e) => setKickoff(e.target.value)}
                    error={!weekdayAllowed(kickoff)}
                    helperText={!weekdayAllowed(kickoff) ? t('friendlies.received.kickoffNotAllowed') : ' '}
                    slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: defaultKickoffValue(MIN_KICKOFF_LEAD_MINUTES) } }}
                  />
                  <Button
                    size="small"
                    variant="contained"
                    onClick={() => handleConfirm(c.challengeId)}
                    disabled={acceptChallenge.isPending || !kickoff || !weekdayAllowed(kickoff)}
                  >
                    {t('friendlies.received.confirmKickoff')}
                  </Button>
                  <Button size="small" onClick={() => setPickingId(null)}>
                    {t('friendlies.received.cancel')}
                  </Button>
                </Stack>
              )}
            </Stack>
          ))}
        </Stack>
      )}
      {acceptChallenge.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('friendlies.received.acceptError')}</Alert>}
    </Paper>
  )
}

function ChallengesSent({ challenges, isLoading }: { challenges?: FriendlyChallenge[]; isLoading: boolean }) {
  const { t } = useTranslation()
  const withdrawChallenge = useWithdrawFriendlyChallenge()

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('friendlies.sent.title')}
      </Typography>
      {isLoading || !challenges ? (
        <Skeleton variant="rectangular" height={80} />
      ) : challenges.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {t('friendlies.sent.empty')}
        </Typography>
      ) : (
        <Stack spacing={1}>
          {challenges.map((c) => (
            <Stack key={c.challengeId} direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Stack>
                <Typography variant="body2" component={RouterLink} to={`/team/${c.opponentTeamId}`}>
                  {c.opponentTeamName}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {c.challengerIsHome ? t('friendlies.sent.youPlayHome') : t('friendlies.sent.youPlayAway')}
                </Typography>
              </Stack>
              <Button
                size="small"
                color="secondary"
                onClick={() => withdrawChallenge.mutate(c.challengeId)}
                disabled={withdrawChallenge.isPending}
              >
                {t('friendlies.sent.withdraw')}
              </Button>
            </Stack>
          ))}
        </Stack>
      )}
    </Paper>
  )
}

function MatchList({
  title,
  emptyLabel,
  matches,
  isLoading,
  myTeamId,
  showResult,
}: {
  title: string
  emptyLabel: string
  matches?: FriendlyMatch[]
  isLoading: boolean
  myTeamId?: number
  showResult: boolean
}) {
  const { t } = useTranslation()
  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {title}
      </Typography>
      {isLoading || !matches ? (
        <Skeleton variant="rectangular" height={80} />
      ) : matches.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {emptyLabel}
        </Typography>
      ) : (
        <Stack spacing={1}>
          {matches.map((m) => {
            const isHome = m.homeTeamId === myTeamId
            // See DivisionDetailsPage's identical mightStillBeLive check - played flips true well
            // before the match's result finishes broadcasting live.
            const stillLive = showResult && m.played && mightStillBeLive(m.matchDate)
            return (
              <Stack
                key={m.matchId}
                component={RouterLink}
                to={`/friendlies/${m.matchId}`}
                direction="row"
                spacing={1}
                sx={{ alignItems: 'center', textDecoration: 'none', color: 'inherit', '&:hover': { opacity: 0.8 } }}
              >
                <Typography variant="body2" sx={{ flex: 1, fontWeight: isHome ? 600 : 400 }}>
                  <NestedTeamNameLink teamId={m.homeTeamId}>{m.homeTeamName}</NestedTeamNameLink>
                </Typography>
                {stillLive ? (
                  <Chip label={t('match.live')} color="error" size="small" sx={{ fontWeight: 700 }} />
                ) : (
                  <Typography variant="body2" sx={{ minWidth: 90, textAlign: 'center' }}>
                    {showResult ? `${m.homeGoals} - ${m.awayGoals}` : formatDateTime(m.matchDate)}
                  </Typography>
                )}
                <Typography variant="body2" sx={{ flex: 1, textAlign: 'right', fontWeight: !isHome ? 600 : 400 }}>
                  <NestedTeamNameLink teamId={m.awayTeamId}>{m.awayTeamName}</NestedTeamNameLink>
                </Typography>
              </Stack>
            )
          })}
        </Stack>
      )}
    </Paper>
  )
}
