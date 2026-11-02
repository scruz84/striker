import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { PLAYER_TRANSFER_LISTING, PLAYER_TRANSFER_LOAN, type LeagueTransfer, type Player } from '../../api/types'
import i18n from '../../i18n'
import { RecentLeagueTransfers } from './RecentLeagueTransfers'

const mocks = vi.hoisted(() => ({ transfers: vi.fn() }))

vi.mock('../../api/hooks', () => ({
  useRecentLeagueTransfers: mocks.transfers,
}))

function transfer(overrides: Partial<LeagueTransfer>): LeagueTransfer {
  return {
    player: { id: 1, firstName: 'Ana', lastName: 'Gil', birthCountry: 'es' } as Player,
    amount: 0,
    kind: PLAYER_TRANSFER_LISTING,
    buyerTeamId: 10,
    buyerTeamName: 'Buyer FC',
    sellerTeamId: 20,
    sellerTeamName: 'Seller FC',
    occurredAt: '2026-10-20T12:00:00Z',
    ...overrides,
  }
}

function renderIt(rows: LeagueTransfer[]) {
  mocks.transfers.mockReturnValue({ data: rows, isLoading: false })
  return render(
    <MemoryRouter>
      <RecentLeagueTransfers leagueId="1" />
    </MemoryRouter>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  mocks.transfers.mockReset()
})
afterEach(cleanup)

describe('RecentLeagueTransfers', () => {
  it('shows a free loan with a Loan chip, "on loan from" and Free', () => {
    renderIt([transfer({ kind: PLAYER_TRANSFER_LOAN, amount: 0 })])
    expect(screen.getByText('Loan')).toBeTruthy()
    expect(screen.getByText(/on loan from/)).toBeTruthy()
    expect(screen.getByText('Free')).toBeTruthy()
  })

  it('shows the fee of a loan that carried one', () => {
    renderIt([transfer({ kind: PLAYER_TRANSFER_LOAN, amount: 150000 })])
    expect(screen.getByText('Loan')).toBeTruthy()
    expect(screen.getByText(/150.*000/)).toBeTruthy()
    expect(screen.queryByText('Free')).toBeNull()
  })

  it('shows a permanent transfer with no chip, "from" and the fee', () => {
    renderIt([transfer({ amount: 2000000 })])
    expect(screen.queryByText('Loan')).toBeNull()
    expect(screen.queryByText(/on loan from/)).toBeNull()
    expect(screen.getByText('Buyer FC').closest('span, p')?.textContent).toContain('from')
    expect(screen.getByText(/2.*000.*000/)).toBeTruthy()
  })
})
