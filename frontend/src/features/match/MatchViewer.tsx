import { useCallback, useEffect, useState, type ReactNode, type SyntheticEvent } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { isPast, mightStillBeLive, useIsBroadcastOver, useIsDue, useIsLineupLocked, useLiveMatch } from '../../api/useLiveMatch'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { withholdSpoilerEvents } from '../../utils/liveMatchEvents'
import { goalScorers } from '../../utils/matchGoals'
import { matchSubtitle } from '../../utils/matchSubtitle'
import { matchTimeline } from '../../utils/matchMinute'
import { useMinuteScrub } from '../../utils/useMinuteScrub'
import { LineupLockedNotice } from '../lineup/LineupLockedNotice'
import { Lineups, goalkeeperRating } from './Lineups'
import { MatchEventsReport } from './MatchEventsReport'
import { MatchHeadToHeadPitch } from './MatchHeadToHeadPitch'
import { MentalityLabel } from './MentalityLabel'
import { MatchMinuteRatings } from './MatchMinuteRatings'
import { MatchMinuteZoneComparison } from './MatchMinuteZoneComparison'
import { MatchRatingsTab } from './MatchRatingsTab'
import { MatchReportTab } from './MatchReportTab'
import { Scoreboard } from './MatchScoreboard'
import { StadiumEntry } from './StadiumEntry'
import type { MatchView } from './matchViewModel'

export interface MatchViewerProps {
  view: MatchView
  // The lineup editor, built by the route's adapter (which owns the competition-specific hooks) and
  // only when the logged-in owner's own team plays the match. The viewer shows it in the scheduled
  // view only, swapped for the "lineup locked" notice once the lock window starts; nobody else sees
  // either, and neither shows live or after full time.
  lineupSection?: ReactNode
  // The opponent preview, shown right below the lineup editor and only while it is shown (never
  // once the lineup lock starts); built by the same adapter, for the same participating owner.
  previewSection?: ReactNode
  // Extra content right under a finished match's scoreboard - the cup tie's penalty-shootout line.
  belowScoreboard?: ReactNode
}

// The single page that presents any match - league, cup tie or friendly - before kickoff, live and
// after full time. The three routes (/match/:id, /cup/ties/:id, /friendlies/:id) are thin adapters
// that load their own endpoint, map it onto a MatchView and render this.
export function MatchViewer({ view, lineupSection, previewSection, belowScoreboard }: MatchViewerProps) {
  const [liveComplete, setLiveComplete] = useState(false)
  const handleLiveComplete = useCallback(() => setLiveComplete(true), [])

  // Show the live view whenever kickoff hasn't happened yet or the broadcast might still be
  // going, not only while `view.played` is still false: that flag flips true in the DB up to 15
  // minutes before matchDate (the match is read into simulation early - see
  // GetSimulableMatchesStmtSql) and again well before the live broadcast finishes pacing its
  // minutes out to a viewer (see liveMatchMinuteDelay in backend/engine/livematch.go) - a viewer
  // who opens this page any time up to matchDate, or in the broadcast window after it, still needs
  // the live view (backend/handlers/liveMatchHandlers.go now catches them up on everything already
  // broadcast, or waits for kickoff), not the long-finished-match view (issue #37). Once shown,
  // stick with the live view until the broadcast itself reports completion via onComplete, rather
  // than switching the moment the next REST poll notices `played`.
  if ((!isPast(view.matchDate) || mightStillBeLive(view.matchDate)) && !liveComplete) {
    return <LiveView view={view} lineupSection={lineupSection} previewSection={previewSection} onComplete={handleLiveComplete} />
  }

  return <FinishedView view={view} belowScoreboard={belowScoreboard} />
}

