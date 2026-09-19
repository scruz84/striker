// Registers public/sw.js - a deliberately no-op service worker (see its own comment) that exists
// only to satisfy the "installable" PWA checks some browsers still run before showing an
// install/add-to-home-screen prompt. Guarded by feature detection since older browsers (and any
// non-browser test environment) simply don't have `navigator.serviceWorker` at all, and skipped
// outside production - registering it in dev would have it intercepting Vite's own dev-server
// requests.
export function registerServiceWorker(): void {
  if (import.meta.env.DEV || !('serviceWorker' in navigator)) {
    return
  }
  window.addEventListener('load', () => {
    void navigator.serviceWorker.register('/sw.js')
  })
}
