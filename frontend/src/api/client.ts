import { getApiEndpoint } from '../config'

export class ApiError extends Error {
  status: number
  // The response's JSON body, when it had one (e.g. a 409's details), otherwise undefined.
  body?: unknown

  constructor(status: number, message: string, body?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

// Sent on every request. The backend rejects state-changing requests without it (CSRF
// protection): a cross-site page cannot add a custom header without a CORS preflight, which the
// origin allow-list refuses.
export const CLIENT_HEADER = 'X-Striker-Client'
export const CLIENT_HEADER_VALUE = 'web'

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(getApiEndpoint(path), {
    credentials: 'include',
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
      [CLIENT_HEADER]: CLIENT_HEADER_VALUE,
    },
  })

  if (!response.ok) {
    let body: unknown
    try {
      const text = await response.text()
      body = text ? JSON.parse(text) : undefined
    } catch {
      body = undefined
    }
    throw new ApiError(response.status, `${options.method ?? 'GET'} ${path} failed with status ${response.status}`, body)
  }

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  return (text ? JSON.parse(text) : undefined) as T
}

export const api = {
  get: <T>(path: string): Promise<T> => request<T>(path),
  post: <T>(path: string, body?: unknown): Promise<T> =>
    request<T>(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown): Promise<T> =>
    request<T>(path, { method: 'PUT', body: body !== undefined ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string, body?: unknown): Promise<T> =>
    request<T>(path, { method: 'DELETE', body: body !== undefined ? JSON.stringify(body) : undefined }),
}

export function isUnauthorized(error: unknown): boolean {
  return error instanceof ApiError && (error.status === 401 || error.status === 403)
}
