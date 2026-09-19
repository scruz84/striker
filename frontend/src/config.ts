declare global {
  interface Window {
    extended?: {
      API_ENDPOINT?: string
      GOOGLE_OAUTH_CLIENT_ID?: string
      VAPID_PUBLIC_KEY?: string
    }
  }
}

// Vite dev server / `npm run build` uses import.meta.env (see .env.development / .env.production.local).
// A container build instead bakes a placeholder into public/config.js, which set-env.sh rewrites at
// container start time from real environment variables - see Dockerfile/nginx.conf/set-env.sh.
function isDevelopment(): boolean {
  return import.meta.env.DEV
}

function runtimeValue(key: 'API_ENDPOINT' | 'GOOGLE_OAUTH_CLIENT_ID' | 'VAPID_PUBLIC_KEY', envKey: string): string {
  if (isDevelopment()) {
    return import.meta.env[envKey] ?? ''
  }
  const value = window.extended?.[key]
  if (!value || value.startsWith('REPLACE_')) {
    return ''
  }
  return value
}

export function getApiBaseUrl(): string {
  return runtimeValue('API_ENDPOINT', 'VITE_API_ENDPOINT')
}

export function getApiEndpoint(path: string): string {
  return getApiBaseUrl() + path
}

// Live match streaming (see api/useLiveMatch.ts) connects directly to the API's WebSocket
// endpoints, so the http(s) base URL needs the equivalent ws(s) one.
export function getWsEndpoint(path: string): string {
  const base = getApiBaseUrl()
  return base.replace(/^http/, 'ws') + path
}

export function getGoogleOAuthClientId(): string {
  return runtimeValue('GOOGLE_OAUTH_CLIENT_ID', 'VITE_GOOGLE_OAUTH_CLIENT_ID')
}

// Empty means no key is configured (see common.ConfigureWebPushNotifier on the backend), in which
// case push notifications are simply unavailable - see src/pushNotifications.ts.
export function getVapidPublicKey(): string {
  return runtimeValue('VAPID_PUBLIC_KEY', 'VITE_VAPID_PUBLIC_KEY')
}
