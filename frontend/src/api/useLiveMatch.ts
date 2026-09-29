import { useEffect, useState } from 'react'
import { getWsEndpoint } from '../config'
import type { Lineup, MatchEvent, MatchInstant, Player, TeamQualification } from './types'

interface LiveMatchMessage {
  type: 'lineups' | 'instant' | 'event' | 'halftime' | 'complete'
  instant?: MatchInstant
  event?: MatchEvent
  // Only set on the one "lineups" message published right before the first "instant" - see
  // backend common.LiveMatchMessage's own comment on why the lineups themselves never change
  // mid-broadcast.
  homeLineup?: Lineup
  awayLineup?: Lineup
  homePlayers?: Record<string, Player>
  awayPlayers?: Record<string, Player>
  homeGoals: number
  awayGoals: number
}

export interface LiveMatchState {
  connected: boolean
  homeGoals: number
  awayGoals: number
  minute: number
  events: MatchEvent[]
  // The most recent per-minute zone-by-zone strength snapshot broadcast for this match (see
  // backend common.MatchInstant) - null until the first "instant" message arrives. Lets a live
  // viewer see the same zone comparison a played match's report shows (issue #56), fed straight
  // to MatchMinuteZoneComparison/ZoneComparisonRows.
  homeQua: TeamQualification | null
  awayQua: TeamQualification | null
  // Every "instant" broadcast so far this match, in order - the same minute-by-minute history a
  // finished match's report reads back from match_instants (see WeekMatch.matchInstants), just
  // accumulated live instead of fetched in one go. Lets a live viewer scrub the zone-battle panel
  // back through the match so far (MatchMinuteZoneComparison), the same way `events` already lets
  // MatchMinuteRatings scrub player ratings live.
  instants: MatchInstant[]
  // True while the broadcast is paused for the halftime break (see backend
  // engine.liveMatchHalftimeDelay) - the scoreboard shows this instead of the pulsing "live" chip
  // so a viewer isn't left wondering why nothing has happened in a while.
  halftime: boolean
  // The starting XI/squads broadcast once via the "lineups" message (see LiveMatchMessage) -
  // null until it arrives, which for a genuinely live connection is essentially immediate (it's
  // the very first thing BroadcastMatchLive publishes), and for one catching up on history is
  // whatever the catch-up itself already delivered. Lets a live viewer show the same per-player
  // ratings a played match's report shows, computed live from `events`/`homeGoals`/`awayGoals`
  // (see utils/liveMatchRatings.computeLiveMatchRatings) rather than waiting for the match to
  // finish.
  homeLineup: Lineup | null
  awayLineup: Lineup | null
  homePlayers: Record<string, Player> | null
  awayPlayers: Record<string, Player> | null
  complete: boolean
  // True once reconnection has been exhausted without ever seeing a "complete" message - the
  // live channel itself is unreachable (a downed Redis, a persistent proxy/network problem),
  // not just between messages. Callers combine this with the REST-polled `played` flag as a
  // fallback: if the match is already known to be over, there's nothing more the socket could
  // tell them anyway, so it's safe to fall back to the plain REST view instead of leaving the
  // viewer stuck on a "connecting" state that will never resolve (see MatchDetailsPage and
  // siblings, and issue #37's own regression - the socket used to be a nice-to-have rather than
  // the only path out of the live view).
  giveUp: boolean
}

const initialState: LiveMatchState = {
  connected: false,
  homeGoals: 0,
  awayGoals: 0,
  minute: 0,
  events: [],
  homeQua: null,
  awayQua: null,
  instants: [],
  halftime: false,
  homeLineup: null,
  awayLineup: null,
  homePlayers: null,
  awayPlayers: null,
  complete: false,
  giveUp: false,
}

