import {
  POSITION_DEFENSE,
  POSITION_GOALKEEPER,
  POSITION_LATERAL_DEFENSE_LEFT,
  POSITION_LATERAL_DEFENSE_RIGHT,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_LATERAL_MIDFIELD_RIGHT,
  POSITION_MIDFIELD,
  POSITION_STRIKER,
} from '../../api/types'

export const PITCH_WIDTH = 300
export const PITCH_HEIGHT = 460

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
}

// Fixed x for the wide (touchline-hugging) slots - always exactly one of each per formation.
const WIDE_X: Record<number, number> = {
  [POSITION_LATERAL_DEFENSE_LEFT]: 30,
  [POSITION_LATERAL_DEFENSE_RIGHT]: 270,
  [POSITION_LATERAL_MIDFIELD_LEFT]: 22,
  [POSITION_LATERAL_MIDFIELD_RIGHT]: 278,
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

/** Computes a pitch coordinate for every slot a formation's position counts call for. */
export function layoutPitchSlots(counts: Partial<Record<number, number>>): PitchSlot[] {
  const slots: PitchSlot[] = []
  for (const [positionStr, count] of Object.entries(counts)) {
    const position = Number(positionStr)
    if (!count) continue
    const y = ROW_Y[position] ?? PITCH_HEIGHT / 2
    if (position in WIDE_X) {
      slots.push({ position, slotIndex: 0, x: WIDE_X[position], y })
      continue
    }
    const band = CENTRAL_BAND[position] ?? [90, 210]
    spreadX(count, band).forEach((x, slotIndex) => slots.push({ position, slotIndex, x, y }))
  }
  return slots
}
