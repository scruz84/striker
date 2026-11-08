import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Player } from '../api/types'
import i18n from '../i18n'
import { formatCurrency } from '../utils/formatCurrency'
import { PayReleaseClauseDialog } from './PayReleaseClauseDialog'

const mocks = vi.hoisted(() => ({ pay: vi.fn() }))
vi.mock('../api/hooks', () => ({
  usePayReleaseClause: () => ({ mutate: mocks.pay, reset: vi.fn(), isPending: false, isError: false, data: undefined }),
}))

const player = { id: 7, teamId: 2, firstName: 'Leo', lastName: 'Vega', wage: 2000, releaseClause: 900000 } as Player

beforeEach(async () => {
  await i18n.changeLanguage('en')
  mocks.pay.mockReset()
})
afterEach(cleanup)

// release-clauses "Paying a clause": the dialog pays the clause on the terms offered.
describe('PayReleaseClauseDialog', () => {
  it('pays the clause with the personal terms and no new clause', () => {
    render(<PayReleaseClauseDialog player={player} open onClose={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: `Pay ${formatCurrency(900000)}` }))
    expect(mocks.pay).toHaveBeenCalledWith({ playerId: 7, wage: 2000, years: 1, releaseClause: undefined })
  })
})
