import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { LoggedUser } from '../api/types'
import { stubViewportWidth } from '../testUtils/viewport'
import { Layout } from './Layout'

const me = vi.hoisted(() => ({ data: undefined as Partial<LoggedUser> | null | undefined }))

vi.mock('../api/hooks', () => ({
  useMe: () => ({ data: me.data, isError: false, isFetching: false, refetch: vi.fn() }),
  useUserTeam: () => ({ data: undefined }),
  useDevLoginUsers: () => ({ data: [] }),
}))
vi.mock('./Header', () => ({ Header: () => null }))
vi.mock('./NavDrawer', () => ({ NavDrawer: ({ variant }: { variant: string }) => <div>nav drawer {variant}</div> }))
vi.mock('./NavRail', () => ({ NavRail: () => <div>nav rail</div> }))
vi.mock('./BottomNav', () => ({ BottomNav: () => <div>bottom nav</div>, BOTTOM_NAV_HEIGHT: 80 }))
vi.mock('./Subheader', () => ({ Subheader: () => null }))
vi.mock('./DetailBackBar', () => ({ DetailBackBar: () => null }))

function renderAt(path: string, user: Partial<LoggedUser> | null | undefined) {
  me.data = user
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/dashboard" element={<div>dashboard page</div>} />
          <Route path="/accept-terms" element={<div>accept terms page</div>} />
          <Route path="/onboarding" element={<div>onboarding page</div>} />
          <Route path="/privacy" element={<div>privacy page</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  )
}

describe('Layout terms gate', () => {
  it('sends a user who has not accepted the terms to /accept-terms before anything else', () => {
    renderAt('/dashboard', { id: '1', termsAccepted: false, pendingCountrySelection: false })
    expect(screen.getByText('accept terms page')).toBeInTheDocument()
  })

  it('puts accepting the terms before onboarding for a brand-new sign-in', () => {
    renderAt('/dashboard', { id: '1', termsAccepted: false, pendingCountrySelection: true })
    expect(screen.getByText('accept terms page')).toBeInTheDocument()
    expect(screen.queryByText('onboarding page')).not.toBeInTheDocument()
  })

  it('still sends an accepted but club-less user to onboarding', () => {
    renderAt('/dashboard', { id: '1', termsAccepted: true, pendingCountrySelection: true })
    expect(screen.getByText('onboarding page')).toBeInTheDocument()
  })

  it('does not bounce a brand-new sign-in between the terms and onboarding gates', () => {
    renderAt('/accept-terms', { id: '1', termsAccepted: false, pendingCountrySelection: true })
    expect(screen.getByText('accept terms page')).toBeInTheDocument()
  })

  it('lets a club-less user read the legal texts', () => {
    renderAt('/privacy', { id: '1', termsAccepted: true, pendingCountrySelection: true })
    expect(screen.getByText('privacy page')).toBeInTheDocument()
  })

  it('lets the legal texts be read before accepting', () => {
    renderAt('/privacy', { id: '1', termsAccepted: false, pendingCountrySelection: false })
    expect(screen.getByText('privacy page')).toBeInTheDocument()
  })

  it('does not interfere for a user who accepted, or for a visitor', () => {
    renderAt('/dashboard', { id: '1', termsAccepted: true, pendingCountrySelection: false })
    expect(screen.getByText('dashboard page')).toBeInTheDocument()
  })

  it('leaves visitors alone', () => {
    renderAt('/privacy', null)
    expect(screen.getByText('privacy page')).toBeInTheDocument()
  })
})

// app-navigation "Navigation layout by window width".
describe('Layout navigation', () => {
  afterEach(() => vi.unstubAllGlobals())
  const player = { id: '1', termsAccepted: true, pendingCountrySelection: false }

  it('uses the navigation bar on a phone (Phone)', () => {
    stubViewportWidth(360)
    renderAt('/dashboard', player)
    expect(screen.getByText('bottom nav')).toBeInTheDocument()
    expect(screen.queryByText('nav rail')).not.toBeInTheDocument()
    expect(screen.getByText('nav drawer modal')).toBeInTheDocument()
  })

  it('uses the rail on a laptop (Laptop)', () => {
    stubViewportWidth(1024)
    renderAt('/dashboard', player)
    expect(screen.getByText('nav rail')).toBeInTheDocument()
    expect(screen.queryByText('bottom nav')).not.toBeInTheDocument()
    expect(screen.getByText('nav drawer modal')).toBeInTheDocument()
  })

  it('uses the standard drawer on a desktop (Desktop)', () => {
    stubViewportWidth(1280)
    renderAt('/dashboard', player)
    expect(screen.getByText('nav drawer standard')).toBeInTheDocument()
    expect(screen.queryByText('nav rail')).not.toBeInTheDocument()
    expect(screen.queryByText('bottom nav')).not.toBeInTheDocument()
  })

  it('shows no bar or rail to a user still choosing a country', () => {
    for (const width of [360, 1024]) {
      stubViewportWidth(width)
      renderAt('/onboarding', { ...player, pendingCountrySelection: true })
      expect(screen.queryByText('bottom nav')).not.toBeInTheDocument()
      expect(screen.queryByText('nav rail')).not.toBeInTheDocument()
    }
  })
})
