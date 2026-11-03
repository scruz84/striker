import { cleanup, render, screen, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api/client'
import { RIVALRY_CLASICO, RIVALRY_RIVAL, type TeamMatchSummary } from '../../api/types'
import i18n from '../../i18n'
import { Scoreboard } from '../match/MatchScoreboard'
import { RivalsSection } from './RivalsSection'
import { MatchesList } from './TeamMatchesSection'

// Rivalry visibility (openspec rivalries): the team page's Rivals section, and the Rival /
// Clásico chip on fixture rows and the viewer's scoreboard.

vi.mock('../../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/client')>()),
  api: { get: vi.fn() },
}))

function renderWithClient(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe('rivalries', () => {
  it('lists the Clásico at 22 and the Rival at 9 on the team page', async () => {
    // The backend only returns pairs with a level, highest first (the pair at 5 is left out).
    vi.mocked(api.get).mockResolvedValue([
      { teamId: 2, teamName: 'Old Enemy', score: 22, level: RIVALRY_CLASICO },
      { teamId: 3, teamName: 'Neighbours', score: 9, level: RIVALRY_RIVAL },
    ])
    renderWithClient(<RivalsSection teamId={1} teamName="Us" />)
    const rows = await screen.findAllByTestId('rival-row')
    expect(api.get).toHaveBeenCalledWith('/v1/teams/team/1/rivalries')
    expect(rows).toHaveLength(2)
    expect(within(rows[0]).getByText('Old Enemy')).toBeInTheDocument()
    expect(within(rows[0]).getByText('Clásico')).toBeInTheDocument()
    expect(within(rows[0]).getByText('22 / 40')).toBeInTheDocument()
    expect(within(rows[1]).getByText('Rival')).toBeInTheDocument()
  })

  it('shows nothing for a club without rivals', async () => {
    vi.mocked(api.get).mockResolvedValue([])
    const { container } = renderWithClient(<RivalsSection teamId={1} teamName="Us" />)
    await vi.waitFor(() => expect(api.get).toHaveBeenCalled())
    expect(container.textContent).toBe('')
  })

  it('marks an upcoming Clásico on its fixture row, and nothing on an ordinary one', async () => {
    await i18n.changeLanguage('es')
    const fixture = (matchId: number, rivalry: number): TeamMatchSummary => ({
      kind: 'league',
      matchId,
      opponentId: 9,
      opponentName: 'Rivals FC',
      isHome: true,
      matchDate: '2030-01-01T18:00:00Z',
      played: false,
      homeGoals: 0,
      awayGoals: 0,
      rivalry,
    })
    renderWithClient(<MatchesList matches={[fixture(1, RIVALRY_CLASICO), fixture(2, 0)]} emptyKey="team.matches.noPlayed" />)
    const chips = screen.getAllByTestId('rivalry-chip')
    expect(chips).toHaveLength(1)
    expect(chips[0].textContent).toBe('Clásico')
  })

  it('shows the Rival chip on the match viewer scoreboard', () => {
    renderWithClient(<Scoreboard homeId={1} homeName="A" awayId={2} awayName="B" homeGoals={0} awayGoals={0} status="scheduled" rivalry={RIVALRY_RIVAL} />)
    expect(screen.getByTestId('rivalry-chip').textContent).toBe('Rival')
  })
})
