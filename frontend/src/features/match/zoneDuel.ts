// Pure formatting for one cell of the zone duel (see ZoneComparisonRows' DuelCell): each side's
// share of the duel as a whole percentage, its raw strength rounded to a whole number under it, and
// a tooltip that keeps both raw values to one decimal.

/** Home's and away's share of a duel, in percent. Both at zero (nobody in that zone) is an even 50/50. */
export function duelShares(home: number, away: number): { home: number; away: number } {
  const total = home + away
  const homeShare = total > 0 ? (home / total) * 100 : 50
  return { home: homeShare, away: 100 - homeShare }
}

/** The raw strength printed under a percentage: rounded to the nearest whole number. */
export function formatStrength(value: number): string {
  return String(Math.round(value))
}

/** The tooltip text for a cell: the matchup caption plus both raw values, always to one decimal. */
export function duelTooltip(caption: string, home: number, away: number): string {
  return `${caption}: ${home.toFixed(1)} - ${away.toFixed(1)}`
}
