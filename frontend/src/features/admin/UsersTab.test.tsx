import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ConfirmContext } from '../../components/confirmDialogContext'
import { NotifyContext } from '../../components/notifyContext'
import i18n from '../../i18n'
import { setPhone } from './testPhone'
import { UsersTab } from './UsersTab'

const setUserAdmin = vi.hoisted(() => vi.fn())
const deleteUser = vi.hoisted(() => vi.fn())
const notify = { success: vi.fn(), error: vi.fn() }

vi.mock('../../api/hooks', () => ({
  useMe: () => ({ data: { id: '1' } }),
  useAdminUsers: () => ({
    isLoading: false,
    data: [
      { userId: 1, name: 'Ada Admin', email: 'ada@example.com', country: 'es', teamId: null, teamName: '', isAdmin: true, lastSeenAt: new Date().toISOString() },
      { userId: 2, name: 'Bob Manager', email: 'bob@example.com', country: 'en', teamId: null, teamName: '', isAdmin: false, lastSeenAt: new Date(Date.now() - 70 * 86_400_000).toISOString() },
    ],
  }),
  useSetUserAdmin: () => ({ mutate: setUserAdmin, isPending: false }),
  useAdminDeleteUser: () => ({ mutate: deleteUser, isPending: false }),
}))
vi.mock('../../components/TeamNameLink', () => ({ TeamNameLink: ({ children }: { children: string }) => <span>{children}</span> }))

const grant = () => i18n.t('admin.usersTable.grantAdmin')
const revoke = () => i18n.t('admin.usersTable.revokeAdmin')

function renderTab() {
  render(
    <ConfirmContext.Provider value={async () => true}>
      <NotifyContext.Provider value={notify}>
        <UsersTab />
      </NotifyContext.Provider>
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
    await waitFor(() => expect(setUserAdmin).toHaveBeenCalledWith({ userId: 2, isAdmin: true }, expect.any(Object)))
  })

  it('tells the admin when the permission change went through, or failed', async () => {
    setUserAdmin.mockImplementation((_vars: unknown, options: { onSuccess: () => void; onError: () => void }) => options.onSuccess())
    setPhone(true)
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: grant() }))
    await waitFor(() => expect(notify.success).toHaveBeenCalledWith(i18n.t('admin.usersTable.adminGranted')))

    setUserAdmin.mockImplementation((_vars: unknown, options: { onSuccess: () => void; onError: () => void }) => options.onError())
    fireEvent.click(screen.getByRole('button', { name: grant() }))
    await waitFor(() => expect(notify.error).toHaveBeenCalledWith(i18n.t('admin.genericError')))
  })

  it('filters the list as you type, and says when nothing matches', () => {
    setPhone(true)
    renderTab()
    const search = screen.getByRole('searchbox', { name: i18n.t('admin.usersSearch.placeholder') })

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

  it('shows how long ago each user was last active', () => {
    setPhone(false)
    renderTab()
    expect(screen.getByText(i18n.t('admin.usersTable.today'))).toBeInTheDocument()
    expect(screen.getByText(i18n.t('admin.usersTable.lastSeenDays', { count: 70 }))).toBeInTheDocument()
  })

  it('erases another user after confirming, and never offers deleting yourself', async () => {
    setPhone(false)
    renderTab()
    const deleteButtons = screen.getAllByRole('button', { name: i18n.t('admin.usersTable.delete') })
    expect(deleteButtons[0]).toBeDisabled() // the signed-in admin (user 1)
    fireEvent.click(deleteButtons[1])
    await waitFor(() => expect(deleteUser).toHaveBeenCalledWith(2, expect.any(Object)))
  })
})
