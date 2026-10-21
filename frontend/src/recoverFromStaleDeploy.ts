// Pages are loaded on demand (see the routes' `lazy` in app/App.tsx) from content-hashed chunks, and
// a deploy replaces every chunk. A tab or installed app still running the previous build then asks
// for chunks that no longer exist, and the page it navigates to never loads. Vite reports that as a
// `vite:preloadError` event; reloading picks up the new build. Only once per minute, so a chunk
// that is genuinely broken (or a server that is down) can't put the app in a reload loop.
const LAST_RELOAD_KEY = 'striker.staleChunkReloadAt'
const MIN_RELOAD_INTERVAL_MS = 60_000

export function recoverFromStaleDeploy(): void {
  window.addEventListener('vite:preloadError', () => {
    let lastReload = 0
    try {
      lastReload = Number(sessionStorage.getItem(LAST_RELOAD_KEY)) || 0
      if (Date.now() - lastReload < MIN_RELOAD_INTERVAL_MS) {
        return
      }
      sessionStorage.setItem(LAST_RELOAD_KEY, String(Date.now()))
    } catch {
      // No sessionStorage (blocked by the browser): reload anyway, at worst once per failure.
    }
    window.location.reload()
  })
}
