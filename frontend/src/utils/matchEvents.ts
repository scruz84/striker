import { EVENT_TYPE_NO_GOAL, SET_PIECE_PENALTY, type MatchEvent } from '../api/types'

// A missed penalty is a key moment, so the report shows it even when the viewer hasn't asked for
// the full commentary. Other missed chances (including missed free kicks) stay hidden then.
export function isMissedPenalty(event: MatchEvent): boolean {
  return event.eventType === EVENT_TYPE_NO_GOAL && event.setPiece === SET_PIECE_PENALTY
}

// Whether a missed chance is part of the full commentary only, rather than a key event.
export function isMinorMissedChance(event: MatchEvent): boolean {
  return event.eventType === EVENT_TYPE_NO_GOAL && !isMissedPenalty(event)
}
