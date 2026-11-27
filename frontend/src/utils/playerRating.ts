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

// Stamina and Form - deliberately not part of SKILL_FIELDS (see PlayerConditionPanel: neither
// measures footballing quality, so neither belongs in overallRating/bestCorePosition), but shown
// alongside the seven skills on the same compact chip rows (SkillChipsRow) since they share the
// same 0-99 scale and tier coloring.
export const CONDITION_FIELDS = ['stamina', 'form'] as const

export const CONDITION_SHORT_LABEL_KEYS: Record<(typeof CONDITION_FIELDS)[number], string> = {
  stamina: 'playerCard.skillsShort.stamina',
  form: 'playerCard.skillsShort.form',
}

/** A single 0-99 headline number summarizing a player's seven core skills - shown as the badge
 * on PlayerCard and inside PlayerStatsPopover. */
export function overallRating(player: Player): number {
  const total = SKILL_FIELDS.reduce((sum, field) => sum + player[field], 0)
  return Math.round(total / SKILL_FIELDS.length)
}

// The DEFENSE, MIDFIELD and STRIKER rows of backend common.SlotWeights (backend/common/
// slotweights.go): each skill's share, in percent, of a player's value at that slot, in the order
// goalKeeper, defense, lateral, pass, midfield, dribbling, shot. Kept in sync by hand - there's no
// shared source between the Go engine and this client-side quick reference (transfer market rows,
// lineup cards) - and pinned by playerRating.test.ts to the same figures as the Go table test.
const CORE_SLOT_WEIGHTS: Record<number, readonly number[]> = {
  [POSITION_DEFENSE]: [0, 90, 0, 0, 10, 0, 0],
  [POSITION_MIDFIELD]: [0, 10, 0, 20, 50, 5, 15],
  [POSITION_STRIKER]: [0, 0, 5, 10, 0, 20, 65],
}

/** A player's value at a core outfield slot (defense, midfield or striker), mirroring backend
 * common.PositionValue: the weighted sum of his skills, rounded like Go's (sum + 50) / 100. */
export function coreSlotValue(player: Player, position: number): number {
  const weights = CORE_SLOT_WEIGHTS[position]
  if (!weights) {
    return 0
  }
  const total = SKILL_FIELDS.reduce((sum, field, index) => sum + player[field] * weights[index], 0)
  return Math.floor((total + 50) / 100)
}

/** Which of the four non-lateral tactical positions (goalkeeper/defense/midfield/striker) a
 * player is naturally best suited to, mirroring backend common.BestCorePosition - shown as a
 * quick-reference "suggested position" badge next to a player's name. Ties favor the earlier
 * position in that list. */
export function bestCorePosition(player: Player): number {
  const values: [number, number][] = [
    [POSITION_GOALKEEPER, player.goalKeeper],
    [POSITION_DEFENSE, coreSlotValue(player, POSITION_DEFENSE)],
    [POSITION_MIDFIELD, coreSlotValue(player, POSITION_MIDFIELD)],
    [POSITION_STRIKER, coreSlotValue(player, POSITION_STRIKER)],
  ]
  return values.reduce((best, current) => (current[1] > best[1] ? current : best))[0]
}

// Lower bounds of the five rating tiers - typical player values sit around 30-50, so the lowest
// tiers are finer than a plain thirds split.
const RATING_TIER_FLOORS = [0, 35, 50, 65, 80] as const

/** 0-4 tier of a 0-99 rating, for the one-tone sequential scale (`palette.rating`). */
export function ratingTier(value: number): number {
  let tier = 0
  RATING_TIER_FLOORS.forEach((floor, index) => {
    if (value >= floor) {
      tier = index
    }
  })
  return tier
}

/** Color for a 0-99 rating or skill value: a five-step scale from neutral to the primary hue
 * (lightness is monotonic, so it still reads in grayscale / with colour-blindness). Red is
 * reserved for states (injury, suspension, relegation) and the number is always shown next to
 * the colour - never rely on the colour alone. */
export function ratingTierColor(value: number, palette: Palette): string {
  return palette.rating[ratingTier(value)]
}
