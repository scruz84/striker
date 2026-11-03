import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../api/client'
import type { Player, WatchlistEntry } from '../api/types'
import i18n from '../i18n'
import { WatchlistTab } from '../features/transfers/WatchlistTab'
import { WatchToggle } from './WatchToggle'

// The watchlist (openspec player-watchlist): every star reads the same cached list, a toggle
// calls the API and moves every star showing that player, and the Watchlist tab shows each
// followed player's market status with its action.

vi.mock('../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api/client')>()),
  api: { get: vi.fn(), put: vi.fn(), delete: vi.fn(), post: vi.fn() },
}))
vi.mock('../components/confirmDialogContext', () => ({ useConfirm: () => () => Promise.resolve(true) }))

const OWN_TEAM = 7
const striker = { id: 42, teamId: 3, firstName: 'Leo', lastName: 'Gol', age: 24, birthCountry: 'es' } as Player
const ownPlayer = { id: 5, teamId: OWN_TEAM, firstName: 'Own', lastName: 'Man', age: 30, birthCountry: 'es' } as Player

let watchlist: WatchlistEntry[] = []

beforeEach(async () => {
  await i18n.changeLanguage('en')
  watchlist = []
  vi.mocked(api.get).mockImplementation((path: string) => {
    if (path === '/v1/auth/me') return Promise.resolve({ id: 1 })
    if (path === '/v1/user/team') return Promise.resolve({ team: { id: OWN_TEAM }, players: [] })
    if (path === '/v1/user/watchlist') return Promise.resolve(watchlist)
    return Promise.reject(new Error(`unexpected GET ${path}`))
  })
  vi.mocked(api.put).mockImplementation((path: string) => {
    if (path === `/v1/user/watchlist/${striker.id}`) {
      watchlist = [{ player: striker, teamName: 'Rivals FC', followedAt: '2026-11-03T10:00:00Z' }]
    }
    return Promise.resolve({})
  })
  vi.mocked(api.delete).mockImplementation(() => {
    watchlist = []
    return Promise.resolve({})
  })
})
afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

function renderWithClient(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('WatchToggle', () => {
  it('follows and unfollows through the API, moving every star of that player', async () => {
    renderWithClient(
      <>
        <WatchToggle player={striker} />
        <WatchToggle player={striker} />
      </>,
    )
    const stars = await screen.findAllByRole('button', { name: 'Follow Leo Gol' })
    expect(stars).toHaveLength(2)

    fireEvent.click(stars[0])
    await waitFor(() => expect(screen.getAllByRole('button', { name: 'Unfollow Leo Gol' })).toHaveLength(2))
    expect(api.put).toHaveBeenCalledWith('/v1/user/watchlist/42')
    expect(screen.getAllByRole('button', { name: 'Unfollow Leo Gol' })[1]).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(screen.getAllByRole('button', { name: 'Unfollow Leo Gol' })[1])
    await waitFor(() => expect(screen.getAllByRole('button', { name: 'Follow Leo Gol' })).toHaveLength(2))
    expect(api.delete).toHaveBeenCalledWith('/v1/user/watchlist/42')
  })

  it('is never shown on one of the manager\'s own players', async () => {
    renderWithClient(
      <>
        <WatchToggle player={ownPlayer} />
        <WatchToggle player={striker} />
      </>,
    )
    await screen.findByRole('button', { name: 'Follow Leo Gol' })
    expect(screen.queryByRole('button', { name: 'Follow Own Man' })).toBeNull()
  })

  it('cannot follow a 31st player', async () => {
    watchlist = Array.from({ length: 30 }, (_, i) => ({
      player: { ...striker, id: 1000 + i },
      followedAt: '2026-11-03T10:00:00Z',
    }))
    renderWithClient(<WatchToggle player={striker} />)
    expect(await screen.findByRole('button', { name: 'Follow Leo Gol' })).toBeDisabled()
  })
})

describe('WatchlistTab', () => {
  it('shows a listed player as "Transfer listed" with his price and a Buy action', async () => {
    watchlist = [
      {
        player: striker,
        teamName: 'Rivals FC',
        followedAt: '2026-11-03T10:00:00Z',
        transferListing: { listingId: 9, player: striker, sellingTeamId: 3, sellingTeamName: 'Rivals FC', askingPrice: 500000, listedAt: '', caps: 0 },
      },
      { player: { ...striker, id: 43, teamId: 0, firstName: 'Free', lastName: 'Kick' }, followedAt: '2026-11-03T10:00:00Z' },
    ]
    renderWithClient(<WatchlistTab />)
    const rows = await screen.findAllByTestId('watchlist-row')
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toMatch(/Transfer listed · .*500/)
    expect(rows[1].textContent).toContain('Free agent')

    fireEvent.click(screen.getByRole('button', { name: 'Buy' }))
    await waitFor(() => expect(api.post).toHaveBeenCalled())
    expect(screen.getByRole('button', { name: 'Sign' })).toBeInTheDocument()
  })
})
