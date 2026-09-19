import { useCallback, useEffect, useState, type SyntheticEvent } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { queryKeys, useFriendlyLineup, useFriendlyMatch, useMe, useSetFriendlyLineup, useUserTeam } from '../../api/hooks'
import type { FriendlyMatch, Player, Team, WeekMatch } from '../../api/types'
import { mightStillBeLive, useIsDue, useIsLineupLocked, useLiveMatch } from '../../api/useLiveMatch'
import { LineupLockedNotice } from '../lineup/LineupLockedNotice'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { Lineups } from '../match/MatchDetailsPage'
import { MatchPositionComparison } from '../match/MatchPositionComparison'
import { MatchRatingsTab } from '../match/MatchRatingsTab'
import { MatchReportTab } from '../match/MatchReportTab'
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

  return <PlayedFriendly match={match} />
}

// friendlyToWeekMatch adapts a played FriendlyMatch into the WeekMatch shape MatchReportTab/
// MatchRatingsTab/Lineups expect, so a friendly's report can reuse the same components a league
// match's report does (see MatchDetailsPage's PlayedMatch) instead of a separate reimplementation.
// divisionId/season/weekNumber/inSimulation are meaningless for a friendly and left at their zero
// values - none of the reused components read them.
function friendlyToWeekMatch(match: FriendlyMatch): WeekMatch {
  return {
    matchId: match.matchId,
    divisionId: 0,
    season: 0,
    weekNumber: 0,
    matchDate: match.matchDate,
    played: match.played,
    inSimulation: false,
    home: match.homeTeamId,
    homeName: match.homeTeamName,
    away: match.awayTeamId,
    awayName: match.awayTeamName,
    homeGoals: match.homeGoals,
    awayGoals: match.awayGoals,
    events: match.events ?? null,
    matchInstants: match.matchInstants ?? null,
    homeLineup: match.homeLineup ?? null,
    awayLineup: match.awayLineup ?? null,
    homePlayers: match.homePlayers ?? null,
    awayPlayers: match.awayPlayers ?? null,
    ratings: match.ratings,
  }
}

function PlayedFriendly({ match }: { match: FriendlyMatch }) {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)
  const weekMatch = friendlyToWeekMatch(match)

  return (
    <Box>
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

      <Paper variant="outlined">
        <Tabs value={tab} onChange={handleChange} variant="scrollable" scrollButtons="auto">
          <Tab label={t('match.tabs.report')} />
          <Tab label={t('match.tabs.summary')} />
          <Tab label={t('match.tabs.ratings')} />
          <Tab label={t('match.tabs.players')} />
        </Tabs>
        <Box sx={{ p: 2 }}>
          {tab === 0 && <MatchReportTab match={weekMatch} />}
          {tab === 1 && (
            <Events events={weekMatch.events ?? []} homeId={match.homeTeamId} homeName={match.homeTeamName} awayName={match.awayTeamName} />
          )}
          {tab === 2 && <MatchRatingsTab match={weekMatch} />}
          {tab === 3 && (
            <Lineups
              homeId={match.homeTeamId}
              homeName={match.homeTeamName}
              awayId={match.awayTeamId}
              awayName={match.awayTeamName}
              homeLineup={weekMatch.homeLineup}
              awayLineup={weekMatch.awayLineup}
              homePlayers={weekMatch.homePlayers}
              awayPlayers={weekMatch.awayPlayers}
              matchInstants={weekMatch.matchInstants}
            />
          )}
        </Box>
      </Paper>
    </Box>
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
  const isLineupLocked = useIsLineupLocked(match.matchDate)

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
        // lineup editor below still renders either way (until it locks near kickoff, see
        // isLineupLocked below), so a manager can set it up ahead of time.
        status={isDue ? (live.connected ? 'live' : 'connecting') : 'scheduled'}
        minute={isDue ? live.minute : undefined}
        subtitle={formatDateTime(match.matchDate)}
      />
      {isDue && (
        <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
          <Events events={live.events} homeId={match.homeTeamId} homeName={match.homeTeamName} awayName={match.awayTeamName} />
        </Paper>
      )}
      {isDue && live.homeQua && live.awayQua && (
        <MatchPositionComparison
          homeQua={live.homeQua}
          awayQua={live.awayQua}
          // See MatchDetailsPage's identical reasoning: goalkeeper ratings aren't available live.
          homeGoalkeeperRating={0}
          awayGoalkeeperRating={0}
          homeName={match.homeTeamName}
          awayName={match.awayTeamName}
        />
      )}
      {id &&
        team &&
        players &&
        (isLineupLocked ? <LineupLockedNotice /> : <FriendlyLineupEditor matchId={id} team={team} players={players} />)}
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
