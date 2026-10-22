import { EVENT_TYPE_GOAL, type MatchEvent } from '../api/types'

export interface ScorerGoal {
  minute: number
  // SET_PIECE_* of the goal; 0 (open play) when the event carries none.
  setPiece: number
}

export interface ScorerLine {
  name: string
  goals: ScorerGoal[]
}

// Each side's goal scorers with the minutes they scored in, for the one-line "Berlana 3' · Lerma 16'"
// summary under a scoreboard - so the goals are visible without scrolling down to the narrated
// report. A scorer with several goals is one entry with several goals; entries keep the order of
// their first goal, and each goal remembers whether it was a penalty or a direct free kick so the
// scoreboard can mark it. Goals with no recorded scorer name are skipped rather than shown blank.
export function goalScorers(events: MatchEvent[], homeId: number): { home: ScorerLine[]; away: ScorerLine[] } {
  const home = new Map<string, ScorerGoal[]>()
  const away = new Map<string, ScorerGoal[]>()
  const goals = events.filter((event) => event.eventType === EVENT_TYPE_GOAL).sort((a, b) => a.minute - b.minute)
  for (const goal of goals) {
    const name = goal.teamOwnerPlayerNames?.[0]
    if (!name) {
      continue
    }
    const side = goal.teamOwner === homeId ? home : away
    side.set(name, [...(side.get(name) ?? []), { minute: goal.minute, setPiece: goal.setPiece ?? 0 }])
  }
  const lines = (side: Map<string, ScorerGoal[]>): ScorerLine[] => Array.from(side, ([name, goals]) => ({ name, goals }))
  return { home: lines(home), away: lines(away) }
}
