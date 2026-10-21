import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

// Cross-Origin-Opener-Policy: same-origin-allow-popups lets the Google sign-in popup (see
// LoginPage.tsx) keep a reference to this window so it can postMessage the credential back on
// success - without it, some Chrome versions isolate the popup into a separate browsing context
// group and silently drop that postMessage (logged as "Cross-Origin-Opener-Policy policy would
// block the window.postMessage call"), breaking login. Applies to both `vite dev` and
// `vite preview`; nginx.conf sets the same header for the production/Docker build.
const coopHeaders = { 'Cross-Origin-Opener-Policy': 'same-origin-allow-popups' }

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: { headers: coopHeaders },
  preview: { headers: coopHeaders },
  test: {
    // e2e/ holds the Playwright smoke test (npm run smoke:mobile), which vitest can't run.
    exclude: ['**/node_modules/**', '**/dist/**', 'e2e/**'],
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.ts'],
    globals: true,
  },
})
