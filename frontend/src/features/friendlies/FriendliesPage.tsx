import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink } from 'react-router-dom'
import {
  useAcceptFriendlyChallenge,
  useChallengeToFriendly,
  useFriendlyChallengesReceived,
  useFriendlyChallengesSent,
  useMe,
  useRecentFriendlies,
  useRejectFriendlyChallenge,
  useUpcomingFriendlies,
  useUserTeam,
  useWithdrawFriendlyChallenge,
} from '../../api/hooks'
import type { FriendlyChallenge, FriendlyMatch } from '../../api/types'
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

  const challenge = useChallengeToFriendly()
  const [opponentTeamId, setOpponentTeamId] = useState('')

  const handleChallenge = (event: FormEvent) => {
    event.preventDefault()
    const id = Number(opponentTeamId)
    if (!id || id <= 0) return
    challenge.mutate(id, { onSuccess: () => setOpponentTeamId('') })
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
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
          <TextField
            size="small"
            label={t('friendlies.challenge.opponentTeamId')}
            value={opponentTeamId}
            onChange={(e) => setOpponentTeamId(e.target.value)}
            sx={{ width: 160 }}
          />
          <Button type="submit" variant="contained" disabled={challenge.isPending}>
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
        <Stack spacing={1}>
          {challenges.map((c) => (
            <Stack key={c.challengeId} direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography variant="body2" component={RouterLink} to={`/team/${c.challengerTeamId}`}>
                {c.challengerTeamName}
              </Typography>
              <Stack direction="row" spacing={1}>
                <Button
                  size="small"
                  variant="contained"
                  onClick={() => acceptChallenge.mutate(c.challengeId)}
                  disabled={acceptChallenge.isPending}
                >
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
              <Typography variant="body2" component={RouterLink} to={`/team/${c.opponentTeamId}`}>
                {c.opponentTeamName}
              </Typography>
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
            return (
              <Stack key={m.matchId} direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                <Typography variant="body2" sx={{ flex: 1, fontWeight: isHome ? 600 : 400 }}>
                  {m.homeTeamName}
                </Typography>
                <Typography variant="body2" sx={{ minWidth: 90, textAlign: 'center' }}>
                  {showResult ? `${m.homeGoals} - ${m.awayGoals}` : formatDateTime(m.matchDate)}
                </Typography>
                <Typography variant="body2" sx={{ flex: 1, textAlign: 'right', fontWeight: !isHome ? 600 : 400 }}>
                  {m.awayTeamName}
                </Typography>
              </Stack>
            )
          })}
        </Stack>
      )}
    </Paper>
  )
}
