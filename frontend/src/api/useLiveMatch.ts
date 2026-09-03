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

// useLiveMatch opens a WebSocket to one of the backend's live match/cup-tie endpoints (see
// backend/handlers/liveMatchHandlers.go) and accumulates its messages into a running score,
// minute, and event list. Connecting to a match that's already finished gets a single
// "complete" message back with the final score - `complete` flips true either way, so callers
// can use it as the signal to switch over to the regular (REST) match details view.
//
// State is not reset when `path` changes - callers that render this for different matches over
// the lifetime of one component (there are none in this app today) should mount it under a
// `key={path}` so a path change gets a genuinely fresh instance instead.
export function useLiveMatch(path: string | undefined, enabled: boolean): LiveMatchState {
  const [state, setState] = useState<LiveMatchState>(initialState)

  useEffect(() => {
    if (!enabled || !path) {
      return
    }
    const socket = new WebSocket(getWsEndpoint(path))

    socket.onopen = () => setState((prev) => ({ ...prev, connected: true }))

    socket.onmessage = (messageEvent) => {
      let msg: LiveMatchMessage
      try {
        msg = JSON.parse(messageEvent.data as string) as LiveMatchMessage
      } catch {
        return
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
    socket.onclose = () => setState((prev) => ({ ...prev, connected: false }))

    return () => socket.close()
  }, [path, enabled])

  return state
}
