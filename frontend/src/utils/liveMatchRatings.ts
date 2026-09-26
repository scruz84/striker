import { EVENT_TYPE_GOAL, EVENT_TYPE_INJURY, EVENT_TYPE_RED_CARD, EVENT_TYPE_YELLOW_CARD, type Lineup, type MatchEvent } from '../api/types'

// Mirrors backend common.ComputeMatchRatings' base/per-event deltas and result bonus closely
// enough for a live view, but driven off however much of the match has broadcast so far
// (`events`/`homeGoals`/`awayGoals`) rather than the final whistle - so unlike the backend
// version, this is expected to keep moving minute to minute, and to land close to but not
// necessarily identical to the final computed rating (which also adds a small deterministic
// per-player jitter this live approximation skips, since there's no stable "final state" yet to
// jitter around).
const RATING_BASE = 6.0
const RATING_MIN = 1.0
const RATING_MAX = 10.0
const RATING_RESULT_BONUS = 0.3

const DELTA_BY_EVENT_TYPE: Partial<Record<number, number>> = {
  [EVENT_TYPE_GOAL]: 1.4,
  [EVENT_TYPE_YELLOW_CARD]: -0.4,
  [EVENT_TYPE_RED_CARD]: -1.5,
  [EVENT_TYPE_INJURY]: -0.2,
}

function clampRating(rating: number): number {
  return Math.round(Math.min(RATING_MAX, Math.max(RATING_MIN, rating)) * 10) / 10
}

/** Computes a 1-10 live rating for every player named in either lineup's positions, the same way
 * MatchRatingsTab shows for a finished match's report but reactive to a live broadcast's
 * `events`/current score as they arrive - see this file's own comment for how it differs from
 * the backend's final computation. Ratings are keyed by playerId as a string, matching
 * WeekMatch.ratings' own shape so both can feed the same rendering (TeamRatingsColumn). */
export function computeLiveMatchRatings(
  homeLineup: Lineup,
  awayLineup: Lineup,
  events: MatchEvent[],
  homeGoals: number,
  awayGoals: number,
): Record<string, number> {
  const ratings: Record<string, number> = {}
  const homePlayerIds = new Set<number>()
  for (const pos of homeLineup.positions) {
    homePlayerIds.add(pos.playerId)
    ratings[String(pos.playerId)] = RATING_BASE
  }
  for (const pos of awayLineup.positions) {
    ratings[String(pos.playerId)] = RATING_BASE
  }

  for (const event of events) {
    const delta = DELTA_BY_EVENT_TYPE[event.eventType]
    if (delta === undefined) {
      continue
    }
    for (const playerId of event.teamOwnerPlayers ?? []) {
      const key = String(playerId)
      if (key in ratings) {
        ratings[key] += delta
      }
    }
  }

  const homeAhead = homeGoals > awayGoals
  const awayAhead = awayGoals > homeGoals
  for (const key of Object.keys(ratings)) {
    const isHome = homePlayerIds.has(Number(key))
    if ((isHome && homeAhead) || (!isHome && awayAhead)) {
      ratings[key] += RATING_RESULT_BONUS
    } else if ((isHome && awayAhead) || (!isHome && homeAhead)) {
      ratings[key] -= RATING_RESULT_BONUS
    }
    ratings[key] = clampRating(ratings[key])
  }

  return ratings
}
