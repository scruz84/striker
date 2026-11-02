import { act, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { queryKeys, useSetCupTieLineup, useSetFriendlyLineup, useSetLineup, useSetMatchLineup } from './hooks'

vi.mock('./client', async (importOriginal) => {
  const original = await importOriginal<typeof import('./client')>()
  return { ...original, api: { ...original.api, put: vi.fn().mockResolvedValue({}) } }
})

describe('saving a lineup refreshes the opponent previews', () => {
  const body = { formation: 0, positions: [], substitutions: [] }
  const cases: Array<[string, () => (() => { mutateAsync: (v: typeof body) => Promise<unknown> })]> = [
    ['default lineup', () => () => useSetLineup()],
    ['league match lineup', () => () => useSetMatchLineup('5')],
    ['cup tie lineup', () => () => useSetCupTieLineup('6')],
    ['friendly lineup', () => () => useSetFriendlyLineup('7')],
  ]

  it.each(cases)('%s', async (_name, makeHook) => {
    const queryClient = new QueryClient()
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries')
    const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    const { result } = renderHook(makeHook(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync(body)
    })

    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.opponentPreviews })
  })
})
