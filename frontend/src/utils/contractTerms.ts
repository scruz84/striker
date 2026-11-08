/** The share of his current wage a player keeps when renewing in his last season (backend's
 * LastSeasonWageFloorShare); with 2 or more seasons left, or when moving club, he keeps it all. */
export const LAST_SEASON_WAGE_FLOOR_SHARE = 0.85

/** The wage below which a player refuses a renewal: his current wage, or 85% of it in his last
 * season. */
export function renewalWageFloor(currentWage: number, contractYearsRemaining: number): number {
  return contractYearsRemaining <= 1 ? Math.ceil(currentWage * LAST_SEASON_WAGE_FLOOR_SHARE) : currentWage
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
