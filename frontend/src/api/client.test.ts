import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError, CLIENT_HEADER, CLIENT_HEADER_VALUE, isUnauthorized } from './client'

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

  it.each([
    ['get', () => api.get('/v1/thing')],
    ['post', () => api.post('/v1/thing', { a: 1 })],
    ['put', () => api.put('/v1/thing', { a: 1 })],
    ['delete', () => api.delete('/v1/thing')],
  ])('sends the CSRF client header on %s', async (_name, call) => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response('{}', { status: 200 }))

    await call()

    const init = vi.mocked(globalThis.fetch).mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>)[CLIENT_HEADER]).toBe(CLIENT_HEADER_VALUE)
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
  })

  it('keeps the CSRF header even when the caller passes its own headers', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response('{}', { status: 200 }))

    await api.post('/v1/thing')

    const init = vi.mocked(globalThis.fetch).mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>)[CLIENT_HEADER]).toBe('web')
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
