import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { api } from './client'
import { divisionLiveScoresRefetchInterval, queryKeys, useDivisionLiveScores } from './hooks'
import type { LeagueLiveScore } from './types'

vi.mock('./client', async (importOriginal) => {
  const original = await importOriginal<typeof import('./client')>()
  return { ...original, api: { ...original.api, get: vi.fn() } }
})

function entry(overrides: Partial<LeagueLiveScore> = {}): LeagueLiveScore {
  return {
    matchId: 1,
    weekNumber: 12,
    matchDate: new Date().toISOString(),
    home: 1,
    homeName: 'Alpha',
    away: 2,
    awayName: 'Beta',
    status: 'live',
    minute: 63,
    homeGoals: 1,
    awayGoals: 0,
    ...overrides,
  }
}

describe('divisionLiveScoresRefetchInterval', () => {
  it('polls every 15 seconds while a match is not over', () => {
    expect(divisionLiveScoresRefetchInterval([entry(), entry({ status: 'finished' })])).toBe(15_000)
    expect(divisionLiveScoresRefetchInterval([entry({ status: 'pending' })])).toBe(15_000)
  })

  it('stops once every entry is finished (Matchday ends)', () => {
    expect(divisionLiveScoresRefetchInterval([entry({ status: 'finished' }), entry({ status: 'finished' })])).toBe(false)
  })

  it('stops on an empty list and before the first response', () => {
    expect(divisionLiveScoresRefetchInterval([])).toBe(false)
    expect(divisionLiveScoresRefetchInterval(undefined)).toBe(false)
  })
})

describe('useDivisionLiveScores', () => {
  const wrapperFor = (queryClient: QueryClient) =>
    function Wrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    }

  it('loads the division live scores', async () => {
    vi.mocked(api.get).mockResolvedValue([entry()])
    const queryClient = new QueryClient()
    const { result } = renderHook(() => useDivisionLiveScores(7, true), { wrapper: wrapperFor(queryClient) })
    await waitFor(() => expect(result.current.data).toHaveLength(1))
    expect(api.get).toHaveBeenCalledWith('/v1/teams/division/7/live')
    expect(queryClient.getQueryData(queryKeys.divisionLiveScores(7))).toHaveLength(1)
  })

  it('makes no request while disabled', async () => {
    vi.mocked(api.get).mockClear()
    const { result } = renderHook(() => useDivisionLiveScores(7, false), { wrapper: wrapperFor(new QueryClient()) })
    expect(result.current.fetchStatus).toBe('idle')
    expect(api.get).not.toHaveBeenCalled()
  })
})
