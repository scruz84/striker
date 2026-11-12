import type { MatchEvent, MatchInstant } from '../api/types'

// A match minute is its clock minute plus an added minute: 1..N inside a half's added time
// (minute then reads 45 or 90), 0 in regulation - see backend openspec/specs/stoppage-time. Every
// event and instant stored before added time existed has no addedMinute, which reads as 0.
export interface MatchMinute {
  minute: number
  addedMinute?: number
}

// matchMinuteText is a minute without the trailing apostrophe - "63" or "90+3" - for the i18n
// strings that add their own (dashboard.liveMatchMinute and friends read "{{minute}}'").
export function matchMinuteText(minute: number, addedMinute?: number): string {
  return addedMinute ? `${minute}+${addedMinute}` : `${minute}`
}

// formatMatchMinute is the label every minute the viewer shows uses: "63'" in regulation, "45+2'"
// or "90+3'" in added time.
export function formatMatchMinute(minute: number, addedMinute?: number): string {
  return addedMinute ? `${minute}+${addedMinute}'` : `${minute}'`
}

// compareMatchMinute orders two minutes in match order: 45 < 45+1 < 45+2 < 46.
export function compareMatchMinute(a: MatchMinute, b: MatchMinute): number {
  if (a.minute !== b.minute) return a.minute - b.minute
  return (a.addedMinute ?? 0) - (b.addedMinute ?? 0)
}

// isAtOrBefore reports whether minute a comes at or before minute b in match order.
export function isAtOrBefore(a: MatchMinute, b: MatchMinute): boolean {
  return compareMatchMinute(a, b) <= 0
}

// matchTimeline is the ordered list of minutes a minute scrubber steps through: one step per
// simulated minute, added minutes included (1..45, 45+1..45+N, 46..90, 90+1..90+N) - the match's
// instants, so a step always has a zone snapshot. A finished match with no stored instants (one
// played before they were stored) steps through 1..90, or to its last event's minute if later.
export function matchTimeline(instants: MatchInstant[], events: MatchEvent[] = []): MatchMinute[] {
  if (instants.length > 0) {
    return instants.map((instant) => ({ minute: instant.minute, addedMinute: instant.addedMinute ?? 0 }))
  }
  if (events.length === 0) {
    return []
  }
  const last = events.reduce((max, event) => Math.max(max, event.minute), 90)
  return Array.from({ length: last }, (_, index) => ({ minute: index + 1, addedMinute: 0 }))
}

// timelineMinute is the minute at a scrubber step: step 0 is before kickoff (minute 0), step n the
// n-th minute of the timeline, clamped to its last one.
export function timelineMinute(timeline: MatchMinute[], step: number): MatchMinute {
  if (step <= 0 || timeline.length === 0) {
    return { minute: 0, addedMinute: 0 }
  }
  return timeline[Math.min(step, timeline.length) - 1]
}
