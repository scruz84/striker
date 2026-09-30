import { useState } from 'react'
import { MAX_SCHEDULED_SUBSTITUTIONS, type Lineup, type LineupSubstitution } from '../../api/types'

// A row being edited in the substitution planner - fields start empty ('') until the manager
// picks them, unlike LineupSubstitution which is only ever complete.
export interface SubstitutionDraft {
  playerOutId: number | ''
  playerInId: number | ''
  minute: number | ''
}

function initialSubstitutionDrafts(lineup: Lineup): SubstitutionDraft[] {
  return (lineup.substitutions ?? []).map((s) => ({ playerOutId: s.playerOutId, playerInId: s.playerInId, minute: s.minute }))
}

// Drives the substitution planner's add/remove/edit state, shared by LineupEditorPage (the
// team's standing default lineup) and MatchLineupForm (a friendly's own per-match substitutions) -
// the two used to each carry their own copy of this exact state, which is what useLineupSelections
// already fixed for the pitch/bench editor half of the form; this is the same fix for the
// substitutions half.
export function useSubstitutionDrafts(lineup: Lineup, usedPlayerIds: Set<number | ''>, unavailability: Map<number, unknown>) {
  const [substitutionDrafts, setSubstitutionDrafts] = useState<SubstitutionDraft[]>(() => initialSubstitutionDrafts(lineup))

  const usedSubOutIds = new Set(substitutionDrafts.map((s) => s.playerOutId).filter((v) => v !== ''))
  const usedSubInIds = new Set(substitutionDrafts.map((s) => s.playerInId).filter((v) => v !== ''))

  const handleAddSubstitution = () => {
    if (substitutionDrafts.length >= MAX_SCHEDULED_SUBSTITUTIONS) return
    setSubstitutionDrafts((prev) => [...prev, { playerOutId: '', playerInId: '', minute: '' }])
  }

  const handleRemoveSubstitution = (index: number) => {
    setSubstitutionDrafts((prev) => prev.filter((_, i) => i !== index))
  }

  const handleSubstitutionChange = (index: number, field: keyof SubstitutionDraft, value: number | '') => {
    setSubstitutionDrafts((prev) => prev.map((s, i) => (i === index ? { ...s, [field]: value } : s)))
  }

  // Clears every draft outright, rather than re-seeding from some other lineup's own
  // substitutions - see LoadPreviousLineupControl, whose "copy a previous match's lineup" always
  // hands back an empty Substitutions array anyway (GetLineupsForMatch never populates it), and
  // whose whole point is a fresh XI, not carrying over an unrelated match's own subs. Needed
  // because loading a different lineup can un-start a player a draft still names as playerOutId,
  // which a lineup change alone wouldn't otherwise clear (this hook's own state, seeded once on
  // mount from the original `lineup` prop, only reacts to draft edits, not to the pitch/bench
  // changing under it).
  const resetSubstitutionDrafts = () => setSubstitutionDrafts([])

  // A draft becomes an actual scheduled substitution only once every field is filled and both
  // players still exist in the current starters/bench (a lineup edit made after picking a
  // substitution can otherwise strand a stale player id) - anything else is silently dropped
  // rather than blocking the whole lineup save over one incomplete row.
  const validSubstitutions: LineupSubstitution[] = substitutionDrafts
    .filter(
      (s): s is { playerOutId: number; playerInId: number; minute: number } =>
        s.playerOutId !== '' &&
        s.playerInId !== '' &&
        s.minute !== '' &&
        usedPlayerIds.has(s.playerOutId) &&
        !usedPlayerIds.has(s.playerInId) &&
        // A suspended/injured player can't come on either (see handlers.validateSubstitutions).
        !unavailability.has(s.playerInId),
    )
    .map((s) => ({ playerOutId: s.playerOutId, playerInId: s.playerInId, minute: s.minute }))

  return {
    substitutionDrafts,
    usedSubOutIds,
    usedSubInIds,
    handleAddSubstitution,
    handleRemoveSubstitution,
    handleSubstitutionChange,
    resetSubstitutionDrafts,
    validSubstitutions,
  }
}
