import { EVENT_TYPE_GOAL, type MatchEvent } from '../api/types'
import { compareMatchMinute } from './matchMinute'

export interface ScorerGoal {
  minute: number
  // 1..N for a goal in a half's added time (minute then reads 45 or 90), 0 otherwise.
  addedMinute: number
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
  const goals = events.filter((event) => event.eventType === EVENT_TYPE_GOAL).sort(compareMatchMinute)
  for (const goal of goals) {
    const name = goal.teamOwnerPlayerNames?.[0]
    if (!name) {
      continue
    }
    const side = goal.teamOwner === homeId ? home : away
    side.set(name, [...(side.get(name) ?? []), { minute: goal.minute, addedMinute: goal.addedMinute ?? 0, setPiece: goal.setPiece ?? 0 }])
  }
  const lines = (side: Map<string, ScorerGoal[]>): ScorerLine[] => Array.from(side, ([name, goals]) => ({ name, goals }))
  return { home: lines(home), away: lines(away) }
}
