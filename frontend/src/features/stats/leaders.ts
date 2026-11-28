import type { LeadersCategory, LeagueLeader } from '../../api/types'
import { formatAverageRating } from './competition'

// The figure a leaders list ranks by.
export function leaderValue(category: LeadersCategory, leader: LeagueLeader): string {
  switch (category) {
    case 'goals':
      return String(leader.goals)
    case 'rating':
      return formatAverageRating(leader.averageRating)
    case 'minutes':
      return `${leader.minutes}'`
    case 'cards':
      return `${leader.yellowCards} 🟨 ${leader.redCards} 🟥`
  }
}

