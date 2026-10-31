import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { DeleteAccountDialog } from './DeleteAccountDialog'

const mutate = vi.hoisted(() => vi.fn())
vi.mock('../../api/hooks', () => ({
  useDeleteAccount: () => ({ mutate, isPending: false, isError: false }),
}))

function renderDialog(teamName: string) {
  render(
    <MemoryRouter>
      <DeleteAccountDialog open onClose={vi.fn()} teamName={teamName} />
    </MemoryRouter>,
  )
}
const deleteButton = () => screen.getByRole('button', { name: i18n.t('deleteAccount.confirm') })
const type = (text: string) => fireEvent.change(screen.getByLabelText(i18n.t('deleteAccount.inputLabel')), { target: { value: text } })

describe('DeleteAccountDialog', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('enables deletion only when the club name is typed exactly', () => {
    renderDialog('CD Norte')
    expect(screen.getByText(/CD Norte/)).toBeInTheDocument()
    expect(deleteButton()).toBeDisabled()
    type('CD Nort')
    expect(deleteButton()).toBeDisabled()
    type('CD Norte')
    expect(deleteButton()).toBeEnabled()
    fireEvent.click(deleteButton())
    expect(mutate).toHaveBeenCalledWith('CD Norte', expect.any(Object))
  })

  it('asks for DELETE when the user has no club', () => {
    renderDialog('')
    expect(screen.getByText(i18n.t('deleteAccount.confirmNoClub'))).toBeInTheDocument()
    type('nope')
    expect(deleteButton()).toBeDisabled()
    type('delete')
    expect(deleteButton()).toBeEnabled()
  })
})
