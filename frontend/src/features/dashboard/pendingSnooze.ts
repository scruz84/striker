// "Remind me tomorrow" for the dashboard's Pending panel: the snooze lives in this browser's
// localStorage (nothing server-side), hides an item until the next local midnight and lapses early
// when the item's content changes (a second offer arrives). Every storage access is wrapped: a
// private window or blocked site data just means nothing is remembered.

export const SNOOZE_STORAGE_KEY = 'striker.pending.snoozed'

export type SnoozeState = Record<string, { contentKey: string; until: number }>

export function readSnoozed(): SnoozeState {
  try {
    const raw = window.localStorage.getItem(SNOOZE_STORAGE_KEY)
    const parsed: unknown = raw ? JSON.parse(raw) : {}
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as SnoozeState) : {}
  } catch {
    return {}
  }
}

export function writeSnoozed(state: SnoozeState): void {
  try {
    window.localStorage.setItem(SNOOZE_STORAGE_KEY, JSON.stringify(state))
  } catch {
    // not remembered: the item simply comes back on the next visit
  }
}

/** The next local midnight after `now` (epoch milliseconds). */
export function nextLocalMidnight(now: number): number {
  const date = new Date(now)
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1).getTime()
}

/** True while `item` is snoozed: stored for the same content and the snooze has not run out. */
export function isSnoozed(state: SnoozeState, item: { id: string; contentKey: string }, now: number): boolean {
  const entry = state[item.id]
  return !!entry && entry.contentKey === item.contentKey && now < entry.until
}

/** `state` with `item` snoozed until the next local midnight; expired entries are dropped. */
export function snoozeItem(state: SnoozeState, item: { id: string; contentKey: string }, now: number): SnoozeState {
  const live = Object.fromEntries(Object.entries(state).filter(([, entry]) => now < entry.until))
  return { ...live, [item.id]: { contentKey: item.contentKey, until: nextLocalMidnight(now) } }
}

/** The clock, wrapped so components can read it without calling the impure Date.now() themselves. */
export function currentTime(): number {
  return Date.now()
}
