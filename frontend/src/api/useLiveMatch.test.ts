import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { maxLiveBroadcastWindowMs, mightStillBeLive, useLiveMatch } from './useLiveMatch'

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

  // Added time (match-scheduling "Live replay of a simulated result"): an instant at 45+2 moves the
  // live clock to 45+2, half time names the minute the first half ended on, and the final whistle
  // the one the match ended on.
  it('follows the broadcast into added time', () => {
    const { result } = renderHook(() => useLiveMatch('/v1/teams/match/1/live', true))
    const socket = FakeWebSocket.instances[0]
    act(() => socket.onopen?.())

    send(socket, { type: 'instant', instant: { matchId: 1, minute: 45, addedMinute: 2 }, homeGoals: 1, awayGoals: 0 })
    expect([result.current.minute, result.current.addedMinute]).toEqual([45, 2])

    send(socket, { type: 'halftime', minute: 45, addedMinute: 2, homeGoals: 1, awayGoals: 0 })
    expect(result.current.halftime).toBe(true)
    expect([result.current.minute, result.current.addedMinute]).toEqual([45, 2])

    send(socket, { type: 'instant', instant: { matchId: 1, minute: 46 }, homeGoals: 1, awayGoals: 0 })
    expect([result.current.minute, result.current.addedMinute]).toEqual([46, 0])

    send(socket, { type: 'event', event: { eventType: 0, minute: 90, addedMinute: 3, teamOwner: 2 }, homeGoals: 1, awayGoals: 1 })
    expect([result.current.minute, result.current.addedMinute]).toEqual([46, 0])
    expect(result.current.events[0].addedMinute).toBe(3)

    send(socket, { type: 'complete', minute: 90, addedMinute: 4, homeGoals: 1, awayGoals: 1 })
    expect(result.current.complete).toBe(true)
    expect([result.current.minute, result.current.addedMinute]).toEqual([90, 4])
  })

  // A phone suspends a backgrounded page's network: every reconnect attempt fails within seconds
  // and the hook gives up, mid-match. Coming back to the page has to try again rather than leave
  // the viewer given up for the rest of the broadcast.
  it('reconnects when the page becomes visible again after giving up', () => {
    const { result } = renderHook(() => useLiveMatch('/v1/friendlies/1/live', true))

    for (let i = 0; i <= 5; i++) {
      act(() => FakeWebSocket.instances.at(-1)?.onclose?.())
      act(() => {
        vi.advanceTimersByTime(3000)
      })
    }
    expect(result.current.giveUp).toBe(true)
    const attempts = FakeWebSocket.instances.length

    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    expect(FakeWebSocket.instances).toHaveLength(attempts + 1)
    expect(result.current.giveUp).toBe(false)
    act(() => FakeWebSocket.instances.at(-1)?.onopen?.())
    expect(result.current.connected).toBe(true)
  })

  it('does not open a second socket while the current one is still alive', () => {
    renderHook(() => useLiveMatch('/v1/friendlies/1/live', true))
    act(() => FakeWebSocket.instances[0].onopen?.())

    act(() => {
      window.dispatchEvent(new Event('online'))
    })

    expect(FakeWebSocket.instances).toHaveLength(1)
  })
})

describe('mightStillBeLive', () => {
  // A league match or friendly is read into simulation (and flips `played` true) up to 15 minutes
  // before its own scheduled matchDate - see GetSimulableFriendliesStmtSql/
  // GetSimulableMatchesStmtSql. A manager who opens the standings, their squad, or a player's page
  // in that window must still see the pre-match state: this is the exact bug reported (the league
  // table, team streak, injuries, and skill updates all showed the post-match state 10 minutes
  // before kickoff).
  it('is true for a match simulated minutes before its own kickoff', () => {
    const matchDate = new Date(Date.now() + 10 * 60 * 1000).toISOString()
    expect(mightStillBeLive(matchDate)).toBe(true)
  })

  it('is true right at the edge of the early-simulation window', () => {
    const matchDate = new Date(Date.now() + 15 * 60 * 1000 - 1000).toISOString()
    expect(mightStillBeLive(matchDate)).toBe(true)
  })

  it('is false for a match further than 15 minutes from kickoff', () => {
    const matchDate = new Date(Date.now() + 20 * 60 * 1000).toISOString()
    expect(mightStillBeLive(matchDate)).toBe(false)
  })

  it('is true right at kickoff', () => {
    expect(mightStillBeLive(new Date().toISOString())).toBe(true)
  })

  it('is true while the broadcast could still be in progress after kickoff', () => {
    const matchDate = new Date(Date.now() - 30 * 60 * 1000).toISOString()
    expect(mightStillBeLive(matchDate)).toBe(true)
  })

  it('is false once the broadcast has certainly finished', () => {
    const matchDate = new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString()
    expect(mightStillBeLive(matchDate)).toBe(false)
  })

  // stoppage-time "Results stay hidden until the final whistle": the longest replay, 90+5 with the
  // second half's maximum added time, blows its final whistle 108 minutes after kickoff - still
  // inside the window, so nothing reveals the result before it.
  it('is still true at the final whistle of the longest match', () => {
    const matchDate = new Date(Date.now() - 108 * 60 * 1000).toISOString()
    expect(mightStillBeLive(matchDate)).toBe(true)
  })

  // Mirrors backend common.LiveBroadcastWindow (2 hours): the backend's
  // TestLongestBroadcastFitsLiveWindow keeps the longest replay at least 5 minutes inside it.
  it('is the same 2-hour window as the backend', () => {
    expect(maxLiveBroadcastWindowMs).toBe(2 * 60 * 60 * 1000)
    expect(maxLiveBroadcastWindowMs - 108 * 60 * 1000).toBeGreaterThanOrEqual(5 * 60 * 1000)
  })
})
