import {
  POSITION_DEFENSE,
  POSITION_GOALKEEPER,
  POSITION_LATERAL_DEFENSE_LEFT,
  POSITION_LATERAL_DEFENSE_RIGHT,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_LATERAL_MIDFIELD_RIGHT,
  POSITION_LATERAL_STRIKER_LEFT,
  POSITION_LATERAL_STRIKER_RIGHT,
  POSITION_MIDFIELD,
  POSITION_STRIKER,
} from '../../api/types'

export const PITCH_WIDTH = 300
export const PITCH_HEIGHT = 460

// A full pitch has a lot of empty opponent-half space above the strikers (nothing is ever drawn
// there - a lineup only ever occupies the own half plus a bit past the halfway line). The
// interactive lineup editor uses this shorter "half pitch" instead, showing just that occupied
// half so the same formation takes noticeably less vertical space - see PitchField's `half` prop
// and layoutPitchSlots' `half` option below. Read-only lineup views (match details) keep using
// the full-height layout, unaffected by any of this.
export const PITCH_HEIGHT_HALF = 260

// Row depth (y) per position, own goal at the bottom (like the classic formation-sheet view).
const ROW_Y: Record<number, number> = {
  [POSITION_GOALKEEPER]: 430,
  [POSITION_DEFENSE]: 350,
  [POSITION_LATERAL_DEFENSE_LEFT]: 350,
  [POSITION_LATERAL_DEFENSE_RIGHT]: 350,
  [POSITION_MIDFIELD]: 240,
  [POSITION_LATERAL_MIDFIELD_LEFT]: 240,
  [POSITION_LATERAL_MIDFIELD_RIGHT]: 240,
  [POSITION_STRIKER]: 110,
  [POSITION_LATERAL_STRIKER_LEFT]: 110,
  [POSITION_LATERAL_STRIKER_RIGHT]: 110,
}

// Same rows, compressed into PITCH_HEIGHT_HALF for the half-pitch layout.
const ROW_Y_HALF: Record<number, number> = {
  [POSITION_GOALKEEPER]: 235,
  [POSITION_DEFENSE]: 178,
  [POSITION_LATERAL_DEFENSE_LEFT]: 178,
  [POSITION_LATERAL_DEFENSE_RIGHT]: 178,
  [POSITION_MIDFIELD]: 112,
  [POSITION_LATERAL_MIDFIELD_LEFT]: 112,
  [POSITION_LATERAL_MIDFIELD_RIGHT]: 112,
  [POSITION_STRIKER]: 45,
  [POSITION_LATERAL_STRIKER_LEFT]: 45,
  [POSITION_LATERAL_STRIKER_RIGHT]: 45,
}

// Fixed x for the wide (touchline-hugging) slots - always exactly one of each per formation.
const WIDE_X: Record<number, number> = {
  [POSITION_LATERAL_DEFENSE_LEFT]: 30,
  [POSITION_LATERAL_DEFENSE_RIGHT]: 270,
  [POSITION_LATERAL_MIDFIELD_LEFT]: 22,
  [POSITION_LATERAL_MIDFIELD_RIGHT]: 278,
  [POSITION_LATERAL_STRIKER_LEFT]: 55,
  [POSITION_LATERAL_STRIKER_RIGHT]: 245,
}

// Horizontal band the central (count-variable) slots at each row spread across, evenly spaced.
const CENTRAL_BAND: Record<number, [number, number]> = {
  [POSITION_GOALKEEPER]: [150, 150],
  [POSITION_DEFENSE]: [95, 205],
  [POSITION_MIDFIELD]: [90, 210],
  [POSITION_STRIKER]: [55, 245],
}

function spreadX(count: number, [left, right]: [number, number]): number[] {
  const step = (right - left) / (count + 1)
  return Array.from({ length: count }, (_, i) => left + step * (i + 1))
}

export interface PitchSlot {
  position: number
  slotIndex: number
  x: number
  y: number
}

/** Computes a pitch coordinate for every slot a formation's position counts call for. Pass
 * `half: true` to lay out against PITCH_HEIGHT_HALF instead (see its comment above). */
export function layoutPitchSlots(counts: Partial<Record<number, number>>, half = false): PitchSlot[] {
  const rowY = half ? ROW_Y_HALF : ROW_Y
  const fallbackY = (half ? PITCH_HEIGHT_HALF : PITCH_HEIGHT) / 2
  const slots: PitchSlot[] = []
  for (const [positionStr, count] of Object.entries(counts)) {
    const position = Number(positionStr)
    if (!count) continue
    const y = rowY[position] ?? fallbackY
    if (position in WIDE_X) {
      slots.push({ position, slotIndex: 0, x: WIDE_X[position], y })
      continue
    }
    const band = CENTRAL_BAND[position] ?? [90, 210]
    spreadX(count, band).forEach((x, slotIndex) => slots.push({ position, slotIndex, x, y }))
  }
  return slots
}

// Total canvas height for the head-to-head pitch - two occupied halves back to back, meeting at
// a shared halfway line (see layoutHeadToHeadSlots below).
export const PITCH_HEIGHT_HEAD_TO_HEAD = PITCH_HEIGHT_HALF * 2

export interface HeadToHeadSlots {
  home: PitchSlot[]
  away: PitchSlot[]
}

/** Lays out both teams' formations on one shared pitch, facing each other, instead of each team
 * getting its own separate pitch - home occupies the bottom half (own goal at the bottom, same
 * orientation as layoutPitchSlots' `half` layout), away is mirrored into the top half (own goal
 * at the top, formation approaching the halfway line from the other side). Left/right x
 * coordinates are NOT mirrored between the two - a team's own left back is drawn on the same
 * side of the pitch for both teams, matching how broadcast head-to-head graphics do it, rather
 * than physically mirroring by pitch geometry (which would make "left vs left" battles harder
 * to read at a glance). */
export function layoutHeadToHeadSlots(
  homeCounts: Partial<Record<number, number>>,
  awayCounts: Partial<Record<number, number>>,
): HeadToHeadSlots {
  const home = layoutPitchSlots(homeCounts, true).map((slot) => ({ ...slot, y: PITCH_HEIGHT_HALF + slot.y }))
  const away = layoutPitchSlots(awayCounts, true).map((slot) => ({ ...slot, y: PITCH_HEIGHT_HALF - slot.y }))
  return { home, away }
}
