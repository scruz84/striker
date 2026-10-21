import { defineConfig } from '@playwright/test'

// Mobile smoke test (see e2e/mobile-smoke.spec.ts). It drives the real app against a running
// backend with a seeded world, so it is NOT part of `npm test` (vitest) nor of CI - run it by hand
// with `npm run smoke:mobile` once the backend is up (backend/app.env sets DEV_LOGIN_ENABLED=true).
//
// SMOKE_API_URL         backend the app talks to (default http://localhost:1984)
// SMOKE_BASE_URL        where the app is served (default http://localhost:5173; started with `vite` when
//                       nothing is listening there). Keep both on `localhost`: the backend's CORS only
//                       allows http://localhost:5173 (and :3000) by default, and its session cookie is
//                       Secure, which Chrome only accepts over plain http on localhost.
// PLAYWRIGHT_CHROMIUM_EXECUTABLE  use this Chromium instead of the one `npx playwright install` fetches.
const apiUrl = process.env.SMOKE_API_URL ?? 'http://localhost:1984'
const baseURL = process.env.SMOKE_BASE_URL ?? 'http://localhost:5173'
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
    command: 'npm run dev -- --host localhost --port 5173 --strictPort',
    url: baseURL,
    reuseExistingServer: true,
    env: { VITE_API_ENDPOINT: apiUrl, VITE_GOOGLE_OAUTH_CLIENT_ID: 'smoke' },
  },
  metadata: { apiUrl },
})
