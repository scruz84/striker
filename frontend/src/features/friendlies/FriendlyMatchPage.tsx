import { useCallback, useEffect, useState } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Typography from '@mui/material/Typography'
import { useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { queryKeys, useFriendlyLineup, useFriendlyMatch, useMe, useSetFriendlyLineup, useUserTeam } from '../../api/hooks'
import type { FriendlyMatch, Player, Team } from '../../api/types'
import { mightStillBeLive, useIsDue, useLiveMatch } from '../../api/useLiveMatch'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { Events, Scoreboard } from '../match/MatchScoreboard'
import { formatDateTime } from '../../utils/formatDate'

export function FriendlyMatchPage() {
  const { id } = useParams<{ id: string }>()
  const { data: match, isLoading } = useFriendlyMatch(id)
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const [liveComplete, setLiveComplete] = useState(false)
  const handleLiveComplete = useCallback(() => setLiveComplete(true), [])

  if (isLoading || !match) {
    return <Skeleton variant="rectangular" height={200} />
  }

  // Show the live view whenever the broadcast might still be going, and stick with it until the
  // broadcast itself reports completion via onComplete - see MatchDetailsPage's identical
  // reasoning (issue #37).
  if ((!match.played || mightStillBeLive(match.matchDate)) && !liveComplete) {
    return (
      <LiveFriendly
        id={id}
        match={match}
        team={userTeam?.team}
        players={userTeam?.players}
        played={match.played}
        onComplete={handleLiveComplete}
      />
    )
  }

  return (
    <Scoreboard
      homeId={match.homeTeamId}
      homeName={match.homeTeamName}
      awayId={match.awayTeamId}
      awayName={match.awayTeamName}
      homeGoals={match.homeGoals}
      awayGoals={match.awayGoals}
      status="played"
      subtitle={formatDateTime(match.matchDate)}
    />
  )
}

function LiveFriendly({
  id,
  match,
  team,
  players,
  played,
  onComplete,
}: {
  id: string | undefined
  match: FriendlyMatch
  team?: Team
  players?: Player[]
  played: boolean
  onComplete: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  // Opened as soon as this mounts, well before kickoff if the viewer got here early - so it's
  // already subscribed and ready the moment the match actually goes live, with no reconnect
  // needed (see useLiveMatch's own comment on backend/handlers/liveMatchHandlers.go subscribing a
  // viewer regardless of how far off matchDate is).
  const live = useLiveMatch(id ? `/v1/friendlies/${id}/live` : undefined, !!id)
  const isDue = useIsDue(match.matchDate)

  useEffect(() => {
    // See MatchDetailsPage's identical reasoning: a broken socket alone isn't enough to bail out
    // while the match might still genuinely be in progress - only once REST also confirms it's over.
    if ((live.complete || (live.giveUp && played)) && id) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyMatch(id) })
      onComplete()
    }
  }, [live.complete, live.giveUp, played, id, queryClient, onComplete])

  if (live.complete) {
    return <Typography color="text.secondary">{t('match.matchFinished')}</Typography>
  }

  return (
    <Box>
      <Scoreboard
        homeId={match.homeTeamId}
        homeName={match.homeTeamName}
        awayId={match.awayTeamId}
        awayName={match.awayTeamName}
        homeGoals={isDue ? live.homeGoals : 0}
        awayGoals={isDue ? live.awayGoals : 0}
        // Before kickoff, show "scheduled" rather than a premature "Live" label (issue #31) - the
        // lineup editor below still renders either way, so a manager can set it up ahead of time.
        status={isDue ? (live.connected ? 'live' : 'connecting') : 'scheduled'}
        minute={isDue ? live.minute : undefined}
        subtitle={formatDateTime(match.matchDate)}
      />
      {isDue && (
        <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
          <Events events={live.events} homeId={match.homeTeamId} homeName={match.homeTeamName} awayName={match.awayTeamName} />
        </Paper>
      )}
      {id && team && players && <FriendlyLineupEditor matchId={id} team={team} players={players} />}
    </Box>
  )
}

function FriendlyLineupEditor({ matchId, team, players }: { matchId: string; team: Team; players: Player[] }) {
  const { data: lineup, isLoading } = useFriendlyLineup(matchId, true)
  const setLineup = useSetFriendlyLineup(matchId)
  if (isLoading || !lineup) {
    return <Skeleton variant="rectangular" height={300} />
  }
  return (
    <MatchLineupForm
      titleKey="friendlies.lineup.title"
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
