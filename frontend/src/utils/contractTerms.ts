/** The shares of his current wage a player keeps when renewing in his last season, from a Naive to
 * a Mastermind negotiator (backend LastSeasonFloorShare); with 2 or more seasons left, or when
 * moving club, he keeps it all. */
export const LAST_SEASON_WAGE_FLOOR_RANGE = [0.75, 0.95] as const

/** The wage below which a player refuses a renewal: his current wage, or - in his last season,
 * where it depends on how shrewd he is - the range from 75% to 95% of it. */
export function renewalWageFloor(currentWage: number, contractYearsRemaining: number): { min: number; max: number } {
  if (contractYearsRemaining > 1) return { min: currentWage, max: currentWage }
  const [low, high] = LAST_SEASON_WAGE_FLOOR_RANGE
  return { min: Math.ceil(currentWage * low), max: Math.ceil(currentWage * high) }
}

/** Whether a club's player is in the last season of his contract: the public marker for another
 * club's player (whose contract years are redacted), or his own contract for the owner. */
export function isLastSeason(player: { teamId: number; contractExpiring?: boolean; contractYearsRemaining: number }): boolean {
  return player.teamId > 0 && (player.contractExpiring === true || player.contractYearsRemaining === 1)
}

/** Mirrors backend PreContractMovePremium: a pre-contract must pay this much more than the
 * player's usual expectation for him to leave his club. */
export const PRE_CONTRACT_MOVE_PREMIUM = 1.1

/** The release clause a contract dialog sends: the typed amount, or undefined for none. */
export function clauseValue(digits: string): number | undefined {
  const amount = Number(digits)
  return digits && amount > 0 ? amount : undefined
}
