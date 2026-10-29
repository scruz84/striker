import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { POSITION_MIDFIELD, type Player, type TransferComparablesResult } from '../api/types'
import i18n from '../i18n'
import { ListPlayerForTransferDialog } from './ListPlayerForTransferDialog'

const mocks = vi.hoisted(() => ({
  comparables: vi.fn(),
  mutate: vi.fn(),
}))

vi.mock('../api/hooks', () => ({
  useListPlayer: () => ({ mutate: mocks.mutate, reset: vi.fn(), isPending: false, isError: false }),
  useTransferComparables: mocks.comparables,
}))

const player = { id: 7, firstName: 'Leo', lastName: 'Vega', wage: 1000 } as Player

const result: TransferComparablesResult = {
  position: POSITION_MIDFIELD,
  age: 26,
  averageSkill: 58,
  broadened: false,
  count: 0,
  comparables: [],
}

function renderDialog(open: boolean) {
  return render(
    <MemoryRouter>
      <ListPlayerForTransferDialog player={player} open={open} onClose={vi.fn()} />
    </MemoryRouter>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  mocks.comparables.mockReset()
  mocks.mutate.mockReset()
  mocks.comparables.mockReturnValue({ data: result, isLoading: false, isError: false })
})
afterEach(cleanup)

describe('ListPlayerForTransferDialog', () => {
  it('only asks for comparables while the dialog is open', () => {
    const { rerender } = renderDialog(false)
    expect(mocks.comparables).toHaveBeenLastCalledWith(7, false)
    rerender(
      <MemoryRouter>
        <ListPlayerForTransferDialog player={player} open onClose={vi.fn()} />
      </MemoryRouter>,
    )
    expect(mocks.comparables).toHaveBeenLastCalledWith(7, true)
  })

  it('shows the comparables section below the asking price', () => {
    renderDialog(true)
    const price = screen.getByLabelText('Asking price')
    const section = screen.getByText('Recent comparable transfers')
    expect(price.compareDocumentPosition(section) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('still lists the player at the typed price when comparables fail', () => {
    mocks.comparables.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderDialog(true)
    expect(screen.getByText('Could not load comparable transfers. You can still list the player.')).toBeTruthy()
    fireEvent.change(screen.getByLabelText('Asking price'), { target: { value: '250000' } })
    fireEvent.click(screen.getByRole('button', { name: 'List' }))
    expect(mocks.mutate).toHaveBeenCalledWith({ playerId: 7, askingPrice: 250000 }, expect.anything())
  })
})
