import { describe, expect, it } from 'vitest'
import { TRANSACTION_SPONSORSHIP, TRANSACTION_TICKET_REVENUE, TRANSACTION_TRANSFER_OUT, TRANSACTION_WAGES, type TeamTransaction } from '../../api/types'
import { computeCashFlowForecast } from './cashFlowForecast'

function tx(overrides: Partial<TeamTransaction>): TeamTransaction {
  return { id: 1, teamId: 1, amount: 0, reason: TRANSACTION_WAGES, description: '', createdAt: '2026-01-01T00:00:00Z', ...overrides }
}

describe('computeCashFlowForecast', () => {
  it('returns null with no transaction history', () => {
    expect(computeCashFlowForecast([], 100000, 200000)).toBeNull()
  })

  it('projects weeks until insolvency when spending outpaces income', () => {
    const transactions = [
      tx({ reason: TRANSACTION_WAGES, amount: -10000, createdAt: '2026-03-01T00:00:00Z' }),
      tx({ reason: TRANSACTION_TICKET_REVENUE, amount: 1000, createdAt: '2026-02-22T00:00:00Z' }),
      tx({ reason: TRANSACTION_SPONSORSHIP, amount: 1000, createdAt: '2026-02-15T00:00:00Z' }),
      tx({ reason: TRANSACTION_TICKET_REVENUE, amount: 1000, createdAt: '2026-02-08T00:00:00Z' }),
      tx({ reason: TRANSACTION_SPONSORSHIP, amount: 1000, createdAt: '2026-02-01T00:00:00Z' }),
    ]

    const forecast = computeCashFlowForecast(transactions, 50000, 200000)

    expect(forecast).not.toBeNull()
    expect(forecast!.weeklyWages).toBe(10000)
    // 4000 income spread over the 3-week span between the oldest and newest of these
    // transactions (Feb 1 to Feb 22) = ~1333/week, well under the 10000/week wage bill.
    expect(forecast!.weeklyIncome).toBeCloseTo(1333, 0)
    expect(forecast!.netWeekly).toBeLessThan(0)
    expect(forecast!.weeksUntilInsolvent).not.toBeNull()
    expect(forecast!.weeksUntilInsolvent).toBeGreaterThan(0)
  })

  it('reports no insolvency projection when income covers wages', () => {
    const transactions = [
      tx({ reason: TRANSACTION_WAGES, amount: -5000, createdAt: '2026-03-01T00:00:00Z' }),
      tx({ reason: TRANSACTION_TICKET_REVENUE, amount: 8000, createdAt: '2026-02-22T00:00:00Z' }),
      tx({ reason: TRANSACTION_SPONSORSHIP, amount: 4000, createdAt: '2026-02-15T00:00:00Z' }),
    ]

    const forecast = computeCashFlowForecast(transactions, 100000, 200000)

    expect(forecast!.netWeekly).toBeGreaterThan(0)
    expect(forecast!.weeksUntilInsolvent).toBeNull()
  })

  it('ignores one-off transfers when estimating recurring weekly income', () => {
    const transactions = [
      tx({ reason: TRANSACTION_WAGES, amount: -5000, createdAt: '2026-03-01T00:00:00Z' }),
      tx({ reason: TRANSACTION_TRANSFER_OUT, amount: -900000, createdAt: '2026-02-28T00:00:00Z' }),
      tx({ reason: TRANSACTION_TICKET_REVENUE, amount: 5000, createdAt: '2026-02-22T00:00:00Z' }),
    ]

    const forecast = computeCashFlowForecast(transactions, 100000, 200000)

    // A one-off 900,000 transfer must never leak into the recurring weekly income estimate.
    expect(forecast!.weeklyIncome).toBeLessThan(10000)
  })

  it('does not project insolvency for a team that is already insolvent', () => {
    const transactions = [tx({ reason: TRANSACTION_WAGES, amount: -5000, createdAt: '2026-03-01T00:00:00Z' })]

    const forecast = computeCashFlowForecast(transactions, -250000, 200000)

    expect(forecast!.weeksUntilInsolvent).toBeNull()
  })
})
