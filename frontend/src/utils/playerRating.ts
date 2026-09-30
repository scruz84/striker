import type { Palette } from '@mui/material/styles'
import { POSITION_DEFENSE, POSITION_GOALKEEPER, POSITION_MIDFIELD, POSITION_STRIKER, type Player } from '../api/types'

export const SKILL_FIELDS = ['goalKeeper', 'defense', 'lateral', 'pass', 'midfield', 'dribbling', 'shot'] as const

// Three-letter abbreviations for SKILL_FIELDS, used wherever all seven skills need to fit on one
// compact line (e.g. a listing card) rather than the full-word labels PlayerStatsPopover's
// SkillBar rows show.
export const SKILL_SHORT_LABEL_KEYS: Record<(typeof SKILL_FIELDS)[number], string> = {
  goalKeeper: 'playerCard.skillsShort.goalkeeper',
  defense: 'playerCard.skillsShort.defense',
  lateral: 'playerCard.skillsShort.lateral',
  pass: 'playerCard.skillsShort.pass',
  midfield: 'playerCard.skillsShort.midfield',
  dribbling: 'playerCard.skillsShort.dribbling',
  shot: 'playerCard.skillsShort.shot',
}

// Full-word labels for SKILL_FIELDS - e.g. a "sort by" dropdown, where the three-letter
// abbreviation alone (SKILL_SHORT_LABEL_KEYS) would be too cryptic for an option label.
export const SKILL_LABEL_KEYS: Record<(typeof SKILL_FIELDS)[number], string> = {
  goalKeeper: 'playerCard.skills.goalkeeper',
  defense: 'playerCard.skills.defense',
  lateral: 'playerCard.skills.lateral',
  pass: 'playerCard.skills.pass',
  midfield: 'playerCard.skills.midfield',
  dribbling: 'playerCard.skills.dribbling',
  shot: 'playerCard.skills.shot',
}

/** A single 0-99 headline number summarizing a player's seven core skills - shown as the badge
 * on PlayerCard and inside PlayerStatsPopover. */
export function overallRating(player: Player): number {
  const total = SKILL_FIELDS.reduce((sum, field) => sum + player[field], 0)
  return Math.round(total / SKILL_FIELDS.length)
}

// Mirrors backend common.GetMidfieldValue/GetStrikerValue (lineup.go/fmteam.go) - the same value
// formulas BestCorePosition compares, kept in sync by hand since there's no shared source between
// the Go engine and this client-side "quick reference" use (transfer market rows, lineup cards).
function midfieldValue(player: Player): number {
  return (player.midfield + player.pass) / 2
}
function strikerValue(player: Player): number {
  return (player.shot + player.pass + player.dribbling) / 3
}

/** Which of the four non-lateral tactical positions (goalkeeper/defense/midfield/striker) a
 * player is naturally best suited to, mirroring backend common.BestCorePosition - shown as a
 * quick-reference "suggested position" badge next to a player's name. Ties favor the earlier
 * position in that list. */
export function bestCorePosition(player: Player): number {
  const values: [number, number][] = [
    [POSITION_GOALKEEPER, player.goalKeeper],
    [POSITION_DEFENSE, player.defense],
    [POSITION_MIDFIELD, midfieldValue(player)],
    [POSITION_STRIKER, strikerValue(player)],
  ]
  return values.reduce((best, current) => (current[1] > best[1] ? current : best))[0]
}

/** Traffic-light color for a 0-99 rating or skill value, same thresholds used everywhere a single
 * number needs an at-a-glance read (overall rating badges, per-skill bars). */
export function ratingTierColor(value: number, palette: Palette): string {
  return value >= 65 ? palette.success.main : value >= 45 ? palette.warning.main : palette.error.main
}
