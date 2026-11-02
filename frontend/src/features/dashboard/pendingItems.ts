// The rules behind the dashboard's Pending panel (see openspec team-dashboard, "Pending panel"):
// which things need the owner's action, how severe each is and where its action leads. Pure, so
// every scenario is a unit test and PendingPanel only renders.

/** Weeks of runway under which the cash-flow forecast raises a warning. */
export const FORECAST_WARNING_WEEKS = 4
/** How far the ticket price may sit from the recommended one before it is flagged. */
export const TICKET_PRICE_GAP = 3
// The last value a rostered player's contractYearsRemaining ever shows before AdvancePlayerContract
// releases him to free agency at season end (see backend common.Player.ContractYearsRemaining) - by
// the time it reads 1, there is exactly one season left to renew or lose him for nothing.
export const CONTRACT_EXPIRY_WARNING_YEARS = 1

export type PendingSeverity = 'error' | 'warning' | 'info' | 'success'

const SEVERITY_RANK: Record<PendingSeverity, number> = { error: 0, warning: 1, info: 2, success: 3 }

export interface PendingItem {
  id: string
  severity: PendingSeverity
  textKey: string
  params: Record<string, string | number>
  /** i18n key of the single action button. */
  actionKey: string
  /** Where the action leads (a router path, query string included). */
  to: string
  /** Changes whenever the item's content changes, which is what un-snoozes it. */
  contentKey: string
  /** Everything but an error can be snoozed until tomorrow. */
  snoozable: boolean
}

export interface UnavailableStarter {
  playerId: number
  name: string
  /** Who the pre-kickoff refill would put in his slot, when the squad has someone. */
  replacementName?: string
}

export interface PendingInput {
  teamId: number
  insolvent: boolean
  /** Weeks until the budget reaches -maxDebt on the current trend, null when it never does. */
  weeksUntilInsolvent: number | null
  /** Starters of the next fixture's lineup who are injured or suspended for it. */
  unavailableStarters: UnavailableStarter[]
  /** The next fixture, when there is one: names the opponent and where its lineup is edited. */
  nextFixture?: { opponentName: string; path: string }
  expiringContracts: number
  /** undefined while the trainer is still loading: nothing is claimed until it is known. */
  hasTrainer: boolean | undefined
  ticketPrice?: number
  recommendedTicketPrice?: number
  offersReceived: number
  challengesReceived: number
  academyCanPromote: boolean
}

export function buildPendingItems(input: PendingInput): PendingItem[] {
  const items: PendingItem[] = []

  if (input.insolvent) {
    items.push({
      id: 'insolvent',
      severity: 'error',
      textKey: 'dashboard.insolvent',
      params: {},
      actionKey: 'dashboard.viewFinances',
      to: '/dashboard/finances',
      contentKey: 'insolvent',
      snoozable: false,
    })
  } else if (input.weeksUntilInsolvent !== null && input.weeksUntilInsolvent <= FORECAST_WARNING_WEEKS) {
    items.push({
      id: 'forecast',
      severity: 'warning',
      textKey: 'dashboard.pending.forecast',
      params: { count: input.weeksUntilInsolvent },
      actionKey: 'dashboard.viewFinances',
      to: '/dashboard/finances',
      contentKey: `forecast:${input.weeksUntilInsolvent}`,
      snoozable: true,
    })
  }

  if (input.unavailableStarters.length > 0 && input.nextFixture) {
    const [first] = input.unavailableStarters
    const single = input.unavailableStarters.length === 1
    const replaced = single && first.replacementName !== undefined
    items.push({
      id: 'unavailableStarters',
      severity: 'warning',
      textKey: replaced ? 'dashboard.pending.unavailableStarterReplaced' : 'dashboard.pending.unavailableStarters',
      params: replaced
        ? { name: first.name, replacement: first.replacementName as string, opponent: input.nextFixture.opponentName }
        : {
            count: input.unavailableStarters.length,
            names: input.unavailableStarters.map((s) => s.name).join(', '),
            opponent: input.nextFixture.opponentName,
          },
      actionKey: 'dashboard.pending.setLineup',
      to: input.nextFixture.path,
      contentKey: `starters:${input.nextFixture.path}:${input.unavailableStarters.map((s) => s.playerId).sort((a, b) => a - b).join(',')}`,
      snoozable: true,
    })
  }

  if (input.expiringContracts > 0) {
    items.push({
      id: 'expiringContracts',
      severity: 'warning',
      textKey: 'dashboard.contractsExpiring',
      params: { count: input.expiringContracts },
      actionKey: 'dashboard.viewTeam',
      to: `/team/${input.teamId}`,
      contentKey: `contracts:${input.expiringContracts}`,
      snoozable: true,
    })
  }

  if (input.hasTrainer === false) {
    items.push({
      id: 'noTrainer',
      severity: 'warning',
      textKey: 'dashboard.pending.noTrainer',
      params: {},
      actionKey: 'dashboard.pending.hireTrainer',
      to: '/transfers?tab=coaching',
      contentKey: 'noTrainer',
      snoozable: true,
    })
  }

  if (
    input.ticketPrice !== undefined &&
    input.recommendedTicketPrice !== undefined &&
    Math.abs(input.ticketPrice - input.recommendedTicketPrice) >= TICKET_PRICE_GAP
  ) {
    items.push({
      id: 'ticketPrice',
      severity: 'info',
      textKey: 'dashboard.pending.ticketPrice',
      params: { price: input.ticketPrice, recommended: input.recommendedTicketPrice },
      actionKey: 'dashboard.pending.adjustPrice',
      to: '/dashboard/finances?ticketPrice=1',
      contentKey: `price:${input.ticketPrice}:${input.recommendedTicketPrice}`,
      snoozable: true,
    })
  }

  if (input.offersReceived > 0) {
    items.push({
      id: 'offers',
      severity: 'info',
      textKey: 'dashboard.offersReceived',
      params: { count: input.offersReceived },
      actionKey: 'dashboard.viewOffers',
      to: '/transfers?tab=offers',
      contentKey: `offers:${input.offersReceived}`,
      snoozable: true,
    })
  }

  if (input.challengesReceived > 0) {
    items.push({
      id: 'challenges',
      severity: 'info',
      textKey: 'dashboard.friendlyChallengesReceived',
      params: { count: input.challengesReceived },
      actionKey: 'dashboard.viewChallenges',
      to: '/friendlies',
      contentKey: `challenges:${input.challengesReceived}`,
      snoozable: true,
    })
  }

  if (input.academyCanPromote) {
    items.push({
      id: 'academy',
      severity: 'success',
      textKey: 'team.academy.available',
      params: {},
      actionKey: 'dashboard.viewTeam',
      to: `/team/${input.teamId}`,
      contentKey: 'academy',
      snoozable: true,
    })
  }

  // Array.prototype.sort is stable, so items of one severity keep the order they were added in.
  return items.sort((a, b) => SEVERITY_RANK[a.severity] - SEVERITY_RANK[b.severity])
}
