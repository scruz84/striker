// Minimal service worker: it exists for Web Push (below) and so the app counts as an installable
// PWA in the browsers that still look for one. Deliberately has no `fetch` handler: this game's
// data changes every minute (live matches, the transfer market, notifications), so an offline
// cache would only risk stale content, and a pass-through handler made every request - page loads
// and API calls alike - wait for this worker to start first (slow on phones, where it is often
// stopped) and turned any network hiccup into a failed page load. Without one the browser skips
// the worker entirely for network requests.
self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
})

// A Web Push message from the backend (see common.WebPushNotifier / common.PushPayload) - shown
// even while the app itself isn't open, since that's the entire point of push over an in-app poll.
self.addEventListener('push', (event) => {
  let payload = { title: 'Striker', body: '' }
  try {
    payload = event.data.json()
  } catch {
    // Malformed/absent payload - fall back to the defaults above rather than dropping the
    // notification silently.
  }
  event.waitUntil(self.registration.showNotification(payload.title, { body: payload.body }))
})

// Focuses an already-open app tab if there is one, otherwise opens a new one - clicking a
// notification should never leave the user staring at whatever tab happened to have focus.
self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ('focus' in client) {
          return client.focus()
        }
      }
      if (self.clients.openWindow) {
        return self.clients.openWindow('/')
      }
    }),
  )
})
