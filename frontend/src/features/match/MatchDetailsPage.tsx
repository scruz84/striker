import { useEffect, useState, type SyntheticEvent } from 'react'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { queryKeys, useMatchDetails } from '../../api/hooks'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_YELLOW_CARD,
  type MatchEvent,
  type WeekMatch,
} from '../../api/types'
import { useLiveMatch } from '../../api/useLiveMatch'
import { TeamBadge } from '../../components/TeamBadge'
import { formatDateTime } from '../../utils/formatDate'
import { commentaryFor } from '../../utils/matchCommentary'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { MatchLineupPitch } from './MatchLineupPitch'

export function MatchDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const { data: match, isLoading } = useMatchDetails(id)

  if (isLoading || !match) {
    return <Skeleton variant="rectangular" height={200} />
  }

  if (!match.played) {
    return (
      <LiveMatch
        key={id}
        matchId={id}
        homeId={match.home}
        homeName={match.homeName}
        awayId={match.away}
        awayName={match.awayName}
      />
    )
  }

  return <PlayedMatch match={match} />
}

interface ScoreboardProps {
  homeId: number
  homeName: string
  awayId: number
  awayName: string
  homeGoals: number
  awayGoals: number
  status: 'live' | 'connecting' | 'played'
  minute?: number
  subtitle?: string
}

function Scoreboard({ homeId, homeName, awayId, awayName, homeGoals, awayGoals, status, minute, subtitle }: ScoreboardProps) {
  const { t } = useTranslation()
  const theme = useTheme()

  return (
    <Paper variant="outlined" sx={{ overflow: 'hidden', mb: 2 }}>
      <Box
        sx={{
          height: 4,
          background: `linear-gradient(90deg, ${theme.palette.primary.main}, ${theme.palette.primary.light})`,
        }}
      />
      <Box sx={{ p: { xs: 2, sm: 3 } }}>
        <Stack direction="row" sx={{ justifyContent: 'center', mb: 1.5 }}>
          {status === 'live' && (
            <Chip
              size="small"
              color="error"
              label={t('match.live')}
              sx={{
                fontWeight: 700,
                animation: 'pulse 1.6s ease-in-out infinite',
                '@keyframes pulse': {
                  '0%, 100%': { opacity: 1 },
                  '50%': { opacity: 0.55 },
                },
              }}
            />
          )}
          {status === 'connecting' && <Chip size="small" label={t('match.connecting')} />}
          {status === 'played' && <Chip size="small" variant="outlined" label={t('match.fullTime')} />}
        </Stack>
        <Grid container sx={{ alignItems: 'center' }}>
          <Grid size={5}>
            <Stack spacing={1} sx={{ alignItems: 'center', textAlign: 'center' }}>
              <TeamBadge teamId={homeId} teamName={homeName} size={56} />
              <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                {homeName}
              </Typography>
            </Stack>
          </Grid>
          <Grid size={2}>
            <Typography variant="h3" align="center" sx={{ fontWeight: 700 }}>
              {homeGoals} - {awayGoals}
            </Typography>
            {typeof minute === 'number' && minute > 0 && (
              <Typography variant="body2" align="center" color="text.secondary">
                {minute}&apos;
              </Typography>
            )}
          </Grid>
          <Grid size={5}>
            <Stack spacing={1} sx={{ alignItems: 'center', textAlign: 'center' }}>
              <TeamBadge teamId={awayId} teamName={awayName} size={56} />
              <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                {awayName}
              </Typography>
            </Stack>
          </Grid>
        </Grid>
        {subtitle && (
          <Typography variant="caption" color="text.secondary" align="center" sx={{ display: 'block', mt: 1.5 }}>
            {subtitle}
          </Typography>
        )}
      </Box>
    </Paper>
  )
}

