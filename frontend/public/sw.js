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
