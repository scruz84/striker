import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Stadium } from '../../api/types'
import '../../i18n'
import { StadiumCard } from './FinancesPage'

const setTicketPrice = vi.hoisted(() => vi.fn())

vi.mock('../../api/hooks', () => {
  const idle = () => ({ mutate: vi.fn(), isPending: false, isSuccess: false, isError: false })
  return {
    useFinances: idle,
    useMe: idle,
    useSetSponsorDeal: idle,
    useUpgradeStadium: idle,
    useRenameStadium: idle,
    useSetTicketPrice: () => ({ mutate: setTicketPrice, isPending: false, isSuccess: false, isError: false }),
  }
})

const stadium = {
  name: 'Estadio Test',
  capacity: 1000,
  ticketPrice: 10,
} as Stadium

function openPriceDialog() {
  render(<StadiumCard stadium={stadium} recommendedTicketPrice={18} maxTicketPrice={39} />)
  fireEvent.click(screen.getByRole('button', { name: /ticket price|precio de entrada/i }))
}

describe('StadiumCard price dialog', () => {
  beforeEach(() => vi.clearAllMocks())

  it('blocks a price above the range and shows the range', () => {
    openPriceDialog()
    const input = screen.getByLabelText(/ticket price|precio de entrada/i, { selector: 'input' })
    fireEvent.change(input, { target: { value: '50' } })

    expect(screen.getByText(/1.*39/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /^(save|guardar)$/i }).hasAttribute('disabled')).toBe(true)
  })

  it('fills the recommended price and allows saving it', () => {
    openPriceDialog()
    fireEvent.click(screen.getByRole('button', { name: /18/ }))

    const input = screen.getByLabelText(/ticket price|precio de entrada/i, { selector: 'input' }) as HTMLInputElement
    expect(input.value).toBe('18')
    const save = screen.getByRole('button', { name: /^(save|guardar)$/i })
    expect(save.hasAttribute('disabled')).toBe(false)
    fireEvent.click(save)
    expect(setTicketPrice).toHaveBeenCalledWith(18, expect.anything())
  })
})
