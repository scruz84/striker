import { describe, expect, it } from 'vitest'
import { POSITION_DEFENSE, POSITION_GOALKEEPER, POSITION_LATERAL_STRIKER_LEFT, POSITION_MIDFIELD, POSITION_STRIKER } from '../../api/types'
import { likelyLines } from './opponentPreviewLines'

describe('likelyLines', () => {
  it('orders the XI attack first, each line with the left flank first, and drops empty lines', () => {
    const lines = likelyLines([
      { playerId: 1, name: 'Keeper', position: POSITION_GOALKEEPER },
      { playerId: 2, name: 'Centre back', position: POSITION_DEFENSE },
      { playerId: 3, name: 'Midfielder', position: POSITION_MIDFIELD },
      { playerId: 4, name: 'Striker', position: POSITION_STRIKER },
      { playerId: 5, name: 'Left forward', position: POSITION_LATERAL_STRIKER_LEFT },
    ])

    expect(lines.map((l) => l.key)).toEqual(['attack', 'midfield', 'defense', 'goalkeeper'])
    expect(lines[0].players.map((p) => p.name)).toEqual(['Left forward', 'Striker'])
  })
})
