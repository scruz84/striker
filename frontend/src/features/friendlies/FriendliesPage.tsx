import { useRef, useState, type FormEvent } from 'react'
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
import HandshakeTwoToneIcon from '@mui/icons-material/HandshakeTwoTone'
import Collapse from '@mui/material/Collapse'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink } from 'react-router-dom'
import { mightStillBeLive } from '../../api/useLiveMatch'
import { EmptyState } from '../../components/EmptyState'
import { PageHeader } from '../../components/PageHeader'
import { TeamAutocomplete } from '../../components/TeamAutocomplete'
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
import type { FriendlyChallenge, FriendlyMatch, TeamSearchResult } from '../../api/types'
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
  const [opponent, setOpponent] = useState<TeamSearchResult | null>(null)
  const opponentInputRef = useRef<HTMLInputElement>(null)
  const [proposedKickoff, setProposedKickoff] = useState('')
  const [challengerIsHome, setChallengerIsHome] = useState(true)

  const proposedKickoffAllowed = isWeekdayAllowed(schedule, proposedKickoff)

  const handleChallenge = (event: FormEvent) => {
    event.preventDefault()
    if (!opponent) return
    if (proposedKickoff && !proposedKickoffAllowed) return
    challenge.mutate(
      {
        opponentTeamId: opponent.id,
        proposedKickoff: proposedKickoff ? new Date(proposedKickoff).toISOString() : undefined,
        challengerIsHome,
      },
      {
        onSuccess: () => {
          setOpponent(null)
          setProposedKickoff('')
          setChallengerIsHome(true)
        },
      },
    )
  }

  const myTeamId = userTeam?.team.id
  const [helpOpen, setHelpOpen] = useState(false)

  const nothingToShow =
    !receivedLoading &&
    !sentLoading &&
    !upcomingLoading &&
    !recentLoading &&
    !received?.length &&
    !sent?.length &&
    !upcoming?.length &&
    !recent?.length

  return (
    <Box>
      <PageHeader title={t('friendlies.title')} />
      <Collapse in={helpOpen} unmountOnExit>
        <Alert severity="info" sx={{ mb: 2 }} onClose={() => setHelpOpen(false)}>
          {t('friendlies.subtitle')}
        </Alert>
      </Collapse>
      {!helpOpen && (
        <Button size="small" onClick={() => setHelpOpen(true)} sx={{ mb: 1 }}>
          {t('friendlies.howItWorks')}
        </Button>
      )}

      <Paper sx={{ p: 2, mb: 3 }} component="form" onSubmit={handleChallenge}>
        <Typography variant="subtitle1" gutterBottom>
          {t('friendlies.challenge.title')}
        </Typography>
        <Stack spacing={2}>
          <TeamAutocomplete
            label={t('friendlies.challenge.opponent')}
            value={opponent}
            onChange={setOpponent}
            excludeTeamId={myTeamId}
            inputRef={opponentInputRef}
          />
          <TextField
            type="datetime-local"
            label={t('friendlies.challenge.proposedKickoff')}
            value={proposedKickoff}
            onChange={(e) => setProposedKickoff(e.target.value)}
            error={!proposedKickoffAllowed}
            helperText={!proposedKickoffAllowed ? t('friendlies.received.kickoffNotAllowed') : undefined}
            slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: defaultKickoffValue(MIN_KICKOFF_LEAD_MINUTES) } }}
            fullWidth
          />
          <ToggleButtonGroup
            exclusive
            fullWidth
            value={challengerIsHome}
            aria-label={t('friendlies.challenge.side')}
            onChange={(_event, value: boolean | null) => {
              if (value !== null) setChallengerIsHome(value)
            }}
          >
            <ToggleButton value={true}>{t('friendlies.challenge.home')}</ToggleButton>
            <ToggleButton value={false}>{t('friendlies.challenge.away')}</ToggleButton>
          </ToggleButtonGroup>
          <Button
            type="submit"
            variant="contained"
            disabled={!opponent || challenge.isPending || (!!proposedKickoff && !proposedKickoffAllowed)}
          >
            {t('friendlies.challenge.button')}
          </Button>
          {!opponent && (
            <Typography variant="caption" color="text.secondary">
              {t('friendlies.challenge.opponentHelper')}
            </Typography>
          )}
        </Stack>
        {challenge.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('friendlies.challenge.success')}</Alert>}
        {challenge.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('friendlies.challenge.error')}</Alert>}
      </Paper>

      {nothingToShow ? (
        <EmptyState
          icon={<HandshakeTwoToneIcon fontSize="inherit" />}
          title={t('friendlies.challenge.emptyTitle')}
          text={t('friendlies.challenge.emptyText')}
          action={
            <Button
              variant="contained"
              onClick={() => {
                opponentInputRef.current?.scrollIntoView({ block: 'center', behavior: 'smooth' })
                opponentInputRef.current?.focus()
              }}
            >
              {t('friendlies.challenge.emptyCta')}
            </Button>
          }
        />
      ) : (
        <>
          {(isLoadingOrHas(received, receivedLoading) || isLoadingOrHas(sent, sentLoading)) && (
            <Grid container spacing={2} sx={{ mb: 2 }}>
              {isLoadingOrHas(received, receivedLoading) && (
                <Grid size={{ xs: 12, sm: 6 }}>
                  <ChallengesReceived challenges={received} isLoading={receivedLoading} />
                </Grid>
              )}
              {isLoadingOrHas(sent, sentLoading) && (
                <Grid size={{ xs: 12, sm: 6 }}>
                  <ChallengesSent challenges={sent} isLoading={sentLoading} />
                </Grid>
              )}
            </Grid>
          )}

          <Grid container spacing={2}>
            {isLoadingOrHas(upcoming, upcomingLoading) && (
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
            )}
            {isLoadingOrHas(recent, recentLoading) && (
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
            )}
          </Grid>
        </>
      )}
    </Box>
  )
}

// A section is only worth its own card while it loads or has something to show - four stacked "no
// hay..." cards are a screenful of nothing on a phone.
function isLoadingOrHas(items: unknown[] | undefined, isLoading: boolean): boolean {
  return isLoading || !items || items.length > 0
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
