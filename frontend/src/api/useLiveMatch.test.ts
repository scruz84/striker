import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useLiveMatch } from './useLiveMatch'

// A minimal fake WebSocket that behaves like the real one for what this hook actually uses:
// onopen/onmessage/onerror/onclose handlers assigned after construction, plus close(). Tracks
// every instance created so a test can drive a specific connection (the initial one, or a
// reconnect's replacement) independently.
class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onerror: (() => void) | null = null
  onclose: (() => void) | null = null
  closed = false
  url: string

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  close() {
    this.closed = true
  }

  send(_data: string) {}
}

function send(socket: FakeWebSocket, payload: unknown) {
  act(() => {
    socket.onmessage?.({ data: JSON.stringify(payload) })
  })
}

describe('useLiveMatch', () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  // Reproduces a real bug: the backend replays a channel's *entire* published history to every
  // freshly connected socket (see backend/handlers/liveMatchHandlers.go's serveLiveMatch),
  // including on a reconnect after a dropped connection - it has no notion of "this viewer
  // already saw messages up to X" carried over from a previous socket. Before this hook reset its
  // accumulated state on each new connection attempt, a reconnect (the exact case its own retry
  // logic exists to handle - a proxy timeout, a network blip) would append that replayed history a
  // second time on top of what the first connection had already accumulated, doubling up every
  // event seen so far.
  it('does not duplicate events replayed again after a reconnect', async () => {
    const { result } = renderHook(() => useLiveMatch('/v1/teams/match/1/live', true))

    expect(FakeWebSocket.instances).toHaveLength(1)
    const first = FakeWebSocket.instances[0]
    act(() => first.onopen?.())

    const goalEvent = {
      type: 'event',
      event: { eventType: 0, minute: 18, teamOwner: 1, teamOwnerPlayers: [1], teamOwnerPlayerNames: ['A'], otherTeam: 2, homeGoals: 1, awayGoals: 0 },
      homeGoals: 1,
      awayGoals: 0,
    }
    send(first, goalEvent)
    expect(result.current.events).toHaveLength(1)
    expect(result.current.homeGoals).toBe(1)

    // The connection drops (not a "complete" message) - the hook's own reconnect logic should
    // kick in.
    act(() => first.onclose?.())
    act(() => {
      vi.advanceTimersByTime(3000)
    })

    expect(FakeWebSocket.instances).toHaveLength(2)
    const second = FakeWebSocket.instances[1]
    act(() => second.onopen?.())

    // The server always resends its full history to a fresh connection - including the same
    // goal event already delivered over the first (now-dead) socket.
    send(second, goalEvent)

    expect(result.current.events).toHaveLength(1)
    expect(result.current.homeGoals).toBe(1)
  })
})
