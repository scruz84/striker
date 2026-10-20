import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { AdminPage } from './AdminPage'

let me: { data?: { isAdmin: boolean }; isLoading: boolean } = { data: { isAdmin: true }, isLoading: false }
vi.mock('../../api/hooks', () => ({ useMe: () => me }))

// Each tab runs its own queries when mounted, so asserting that only the active tab's component is
// rendered is what guarantees the others' data is never requested.
vi.mock('./OverviewTab', () => ({ OverviewTab: () => <div>overview-content</div> }))
vi.mock('./TeamsTab', () => ({ TeamsTab: () => <div>teams-content</div> }))
vi.mock('./RulesTab', () => ({ RulesTab: () => <div>rules-content</div> }))
vi.mock('./WorldTab', () => ({ WorldTab: () => <div>world-content</div> }))
vi.mock('./UsersTab', () => ({ UsersTab: () => <div>users-content</div> }))

function Search() {
  return <div data-testid="search">{useLocation().search}</div>
}

function renderAt(url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <AdminPage />
      <Search />
    </MemoryRouter>,
  )
}

describe('AdminPage', () => {
  beforeEach(() => {
    me = { data: { isAdmin: true }, isLoading: false }
  })

  it('opens on the overview tab and mounts only that tab', () => {
    renderAt('/admin')
    expect(screen.getByText('overview-content')).toBeInTheDocument()
    for (const other of ['teams', 'rules', 'world', 'users']) {
      expect(screen.queryByText(`${other}-content`)).not.toBeInTheDocument()
    }
  })

  it('opens the tab named in ?tab=', () => {
    renderAt('/admin?tab=users')
    expect(screen.getByText('users-content')).toBeInTheDocument()
    expect(screen.queryByText('overview-content')).not.toBeInTheDocument()
  })

  it('falls back to the overview for an unknown ?tab=', () => {
    renderAt('/admin?tab=bogus')
    expect(screen.getByText('overview-content')).toBeInTheDocument()
  })

  it('switches tab and writes it to the URL, dropping the param for the default tab', () => {
    renderAt('/admin')
    fireEvent.click(screen.getByRole('tab', { name: i18n.t('admin.tabs.world') }))
    expect(screen.getByText('world-content')).toBeInTheDocument()
    expect(screen.queryByText('overview-content')).not.toBeInTheDocument()
    expect(screen.getByTestId('search')).toHaveTextContent('?tab=world')

    fireEvent.click(screen.getByRole('tab', { name: i18n.t('admin.tabs.overview') }))
    expect(screen.getByText('overview-content')).toBeInTheDocument()
    expect(screen.getByTestId('search')).toHaveTextContent('')
  })

  it('shows no tabs to a non-admin', () => {
    me = { data: { isAdmin: false }, isLoading: false }
    renderAt('/admin')
    expect(screen.getByText(i18n.t('admin.noAccess'))).toBeInTheDocument()
    expect(screen.queryByRole('tab')).not.toBeInTheDocument()
  })
})
