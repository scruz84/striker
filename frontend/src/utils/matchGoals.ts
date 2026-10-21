import { EVENT_TYPE_GOAL, type MatchEvent } from '../api/types'

export interface ScorerLine {
  name: string
  minutes: number[]
}

// Each side's goal scorers with the minutes they scored in, for the one-line "Berlana 3' · Lerma 16'"
// summary under a scoreboard - so the goals are visible without scrolling down to the narrated
// report. A scorer with several goals is one entry with several minutes; entries keep the order of
// their first goal. Goals with no recorded scorer name are skipped rather than shown blank.
export function goalScorers(events: MatchEvent[], homeId: number): { home: ScorerLine[]; away: ScorerLine[] } {
  const home = new Map<string, number[]>()
  const away = new Map<string, number[]>()
  const goals = events.filter((event) => event.eventType === EVENT_TYPE_GOAL).sort((a, b) => a.minute - b.minute)
  for (const goal of goals) {
    const name = goal.teamOwnerPlayerNames?.[0]
    if (!name) {
      continue
    }
    const side = goal.teamOwner === homeId ? home : away
    side.set(name, [...(side.get(name) ?? []), goal.minute])
  }
  const lines = (side: Map<string, number[]>): ScorerLine[] => Array.from(side, ([name, minutes]) => ({ name, minutes }))
  return { home: lines(home), away: lines(away) }
}