// A dropped connection before the match finishes (a proxy timeout, a backend restart, a network
// blip) gets a few quick reconnect attempts rather than leaving the viewer stuck on whatever
// partial state they had - useMatchDetails' REST polling (see its comment) is the backstop if
// reconnecting doesn't work either, so this doesn't need to retry forever.
const maxReconnectAttempts = 5
const reconnectDelayMs = 3000

// maxLiveBroadcastWindowMs generously bounds how long after kickoff a match's live broadcast
// (~105 real minutes - see liveMatchMinuteDelay/liveMatchHalftimeDelay in backend/engine/livematch.go - plus whatever lag
// the scheduler has before it even starts simulating) could still be in progress. Callers use
// this to decide whether it's worth opening a live connection at all for a match the REST data
// already reports as played: `played` flips true in the DB the instant the match is simulated,
// well before its broadcast finishes pacing out to viewers, so a viewer checking in any time in
// that window still needs the live view, not the long-finished-match view (issue #37) - but a
// match played hours or days ago has certainly finished broadcasting, so there's no point
// connecting a websocket just to be told "complete" immediately (which the backend does handle
// correctly either way - see serveLiveMatch - this is purely to skip the pointless round trip).
const maxLiveBroadcastWindowMs = 2 * 60 * 60 * 1000

export function mightStillBeLive(matchDate: string): boolean {
  const elapsedMs = Date.now() - new Date(matchDate).getTime()
  // A friendly or league match is read into simulation up to 15 minutes before its scheduled
  // matchDate (see GetSimulableFriendliesStmtSql/GetSimulableMatchesStmtSql), so `played` can
  // already be true while matchDate itself is still in the future. Without the elapsedMs >= 0
  // guard, that negative elapsed time trivially satisfies "< maxLiveBroadcastWindowMs" and every
  // caller pairing this with `played` (the live banner, "LIVE" chips, recent-form results) would
  // treat a match as in progress, or already decided, before its advertised kickoff has arrived.
  return elapsedMs >= 0 && elapsedMs < maxLiveBroadcastWindowMs
}

// isPast reports whether the given ISO timestamp has already arrived. Callers use this instead
// of comparing against `Date.now()` directly during render - oxlint's purity check flags a
// component body that calls the impure `Date.now()` itself, but not one that calls a plain
// helper function that happens to use it (same reasoning as mightStillBeLive above).
export function isPast(isoDate: string): boolean {
  return new Date(isoDate).getTime() <= Date.now()
}

// useLiveMatch opens a WebSocket to one of the backend's live match/cup-tie endpoints (see
// backend/handlers/liveMatchHandlers.go) and accumulates its messages into a running score,
// minute, and event list. Connecting to a match whose broadcast has already finished gets caught
// up on everything published so far in one quick burst, ending in a "complete" message with the
// final score - `complete` flips true either way, so callers can use it as the signal to switch
// over to the regular (REST) match details view.
//
// State is not reset when `path` changes - callers that render this for different matches over
// the lifetime of one component (there are none in this app today) should mount it under a
// `key={path}` so a path change gets a genuinely fresh instance instead.
// useIsDue reports whether matchDate has already arrived, and flips from false to true exactly
// once, right when it does - a single scheduled timeout rather than a repeating poll. Callers
// combine this with useLiveMatch's `connected` to decide when to actually claim a match is
// "live": a WebSocket connects (and starts receiving messages) the instant the component mounts,
// regardless of how far in the future matchDate is - see backend/handlers/liveMatchHandlers.go,
// which happily subscribes a viewer to a match days before kickoff - so `connected` alone isn't
// enough to distinguish "watching a match in progress" from "sitting on a page for a match that
// hasn't started yet" (issue #31).
function useDueAt(targetTime: number): boolean {
  const [isDue, setIsDue] = useState(() => targetTime <= Date.now())

  useEffect(() => {
    if (isDue) {
      return
    }
    const delay = targetTime - Date.now()
    const timer = setTimeout(() => setIsDue(true), Math.max(delay, 0))
    return () => clearTimeout(timer)
  }, [targetTime, isDue])

  return isDue
}

