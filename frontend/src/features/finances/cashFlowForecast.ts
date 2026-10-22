import {
  TRANSACTION_SPONSORSHIP,
  TRANSACTION_STADIUM_MAINTENANCE,
  TRANSACTION_TICKET_REVENUE,
  TRANSACTION_TRAINER_WAGES,
  TRANSACTION_WAGES,
  type TeamTransaction,
} from '../../api/types'

const MILLISECONDS_PER_WEEK = 7 * 24 * 60 * 60 * 1000

export interface CashFlowForecast {
  weeklyWages: number
  weeklyIncome: number
  netWeekly: number
  // Weeks until budget reaches -maxDebt at the current net weekly rate - only set when netWeekly
  // is negative and the team isn't already insolvent (see FinancesPage's own insolvent banner for
  // that case).
  weeksUntilInsolvent: number | null
}

// computeCashFlowForecast estimates a team's ongoing weekly cash flow from its recent transaction
// history alone (see openspec/specs/team-finances, "Financial planning tools") - no separate wage-bill/attendance endpoint
// needed, since the numbers those would return are already implicit in the transactions
// PayMatchdayWages/creditTicketRevenue/awardSponsorship already write.
//
// weeklyWages is the average of WAGES, TRAINER_WAGES, and STADIUM_MAINTENANCE transactions over
// however many weeks the visible window spans, the same way weeklyIncome already averages
// ticket/sponsorship - all three are charged once per league match now (see
// engine.PayMatchdayWages), not once a calendar week, so a league scheduled to play more than
// once a week would otherwise make reading the single most recent transaction of each as "the
// weekly bill" undercount how often it actually recurs. Stadium maintenance isn't a wage at all,
// but it's folded into the same bucket here rather than given its own line, since both are
// recurring per-round expenses this forecast nets against income the same way.
// weeklyIncome is the true average of TICKET_REVENUE and SPONSORSHIP transactions (the two
// genuinely recurring, roughly-weekly income sources) over however many weeks the visible
// transaction window actually spans - deliberately excluding once-a-season lumps (prize money,
// season tickets, merchandising) and transfers, which would otherwise wildly skew a "weekly"
// estimate depending on when in the season this happens to run.
export function computeCashFlowForecast(transactions: TeamTransaction[], budget: number, maxDebt: number): CashFlowForecast | null {
  if (transactions.length === 0) {
    return null
  }

  const wageTransactions = transactions.filter(
    (tx) => tx.reason === TRANSACTION_WAGES || tx.reason === TRANSACTION_TRAINER_WAGES || tx.reason === TRANSACTION_STADIUM_MAINTENANCE,
  )
  const weeklyWages = Math.abs(averagePerWeek(wageTransactions))

  const recurringIncome = transactions.filter((tx) => tx.reason === TRANSACTION_TICKET_REVENUE || tx.reason === TRANSACTION_SPONSORSHIP)
  const weeklyIncome = averagePerWeek(recurringIncome)

  const netWeekly = weeklyIncome - weeklyWages

  let weeksUntilInsolvent: number | null = null
  if (netWeekly < 0 && budget >= -maxDebt) {
    weeksUntilInsolvent = Math.max(0, Math.ceil((budget - -maxDebt) / -netWeekly))
  }

  return { weeklyWages, weeklyIncome, netWeekly, weeksUntilInsolvent }
}

// averagePerWeek sums every transaction's amount and divides by however many weeks separate the
// oldest and the most recent one in the list (at least 1, so a single data point isn't divided by
// zero) - not by the transaction count, since ticket revenue only lands on home matches.
function averagePerWeek(transactions: TeamTransaction[]): number {
  if (transactions.length === 0) {
    return 0
  }
  const timestamps = transactions.map((tx) => new Date(tx.createdAt).getTime())
  const spanWeeks = Math.max(1, (Math.max(...timestamps) - Math.min(...timestamps)) / MILLISECONDS_PER_WEEK)
  const total = transactions.reduce((sum, tx) => sum + tx.amount, 0)
  return total / spanWeeks
}
