import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_YELLOW_CARD,
  type MatchEvent,
} from '../api/types'

const COMMENTARY_KEYS: Partial<Record<number, string>> = {
  [EVENT_TYPE_GOAL]: 'match.commentary.goal',
  [EVENT_TYPE_NO_GOAL]: 'match.commentary.noGoal',
  [EVENT_TYPE_YELLOW_CARD]: 'match.commentary.yellowCard',
  [EVENT_TYPE_RED_CARD]: 'match.commentary.redCard',
  [EVENT_TYPE_INJURY]: 'match.commentary.injury',
  [EVENT_TYPE_SUBSTITUTION]: 'match.commentary.substitution',
}

// Picks a stable index into a list of phrasing variants from a few small integers - the same
// event (same minute/type/player) always narrates the same way across re-renders, but different
// events land on different variants, without needing to store which one was picked anywhere.
function pickVariantIndex(count: number, seedParts: number[]): number {
  let hash = 0
  for (const part of seedParts) {
    hash = (hash * 31 + part) % 1000003
  }
  return Math.abs(hash) % count
}

function interpolate(template: string, vars: Record<string, string>): string {
  return template.replace(/\{\{(\w+)\}\}/g, (_, key: string) => vars[key] ?? '')
}

type Translate = (key: string, options: { returnObjects: true }) => unknown

/** Turns one match event into a single radio-commentator-style line, picking one of several
 * i18n-translated phrasings for its event type (see match.commentary.* in the locale files).
 * Returns undefined for event types with no commentary (or missing player data to fill one in) -
 * callers should fall back to their own rendering in that case. */
export function commentaryFor(event: MatchEvent, t: Translate, teamName: string, opponentName: string): string | undefined {
  const key = COMMENTARY_KEYS[event.eventType]
  if (!key) {
    return undefined
  }

  const variants = t(key, { returnObjects: true })
  if (!Array.isArray(variants) || variants.length === 0) {
    return undefined
  }

  const playerIds = event.teamOwnerPlayers ?? []
  const playerNames = event.teamOwnerPlayerNames ?? []
  const template = variants[pickVariantIndex(variants.length, [event.minute, event.eventType, playerIds[0] ?? 0])] as string

  if (event.eventType === EVENT_TYPE_SUBSTITUTION) {
    if (playerNames.length < 2) {
      return undefined
    }
    return interpolate(template, { team: teamName, opponent: opponentName, outPlayer: playerNames[0], inPlayer: playerNames[1] })
  }

  const player = playerNames[0]
  if (!player) {
    return undefined
  }
  return interpolate(template, { player, team: teamName, opponent: opponentName })
}
