import { useEffect, useState } from 'react'
import { getWsEndpoint } from '../config'
import type { MatchEvent, MatchInstant } from './types'

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
  complete: boolean
}

const initialState: LiveMatchState = {
  connected: false,
  homeGoals: 0,
  awayGoals: 0,
  minute: 0,
  events: [],
  complete: false,
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
export function useIsDue(matchDate: string): boolean {
  const [isDue, setIsDue] = useState(() => new Date(matchDate).getTime() <= Date.now())

  useEffect(() => {
    if (isDue) {
      return
    }
    const delay = new Date(matchDate).getTime() - Date.now()
    const timer = setTimeout(() => setIsDue(true), Math.max(delay, 0))
    return () => clearTimeout(timer)
  }, [matchDate, isDue])

  return isDue
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
          complete: msg.type === 'complete',
        }))
      }

      socket.onerror = () => setState((prev) => ({ ...prev, connected: false }))

      socket.onclose = () => {
        setState((prev) => ({ ...prev, connected: false }))
        if (cancelled || isComplete || attempt >= maxReconnectAttempts) {
          return
        }
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
