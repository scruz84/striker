import {
  BEST_XI_LINE_ATTACK,
  BEST_XI_LINE_DEFENCE,
  BEST_XI_LINE_GOALKEEPER,
  BEST_XI_LINE_MIDFIELD,
  POSITION_DEFENSE,
  POSITION_GOALKEEPER,
  POSITION_LATERAL_DEFENSE_LEFT,
  POSITION_LATERAL_DEFENSE_RIGHT,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_LATERAL_MIDFIELD_RIGHT,
  POSITION_MIDFIELD,
  POSITION_STRIKER,
  type BestXiPlayer,
} from '../../api/types'
import { layoutPitchSlots } from '../../components/pitch/pitchLayout'

// The pitch positions a line of n players takes, left to right: a back or midfield four uses
// both wide lanes, as the lineup editor draws a 4-4-2, so the chips keep apart on a phone.
function linePositions(line: number, n: number): number[] {
  const wide = (left: number, centre: number, right: number) =>
    n >= 3 ? [left, ...Array<number>(n - 2).fill(centre), right] : Array<number>(n).fill(centre)
  switch (line) {
    case BEST_XI_LINE_GOALKEEPER:
      return Array<number>(n).fill(POSITION_GOALKEEPER)
    case BEST_XI_LINE_DEFENCE:
      return wide(POSITION_LATERAL_DEFENSE_LEFT, POSITION_DEFENSE, POSITION_LATERAL_DEFENSE_RIGHT)
    case BEST_XI_LINE_MIDFIELD:
      return wide(POSITION_LATERAL_MIDFIELD_LEFT, POSITION_MIDFIELD, POSITION_LATERAL_MIDFIELD_RIGHT)
    case BEST_XI_LINE_ATTACK:
    default:
      return Array<number>(n).fill(POSITION_STRIKER)
  }
}

// Places the stored players on the lineup editor's half pitch (PITCH_HEIGHT_EDITOR): each line's
// players, in slot order, take that row's places from left to right.
export function placeBestXi(players: BestXiPlayer[]) {
  const sorted = [...players].sort((a, b) => a.slot - b.slot)
  const byLine = new Map<number, BestXiPlayer[]>()
  for (const player of sorted) {
    byLine.set(player.line, [...(byLine.get(player.line) ?? []), player])
  }
  const assigned: { player: BestXiPlayer; position: number }[] = []
  for (const [line, members] of byLine) {
    linePositions(line, members.length).forEach((position, i) => assigned.push({ player: members[i], position }))
  }
  const counts: Record<number, number> = {}
  for (const { position } of assigned) counts[position] = (counts[position] ?? 0) + 1
  const slots = layoutPitchSlots(counts, true, true)
  const taken: Record<number, number> = {}
  return assigned
    .sort((a, b) => a.player.slot - b.player.slot)
    .flatMap(({ player, position }) => {
      const index = taken[position] ?? 0
      taken[position] = index + 1
      const slot = slots.find((s) => s.position === position && s.slotIndex === index)
      return slot ? [{ player, x: slot.x, y: slot.y }] : []
    })
}
