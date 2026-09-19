import { useEffect, useState } from 'react'
import { getWsEndpoint } from '../config'
import type { MatchEvent, MatchInstant, TeamQualification } from './types'

interface LiveMatchMessage {
  type: 'instant' | 'event' | 'complete'
  instant?: MatchInstant
  event?: MatchEvent
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
  // to MatchPositionComparison.
  homeQua: TeamQualification | null
  awayQua: TeamQualification | null
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
// (~90 real minutes - see liveMatchMinuteDelay in backend/engine/livematch.go - plus whatever lag
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
  return Date.now() - new Date(matchDate).getTime() < maxLiveBroadcastWindowMs
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
