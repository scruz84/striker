import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { POSITION_MIDFIELD, type Player, type TransferComparablesResult } from '../../api/types'
import i18n from '../../i18n'
import { ListPlayerDialog } from './ListPlayerDialog'

const mocks = vi.hoisted(() => ({
  comparables: vi.fn(),
  mutate: vi.fn(),
}))

vi.mock('../../api/hooks', () => ({
  useListPlayer: () => ({ mutate: mocks.mutate, reset: vi.fn(), isPending: false, isError: false }),
  useTransferComparables: mocks.comparables,
}))

const players = [
  { id: 1, firstName: 'Ana', lastName: 'Gil', wage: 1000, birthCountry: 'ES' },
  { id: 2, firstName: 'Luis', lastName: 'Mora', wage: 2000, birthCountry: 'ES' },
] as Player[]

const emptyResult: TransferComparablesResult = {
  position: POSITION_MIDFIELD,
  age: 26,
  averageSkill: 58,
  broadened: false,
  count: 0,
  comparables: [],
}

function renderDialog() {
  return render(
    <MemoryRouter>
      <ListPlayerDialog open onClose={vi.fn()} players={players} />
    </MemoryRouter>,
  )
}

function pick(name: string) {
  fireEvent.mouseDown(screen.getByRole('combobox'))
  fireEvent.click(within(screen.getByRole('listbox')).getByText(name))
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  mocks.comparables.mockReset()
  mocks.mutate.mockReset()
  mocks.comparables.mockReturnValue({ data: emptyResult, isLoading: false, isError: false })
})
afterEach(cleanup)

describe('ListPlayerDialog', () => {
  it('shows no section and requests nothing until a player is picked', () => {
    renderDialog()
    expect(screen.queryByText('Recent comparable transfers')).toBeNull()
    expect(mocks.comparables).toHaveBeenLastCalledWith(undefined, true)
  })

  it('shows the picked player\'s comparables', () => {
    renderDialog()
    pick('Ana Gil')
    expect(mocks.comparables).toHaveBeenLastCalledWith(1, true)
    expect(screen.getByText('Recent comparable transfers')).toBeTruthy()
    expect(screen.getByText('No similar players have been transferred in the last 60 days')).toBeTruthy()
  })

  it('follows the pick when another player is chosen', () => {
    renderDialog()
    pick('Ana Gil')
    pick('Luis Mora')
    expect(mocks.comparables).toHaveBeenLastCalledWith(2, true)
  })

  it('still lists the player when the comparables fail', () => {
    mocks.comparables.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderDialog()
    pick('Ana Gil')
    expect(screen.getByText('Could not load comparable transfers. You can still list the player.')).toBeTruthy()
    fireEvent.change(screen.getByLabelText('Asking price'), { target: { value: '250000' } })
    fireEvent.click(screen.getByRole('button', { name: 'List' }))
    expect(mocks.mutate).toHaveBeenCalledWith({ playerId: 1, askingPrice: 250000 }, expect.anything())
  })
})
