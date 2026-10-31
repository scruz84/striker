declare global {
  interface Window {
    extended?: {
      API_ENDPOINT?: string
      GOOGLE_OAUTH_CLIENT_ID?: string
      VAPID_PUBLIC_KEY?: string
      LEGAL_CONTROLLER_NAME?: string
      LEGAL_CONTACT_EMAIL?: string
      LEGAL_CONTROLLER_TAX_ID?: string
      LEGAL_POSTAL_ADDRESS?: string
      LEGAL_HOSTING_PROVIDER?: string
      LEGAL_HOSTING_REGION?: string
      LEGAL_NOTICE_ENABLED?: string
    }
  }
}

// Vite dev server / `npm run build` uses import.meta.env (see .env.development / .env.production.local).
// A container build instead bakes a placeholder into public/config.js, which set-env.sh rewrites at
// container start time from real environment variables - see Dockerfile/nginx.conf/set-env.sh.
function isDevelopment(): boolean {
  return import.meta.env.DEV
}

type RuntimeKey = keyof NonNullable<Window['extended']>

function runtimeValue(key: RuntimeKey, envKey: string): string {
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

// Who runs this deployment, shown in the Privacy Policy, Terms and legal notice. Deployment
// properties (LEGAL_* in the container's environment, or VITE_LEGAL_* for `npm run dev`), never
// code, so one image serves any operator. Unset values are empty strings.
export interface LegalConfig {
  controllerName: string
  contactEmail: string
  taxId: string
  postalAddress: string
  hostingProvider: string
  hostingRegion: string
  /** LEGAL_NOTICE_ENABLED=true turns on the public legal notice page and its links. */
  noticeEnabled: boolean
}

export function getLegalConfig(): LegalConfig {
  return {
    controllerName: runtimeValue('LEGAL_CONTROLLER_NAME', 'VITE_LEGAL_CONTROLLER_NAME'),
    contactEmail: runtimeValue('LEGAL_CONTACT_EMAIL', 'VITE_LEGAL_CONTACT_EMAIL'),
    taxId: runtimeValue('LEGAL_CONTROLLER_TAX_ID', 'VITE_LEGAL_CONTROLLER_TAX_ID'),
    postalAddress: runtimeValue('LEGAL_POSTAL_ADDRESS', 'VITE_LEGAL_POSTAL_ADDRESS'),
    hostingProvider: runtimeValue('LEGAL_HOSTING_PROVIDER', 'VITE_LEGAL_HOSTING_PROVIDER'),
    hostingRegion: runtimeValue('LEGAL_HOSTING_REGION', 'VITE_LEGAL_HOSTING_REGION'),
    noticeEnabled: runtimeValue('LEGAL_NOTICE_ENABLED', 'VITE_LEGAL_NOTICE_ENABLED').trim().toLowerCase() === 'true',
  }
}
