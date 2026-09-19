// Browser-side half of Web Push: turning the VAPID public key (see config.ts) into a
// PushManager subscription and shaping it into the payload the backend expects (see
// handlers.SubscribeToPushHandler). See public/sw.js for the service worker side that actually
// shows a notification once the push service delivers one, and api/hooks.ts for the mutations
// that send a subscription/unsubscription to the backend.
import { getVapidPublicKey } from './config'

export interface PushSubscriptionPayload {
  endpoint: string
  keys: { p256dh: string; auth: string }
}

// Push notifications need a service worker, PushManager support, and a VAPID key configured on
// this deployment (see common.ConfigureWebPushNotifier on the backend) - any of those missing
// means there's nothing to subscribe to.
export function isPushSupported(): boolean {
  return 'serviceWorker' in navigator && 'PushManager' in window && !!getVapidPublicKey()
}

// PushManager.subscribe wants its applicationServerKey as a Uint8Array, but VAPID keys are handed
// out base64url-encoded - see
// https://developer.mozilla.org/en-US/docs/Web/API/PushManager/subscribe.
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = window.atob(base64)
  return Uint8Array.from([...rawData].map((char) => char.charCodeAt(0)))
}

// Requests notification permission, then subscribes this browser to push via the service worker
// registered by registerServiceWorker. Throws if permission is denied or the browser hands back
// an incomplete subscription - the caller (see api/hooks.ts useSubscribeToPush) leaves the
// preference off in that case rather than pretending it succeeded.
export async function subscribeToPush(): Promise<PushSubscriptionPayload> {
  const permission = await Notification.requestPermission()
  if (permission !== 'granted') {
    throw new Error('notification permission not granted')
  }

  const registration = await navigator.serviceWorker.ready
  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    // lib.dom's PushSubscriptionOptionsInit wants a BufferSource backed by ArrayBuffer
    // specifically, but Uint8Array's type param widened to ArrayBufferLike (which also covers
    // SharedArrayBuffer) in newer TS - a plain Uint8Array from Uint8Array.from is always
    // ArrayBuffer-backed in practice, so this cast is safe.
    applicationServerKey: urlBase64ToUint8Array(getVapidPublicKey()) as BufferSource,
  })

  const json = subscription.toJSON()
  if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) {
    throw new Error('browser returned an incomplete push subscription')
  }
  return { endpoint: json.endpoint, keys: { p256dh: json.keys.p256dh, auth: json.keys.auth } }
}

// Unsubscribes this browser from push, returning the endpoint that was dropped (for the caller to
// tell the backend about - see api/hooks.ts useUnsubscribeFromPush) or undefined if it was never
// subscribed here in the first place.
export async function unsubscribeFromPush(): Promise<string | undefined> {
  const registration = await navigator.serviceWorker.ready
  const subscription = await registration.pushManager.getSubscription()
  if (!subscription) {
    return undefined
  }
  const endpoint = subscription.endpoint
  await subscription.unsubscribe()
  return endpoint
}
