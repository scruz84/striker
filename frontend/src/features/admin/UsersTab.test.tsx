import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ConfirmContext } from '../../components/confirmDialogContext'
import i18n from '../../i18n'
import { setPhone } from './testPhone'
import { UsersTab } from './UsersTab'

const setUserAdmin = vi.hoisted(() => vi.fn())

vi.mock('../../api/hooks', () => ({
  useMe: () => ({ data: { id: '1' } }),
  useAdminUsers: () => ({
    isLoading: false,
    data: [
      { userId: 1, name: 'Ada Admin', email: 'ada@example.com', country: 'es', teamId: null, teamName: '', isAdmin: true },
      { userId: 2, name: 'Bob Manager', email: 'bob@example.com', country: 'en', teamId: null, teamName: '', isAdmin: false },
    ],
  }),
  useSetUserAdmin: () => ({ mutate: setUserAdmin, isPending: false }),
}))
vi.mock('../../components/TeamNameLink', () => ({ TeamNameLink: ({ children }: { children: string }) => <span>{children}</span> }))

const grant = () => i18n.t('admin.usersTable.grantAdmin')
const revoke = () => i18n.t('admin.usersTable.revokeAdmin')

function renderTab() {
  render(
    <ConfirmContext.Provider value={async () => true}>
      <UsersTab />
    </ConfirmContext.Provider>,
  )
}

describe('UsersTab', () => {
  beforeEach(() => vi.clearAllMocks())

  it('lists users in a table on a wide screen', () => {
    setPhone(false)
    renderTab()
    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(screen.getByText('bob@example.com')).toBeInTheDocument()
  })

  it('stacks users as cards on a phone, with no table to scroll sideways', () => {
    setPhone(true)
    renderTab()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByText('Ada Admin')).toBeInTheDocument()
    expect(screen.getByText('bob@example.com')).toBeInTheDocument()
  })

  it('grants admin from a phone card after confirming', async () => {
    setPhone(true)
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: grant() }))
    await waitFor(() => expect(setUserAdmin).toHaveBeenCalledWith({ userId: 2, isAdmin: true }))
  })

  it('filters the list as you type, and says when nothing matches', () => {
    setPhone(true)
    renderTab()
    const search = screen.getByRole('textbox', { name: i18n.t('admin.usersSearch.placeholder') })

    fireEvent.change(search, { target: { value: 'bob' } })
    expect(screen.getByText('Bob Manager')).toBeInTheDocument()
    expect(screen.queryByText('Ada Admin')).not.toBeInTheDocument()

    fireEvent.change(search, { target: { value: 'nobody' } })
    expect(screen.getByText(i18n.t('admin.usersSearch.none', { query: 'nobody' }))).toBeInTheDocument()
    expect(screen.queryByText('Bob Manager')).not.toBeInTheDocument()
  })

  it("won't let you revoke your own admin access", () => {
    setPhone(true)
    renderTab()
    expect(screen.getByRole('button', { name: revoke() })).toBeDisabled()
  })
})