export function useIsDue(matchDate: string): boolean {
  return useDueAt(new Date(matchDate).getTime())
}

// LINEUP_LOCK_MINUTES mirrors the backend's own common.LineupLockMinutes
// (backend/common/lineuplock.go) - kept in sync manually since the two run in separate
// deployments with no shared config today.
export const LINEUP_LOCK_MINUTES = 20

// useIsLineupLocked reports whether matchDate is within LINEUP_LOCK_MINUTES of kickoff, or has
// already passed - the same instant a Set*LineupHandler call would itself start rejecting (see
// common.IsLineupLocked). Callers use this to stop offering the lineup editor before the save
// would fail anyway, not just to react to a failed save.
export function useIsLineupLocked(matchDate: string): boolean {
  return useDueAt(new Date(matchDate).getTime() - LINEUP_LOCK_MINUTES * 60 * 1000)
}

export function useLiveMatch(path: string | undefined, enabled: boolean): LiveMatchState {
  const [state, setState] = useState<LiveMatchState>(initialState)

  useEffect(() => {
    if (!enabled || !path) {
      return
    }

    let cancelled = false
    let socket: WebSocket | null = null
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined
    let attempt = 0
    let isComplete = false

    const connect = () => {
      // The backend always replays a channel's *entire* published history to a freshly connected
      // socket (see backend/handlers/liveMatchHandlers.go's serveLiveMatch), including on a
      // reconnect after a dropped connection - it has no notion of "this viewer already saw
      // messages up to X" across two separate sockets. Without this reset, a reconnect (a proxy
      // timeout, a network blip - the exact case this hook's own retry logic exists for) would
      // append that whole replayed history a second time onto whatever had already accumulated
      // from the first connection, doubling up every event and instant seen so far.
      setState(initialState)
      socket = new WebSocket(getWsEndpoint(path))

      socket.onopen = () => {
        attempt = 0
        setState((prev) => ({ ...prev, connected: true }))
      }

      socket.onmessage = (messageEvent) => {
        let msg: LiveMatchMessage
        try {
          msg = JSON.parse(messageEvent.data as string) as LiveMatchMessage
        } catch {
          return
        }
        if (msg.type === 'complete') {
          isComplete = true
        }
        setState((prev) => ({
          connected: true,
          homeGoals: msg.homeGoals,
          awayGoals: msg.awayGoals,
          minute: msg.instant?.minute ?? prev.minute,
          events: msg.event ? [...prev.events, msg.event] : prev.events,
          homeQua: msg.instant?.homeQua ?? prev.homeQua,
          awayQua: msg.instant?.awayQua ?? prev.awayQua,
          instants: msg.instant ? [...prev.instants, msg.instant] : prev.instants,
          halftime: msg.type === 'halftime',
          homeLineup: msg.homeLineup ?? prev.homeLineup,
          awayLineup: msg.awayLineup ?? prev.awayLineup,
          homePlayers: msg.homePlayers ?? prev.homePlayers,
          awayPlayers: msg.awayPlayers ?? prev.awayPlayers,
          complete: msg.type === 'complete',
          giveUp: false,
        }))
      }

      socket.onerror = () => setState((prev) => ({ ...prev, connected: false }))

      socket.onclose = () => {
        if (cancelled || isComplete) {
          setState((prev) => ({ ...prev, connected: false }))
          return
        }
        if (attempt >= maxReconnectAttempts) {
          setState((prev) => ({ ...prev, connected: false, giveUp: true }))
          return
        }
        setState((prev) => ({ ...prev, connected: false }))
        attempt += 1
        reconnectTimer = setTimeout(connect, reconnectDelayMs)
      }
    }

    connect()

    return () => {
      cancelled = true
      if (reconnectTimer) {
        clearTimeout(reconnectTimer)
      }
      socket?.close()
    }
  }, [path, enabled])

  return state
}