function LiveView({
  view,
  lineupSection,
  previewSection,
  onComplete,
}: {
  view: MatchView
  lineupSection?: ReactNode
  previewSection?: ReactNode
  onComplete: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  // Opened as soon as this mounts, well before kickoff if the viewer got here early - so it's
  // already subscribed and ready the moment the match actually goes live, with no reconnect
  // needed (see useLiveMatch's own comment on backend/handlers/liveMatchHandlers.go subscribing a
  // viewer regardless of how far off matchDate is).
  const live = useLiveMatch(view.liveUrl, true)
  const isDue = useIsDue(view.matchDate)
  const isBroadcastOver = useIsBroadcastOver(view.matchDate)
  const isLineupLocked = useIsLineupLocked(view.matchDate)
  // Shared by the zone-comparison panel below, so scrubbing it moves any other scrub-aware panel
  // in lockstep instead of each one tracking its own independent minute.
  // Steps through every minute broadcast so far, added minutes included (see matchTimeline).
  const liveTimeline = matchTimeline(live.instants)
  const minuteScrub = useMinuteScrub(liveTimeline.length)
  // Injury/red-card events are withheld from everything fed live events below (the narrated
  // report and MatchMinuteRatings alike) - a live viewer shouldn't learn a team just went down to
  // ten men, or which player just got injured, before the match itself is over (see
  // withholdSpoilerEvents' own comment). Once the match completes, the page switches over to the
  // finished-match view, which reads the real, unfiltered events straight from the backend.
  const liveEvents = withholdSpoilerEvents(live.events)
  const liveScorers = goalScorers(liveEvents, view.homeId)
  const subtitle = matchSubtitle(view.subtitleLabel, view.matchDate, view.weather)

  useEffect(() => {
    // live.giveUp alone isn't enough - the socket can be entirely unreachable (a downed Redis,
    // a persistent network problem) while the match is still genuinely in progress, and there's
    // nothing useful to fall back to yet in that case. Only bail out once the REST side also
    // confirms the match is actually over, restoring the pre-#37-fix resilience where a broken
    // socket never stranded a viewer once the match had genuinely finished.
    //
    // `played` alone isn't "actually over" either, and neither is `played` plus kickoff having
    // arrived: the match is simulated (and `played` flips true) up to 15 minutes before matchDate
    // (see GetSimulableMatchesStmtSql), and its broadcast then runs up to 108 more minutes (added time included) after
    // kickoff. A viewer who hits a flaky socket anywhere in that span (a proxy timeout, a network
    // blip, a phone that suspended the page while the screen was locked - anything that burns
    // through the reconnect attempts) would otherwise be dropped straight into the finished-match
    // report, final score and every goal included, while the match is still being broadcast live.
    // Only the broadcast window itself having elapsed (isBroadcastOver) says it's really over;
    // until then a given-up socket stays on the live view, which useLiveMatch retries as soon as
    // the page is visible or back online again.
    if (live.complete || (live.giveUp && view.played && isBroadcastOver)) {
      void queryClient.invalidateQueries({ queryKey: view.queryKey })
      onComplete()
    }
  }, [live.complete, live.giveUp, view.played, isBroadcastOver, view.queryKey, queryClient, onComplete])

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
          homeId={view.homeId}
          homeName={view.homeName}
          stadiumName={view.homeStadiumName}
          rivalry={view.rivalry}
          awayId={view.awayId}
          awayName={view.awayName}
          homePosition={view.homePosition}
          awayPosition={view.awayPosition}
          homeGoals={0}
          awayGoals={0}
          status="scheduled"
          subtitle={subtitle}
        />
        {lineupSection &&
          (isLineupLocked ? (
            <LineupLockedNotice />
          ) : (
            <>
              {lineupSection}
              {previewSection}
            </>
          ))}
      </Box>
    )
  }

  const lineups =
    live.homeLineup && live.awayLineup && live.homePlayers && live.awayPlayers
      ? { homeLineup: live.homeLineup, awayLineup: live.awayLineup, homePlayers: live.homePlayers, awayPlayers: live.awayPlayers }
      : null

  return (
    <Box>
      <Scoreboard
        homeId={view.homeId}
        homeName={view.homeName}
        stadiumName={view.homeStadiumName}
        rivalry={view.rivalry}
        awayId={view.awayId}
        awayName={view.awayName}
        homePosition={view.homePosition}
        awayPosition={view.awayPosition}
        homeGoals={live.homeGoals}
        awayGoals={live.awayGoals}
        status={live.connected ? (live.halftime ? 'halftime' : 'live') : 'connecting'}
        minute={live.minute}
        addedMinute={live.addedMinute}
        homeScorers={liveScorers.home}
        awayScorers={liveScorers.away}
        subtitle={subtitle}
      />
      {lineups && (
        <>
          <StadiumEntry stadiumName={view.homeStadiumName} homeName={view.homeName} awayName={view.awayName} />
          <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
              {t('match.tabs.lineups')}
            </Typography>
            <Typography variant="caption" color="text.secondary" component="div" sx={{ textAlign: 'center', mb: 1 }}>
              {view.awayName}
              <MentalityLabel mentality={lineups.awayLineup.mentality} />
            </Typography>
            <MatchHeadToHeadPitch {...lineups} />
            <Typography variant="caption" color="text.secondary" component="div" sx={{ textAlign: 'center', mt: 1 }}>
              {view.homeName}
              <MentalityLabel mentality={lineups.homeLineup.mentality} />
            </Typography>
          </Paper>
        </>
      )}
      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <MatchEventsReport
          events={liveEvents}
          homeId={view.homeId}
          homeName={view.homeName}
          awayId={view.awayId}
          awayName={view.awayName}
          // Report flavor (see MatchEventsReport) - already known by the time a viewer reaches the
          // live view, since a match is fully simulated (and its flavor persisted) before its live
          // broadcast starts pacing events out.
          weather={view.weather}
          attendance={view.attendance}
          refereeName={view.refereeName}
          assistant1Name={view.assistant1Name}
          assistant2Name={view.assistant2Name}
        />
      </Paper>
      {live.instants.length > 0 && (
        <MatchMinuteZoneComparison
          homeId={view.homeId}
          homeName={view.homeName}
          awayId={view.awayId}
          awayName={view.awayName}
          // The starting goalkeepers, counted exactly as the finished view counts them (see
          // Lineups) once the lineups have arrived - 0 only in the instant before they do - so a
          // minute's figures don't jump when the page switches over at full time.
          homeGoalkeeperRating={goalkeeperRating(live.homeLineup, live.homePlayers)}
          awayGoalkeeperRating={goalkeeperRating(live.awayLineup, live.awayPlayers)}
          matchInstants={live.instants}
          timeline={liveTimeline}
          live
          scrub={minuteScrub}
        />
      )}
      {lineups && (
        <MatchMinuteRatings
          homeId={view.homeId}
          homeName={view.homeName}
          awayId={view.awayId}
          awayName={view.awayName}
          {...lineups}
          events={liveEvents}
          instants={live.instants}
          timeline={liveTimeline}
          live
          scrub={minuteScrub}
        />
      )}
      <HeadToHeadCard teamAId={view.homeId} teamAName={view.homeName} teamBId={view.awayId} teamBName={view.awayName} />
    </Box>
  )
}

