import { EVENT_TYPE_INJURY, EVENT_TYPE_RED_CARD, type MatchEvent } from '../api/types'

// A red card or injury changes what a team can do for the rest of the match (a man down, an
// emergency substitution) - information a live viewer shouldn't get before the match itself is
// over, the same way its final score/ratings aren't shown early either (see
// backend/engine/livematch.go's onComplete comment for the equivalent backend-side rule on match
// result notifications). Everything else broadcast live (goals, yellow cards, substitutions,
// missed chances) is unaffected.
const WITHHELD_WHILE_LIVE = new Set<number>([EVENT_TYPE_INJURY, EVENT_TYPE_RED_CARD])

/** Drops injury/red-card events from a still-broadcasting match's event list - callers pass this
 * to MatchEventsReport (and anywhere else live events are surfaced) instead of `live.events`
 * directly. Once the match completes, the page switches over to the regular finished-match view,
 * which reads events straight from the backend and isn't filtered. */
export function withholdSpoilerEvents(events: MatchEvent[]): MatchEvent[] {
  return events.filter((event) => !WITHHELD_WHILE_LIVE.has(event.eventType))
}
