import type { TeamPlayerStats } from '../../api/types'

export type TeamStatsSort = 'minutes' | 'appearances' | 'goals' | 'rating' | 'cards'

export const TEAM_STATS_SORTS: TeamStatsSort[] = ['minutes', 'appearances', 'goals', 'rating', 'cards']

const value: Record<TeamStatsSort, (p: TeamPlayerStats) => number> = {
  minutes: (p) => p.minutes,
  appearances: (p) => p.appearances,
  goals: (p) => p.goals,
  rating: (p) => p.averageRating ?? -1,
  cards: (p) => p.yellowCards + 3 * p.redCards,
}

// Orders a squad's lines by the chosen figure, highest first, then by minutes and player id.
export function sortTeamStats(players: TeamPlayerStats[], sort: TeamStatsSort): TeamPlayerStats[] {
  return [...players].sort(
    (a, b) => value[sort](b) - value[sort](a) || b.minutes - a.minutes || a.playerId - b.playerId,
  )
}
