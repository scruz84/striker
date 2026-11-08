import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Player, PreContractOffer } from '../../api/types'
import i18n from '../../i18n'
import { formatCurrency } from '../../utils/formatCurrency'
import { PreContractsSection } from './PreContractCards'

vi.mock('../../api/hooks', () => ({
  useRenewContract: () => ({ mutate: vi.fn(), reset: vi.fn(), isPending: false, isError: false, data: undefined }),
}))

const pending: PreContractOffer = {
  offerId: 5,
  player: { id: 7, teamId: 2, firstName: 'Leo', lastName: 'Vega', wage: 2000, contractYearsRemaining: 1 } as Player,
  fromTeamId: 1,
  fromTeamName: 'Rival FC',
  toTeamId: 2,
  toTeamName: 'Home FC',
  wage: 3300,
  years: 2,
  createdAt: '2026-10-01T10:00:00Z',
  decidesAt: '2026-10-04T10:00:00Z',
  keepWage: 3000,
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(cleanup)

describe('PreContractsSection', () => {
  // pre-contracts "Right to counter-offer": the owning club sees the rival's terms, when the player
  // decides and the wage that keeps him, with a Renew action pre-filled with it.
  it('shows a received offer with the wage that keeps the player', () => {
    render(
      <MemoryRouter>
        <PreContractsSection received={[pending]} sent={[]} onWithdraw={vi.fn()} withdrawPending={false} />
      </MemoryRouter>,
    )
    expect(screen.getByText('Rival FC')).toBeTruthy()
    expect(screen.getByText(`Renew him at ${formatCurrency(3000)}/match to keep him`)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Renew' }))
    expect((screen.getByLabelText('Wage per match') as HTMLInputElement).value.replace(/\D/g, '')).toBe('3000')
  })

  it('shows a signed offer as leaving, without a Renew action', () => {
    render(
      <MemoryRouter>
        <PreContractsSection received={[{ ...pending, signedAt: '2026-10-04T10:00:00Z', decidesAt: undefined, keepWage: undefined }]} sent={[]} onWithdraw={vi.fn()} withdrawPending={false} />
      </MemoryRouter>,
    )
    expect(screen.getByText('Signed with Rival FC: he leaves at the end of the season')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Renew' })).toBeNull()
  })

  it('lets the buyer withdraw a pending offer', () => {
    const onWithdraw = vi.fn()
    render(
      <MemoryRouter>
        <PreContractsSection received={[]} sent={[{ ...pending, keepWage: undefined }]} onWithdraw={onWithdraw} withdrawPending={false} />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Withdraw' }))
    expect(onWithdraw).toHaveBeenCalledWith(expect.objectContaining({ offerId: 5 }))
  })
})