function LiveMatch({
  matchId,
  homeId,
  homeName,
  awayId,
  awayName,
}: {
  matchId: string | undefined
  homeId: number
  homeName: string
  awayId: number
  awayName: string
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const live = useLiveMatch(matchId ? `/v1/teams/match/${matchId}/live` : undefined, !!matchId)

  useEffect(() => {
    if (live.complete && matchId) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.match(matchId) })
    }
  }, [live.complete, matchId, queryClient])

  if (live.complete) {
    return <Typography color="text.secondary">{t('match.matchFinished')}</Typography>
  }

  return (
    <Box>
      <Scoreboard
        homeId={homeId}
        homeName={homeName}
        awayId={awayId}
        awayName={awayName}
        homeGoals={live.homeGoals}
        awayGoals={live.awayGoals}
        status={live.connected ? 'live' : 'connecting'}
        minute={live.minute}
      />
      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Events events={live.events} homeId={homeId} homeName={homeName} awayName={awayName} />
      </Paper>
      <HeadToHeadCard teamAId={homeId} teamAName={homeName} teamBId={awayId} teamBName={awayName} />
    </Box>
  )
}

function PlayedMatch({ match }: { match: WeekMatch }) {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)

  return (
    <Box>
      <Scoreboard
        homeId={match.home}
        homeName={match.homeName}
        awayId={match.away}
        awayName={match.awayName}
        homeGoals={match.homeGoals}
        awayGoals={match.awayGoals}
        status="played"
        subtitle={formatDateTime(match.matchDate)}
      />

      <Paper variant="outlined" sx={{ mb: 2 }}>
        <Tabs value={tab} onChange={handleChange}>
          <Tab label={t('match.tabs.events')} />
          <Tab label={t('match.tabs.lineups')} />
        </Tabs>
        <Box sx={{ p: 2 }}>
          {tab === 0 && (
            <Events events={match.events ?? []} homeId={match.home} homeName={match.homeName} awayName={match.awayName} />
          )}
          {tab === 1 && (
            <Lineups
              homeName={match.homeName}
              awayName={match.awayName}
              homeLineup={match.homeLineup}
              awayLineup={match.awayLineup}
              homePlayers={match.homePlayers}
              awayPlayers={match.awayPlayers}
            />
          )}
        </Box>
      </Paper>
      <HeadToHeadCard teamAId={match.home} teamAName={match.homeName} teamBId={match.away} teamBName={match.awayName} />
    </Box>
  )
}

const NOTABLE_EVENT_TYPES: number[] = [
  EVENT_TYPE_GOAL,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_YELLOW_CARD,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_SUBSTITUTION,
]

/** A vertical timeline with home-team events on the left, away-team events on the right, and
 * a minute marker down the center - the layout real match-center timelines use. */
function Events({
  events,
  homeId,
  homeName,
  awayName,
}: {
  events: WeekMatch['events']
  homeId: number
  homeName: string
  awayName: string
}) {
  const { t } = useTranslation()
  const notable = (events ?? [])
    .filter((event) => NOTABLE_EVENT_TYPES.includes(event.eventType))
    .sort((a, b) => a.minute - b.minute)

  if (notable.length === 0) {
    return <Typography color="text.secondary">{t('match.noNotableEvents')}</Typography>
  }

  return (
    <Box sx={{ position: 'relative' }}>
      <Box
        sx={{
          position: 'absolute',
          left: '50%',
          top: 0,
          bottom: 0,
          width: '2px',
          bgcolor: 'divider',
          transform: 'translateX(-50%)',
        }}
      />
      <Stack spacing={1.5}>
        {notable.map((event, idx) => {
          const isHome = event.teamOwner === homeId
          const teamName = isHome ? homeName : awayName
          const opponentName = isHome ? awayName : homeName
          return (
            <Grid container key={idx} sx={{ alignItems: 'center' }}>
              <Grid size={5.5} sx={{ textAlign: 'right' }}>
                {isHome && <EventContent event={event} align="right" teamName={teamName} opponentName={opponentName} />}
              </Grid>
              <Grid size={1} sx={{ display: 'flex', justifyContent: 'center' }}>
                <Chip size="small" label={`${event.minute}'`} sx={{ position: 'relative', zIndex: 1 }} />
              </Grid>
              <Grid size={5.5}>
                {!isHome && <EventContent event={event} align="left" teamName={teamName} opponentName={opponentName} />}
              </Grid>
            </Grid>
          )
        })}
      </Stack>
    </Box>
  )
}

