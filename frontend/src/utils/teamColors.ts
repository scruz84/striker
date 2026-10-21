import { createRng, pick } from '../components/avatar/random'

const BADGE_COLORS = ['#4CAF50', '#2196F3', '#FF9800', '#E91E63', '#9C27B0', '#00BCD4', '#795548', '#3F51B5'] as const

/** The badge color for a team id - also used as the team's color in charts that pit two teams against each other. */
export function teamColor(teamId: number): string {
  return pick(createRng(teamId), BADGE_COLORS)
}

/** Colors for a home/away pair; if the two badge colors collide the away side takes the next palette entry. */
export function matchTeamColors(homeId: number, awayId: number): { home: string; away: string } {
  const home = teamColor(homeId)
  const away = teamColor(awayId)
  if (home !== away) {
    return { home, away }
  }
  return { home, away: BADGE_COLORS[(BADGE_COLORS.indexOf(away as (typeof BADGE_COLORS)[number]) + 1) % BADGE_COLORS.length] }
}
