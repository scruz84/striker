import { useMe, useUserTeam } from '../../api/hooks'
import type { PlayerUnavailability } from '../../api/types'

/** The caller's own suspended/injured players, keyed by player id - every lineup editor (default
 * or per-match) only ever edits the caller's own squad, so this reads straight from the same
 * cached /v1/user/team query those pages already load their players from. */
export function useSquadUnavailability(): Map<number, PlayerUnavailability> {
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  return new Map((userTeam?.unavailability ?? []).map((u) => [u.playerId, u]))
}
