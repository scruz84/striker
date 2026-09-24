import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import {
  FORMATION_BALANCED,
  POSITION_GOALKEEPER,
  UNAVAILABLE_INJURY,
  type Lineup,
  type Player,
  type PlayerUnavailability,
} from '../../api/types'
import { useLineupSelections } from './useLineupSelections'

function player(id: number): Player {
  return {
    id,
    teamId: 1,
    firstName: 'P',
    lastName: String(id),
    birthDay: '2000-01-01',
    goalKeeper: 50,
    defense: 50,
    lateral: 50,
    pass: 50,
    midfield: 50,
    dribbling: 50,
    shot: 50,
    header: false,
    quick: false,
    creative: false,
    technical: false,
    wage: 0,
    age: 25,
    contractYearsRemaining: 1,
  }
}

const players = [player(1), player(2), player(3)]
const injured = (playerId: number): Map<number, PlayerUnavailability> =>
  new Map([[playerId, { playerId, reason: UNAVAILABLE_INJURY, matchesRemaining: 2 }]])
const lineupWithGoalkeeper = (playerId: number): Lineup => ({
  matchId: 0,
  teamId: 1,
  formation: FORMATION_BALANCED,
  positions: [{ position: POSITION_GOALKEEPER, playerId }],
  substitutions: [],
})

describe('useLineupSelections', () => {
  it("won't let a suspended/injured bench player be picked up or swapped in", () => {
    const { result } = renderHook(() => useLineupSelections(players, lineupWithGoalkeeper(1), injured(2)))

    act(() => result.current.handleBenchPlayerClick(2))
    expect(result.current.selectedPlayerId).toBeNull()

    act(() => result.current.handleSlotClick(POSITION_GOALKEEPER, 0))
    act(() => result.current.handleBenchPlayerClick(2))
    expect(result.current.getPlayer(POSITION_GOALKEEPER, 0)?.id).toBe(1)
  })

  it('reports an already-fielded unavailable player and lets them be replaced', () => {
    const { result } = renderHook(() => useLineupSelections(players, lineupWithGoalkeeper(1), injured(1)))
    expect(result.current.unavailableStarters.map((p) => p.id)).toEqual([1])
    expect(result.current.isComplete).toBe(false)

    act(() => result.current.handleSlotClick(POSITION_GOALKEEPER, 0))
    act(() => result.current.handleBenchPlayerClick(3))
    expect(result.current.getPlayer(POSITION_GOALKEEPER, 0)?.id).toBe(3)
    expect(result.current.unavailableStarters).toEqual([])
  })
})
