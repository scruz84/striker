import type { ReactNode } from 'react'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_MATCH_ABANDONED,
  EVENT_TYPE_MATCH_DEFAULT_RESULT,
  EVENT_TYPE_MENTALITY_CHANGE,
  EVENT_TYPE_NO_GOAL,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_SUBSTITUTION,
  EVENT_TYPE_YELLOW_CARD,
  SET_PIECE_DIRECT_FREE_KICK,
  SET_PIECE_PENALTY,
  MENTALITY_KEYS,
  type MatchEvent,
} from '../api/types'
import { PlayerNameLink } from '../components/PlayerNameLink'
import { TeamNameLink } from '../components/TeamNameLink'

const COMMENTARY_KEYS: Partial<Record<number, string>> = {
  [EVENT_TYPE_GOAL]: 'match.commentary.goal',
  [EVENT_TYPE_NO_GOAL]: 'match.commentary.noGoal',
  [EVENT_TYPE_YELLOW_CARD]: 'match.commentary.yellowCard',
  [EVENT_TYPE_RED_CARD]: 'match.commentary.redCard',
  [EVENT_TYPE_INJURY]: 'match.commentary.injury',
  [EVENT_TYPE_SUBSTITUTION]: 'match.commentary.substitution',
  [EVENT_TYPE_MATCH_ABANDONED]: 'match.commentary.abandoned',
  [EVENT_TYPE_MATCH_DEFAULT_RESULT]: 'match.commentary.defaultResult',
}

// A penalty or direct free kick narrates with its own phrasing; a missing setPiece (old or cached
// payload) falls through to the open-play key.
const SET_PIECE_COMMENTARY_KEYS: Partial<Record<number, Partial<Record<number, string>>>> = {
  [EVENT_TYPE_GOAL]: {
    [SET_PIECE_PENALTY]: 'match.commentary.penaltyGoal',
    [SET_PIECE_DIRECT_FREE_KICK]: 'match.commentary.freeKickGoal',
  },
  [EVENT_TYPE_NO_GOAL]: {
    [SET_PIECE_PENALTY]: 'match.commentary.penaltyMiss',
    [SET_PIECE_DIRECT_FREE_KICK]: 'match.commentary.freeKickMiss',
  },
}

function commentaryKey(event: MatchEvent): string | undefined {
  // A mentality change narrates the side's new approach (match.commentary.mentalityChange.*).
  if (event.eventType === EVENT_TYPE_MENTALITY_CHANGE) {
    const mentality = event.mentality === undefined ? undefined : MENTALITY_KEYS[event.mentality]
    return mentality ? `match.commentary.mentalityChange.${mentality}` : undefined
  }
  return SET_PIECE_COMMENTARY_KEYS[event.eventType]?.[event.setPiece ?? 0] ?? COMMENTARY_KEYS[event.eventType]
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

// A {{key}} placeholder's fill-in text, plus the player or team it should link to when it names
// one - a placeholder with neither renders as plain text.
interface CommentaryVar {
  text: string
  playerId?: number
  teamId?: number
}

// Splits template on its {{...}} placeholders and fills each one in from vars, rendering a
// placeholder with a playerId/teamId as a clickable PlayerNameLink/TeamNameLink (issue: clickable
// player and team names in the match report) and everything else - including a placeholder with
// no matching var - as plain text, same as interpolate's old `?? ''` fallback.
function interpolateNodes(template: string, vars: Record<string, CommentaryVar>): ReactNode[] {
  return template.split(/(\{\{\w+\}\})/g).map((part, index) => {
    const match = /^\{\{(\w+)\}\}$/.exec(part)
    if (!match) {
      return part
    }
    const v = vars[match[1]]
    if (!v) {
      return ''
    }
    if (v.playerId !== undefined) {
      return (
        <PlayerNameLink key={index} playerId={v.playerId}>
          {v.text}
        </PlayerNameLink>
      )
    }
    if (v.teamId !== undefined) {
      return (
        <TeamNameLink key={index} teamId={v.teamId}>
          {v.text}
        </TeamNameLink>
      )
    }
    return v.text
  })
}

type Translate = (key: string, options?: { returnObjects: true }) => unknown

/** Turns one match event into a single radio-commentator-style line, picking one of several
 * i18n-translated phrasings for its event type (see match.commentary.* in the locale files) -
 * with any player or team name in that phrasing rendered as a clickable link to that player's/
 * team's page rather than plain text. Returns undefined for event types with no commentary (or
 * missing player data to fill one in) - callers should fall back to their own rendering in that
 * case. */
export function commentaryFor(
  event: MatchEvent,
  t: Translate,
  teamId: number,
  teamName: string,
  opponentId: number,
  opponentName: string,
): ReactNode[] | undefined {
  const key = commentaryKey(event)
  if (!key) {
    return undefined
  }

  const variants = t(key, { returnObjects: true })
  if (!Array.isArray(variants) || variants.length === 0) {
    return undefined
  }

  const playerIds = event.teamOwnerPlayers ?? []
  const playerNames = event.teamOwnerPlayerNames ?? []
  const template = variants[pickVariantIndex(variants.length, [event.minute, event.eventType, event.setPiece ?? 0, playerIds[0] ?? 0])] as string

  if (event.eventType === EVENT_TYPE_SUBSTITUTION) {
    if (playerNames.length < 2) {
      return undefined
    }
    return interpolateNodes(template, {
      team: { text: teamName, teamId },
      opponent: { text: opponentName, teamId: opponentId },
      outPlayer: { text: playerNames[0], playerId: playerIds[0] },
      inPlayer: { text: playerNames[1], playerId: playerIds[1] },
    })
  }

  // Unlike every other event type, an abandoned match names no player - just the short-handed
  // team (event.teamOwner, resolved to team/teamName by the caller) and the side awarded the win.
  // A default result likewise names no player: just the two sides (team is the home side).
  if (
    event.eventType === EVENT_TYPE_MATCH_ABANDONED ||
    event.eventType === EVENT_TYPE_MATCH_DEFAULT_RESULT ||
    event.eventType === EVENT_TYPE_MENTALITY_CHANGE
  ) {
    return interpolateNodes(template, {
      team: { text: teamName, teamId },
      opponent: { text: opponentName, teamId: opponentId },
    })
  }

  const player = playerNames[0]
  if (!player) {
    return undefined
  }
  // The defending goalkeeper of a set piece. A cup tie or friendly doesn't store it, so a missing
  // name reads as "the goalkeeper" instead of leaving a hole in the sentence.
  const keeperId = event.otherTeamPlayers?.[0]
  const keeperName = event.otherTeamPlayerNames?.[0]
  const keeper: CommentaryVar = keeperName
    ? { text: keeperName, playerId: keeperId }
    : { text: String(t('match.setPiece.theKeeper')) }
  return interpolateNodes(template, {
    player: { text: player, playerId: playerIds[0] },
    team: { text: teamName, teamId },
    opponent: { text: opponentName, teamId: opponentId },
    keeper,
  })
}
