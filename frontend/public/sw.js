// Minimal service worker, registered only so the app satisfies "installable" PWA criteria
// (see src/registerServiceWorker.ts) across the browsers that still check for one (Chrome's own
// install-banner heuristic dropped this requirement, but others haven't). Deliberately does no
// caching: this game's data changes every minute (live matches, the transfer market,
// notifications), so an offline-first cache would risk serving stale content instead of actually
// helping - every request is passed straight through to the network unmodified.
self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
})

self.addEventListener('fetch', (event) => {
  event.respondWith(fetch(event.request))
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
