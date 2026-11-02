import { describe, expect, it } from 'vitest'
import { buildPendingItems, type PendingInput } from './pendingItems'

function input(overrides: Partial<PendingInput> = {}): PendingInput {
  return {
    teamId: 7,
    insolvent: false,
    weeksUntilInsolvent: null,
    unavailableStarters: [],
    expiringContracts: 0,
    hasTrainer: true,
    offersReceived: 0,
    challengesReceived: 0,
    academyCanPromote: false,
    ...overrides,
  }
}

const ids = (overrides: Partial<PendingInput>) => buildPendingItems(input(overrides)).map((i) => i.id)

describe('buildPendingItems', () => {
  it('lists nothing when nothing is pending', () => {
    expect(buildPendingItems(input())).toEqual([])
  })

  it('orders by severity, then by the order of the item list', () => {
    expect(ids({ insolvent: true, offersReceived: 1, hasTrainer: false })).toEqual(['insolvent', 'noTrainer', 'offers'])
    expect(
      ids({ academyCanPromote: true, challengesReceived: 2, offersReceived: 1, expiringContracts: 3, hasTrainer: false, weeksUntilInsolvent: 2 }),
    ).toEqual(['forecast', 'expiringContracts', 'noTrainer', 'offers', 'challenges', 'academy'])
  })

  it('names the replacement of a single injured starter and links to the fixture lineup', () => {
    const [item] = buildPendingItems(
      input({
        unavailableStarters: [{ playerId: 3, name: 'Left Back', replacementName: 'Bea B' }],
        nextFixture: { opponentName: 'Rivals', path: '/match/12' },
      }),
    )

    expect(item).toMatchObject({
      id: 'unavailableStarters',
      severity: 'warning',
      textKey: 'dashboard.pending.unavailableStarterReplaced',
      params: { name: 'Left Back', replacement: 'Bea B', opponent: 'Rivals' },
      to: '/match/12',
    })
  })

  it('counts several unavailable starters without naming replacements', () => {
    const [item] = buildPendingItems(
      input({
        unavailableStarters: [
          { playerId: 3, name: 'A', replacementName: 'X' },
          { playerId: 4, name: 'B' },
        ],
        nextFixture: { opponentName: 'Rivals', path: '/match/12' },
      }),
    )

    expect(item.textKey).toBe('dashboard.pending.unavailableStarters')
    expect(item.params).toEqual({ count: 2, names: 'A, B', opponent: 'Rivals' })
  })

  it('shows no unavailable-starters item when only reserves are hurt (no starters given)', () => {
    expect(ids({ unavailableStarters: [], nextFixture: { opponentName: 'Rivals', path: '/match/12' } })).toEqual([])
  })

  it('warns when the forecast reaches the debt ceiling within 4 weeks, not beyond, not when already insolvent', () => {
    expect(ids({ weeksUntilInsolvent: 3 })).toEqual(['forecast'])
    expect(ids({ weeksUntilInsolvent: 4 })).toEqual(['forecast'])
    expect(ids({ weeksUntilInsolvent: 5 })).toEqual([])
    expect(ids({ weeksUntilInsolvent: 3, insolvent: true })).toEqual(['insolvent'])
    expect(buildPendingItems(input({ weeksUntilInsolvent: 3 }))[0].to).toBe('/dashboard/finances')
  })

  it('flags a ticket price 3 or more from the recommendation and opens the price dialog', () => {
    const [item] = buildPendingItems(input({ ticketPrice: 15, recommendedTicketPrice: 11 }))
    expect(item).toMatchObject({ id: 'ticketPrice', severity: 'info', params: { price: 15, recommended: 11 }, to: '/dashboard/finances?ticketPrice=1' })
    expect(ids({ ticketPrice: 13, recommendedTicketPrice: 11 })).toEqual([])
    expect(ids({ ticketPrice: 8, recommendedTicketPrice: 11 })).toEqual(['ticketPrice'])
    expect(ids({ ticketPrice: 15 })).toEqual([])
  })

  it('links a missing trainer to the coaching tab, and claims nothing while it is loading', () => {
    expect(buildPendingItems(input({ hasTrainer: false }))[0].to).toBe('/transfers?tab=coaching')
    expect(ids({ hasTrainer: undefined })).toEqual([])
  })

  it('changes an item\'s content key when its content changes, and never lets insolvency be snoozed', () => {
    const one = buildPendingItems(input({ offersReceived: 1 }))[0]
    const two = buildPendingItems(input({ offersReceived: 2 }))[0]
    expect(one.contentKey).not.toBe(two.contentKey)
    expect(one.snoozable).toBe(true)
    expect(buildPendingItems(input({ insolvent: true }))[0].snoozable).toBe(false)
  })
})
