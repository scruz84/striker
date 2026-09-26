import { useMe, useUserTeam } from '../../api/hooks'
import { UNAVAILABLE_SUSPENSION, type PlayerUnavailability } from '../../api/types'

/** The caller's own suspended/injured players, keyed by player id - every lineup editor (default
 * or per-match) only ever edits the caller's own squad, so this reads straight from the same
 * cached /v1/user/team query those pages already load their players from.
 *
 * A red-card suspension carries no sanction into a friendly (see backend
 * common.GetUnavailablePlayerIdsForFriendly/engine.DisciplineSimulation.Apply), so the friendly
 * lineup editor (isFriendly=true) must not flag or block a suspended-only player either -
 * otherwise it would show a warning icon and refuse a starter the backend will happily field.
 * Injuries still apply everywhere, friendly included. */
export function useSquadUnavailability(isFriendly = false): Map<number, PlayerUnavailability> {
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const unavailability = (userTeam?.unavailability ?? []).filter(
    (u) => !isFriendly || u.reason !== UNAVAILABLE_SUSPENSION,
  )
  return new Map(unavailability.map((u) => [u.playerId, u]))
}
