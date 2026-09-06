import type { Player } from '../api/types'

export const SKILL_FIELDS = ['goalKeeper', 'defense', 'lateral', 'pass', 'midfield', 'dribbling', 'shot'] as const

/** A single 0-99 headline number summarizing a player's seven core skills - shown as the badge
 * on PlayerCard and inside PlayerStatsPopover. */
export function overallRating(player: Player): number {
  const total = SKILL_FIELDS.reduce((sum, field) => sum + player[field], 0)
  return Math.round(total / SKILL_FIELDS.length)
}
