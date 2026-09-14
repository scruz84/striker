import { useCallback, useEffect, useState, type SyntheticEvent } from 'react'
import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { queryKeys, useMatchDetails, useMatchLineup, useMe, useSetMatchLineup, useUserTeam } from '../../api/hooks'
import { POSITION_GOALKEEPER, type Player, type Team, type WeekMatch } from '../../api/types'
import { mightStillBeLive, useIsDue, useLiveMatch } from '../../api/useLiveMatch'
import { formatDateTime } from '../../utils/formatDate'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { MatchHeadToHeadPitch } from './MatchHeadToHeadPitch'
import { MatchPositionComparison } from './MatchPositionComparison'
import { Events, Scoreboard } from './MatchScoreboard'

export function MatchDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const { data: match, isLoading } = useMatchDetails(id)
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const [liveComplete, setLiveComplete] = useState(false)
  const handleLiveComplete = useCallback(() => setLiveComplete(true), [])

  if (isLoading || !match) {
    return <Skeleton variant="rectangular" height={200} />
  }

  // Show the live view whenever the broadcast might still be going, not only while `match.played`
  // is still false: that flag flips true in the DB the instant the match is simulated, well
  // before the live broadcast finishes pacing its minutes out to a viewer (see
  // liveMatchMinuteDelay in backend/engine/livematch.go) - a viewer who opens this page any time
  // in that window still needs the live view (backend/handlers/liveMatchHandlers.go now catches
  // them up on everything already broadcast), not the long-finished-match view (issue #37). Once
  // shown, stick with the live view until the broadcast itself reports completion via onComplete,
  // rather than switching the moment the next REST poll notices `played`.
  if ((!match.played || mightStillBeLive(match.matchDate)) && !liveComplete) {
    const isParticipant = userTeam?.team.id === match.home || userTeam?.team.id === match.away
    return (
      <LiveMatch
        key={id}
        matchId={id}
        homeId={match.home}
        homeName={match.homeName}
        awayId={match.away}
        awayName={match.awayName}
        matchDate={match.matchDate}
        team={isParticipant ? userTeam?.team : undefined}
        players={isParticipant ? userTeam?.players : undefined}
        onComplete={handleLiveComplete}
      />
    )
  }

  return <PlayedMatch match={match} />
}

function LiveMatch({
  matchId,
  homeId,
  homeName,
  awayId,
  awayName,
  matchDate,
  team,
  players,
  onComplete,
}: {
  matchId: string | undefined
  homeId: number
  homeName: string
  awayId: number
  awayName: string
  matchDate: string
  team?: Team
  players?: Player[]
  onComplete: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  // Opened as soon as this mounts, well before kickoff if the viewer got here early - so it's
  // already subscribed and ready the moment the match actually goes live, with no reconnect
  // needed (see useLiveMatch's own comment on backend/handlers/liveMatchHandlers.go subscribing a
  // viewer regardless of how far off matchDate is).
  const live = useLiveMatch(matchId ? `/v1/teams/match/${matchId}/live` : undefined, !!matchId)
  const isDue = useIsDue(matchDate)

  useEffect(() => {
    if (live.complete && matchId) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.match(matchId) })
      onComplete()
    }
  }, [live.complete, matchId, queryClient, onComplete])

  if (live.complete) {
    return <Typography color="text.secondary">{t('match.matchFinished')}</Typography>
  }

  // Before kickoff, show the scheduled date instead of a premature "Live" label (issue #31) -
  // the WebSocket above is already connected regardless, so once isDue flips true this switches
  // straight into the live view with no gap.
  if (!isDue) {
    return (
      <Box>
        <Scoreboard
          homeId={homeId}
          homeName={homeName}
          awayId={awayId}
          awayName={awayName}
          homeGoals={0}
          awayGoals={0}
          status="scheduled"
          subtitle={formatDateTime(matchDate)}
        />
        {matchId && team && players && <MatchLineupSection matchId={matchId} team={team} players={players} />}
      </Box>
    )
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

function MatchLineupSection({ matchId, team, players }: { matchId: string; team: Team; players: Player[] }) {
  const { data: lineup, isLoading } = useMatchLineup(matchId, true)
  const setLineup = useSetMatchLineup(matchId)
  if (isLoading || !lineup) {
    return <Skeleton variant="rectangular" height={300} />
  }
  return (
    <MatchLineupForm
      titleKey="match.lineup.title"
      titleValues={{ teamName: team.name }}
      players={players}
      lineup={lineup}
      onSave={(formation, positions) => setLineup.mutate({ formation, positions })}
      isSaving={setLineup.isPending}
      isError={setLineup.isError}
      isSuccess={setLineup.isSuccess}
    />
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
              matchInstants={match.matchInstants}
            />
          )}
        </Box>
      </Paper>
      <HeadToHeadCard teamAId={match.home} teamAName={match.homeName} teamBId={match.away} teamBName={match.awayName} />
    </Box>
  )
}

export interface LineupsProps {
  homeName: string
  awayName: string
  homeLineup: WeekMatch['homeLineup']
  awayLineup: WeekMatch['awayLineup']
  homePlayers: WeekMatch['homePlayers']
  awayPlayers: WeekMatch['awayPlayers']
  matchInstants: WeekMatch['matchInstants']
}

// The starting goalkeeper's own rating - not part of TeamQualification (see its own comment),
// but this is exactly what engine.processChance reads live for goalkeeping strength. Exported for
// reuse by CupTieDetailsPage's own played-tie lineups tab.
export function goalkeeperRating(lineup: WeekMatch['homeLineup'], players: WeekMatch['homePlayers']): number {
  const gk = lineup?.positions.find((p) => p.position === POSITION_GOALKEEPER)
  if (!gk || !players) return 0
  return players[String(gk.playerId)]?.goalKeeper ?? 0
}

// Exported for reuse by CupTieDetailsPage - a cup tie's lineups tab is otherwise identical, minus
// per-minute instants (cup ties don't record those - see backend/common.CupTieDetails), so
// callers there simply pass matchInstants={null}.
export function Lineups({ homeName, awayName, homeLineup, awayLineup, homePlayers, awayPlayers, matchInstants }: LineupsProps) {
  // The kickoff instant (minute 1) reflects the same starting XI the pitch below shows - later
  // instants would drift as the match itself changes who's on the pitch (cards, injuries, subs).
  const kickoff = matchInstants?.[0]

  return (
    <Grid container spacing={3}>
      {kickoff && (
        <Grid size={{ xs: 12, md: 5 }}>
          <MatchPositionComparison
            homeQua={kickoff.homeQua}
            awayQua={kickoff.awayQua}
            homeGoalkeeperRating={goalkeeperRating(homeLineup, homePlayers)}
            awayGoalkeeperRating={goalkeeperRating(awayLineup, awayPlayers)}
            homeName={homeName}
            awayName={awayName}
          />
        </Grid>
      )}
      <Grid size={{ xs: 12, md: kickoff ? 7 : 12 }}>
        <Box sx={{ maxWidth: 380, mx: 'auto' }}>
          <Typography variant="subtitle1" gutterBottom sx={{ textAlign: 'center' }}>
            {awayName}
          </Typography>
          <MatchHeadToHeadPitch
            homeLineup={homeLineup}
            homePlayers={homePlayers}
            awayLineup={awayLineup}
            awayPlayers={awayPlayers}
          />
          <Typography variant="subtitle1" sx={{ textAlign: 'center' }}>
            {homeName}
          </Typography>
        </Box>
      </Grid>
    </Grid>
  )
}
