import { act, cleanup, fireEvent, render, renderHook, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api/client'
import type { LeagueLiveScore, TeamMatchSummary } from '../../api/types'
import type { LiveMatchState } from '../../api/useLiveMatch'
import i18n from '../../i18n'
import { setPhone } from '../admin/testPhone'
import { LiveMatchHero, useOwnLiveMatch } from './DashboardPage'

vi.mock('../../api/client', async (importOriginal) => {
  const original = await importOriginal<typeof import('../../api/client')>()
  return { ...original, api: { ...original.api, get: vi.fn() } }
})

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
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <LiveMatchHero match={match} live={live} team={team} divisionId={7} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(async () => {
  setPhone(false)
  vi.mocked(api.get).mockReset()
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

describe('LiveMatchHero rest of the matchday', () => {
  const leagueMatch = () => summary({ kind: 'league', matchId: 100 })

  function liveScore(overrides: Partial<LeagueLiveScore> = {}): LeagueLiveScore {
    return {
      matchId: 101,
      weekNumber: 12,
      matchDate: new Date(Date.now() - 30 * 60_000).toISOString(),
      home: 11,
      homeName: 'Alpha FC',
      away: 12,
      awayName: 'Beta FC',
      status: 'live',
      minute: 63,
      homeGoals: 1,
      awayGoals: 2,
      ...overrides,
    }
  }

  // The owner's own match (week 12) plus `others` more matches of that week.
  function matchday(others: number, overrides: Partial<LeagueLiveScore> = {}): LeagueLiveScore[] {
    const own = liveScore({ matchId: 100, home: 1, homeName: 'Mine FC', away: 2, awayName: 'Rivals FC', homeGoals: 9, awayGoals: 9 })
    const rest = Array.from({ length: others }, (_, i) =>
      liveScore({ matchId: 101 + i, homeName: `Home ${i}`, awayName: `Away ${i}`, ...overrides }),
    )
    return [own, ...rest]
  }

  // The hero's own "Watch live" button also links to the own match, so list only the other rows.
  const listedMatchLinks = () =>
    screen
      .getAllByRole('link')
      .map((link) => link.getAttribute('href'))
      .filter((href) => href?.startsWith('/match/') && href !== '/match/100')

  it('lists the other matches of the matchday, not the owner\'s own (League match live)', async () => {
    vi.mocked(api.get).mockResolvedValue([...matchday(2), liveScore({ matchId: 300, weekNumber: 13, homeName: 'Stray', awayName: 'Match' })])
    renderHero(leagueMatch(), liveState({ minute: 63 }))
    await waitFor(() => expect(screen.getByText('Rest of the matchday')).toBeTruthy())
    expect(api.get).toHaveBeenCalledWith('/v1/teams/division/7/live')
    expect(listedMatchLinks()).toEqual(['/match/101', '/match/102'])
    expect(screen.getByText('Home 0')).toBeTruthy()
  })

  it('shows the score as broadcast, never the stored one (Score as broadcast)', async () => {
    vi.mocked(api.get).mockResolvedValue(matchday(1, { homeGoals: 1, awayGoals: 2, minute: 63 }))
    renderHero(leagueMatch(), liveState({ minute: 63 }))
    await waitFor(() => expect(screen.getByText('Home 0')).toBeTruthy())
    const row = screen.getByText('Home 0').closest('a') as HTMLElement
    expect(row.textContent).toContain("63'")
    // Home 0 (1) - Away 0 (2) at 63', never the stored 9 - 9 of the own match or any final score.
    expect(row.textContent).toBe("H0Home 01A0Away 0263'")
  })

  it('makes no live scores request and shows no section for a cup tie (Cup tie live)', () => {
    renderHero(summary({ kind: 'cup' }), liveState({ minute: 20 }))
    expect(api.get).not.toHaveBeenCalled()
    expect(screen.queryByText('Rest of the matchday')).toBeNull()
  })

  it('keeps the own match and hides the section when the endpoint is down (Endpoint down)', async () => {
    vi.mocked(api.get).mockRejectedValue(new Error('boom'))
    renderHero(leagueMatch(), liveState({ homeGoals: 1, awayGoals: 0, minute: 30 }))
    await waitFor(() => expect(api.get).toHaveBeenCalled())
    expect(screen.getByText('1 - 0')).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Watch live' })).toBeTruthy()
    expect(screen.queryByText('Rest of the matchday')).toBeNull()
  })

  it('hides the section when no other match shares the matchday or the own match is missing', async () => {
    vi.mocked(api.get).mockResolvedValue(matchday(0))
    const { unmount } = renderHero(leagueMatch(), liveState({ minute: 30 }))
    await waitFor(() => expect(api.get).toHaveBeenCalled())
    expect(screen.queryByText('Rest of the matchday')).toBeNull()
    unmount()

    vi.mocked(api.get).mockResolvedValue(matchday(3).slice(1))
    renderHero(leagueMatch(), liveState({ minute: 30 }))
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2))
    expect(screen.queryByText('Rest of the matchday')).toBeNull()
  })

  it('starts collapsed on a phone behind the "N matches · G goals" toggle (Phone summary)', async () => {
    setPhone(true)
    vi.mocked(api.get).mockResolvedValue(matchday(7, { homeGoals: 1, awayGoals: 0 }).map((e, i) => (i === 1 ? { ...e, homeGoals: 6, awayGoals: 0 } : e)))
    renderHero(leagueMatch(), liveState({ minute: 30 }))
    const toggle = await screen.findByRole('button', { name: '7 matches · 12 goals' })
    expect(screen.queryByText('Home 0')).toBeNull()
    fireEvent.click(toggle)
    expect(screen.getByText('Home 0')).toBeTruthy()
    expect(listedMatchLinks()).toHaveLength(7)
  })

  it('lists the matches without tapping anything on desktop (Desktop)', async () => {
    vi.mocked(api.get).mockResolvedValue(matchday(7))
    renderHero(leagueMatch(), liveState({ minute: 30 }))
    await waitFor(() => expect(screen.getByText('Home 6')).toBeTruthy())
    expect(listedMatchLinks()).toHaveLength(7)
  })
})
