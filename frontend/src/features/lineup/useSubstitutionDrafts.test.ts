import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { FORMATION_BALANCED, UNAVAILABLE_INJURY, type Lineup, type PlayerUnavailability } from '../../api/types'
import { useSubstitutionDrafts } from './useSubstitutionDrafts'

const emptyLineup: Lineup = { matchId: 0, teamId: 1, formation: FORMATION_BALANCED, positions: [], substitutions: [] }
const usedPlayerIds = new Set<number | ''>([1, 2, 3])
const noUnavailability = new Map<number, PlayerUnavailability>()
const injured = (playerId: number): Map<number, PlayerUnavailability> =>
  new Map([[playerId, { playerId, reason: UNAVAILABLE_INJURY, matchesRemaining: 2 }]])

describe('useSubstitutionDrafts', () => {
  it('starts from the lineup already loaded and can add/remove drafts up to the max', () => {
    const { result } = renderHook(() => useSubstitutionDrafts(emptyLineup, usedPlayerIds, noUnavailability))
    expect(result.current.substitutionDrafts).toEqual([])

    act(() => result.current.handleAddSubstitution())
    act(() => result.current.handleAddSubstitution())
    act(() => result.current.handleAddSubstitution())
    expect(result.current.substitutionDrafts).toHaveLength(3)

    // A fourth draft is refused once MAX_SCHEDULED_SUBSTITUTIONS (3) is reached.
    act(() => result.current.handleAddSubstitution())
    expect(result.current.substitutionDrafts).toHaveLength(3)

    act(() => result.current.handleRemoveSubstitution(0))
    expect(result.current.substitutionDrafts).toHaveLength(2)
  })

  it('only turns a fully-filled draft into a valid substitution', () => {
    const { result } = renderHook(() => useSubstitutionDrafts(emptyLineup, usedPlayerIds, noUnavailability))

    act(() => result.current.handleAddSubstitution())
    expect(result.current.validSubstitutions).toEqual([])

    act(() => result.current.handleSubstitutionChange(0, 'playerOutId', 1))
    act(() => result.current.handleSubstitutionChange(0, 'playerInId', 4))
    expect(result.current.validSubstitutions).toEqual([])

    act(() => result.current.handleSubstitutionChange(0, 'minute', 60))
    expect(result.current.validSubstitutions).toEqual([{ playerOutId: 1, playerInId: 4, minute: 60 }])
  })

  it('drops a draft whose outgoing player is no longer among the current starters', () => {
    const { result } = renderHook(() => useSubstitutionDrafts(emptyLineup, usedPlayerIds, noUnavailability))

    act(() => result.current.handleAddSubstitution())
    act(() => result.current.handleSubstitutionChange(0, 'playerOutId', 99))
    act(() => result.current.handleSubstitutionChange(0, 'playerInId', 4))
    act(() => result.current.handleSubstitutionChange(0, 'minute', 60))

    expect(result.current.validSubstitutions).toEqual([])
  })

  it('drops a draft bringing in a player already fielded, or one suspended/injured', () => {
    const { result: alreadyFielded } = renderHook(() => useSubstitutionDrafts(emptyLineup, usedPlayerIds, noUnavailability))
    act(() => alreadyFielded.current.handleAddSubstitution())
    act(() => alreadyFielded.current.handleSubstitutionChange(0, 'playerOutId', 1))
    act(() => alreadyFielded.current.handleSubstitutionChange(0, 'playerInId', 2))
    act(() => alreadyFielded.current.handleSubstitutionChange(0, 'minute', 60))
    expect(alreadyFielded.current.validSubstitutions).toEqual([])

    const { result: unavailableIn } = renderHook(() => useSubstitutionDrafts(emptyLineup, usedPlayerIds, injured(4)))
    act(() => unavailableIn.current.handleAddSubstitution())
    act(() => unavailableIn.current.handleSubstitutionChange(0, 'playerOutId', 1))
    act(() => unavailableIn.current.handleSubstitutionChange(0, 'playerInId', 4))
    act(() => unavailableIn.current.handleSubstitutionChange(0, 'minute', 60))
    expect(unavailableIn.current.validSubstitutions).toEqual([])
  })
})
