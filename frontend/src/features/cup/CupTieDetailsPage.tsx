import { type SyntheticEvent, useCallback, useEffect, useState } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { queryKeys, useCupTieDetails, useCupTieLineup, useMe, useSetCupTieLineup, useUserTeam } from '../../api/hooks'
import type { CupTieDetails, Player, Team, WeekMatch } from '../../api/types'
import { isPast, mightStillBeLive, useIsDue, useIsLineupLocked, useLiveMatch } from '../../api/useLiveMatch'
import { formatDateTime } from '../../utils/formatDate'
import { computeRatingsAsOfMinute } from '../../utils/liveMatchRatings'
import { useMinuteScrub } from '../../utils/useMinuteScrub'
import { LineupLockedNotice } from '../lineup/LineupLockedNotice'
import { MatchLineupForm } from '../lineup/MatchLineupForm'
import { Scoreboard } from '../match/MatchScoreboard'
import { Lineups } from '../match/MatchDetailsPage'
import { MatchEventsReport } from '../match/MatchEventsReport'
import { MatchHeadToHeadPitch } from '../match/MatchHeadToHeadPitch'
import { MatchMinuteZoneComparison } from '../match/MatchMinuteZoneComparison'
import { MatchReportTab } from '../match/MatchReportTab'

export function CupTieDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const { data: tie, isLoading } = useCupTieDetails(id)
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const [liveComplete, setLiveComplete] = useState(false)
  const handleLiveComplete = useCallback(() => setLiveComplete(true), [])

  if (isLoading || !tie) {
    return <Skeleton variant="rectangular" height={200} />
  }

  if (tie.homeTeamId === null || tie.awayTeamId === null) {
    return <ByeTie tie={tie} />
  }

  // Show the live view whenever kickoff hasn't happened yet or the broadcast might still be
  // going, and stick with it until the broadcast itself reports completion via onComplete - see
  // MatchDetailsPage's identical reasoning (issue #37).
  if ((!isPast(tie.matchDate ?? '') || mightStillBeLive(tie.matchDate ?? '')) && !liveComplete) {
    const isParticipant = userTeam?.team.id === tie.homeTeamId || userTeam?.team.id === tie.awayTeamId
    return (
      <LiveTie
        key={id}
        id={id}
        tie={tie}
        team={isParticipant ? userTeam?.team : undefined}
        players={isParticipant ? userTeam?.players : undefined}
        played={tie.played}
        onComplete={handleLiveComplete}
      />
    )
  }

  return <PlayedTie tie={tie} />
}

function ByeTie({ tie }: { tie: CupTieDetails }) {
  const { t } = useTranslation()
  return (
    <Paper variant="outlined" sx={{ p: 3, textAlign: 'center' }}>
      <Typography>{t('cup.bye')}</Typography>
      <Typography color="text.secondary">{tie.homeTeamName || tie.awayTeamName}</Typography>
    </Paper>
  )
}

// LiveTie mirrors MatchDetailsPage's LiveMatch / FriendlyMatchPage's LiveFriendly - a cup tie
// gets the same live-websocket treatment as a league match or friendly (see
// handlers.LiveCupTieHandler, which broadcasts to the same per-match channel shape), so a tie in
// progress isn't stuck showing a static "Scheduled" scoreboard until the page is reloaded.
function LiveTie({
  id,
  tie,
  team,
  players,
  played,
  onComplete,
}: {
  id: string | undefined
  tie: CupTieDetails
  team?: Team
  players?: Player[]
  played: boolean
  onComplete: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const homeId = tie.homeTeamId ?? 0
  const awayId = tie.awayTeamId ?? 0
  // Opened as soon as this mounts, well before kickoff if the viewer got here early - so it's
  // already subscribed and ready the moment the tie actually goes live, with no reconnect needed
  // (see useLiveMatch's own comment on backend/handlers/liveMatchHandlers.go subscribing a viewer
  // regardless of how far off matchDate is).
  const live = useLiveMatch(id ? `/v1/cups/ties/${id}/live` : undefined, !!id)
  const isDue = useIsDue(tie.matchDate ?? '')
  const isLineupLocked = useIsLineupLocked(tie.matchDate ?? '')
  // Shared with the zone-comparison panel below, so scrubbing it also updates the pitch's
  // per-player ratings instead of the pitch staying frozen on the live/current minute.
  const minuteScrub = useMinuteScrub(live.minute)
  const pitchRatings =
    live.homeLineup && live.awayLineup ? computeRatingsAsOfMinute(live.homeLineup, live.awayLineup, live.events, minuteScrub.minute) : undefined

  useEffect(() => {
    // See MatchDetailsPage's identical reasoning: a broken socket alone isn't enough to bail out
    // while the tie might still genuinely be in progress - only once REST also confirms it's over.
    // `played` alone isn't "over" either, since it flips true up to 15 minutes before matchDate -
    // `isDue` guards against falling into the finished-tie report before the tie's own kickoff.
    if ((live.complete || (live.giveUp && played && isDue)) && id) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cupTie(id) })
      onComplete()
    }
  }, [live.complete, live.giveUp, played, isDue, id, queryClient, onComplete])

  if (live.complete) {
    return <Typography color="text.secondary">{t('match.matchFinished')}</Typography>
  }

  // Before kickoff, show the scheduled date instead of a premature "Live" label (same reasoning
  // as MatchDetailsPage/FriendlyMatchPage) - the WebSocket above is already connected regardless.
  if (!isDue) {
    return (
      <Box>
        <Scoreboard
          homeId={homeId}
          homeName={tie.homeTeamName}
          awayId={awayId}
          awayName={tie.awayTeamName}
          homeGoals={0}
          awayGoals={0}
          status="scheduled"
          subtitle={tie.matchDate ? formatDateTime(tie.matchDate) : undefined}
        />
        {id &&
          team &&
          players &&
          (isLineupLocked ? <LineupLockedNotice /> : <CupTieLineupSection tieId={id} team={team} players={players} />)}
      </Box>
    )
  }

  return (
    <Box>
      <Scoreboard
        homeId={homeId}
        homeName={tie.homeTeamName}
        awayId={awayId}
        awayName={tie.awayTeamName}
        homeGoals={live.homeGoals}
        awayGoals={live.awayGoals}
        status={live.connected ? (live.halftime ? 'halftime' : 'live') : 'connecting'}
        minute={live.minute}
      />
      {live.homeLineup && live.awayLineup && live.homePlayers && live.awayPlayers && (
        <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            {t('match.tabs.lineups')}
          </Typography>
          <MatchHeadToHeadPitch
            homeLineup={live.homeLineup}
            homePlayers={live.homePlayers}
            awayLineup={live.awayLineup}
            awayPlayers={live.awayPlayers}
            ratings={pitchRatings}
          />
        </Paper>
      )}
      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <MatchEventsReport events={live.events} homeId={homeId} homeName={tie.homeTeamName} awayId={awayId} awayName={tie.awayTeamName} />
      </Paper>
      {live.instants.length > 0 && (
        <MatchMinuteZoneComparison
          homeId={homeId}
          homeName={tie.homeTeamName}
          awayId={awayId}
          awayName={tie.awayTeamName}
          // See MatchDetailsPage's identical reasoning: goalkeeper ratings aren't available live.
          homeGoalkeeperRating={0}
          awayGoalkeeperRating={0}
          matchInstants={live.instants}
          currentMinute={live.minute}
          live
          scrub={minuteScrub}
        />
      )}
      {id &&
        team &&
        players &&
        (isLineupLocked ? <LineupLockedNotice /> : <CupTieLineupSection tieId={id} team={team} players={players} />)}
    </Box>
  )
}