function FinishedView({ view, belowScoreboard }: { view: MatchView; belowScoreboard?: ReactNode }) {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)
  const scorers = goalScorers(view.events ?? [], view.homeId)

  return (
    <Box>
      <Scoreboard
        homeId={view.homeId}
        homeName={view.homeName}
        stadiumName={view.homeStadiumName}
        rivalry={view.rivalry}
        awayId={view.awayId}
        awayName={view.awayName}
        homePosition={view.homePosition}
        awayPosition={view.awayPosition}
        homeGoals={view.homeGoals}
        awayGoals={view.awayGoals}
        status="played"
        homeScorers={scorers.home}
        awayScorers={scorers.away}
        subtitle={matchSubtitle(view.subtitleLabel, view.matchDate, view.weather)}
      />
      {belowScoreboard}

      <Paper variant="outlined" sx={{ mb: 2 }}>
        <Tabs value={tab} onChange={handleChange} variant="scrollable" scrollButtons="auto">
          <Tab label={t('match.tabs.chronicle')} />
          <Tab label={t('match.tabs.notes')} />
          <Tab label={t('match.tabs.tactics')} />
        </Tabs>
        <Box sx={{ p: 2 }}>
          {tab === 0 && <MatchReportTab match={view} />}
          {tab === 1 && <MatchRatingsTab match={view} />}
          {tab === 2 && (
            <Lineups
              homeId={view.homeId}
              homeName={view.homeName}
              awayId={view.awayId}
              awayName={view.awayName}
              homeLineup={view.homeLineup}
              awayLineup={view.awayLineup}
              homePlayers={view.homePlayers}
              awayPlayers={view.awayPlayers}
              matchInstants={view.matchInstants}
              ratings={view.ratings}
            />
          )}
        </Box>
      </Paper>
      <HeadToHeadCard teamAId={view.homeId} teamAName={view.homeName} teamBId={view.awayId} teamBName={view.awayName} />
    </Box>
  )
}
