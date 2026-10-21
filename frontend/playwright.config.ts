import { defineConfig } from '@playwright/test'

// Mobile smoke test (see e2e/mobile-smoke.spec.ts). It drives the real app against a running
// backend with a seeded world, so it is NOT part of `npm test` (vitest) nor of CI - run it by hand
// with `npm run smoke:mobile` once the backend is up (backend/app.env sets DEV_LOGIN_ENABLED=true).
//
// SMOKE_API_URL         backend the app talks to (default http://127.0.0.1:1984)
// SMOKE_BASE_URL        where the app is served (default http://127.0.0.1:5173; started with `vite` when
//                       nothing is listening there). Keep both on the same host name - the session
//                       cookie is host-bound, and the backend only allows its known dev origins.
// PLAYWRIGHT_CHROMIUM_EXECUTABLE  use this Chromium instead of the one `npx playwright install` fetches.
const apiUrl = process.env.SMOKE_API_URL ?? 'http://127.0.0.1:1984'
const baseURL = process.env.SMOKE_BASE_URL ?? 'http://127.0.0.1:5173'
const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE

export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL,
    // 360x740 is the smallest phone the epic (#268) targets; touch and DPR 2 like a real device.
    viewport: { width: 360, height: 740 },
    deviceScaleFactor: 2,
    hasTouch: true,
    isMobile: true,
    launchOptions: executablePath ? { executablePath } : {},
  },
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1 --port 5173 --strictPort',
    url: baseURL,
    reuseExistingServer: true,
    env: { VITE_API_ENDPOINT: apiUrl, VITE_GOOGLE_OAUTH_CLIENT_ID: 'smoke' },
  },
  metadata: { apiUrl },
})
