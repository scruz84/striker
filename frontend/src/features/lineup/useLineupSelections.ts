import { useMemo, useState } from 'react'
import {
  FORMATION_BALANCED,
  FORMATION_POSITION_COUNTS,
  type Lineup,
  type Player,
  type PlayerUnavailability,
} from '../../api/types'

// Slots per position, keyed the same way FORMATION_POSITION_COUNTS is: a position not present
// for a formation simply isn't used by it.
type Selections = Record<number, (number | '')[]>

function initialSelections(lineup: Lineup): Selections {
  const next: Selections = {}
  for (const pos of lineup.positions) {
    if (!next[pos.position]) {
      next[pos.position] = []
    }
    next[pos.position].push(pos.playerId)
  }
  return next
}

// Drives the pitch/bench editor's tap-to-select-then-tap-to-place interaction, shared by
// LineupEditorPage (the team's standing default lineup, plus scheduled substitutions) and
// MatchLineupForm (a one-off per-match override) - the two used to each carry their own copy of
// this exact state and swap logic, which is what let a bug (a stale squadSize-style guard, a
// missed edge case) slip into one without the other; now there's only one copy to get right.
//
// Replaces the previous drag-and-drop editor: dragging a small avatar a few dozen pixels across a
// crowded pitch gave no clear sense of which slot you were about to drop onto (an occupied slot
// never highlighted at all, only an empty one did), and touch dragging needed a press-and-hold
// delay that fought with scrolling the page. Tapping a player to select them, then tapping a
// second target to place them, needs no timing heuristics and lets every valid target be
// highlighted the whole time a selection is active - see PlayerChip/SoccerPitch/PlayerBench's own
// "highlight" props for that half of the fix.
//
// A suspended/injured player (see useSquadUnavailability) can't be put on the pitch: tapping them
// on the bench does nothing, so they can never be picked up nor swapped in. One already fielded by
// a lineup saved before their suspension/injury can still be moved (to the bench, or elsewhere on
// the pitch); whether that blocks saving depends on enforceAvailability - true for a per-match
// lineup (isComplete stays false until they're replaced, matching the backend rejecting the save -
// see handlers.validateLineup), false for a team's standing default (league/cup or friendly),
// which is general-purpose rather than tied to a specific matchday's squad and so may legitimately
// carry a player who's currently out.
export function useLineupSelections(
  players: Player[],
  lineup: Lineup,
  unavailability: Map<number, PlayerUnavailability>,
  enforceAvailability = true,
) {
  // Lazy initializers run once on mount, seeded from the lineup already loaded by the caller - no
  // effect needed to sync them in after the fact. Switching formations doesn't reset selections:
  // it just changes how many slots per position are shown/required, so a player picked under one
  // formation stays picked (just off-pitch, back on the bench) if you switch away and later
  // switch back to it.
  const [formation, setFormationState] = useState(lineup.formation)
  const [selections, setSelections] = useState<Selections>(() => initialSelections(lineup))
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | null>(null)

  const counts = FORMATION_POSITION_COUNTS[formation] ?? FORMATION_POSITION_COUNTS[FORMATION_BALANCED]
  const positionsToShow = Object.keys(counts).map(Number)

  const visibleSelections = useMemo(() => {
    const flat: { position: number; playerId: number | '' }[] = []
    for (const position of positionsToShow) {
      const count = counts[position] ?? 0
      for (let i = 0; i < count; i++) {
        flat.push({ position, playerId: selections[position]?.[i] ?? '' })
      }
    }
    return flat
  }, [selections, counts, positionsToShow])

  const usedPlayerIds = useMemo(
    () => new Set(visibleSelections.map((s) => s.playerId).filter((v) => v !== '')),
    [visibleSelections],
  )
  const unavailableStarters = players.filter((p) => usedPlayerIds.has(p.id) && unavailability.has(p.id))
  const isComplete =
    visibleSelections.every((s) => s.playerId !== '') && (!enforceAvailability || unavailableStarters.length === 0)
  const benchPlayers = players.filter((p) => !usedPlayerIds.has(p.id))

  const playersById = useMemo(() => new Map(players.map((p) => [p.id, p])), [players])
  const getPlayer = (position: number, slotIndex: number): Player | undefined => {
    const id = selections[position]?.[slotIndex]
    return id === undefined || id === '' ? undefined : playersById.get(id)
  }

  function findSlotForPlayer(playerId: number): { position: number; slotIndex: number } | null {
    for (const position of positionsToShow) {
      const count = counts[position] ?? 0
      const arr = selections[position] ?? []
      for (let i = 0; i < count; i++) {
        if (arr[i] === playerId) {
          return { position, slotIndex: i }
        }
      }
    }
    return null
  }

  // Moves movingPlayerId into (targetPosition, targetSlotIndex), swapping in whoever was already
  // there (if anyone) into movingPlayerId's own former slot - or, if movingPlayerId came from the
  // bench (no former slot), the displaced occupant simply isn't written back anywhere, which is
  // exactly what leaves them benched (benchPlayers is derived as "not in usedPlayerIds").
  function movePlayerIntoSlot(movingPlayerId: number, targetPosition: number, targetSlotIndex: number) {
    const movingSource = findSlotForPlayer(movingPlayerId)
    setSelections((prev) => {
      const next = { ...prev }
      const targetArr = [...(next[targetPosition] ?? [])]
      const displaced = targetArr[targetSlotIndex] ?? ''
      targetArr[targetSlotIndex] = movingPlayerId
      next[targetPosition] = targetArr

      if (movingSource) {
        const sourceArr = movingSource.position === targetPosition ? targetArr : [...(next[movingSource.position] ?? [])]
        sourceArr[movingSource.slotIndex] = displaced
        next[movingSource.position] = sourceArr
      }
      return next
    })
  }

  // Tapping a pitch slot (empty or occupied): nothing selected yet just picks up whoever's there
  // (a no-op on an empty slot); tapping the already-selected player's own slot again cancels the
  // selection; otherwise the selected player moves here.
  function handleSlotClick(position: number, slotIndex: number) {
    const occupantId = selections[position]?.[slotIndex]
    if (selectedPlayerId === null) {
      if (occupantId !== undefined && occupantId !== '') {
        setSelectedPlayerId(occupantId)
      }
      return
    }
    if (occupantId === selectedPlayerId) {
      setSelectedPlayerId(null)
      return
    }
    movePlayerIntoSlot(selectedPlayerId, position, slotIndex)
    setSelectedPlayerId(null)
  }

  // Tapping a bench player: same select/cancel behavior as a pitch slot. Tapping one bench player
  // while a *fielded* player is selected sends the fielded player to the bench and puts this one
  // in their old slot - the bench equivalent of a pitch-to-pitch swap. Two bench players have
  // nothing to swap (the bench has no order), so that case just moves the selection instead.
  function handleBenchPlayerClick(playerId: number) {
    if (unavailability.has(playerId)) {
      return
    }
    if (selectedPlayerId === playerId) {
      setSelectedPlayerId(null)
      return
    }
    if (selectedPlayerId === null) {
      setSelectedPlayerId(playerId)
      return
    }
    const source = findSlotForPlayer(selectedPlayerId)
    if (!source) {
      setSelectedPlayerId(playerId)
      return
    }
    movePlayerIntoSlot(playerId, source.position, source.slotIndex)
    setSelectedPlayerId(null)
  }

  // Tapping the bench's empty background (not a specific player) benches whichever fielded player
  // is currently selected - a no-op if the selection is already a bench player or nothing at all.
  function handleBenchAreaClick() {
    if (selectedPlayerId === null) return
    const source = findSlotForPlayer(selectedPlayerId)
    if (!source) {
      setSelectedPlayerId(null)
      return
    }
    setSelections((prev) => {
      const arr = [...(prev[source.position] ?? [])]
      arr[source.slotIndex] = ''
      return { ...prev, [source.position]: arr }
    })
    setSelectedPlayerId(null)
  }

  // A formation change can shrink or reshuffle which slots are visible, so any in-progress
  // selection is cleared rather than risk it pointing at a slot that no longer shows.
  function setFormation(next: number) {
    setFormationState(next)
    setSelectedPlayerId(null)
  }

  // Replaces the whole editor state with another lineup - e.g. "copy a previous match's lineup"
  // (see LoadPreviousLineupControl). A player that lineup fielded but who's no longer on this
  // squad (sold, retired, released) is silently dropped rather than left as a phantom occupant of
  // that slot: isComplete already requires every visible slot to hold a real playerId, so the slot
  // just comes back empty for the manager to fill with someone actually on the roster today.
  function loadLineup(next: Lineup) {
    const squadIds = new Set(players.map((p) => p.id))
    const filtered: Selections = {}
    for (const pos of next.positions) {
      if (!squadIds.has(pos.playerId)) continue
      if (!filtered[pos.position]) {
        filtered[pos.position] = []
      }
      filtered[pos.position].push(pos.playerId)
    }
    setFormationState(next.formation)
    setSelections(filtered)
    setSelectedPlayerId(null)
  }

  return {
    formation,
    setFormation,
    loadLineup,
    counts,
    positionsToShow,
    visibleSelections,
    usedPlayerIds,
    isComplete,
    unavailableStarters,
    benchPlayers,
    getPlayer,
    selectedPlayerId,
    handleSlotClick,
    handleBenchPlayerClick,
    handleBenchAreaClick,
  }
}
