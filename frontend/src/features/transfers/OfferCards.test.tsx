import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Player, TransferOffer } from '../../api/types'
import { ConfirmContext } from '../../components/confirmDialogContext'
import i18n from '../../i18n'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { ReceivedOfferActions, SentOfferCard } from './OfferCards'

const mocks = vi.hoisted(() => ({
  acceptCounter: vi.fn(),
  declineCounter: vi.fn(),
  counter: vi.fn(),
  confirm: vi.fn(),
}))

vi.mock('../../api/hooks', () => ({
  useAcceptCounterOffer: () => ({ mutate: mocks.acceptCounter, isPending: false, isError: false }),
  useDeclineCounterOffer: () => ({ mutate: mocks.declineCounter, isPending: false, isError: false }),
  useCounterOffer: () => ({ mutate: mocks.counter, reset: vi.fn(), isPending: false, isError: false }),
  useTransferComparables: () => ({ data: undefined, isLoading: false, isError: true }),
}))

const offer: TransferOffer = {
  offerId: 3,
  player: { id: 7, firstName: 'Leo', lastName: 'Vega', wage: 1000 } as Player,
  fromTeamId: 1,
  fromTeamName: 'Buyer FC',
  toTeamId: 2,
  toTeamName: 'Seller FC',
  amount: 800000,
  offeredWage: 1500,
  offeredYears: 2,
  createdAt: '2026-10-01T10:00:00Z',
  acceptableAt: '2026-10-01T12:00:00Z',
}

const countered: TransferOffer = {
  ...offer,
  counterAmount: 1000000,
  counteredAt: '2026-10-02T10:00:00Z',
  counterExpiresAt: '2026-10-05T10:00:00Z',
}

function renderWithConfirm(node: React.ReactNode) {
  return render(
    <MemoryRouter>
      <ConfirmContext.Provider value={mocks.confirm}>{node}</ConfirmContext.Provider>
    </MemoryRouter>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  Object.values(mocks).forEach((m) => m.mockReset())
  mocks.confirm.mockResolvedValue(true)
})
afterEach(cleanup)

describe('SentOfferCard', () => {
  it("shows the buyer the counter fee, when it expires, and Accept and Decline", async () => {
    renderWithConfirm(<SentOfferCard offer={countered} onWithdraw={vi.fn()} withdrawPending={false} />)

    expect(screen.getByText(`Seller FC countered with ${formatCurrency(1000000)}`)).toBeTruthy()
    expect(screen.getByText(`Expires ${formatDateTime('2026-10-05T10:00:00Z')}`)).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Withdraw' })).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Accept counter' }))
    await waitFor(() => expect(mocks.acceptCounter).toHaveBeenCalledWith(3))

    fireEvent.click(screen.getByRole('button', { name: 'Decline' }))
    await waitFor(() => expect(mocks.declineCounter).toHaveBeenCalledWith(3))
  })

  it('offers only Withdraw on an offer that was not countered', () => {
    const onWithdraw = vi.fn()
    renderWithConfirm(<SentOfferCard offer={offer} onWithdraw={onWithdraw} withdrawPending={false} />)
    expect(screen.queryByRole('button', { name: 'Accept counter' })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Withdraw' }))
    expect(onWithdraw).toHaveBeenCalledWith(offer)
  })
})

describe('ReceivedOfferActions', () => {
  it('shows the seller a countered offer as waiting for the buyer', () => {
    renderWithConfirm(<ReceivedOfferActions offer={countered} onAccept={vi.fn()} onReject={vi.fn()} acceptPending={false} rejectPending={false} />)
    expect(screen.getByText(`You countered with ${formatCurrency(1000000)} - waiting for Buyer FC`)).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Accept' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Counter' })).toBeNull()
  })

  it('counters a pending offer with a fee above it', async () => {
    renderWithConfirm(<ReceivedOfferActions offer={offer} onAccept={vi.fn()} onReject={vi.fn()} acceptPending={false} rejectPending={false} />)
    fireEvent.click(screen.getByRole('button', { name: 'Counter' }))

    const field = await screen.findByLabelText('Your counter fee')
    const send = screen.getByRole('button', { name: 'Send counter-offer' }) as HTMLButtonElement
    fireEvent.change(field, { target: { value: '800000' } })
    expect(send.disabled).toBe(true)
    fireEvent.change(field, { target: { value: '1000000' } })
    expect(send.disabled).toBe(false)
    fireEvent.click(send)
    expect(mocks.counter).toHaveBeenCalledWith({ offerId: 3, amount: 1000000 }, expect.anything())
  })
})
