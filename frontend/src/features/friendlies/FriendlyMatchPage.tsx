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
import { isPast, mightStillBeLive, useIsDue, useIsLineupLocked, useLiveMatch } from '../../api/useLiveMatch'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { formatDateTime } from '../../utils/formatDate'
import { withholdSpoilerEvents } from '../../utils/liveMatchEvents'
import { useMinuteScrub } from '../../utils/useMinuteScrub'
import { LineupLockedNotice } from '../lineup/LineupLockedNotice'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { Lineups } from '../match/MatchDetailsPage'
import { MatchEventsReport } from '../match/MatchEventsReport'
import { MatchHeadToHeadPitch } from '../match/MatchHeadToHeadPitch'
import { MatchMinuteRatings } from '../match/MatchMinuteRatings'
import { MatchMinuteZoneComparison } from '../match/MatchMinuteZoneComparison'
import { MatchRatingsTab } from '../match/MatchRatingsTab'
import { MatchReportTab } from '../match/MatchReportTab'
import { Scoreboard } from '../match/MatchScoreboard'

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

  // Show the live view whenever kickoff hasn't happened yet or the broadcast might still be
  // going, and stick with it until the broadcast itself reports completion via onComplete - see
  // MatchDetailsPage's identical reasoning (issue #37).
  if ((!isPast(match.matchDate) || mightStillBeLive(match.matchDate)) && !liveComplete) {
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
    // A friendly doesn't carry stadium flavor the way a league match does (see WeekMatch's own
    // comment) - Scoreboard simply omits the venue caption when this is empty.
    homeStadiumName: '',
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
          <Tab label={t('match.tabs.players')} />
          <Tab label={t('match.tabs.ratings')} />
        </Tabs>
        <Box sx={{ p: 2 }}>
          {tab === 0 && <MatchReportTab match={weekMatch} />}
          {tab === 1 && <MatchRatingsTab match={weekMatch} />}
          {tab === 2 && (
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
              ratings={weekMatch.ratings}
            />
          )}
        </Box>
      </Paper>
      <HeadToHeadCard
        teamAId={match.homeTeamId}
        teamAName={match.homeTeamName}
        teamBId={match.awayTeamId}
        teamBName={match.awayTeamName}
      />
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
  // Shared with the zone-comparison panel below, so scrubbing it moves any other scrub-aware panel
  // in lockstep instead of each one tracking its own independent minute.
  const minuteScrub = useMinuteScrub(live.minute)
  // Player evaluations (ratings) aren't shown at all while a match is still broadcasting live -
  // withheld until full time the same way the events below withhold injuries/red cards, so the
  // pitch simply renders without a `ratings` prop; it comes back once the page switches to the
  // finished-match view (played=true) with the real thing.
  const liveEvents = withholdSpoilerEvents(live.events)

  useEffect(() => {
    // See MatchDetailsPage's identical reasoning: a broken socket alone isn't enough to bail out
    // while the match might still genuinely be in progress - only once REST also confirms it's over.
    // `played` alone isn't "over" either, since it flips true up to 15 minutes before matchDate -
    // `isDue` guards against falling into the finished-match report before the match's own kickoff.
    if ((live.complete || (live.giveUp && played && isDue)) && id) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.friendlyMatch(id) })
      onComplete()
    }
  }, [live.complete, live.giveUp, played, isDue, id, queryClient, onComplete])

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
        status={isDue ? (live.connected ? (live.halftime ? 'halftime' : 'live') : 'connecting') : 'scheduled'}
        minute={isDue ? live.minute : undefined}
        subtitle={formatDateTime(match.matchDate)}
      />
      {isDue && live.homeLineup && live.awayLineup && live.homePlayers && live.awayPlayers && (
        <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            {t('match.tabs.lineups')}
          </Typography>
          <MatchHeadToHeadPitch
            homeLineup={live.homeLineup}
            homePlayers={live.homePlayers}
            awayLineup={live.awayLineup}
            awayPlayers={live.awayPlayers}
          />
        </Paper>
      )}
      {isDue && (
        <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
          <MatchEventsReport
            events={liveEvents}
            homeId={match.homeTeamId}
            homeName={match.homeTeamName}
            awayId={match.awayTeamId}
            awayName={match.awayTeamName}
          />
        </Paper>
      )}
      {isDue && live.instants.length > 0 && (
        <MatchMinuteZoneComparison
          homeId={match.homeTeamId}
          homeName={match.homeTeamName}
          awayId={match.awayTeamId}
          awayName={match.awayTeamName}
          // See MatchDetailsPage's identical reasoning: goalkeeper ratings aren't available live.
          homeGoalkeeperRating={0}
          awayGoalkeeperRating={0}
          matchInstants={live.instants}
          currentMinute={live.minute}
          live
          scrub={minuteScrub}
        />
      )}
      {isDue && live.homeLineup && live.awayLineup && live.homePlayers && live.awayPlayers && (
        <MatchMinuteRatings
          homeId={match.homeTeamId}
          homeName={match.homeTeamName}
          awayId={match.awayTeamId}
          awayName={match.awayTeamName}
          homeLineup={live.homeLineup}
          awayLineup={live.awayLineup}
          homePlayers={live.homePlayers}
          awayPlayers={live.awayPlayers}
          events={liveEvents}
          currentMinute={live.minute}
          live
          scrub={minuteScrub}
        />
      )}
      {id &&
        team &&
        players &&
        (isLineupLocked ? <LineupLockedNotice /> : <FriendlyLineupEditor matchId={id} team={team} players={players} />)}
      {isDue && (
        <HeadToHeadCard
          teamAId={match.homeTeamId}
          teamAName={match.homeTeamName}
          teamBId={match.awayTeamId}
          teamBName={match.awayTeamName}
        />
      )}
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
      teamId={team.id}
      players={players}
      lineup={lineup}
      onSave={(formation, positions, substitutions) => setLineup.mutate({ formation, positions, substitutions })}
      isSaving={setLineup.isPending}
      isError={setLineup.isError}
      isSuccess={setLineup.isSuccess}
      substitutionsEnabled
      isFriendly
    />
  )
}
