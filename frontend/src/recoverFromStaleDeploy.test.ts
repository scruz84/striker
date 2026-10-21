import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { recoverFromStaleDeploy } from './recoverFromStaleDeploy'

describe('recoverFromStaleDeploy', () => {
  const reload = vi.fn()
  const originalLocation = window.location

  beforeEach(() => {
    sessionStorage.clear()
    reload.mockClear()
    Object.defineProperty(window, 'location', { configurable: true, value: { ...originalLocation, reload } })
    vi.useFakeTimers()
    recoverFromStaleDeploy()
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
    vi.useRealTimers()
  })

  const chunkFails = () => window.dispatchEvent(new Event('vite:preloadError', { cancelable: true }))

  it('reloads to pick up the new build when a page chunk fails to load', () => {
    chunkFails()
    expect(reload).toHaveBeenCalled()
  })

  it('reloads at most once a minute, so a chunk that keeps failing cannot loop', () => {
    chunkFails()
    const reloadsAfterFirst = reload.mock.calls.length
    vi.advanceTimersByTime(30_000)
    chunkFails()
    expect(reload.mock.calls.length).toBe(reloadsAfterFirst)
    vi.advanceTimersByTime(31_000)
    chunkFails()
    expect(reload.mock.calls.length).toBeGreaterThan(reloadsAfterFirst)
  })
})