function EventContent({
  event,
  align,
  teamName,
  opponentName,
}: {
  event: MatchEvent
  align: 'left' | 'right'
  teamName: string
  opponentName: string
}) {
  const { t } = useTranslation()

  const icon = (() => {
    switch (event.eventType) {
      case EVENT_TYPE_GOAL:
        return '⚽'
      case EVENT_TYPE_NO_GOAL:
        return '🚫'
      case EVENT_TYPE_YELLOW_CARD:
        return '🟨'
      case EVENT_TYPE_RED_CARD:
        return '🟥'
      case EVENT_TYPE_INJURY:
        return '🚑'
      case EVENT_TYPE_SUBSTITUTION:
        return '🔄'
      default:
        return null
    }
  })()

  const label = (() => {
    switch (event.eventType) {
      case EVENT_TYPE_NO_GOAL:
        return t('match.missedChance')
      case EVENT_TYPE_YELLOW_CARD:
        return t('match.yellowCard')
      case EVENT_TYPE_RED_CARD:
        return t('match.redCard')
      case EVENT_TYPE_INJURY:
        return t('match.injury')
      case EVENT_TYPE_SUBSTITUTION:
        return t('match.substitution')
      default:
        return undefined
    }
  })()

  // Radio-commentator-style narration, one of several phrasings per event type (see
  // match.commentary.* in the locale files) - the same event always picks the same variant, so
  // it stays put across re-renders rather than re-rolling on every refetch.
  const commentary = commentaryFor(event, t, teamName, opponentName)

  return (
    <Stack
      direction={align === 'right' ? 'row-reverse' : 'row'}
      spacing={1}
      sx={{ alignItems: 'center', justifyContent: align === 'right' ? 'flex-start' : 'flex-start' }}
    >
      <Typography component="span" aria-label={label} title={label} sx={{ fontSize: '1.1rem', lineHeight: 1 }}>
        {icon}
      </Typography>
      <Box sx={{ textAlign: align }}>
        {event.eventType === EVENT_TYPE_GOAL && (
          <Typography variant="body2" sx={{ fontWeight: 700 }}>
            {event.homeGoals} - {event.awayGoals}
          </Typography>
        )}
        {commentary && (
          <Typography variant="body2" color="text.secondary">
            {commentary}
          </Typography>
        )}
      </Box>
    </Stack>
  )
}

interface LineupsProps {
  homeName: string
  awayName: string
  homeLineup: WeekMatch['homeLineup']
  awayLineup: WeekMatch['awayLineup']
  homePlayers: WeekMatch['homePlayers']
  awayPlayers: WeekMatch['awayPlayers']
}

function Lineups({ homeName, awayName, homeLineup, awayLineup, homePlayers, awayPlayers }: LineupsProps) {
  return (
    <Grid container spacing={2}>
      <Grid size={{ xs: 12, sm: 6 }}>
        <Typography variant="subtitle1" gutterBottom sx={{ textAlign: 'center' }}>
          {homeName}
        </Typography>
        <MatchLineupPitch lineup={homeLineup} players={homePlayers} />
      </Grid>
      <Grid size={{ xs: 12, sm: 6 }}>
        <Typography variant="subtitle1" gutterBottom sx={{ textAlign: 'center' }}>
          {awayName}
        </Typography>
        <MatchLineupPitch lineup={awayLineup} players={awayPlayers} />
      </Grid>
    </Grid>
  )
}
