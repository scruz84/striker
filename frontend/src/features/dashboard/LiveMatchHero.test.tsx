import { act, cleanup, render, renderHook, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { TeamMatchSummary } from '../../api/types'
import type { LiveMatchState } from '../../api/useLiveMatch'
import i18n from '../../i18n'
import { LiveMatchHero, useOwnLiveMatch } from './DashboardPage'

const socket = vi.hoisted(() => ({ state: {} as unknown, path: undefined as string | undefined, enabled: false }))

vi.mock('../../api/useLiveMatch', async (importActual) => ({
  ...(await importActual<typeof import('../../api/useLiveMatch')>()),
  useLiveMatch: (path: string | undefined, enabled: boolean) => {
    socket.path = path
    socket.enabled = enabled
    return socket.state as LiveMatchState
  },
}))

const liveState = (overrides: Partial<LiveMatchState> = {}) =>
  ({ connected: true, homeGoals: 0, awayGoals: 0, minute: 0, halftime: false, complete: false, giveUp: false, ...overrides }) as LiveMatchState

function summary(overrides: Partial<TeamMatchSummary> = {}): TeamMatchSummary {
  return {
    kind: 'cup',
    matchId: 5,
    opponentId: 2,
    opponentName: 'Rivals FC',
    isHome: true,
    matchDate: new Date(Date.now() - 30 * 60_000).toISOString(),
    played: true,
    homeGoals: 3,
    awayGoals: 2,
    ...overrides,
  }
}

const team = { id: 1, name: 'Mine FC' }

function renderHero(match: TeamMatchSummary, live: LiveMatchState) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <LiveMatchHero match={match} live={live} team={team} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(cleanup)

describe('LiveMatchHero', () => {
  it('shows the broadcast score and minute, never the stored final result', () => {
    renderHero(summary(), liveState({ homeGoals: 1, awayGoals: 2, minute: 63 }))
    expect(screen.getByText('1 - 2')).toBeTruthy()
    expect(screen.getByText("63'")).toBeTruthy()
    expect(screen.queryByText('3 - 2')).toBeNull()
    expect(screen.getByText('LIVE')).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Watch live' }).getAttribute('href')).toBe('/cup/ties/5')
  })

  it('reads Half-time instead of the minute during the break', () => {
    renderHero(summary(), liveState({ homeGoals: 1, awayGoals: 0, minute: 45, halftime: true }))
    expect(screen.getByText('Half-time')).toBeTruthy()
    expect(screen.queryByText("45'")).toBeNull()
  })

  it('shows 0 - 0 and no minute before the first broadcast minute', () => {
    renderHero(summary(), liveState())
    expect(screen.getByText('0 - 0')).toBeTruthy()
    expect(screen.queryByText(/'$/)).toBeNull()
  })

  it('drops the numbers when the live channel is unreachable', () => {
    renderHero(summary(), liveState({ giveUp: true, homeGoals: 1, minute: 20 }))
    expect(screen.queryByText(/ - /)).toBeNull()
    expect(screen.getByText('LIVE')).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Watch live' })).toBeTruthy()
  })

  it('orders the teams home/away', () => {
    renderHero(summary({ isHome: false }), liveState())
    const text = document.body.textContent ?? ''
    expect(text.indexOf('Rivals FC')).toBeLessThan(text.indexOf('Mine FC'))
  })
})

describe('useOwnLiveMatch', () => {
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={new QueryClient()}>{children}</QueryClientProvider>
  }

  it('opens no socket without a live match', () => {
    socket.state = liveState()
    const old = summary({ matchDate: new Date(Date.now() - 5 * 3600_000).toISOString() })
    const { result } = renderHook(() => useOwnLiveMatch([old], '1'), { wrapper })
    expect(result.current.match).toBeUndefined()
    expect(socket.enabled).toBe(false)
  })

  it('follows the live match and turns it off when the broadcast completes', () => {
    socket.state = liveState({ minute: 10 })
    const live = summary()
    const { result, rerender } = renderHook(() => useOwnLiveMatch([live], '1'), { wrapper })
    expect(result.current.match).toBe(live)
    expect(socket.path).toBe('/v1/cups/ties/5/live')
    expect(socket.enabled).toBe(true)

    socket.state = liveState({ complete: true })
    act(() => rerender())
    expect(result.current.match).toBeUndefined()
    expect(result.current.finished.has('cup:5')).toBe(true)
  })
})
