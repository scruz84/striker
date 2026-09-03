import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError, isUnauthorized } from './client'

describe('api client', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = vi.fn()
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('parses a successful JSON response', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(
      new Response(JSON.stringify({ id: 1 }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )

    const result = await api.get<{ id: number }>('/v1/thing')
    expect(result).toEqual({ id: 1 })
  })

  it('sends credentials and a JSON body on POST', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(JSON.stringify({ ok: true }), { status: 200 }))

    await api.post('/v1/login', { jwt: 'token', provider: 'google' })

    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/v1/login'),
      expect.objectContaining({
        method: 'POST',
        credentials: 'include',
        body: JSON.stringify({ jwt: 'token', provider: 'google' }),
      }),
    )
  })

  it('throws an ApiError with the response status on a non-ok response', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 403 }))

    await expect(api.get('/v1/auth/me')).rejects.toBeInstanceOf(ApiError)
  })

  it('isUnauthorized recognizes 401 and 403 but not other errors', () => {
    expect(isUnauthorized(new ApiError(401, 'nope'))).toBe(true)
    expect(isUnauthorized(new ApiError(403, 'nope'))).toBe(true)
    expect(isUnauthorized(new ApiError(500, 'boom'))).toBe(false)
    expect(isUnauthorized(new Error('unrelated'))).toBe(false)
  })
})
