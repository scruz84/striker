import { useMe, useUserTeam } from '../../api/hooks'

/** The caller's own players currently listed for transfer, as a set of player ids - mirrors
 * useSquadUnavailability, reading from the same cached /v1/user/team query so the squad page and
 * every lineup editor (which only ever edit the caller's own squad) can flag a for-sale player the
 * same way they already flag an injured/suspended one. */
export function useSquadTransferListed(): Set<number> {
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  return new Set(userTeam?.transferListedPlayerIds ?? [])
}
