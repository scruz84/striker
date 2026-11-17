import { render, screen, waitFor } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api/client'
import i18n from '../../i18n'
import { lightTheme } from '../../theme'
import { TeamDetailsPage } from './TeamDetailsPage'

// A dissolved club's page (openspec league-pyramid "Dissolved team"): it reads "Dissolved in
// season N" instead of its division, and offers no friendly challenge.

vi.mock('../../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/client')>()),
  api: { get: vi.fn(), put: vi.fn(), delete: vi.fn(), post: vi.fn() },
}))

const DISSOLVED = 9

beforeEach(async () => {
  await i18n.changeLanguage('en')
  vi.mocked(api.get).mockImplementation((path: string) => {
    if (path === '/v1/auth/me') return Promise.resolve({ id: 1 })
    if (path === '/v1/user/team') return Promise.resolve({ team: { id: 7, name: 'Mine', country: 'es' }, players: [] })
    if (path.startsWith(`/v1/teams/team/${DISSOLVED}/`) && !path.includes('/history') && !path.includes('/matches') && !path.includes('/rivalries')) {
      return Promise.resolve({
        team: { id: DISSOLVED, ownerId: 0, country: 'es', name: 'Old Club', divisionId: -1, divisionName: '', classificationPosition: 0, nameChanged: false, dissolvedSeason: 4 },
        players: [],
        lastMatches: [],
        nextMatches: [],
        fanbase: 0,
      })
    }
    return Promise.resolve([])
  })
})

describe('TeamDetailsPage for a dissolved club', () => {
  it('reads "Dissolved in season N" and offers no challenge', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={client}>
        <ThemeProvider theme={lightTheme}>
          <MemoryRouter initialEntries={[`/team/${DISSOLVED}`]}>
            <Routes>
              <Route path="/team/:id" element={<TeamDetailsPage />} />
            </Routes>
          </MemoryRouter>
        </ThemeProvider>
      </QueryClientProvider>,
    )
    await waitFor(() => expect(screen.getByText(i18n.t('team.dissolved', { season: 4 }))).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: i18n.t('friendlies.challenge.challengeTeam') })).not.toBeInTheDocument()
  })
})
