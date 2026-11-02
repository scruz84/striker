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
  type OpponentPreviewPlayer,
} from '../../api/types'

// The likely XI is drawn line by line, attack first, each line left to right (the owner's left
// flank first), so it reads like the pitch from the opponent's goal down to its own.
const XI_LINES: { key: string; positions: number[] }[] = [
  { key: 'attack', positions: [POSITION_LATERAL_STRIKER_LEFT, POSITION_STRIKER, POSITION_LATERAL_STRIKER_RIGHT] },
  { key: 'midfield', positions: [POSITION_LATERAL_MIDFIELD_LEFT, POSITION_MIDFIELD, POSITION_LATERAL_MIDFIELD_RIGHT] },
  { key: 'defense', positions: [POSITION_LATERAL_DEFENSE_LEFT, POSITION_DEFENSE, POSITION_LATERAL_DEFENSE_RIGHT] },
  { key: 'goalkeeper', positions: [POSITION_GOALKEEPER] },
]

/** Groups the likely XI into its lines (attack to goalkeeper), dropping empty ones. */
export function likelyLines(players: OpponentPreviewPlayer[]): { key: string; players: OpponentPreviewPlayer[] }[] {
  return XI_LINES.map((line) => ({
    key: line.key,
    players: line.positions.flatMap((position) => players.filter((p) => p.position === position)),
  })).filter((line) => line.players.length > 0)
}