function CupTieLineupSection({ tieId, team, players }: { tieId: string; team: Team; players: Player[] }) {
  const { data: lineup, isLoading } = useCupTieLineup(tieId, true)
  const setLineup = useSetCupTieLineup(tieId)
  if (isLoading || !lineup) {
    return <Skeleton variant="rectangular" height={300} />
  }
  return (
    <MatchLineupForm
      titleKey="cup.lineup.title"
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

// cupTieToWeekMatch adapts a played CupTieDetails into the WeekMatch shape MatchReportTab
// expects, so a tie's report can reuse the same Hattrick-style narrative a league match's or
// friendly's report does (see MatchDetailsPage's PlayedMatch / FriendlyMatchPage's
// friendlyToWeekMatch) instead of the bare live-style event ticker this used to fall back to.
// divisionId/season/weekNumber/inSimulation/matchInstants/ratings are meaningless or unavailable
// for a cup tie and left at their zero/null values - MatchReportTab never reads them, and a tie
// doesn't carry weather/attendance/referee flavor either, so its report simply omits that intro
// paragraph, same as a friendly's.
function cupTieToWeekMatch(tie: CupTieDetails): WeekMatch {
  return {
    matchId: tie.tieId,
    divisionId: 0,
    season: 0,
    weekNumber: 0,
    matchDate: tie.matchDate ?? '',
    played: tie.played,
    inSimulation: false,
    home: tie.homeTeamId ?? 0,
    homeName: tie.homeTeamName,
    homeStadiumName: '',
    away: tie.awayTeamId ?? 0,
    awayName: tie.awayTeamName,
    homeGoals: tie.homeGoals,
    awayGoals: tie.awayGoals,
    events: tie.events,
    matchInstants: null,
    homeLineup: tie.homeLineup,
    awayLineup: tie.awayLineup,
    homePlayers: tie.homePlayers,
    awayPlayers: tie.awayPlayers,
  }
}

function PlayedTie({ tie }: { tie: CupTieDetails }) {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)
  const weekMatch = cupTieToWeekMatch(tie)

  return (
    <Box>
      <Scoreboard
        homeId={tie.homeTeamId ?? 0}
        homeName={tie.homeTeamName}
        awayId={tie.awayTeamId ?? 0}
        awayName={tie.awayTeamName}
        homeGoals={tie.homeGoals}
        awayGoals={tie.awayGoals}
        status="played"
        subtitle={tie.matchDate ? formatDateTime(tie.matchDate) : undefined}
      />
      {tie.wentToPenalties && (
        <Typography color="text.secondary" align="center" sx={{ mb: 2 }}>
          {t('cup.pens')}
        </Typography>
      )}

      <Paper variant="outlined" sx={{ mb: 2 }}>
        <Tabs value={tab} onChange={handleChange}>
          <Tab label={t('match.tabs.report')} />
          <Tab label={t('match.tabs.lineups')} />
        </Tabs>
        <Box sx={{ p: 2 }}>
          {tab === 0 && <MatchReportTab match={weekMatch} />}
          {tab === 1 && (
            <Lineups
              homeId={tie.homeTeamId ?? 0}
              homeName={tie.homeTeamName}
              awayId={tie.awayTeamId ?? 0}
              awayName={tie.awayTeamName}
              homeLineup={tie.homeLineup}
              awayLineup={tie.awayLineup}
              homePlayers={tie.homePlayers}
              awayPlayers={tie.awayPlayers}
              matchInstants={null}
            />
          )}
        </Box>
      </Paper>
    </Box>
  )
}
