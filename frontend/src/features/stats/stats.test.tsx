import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api/client'
import type { BestXi, BestXiPlayer, ClubRecords, RetiredPlayerDetails, TeamPlayerStats } from '../../api/types'
import i18n from '../../i18n'
import { lightTheme } from '../../theme'
import { PlayerDetailsPage } from '../player/PlayerDetailsPage'
import { BestXiCard } from './BestXiCard'
import { placeBestXi } from './bestXiLayout'
import { ClubRecordsSection } from './ClubRecordsSection'
import { competitionLabel } from './competition'
import { LeagueLeadersCard } from './LeagueLeadersCard'
import { sortTeamStats } from './teamStatsSort'

vi.mock('../../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/client')>()),
  api: { get: vi.fn(), put: vi.fn(), delete: vi.fn(), post: vi.fn() },
}))

const get = vi.mocked(api.get)

function renderAt(element: ReactNode, url = '/', path = '/') {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ThemeProvider theme={lightTheme}>
        <MemoryRouter initialEntries={[url]}>
          <Routes>
            <Route path={path} element={element} />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

const line = { appearances: 10, starts: 8, minutes: 800, goals: 4, penaltyGoals: 1, yellowCards: 2, redCards: 0, averageRating: 7.25, ratedMatches: 10 }

beforeEach(async () => {
  await i18n.changeLanguage('en')
  get.mockReset()
})

describe('retired player profile', () => {
  it('reads the retirement and the kept record instead of a player card', async () => {
    const retired: RetiredPlayerDetails = {
      retired: { id: 5, firstName: 'Old', lastName: 'Legend', bestPosition: 7, age: 37, lastTeamId: 3, lastTeamName: 'Old Club', retiredSeason: 9, retiredAt: '2026-11-01T00:00:00Z' },
      appearances: 300, minutes: 25000, goals: 150, yellowCards: 20, redCards: 1, caps: 12,
      seasonAwards: { goldenBoots: 2, playerOfSeasons: 1 },
      seasonStats: [{ ...line, season: 9, competition: 1, competitionId: 4, competitionName: 'Primera A', teamId: 3, teamName: 'Old Club' }],
      clubHistory: [], transfers: [], bestXi: { weeks: 6, seasons: 1 },
    }
    get.mockImplementation(async (path: string) => {
      if (path === '/v1/teams/player/5/') return retired
      if (path === '/v1/user/team') return { team: { id: 1 } }
      return []
    })
    renderAt(<PlayerDetailsPage />, '/player/5', '/player/:id')
    expect(await screen.findByText(i18n.t('retiredPlayer.retiredInSeason', { season: 9, age: 37 }), { exact: false })).toBeInTheDocument()
    expect(screen.getByText(i18n.t('retiredPlayer.chip'))).toBeInTheDocument()
    expect(screen.getByText('150')).toBeInTheDocument()
    expect(screen.getByText(i18n.t('stats.playerSeasons.title'))).toBeInTheDocument()
    expect(screen.queryByText(i18n.t('team.makeOffer'))).not.toBeInTheDocument()
  })
})

describe('LeagueLeadersCard', () => {
  it('switches category and asks for that list', async () => {
    get.mockImplementation(async (path: string) => {
      const category = new URL(path, 'http://x').searchParams.get('category')
      return {
        season: 3,
        category,
        leaders: [{ ...line, playerId: 1, firstName: 'Top', lastName: category === 'rating' ? 'Rated' : 'Scorer', teamId: 2, teamName: 'Club' }],
      }
    })
    renderAt(<LeagueLeadersCard leagueId="7" />)
    expect(await screen.findByText('Top Scorer')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('stats.leaders.categories.rating') }))
    expect(await screen.findByText('Top Rated')).toBeInTheDocument()
    expect(screen.getByText('7.3')).toBeInTheDocument()
    expect(get).toHaveBeenCalledWith('/v1/teams/division/7/leaders?category=rating')
  })
})

const xiPlayer = (slot: number, line: number, lastName: string): BestXiPlayer => ({
  slot, line, playerId: 100 + slot, firstName: 'P', lastName, teamId: 2, teamName: 'Club', rating: 8, goals: 0, matches: 1,
})

describe('BestXiCard', () => {
  it('places the 4-4-2 on the pitch and steps to the team of the season', async () => {
    const week: BestXi = {
      season: 3, week: 5, weeks: [0, 4, 5],
      players: [xiPlayer(1, 1, 'Keeper'), xiPlayer(2, 2, 'Back'), xiPlayer(6, 3, 'Mid'), xiPlayer(10, 4, 'Nine')],
    }
    const seasonXi: BestXi = { ...week, week: 0, players: [xiPlayer(1, 1, 'SeasonKeeper')] }
    get.mockImplementation(async (path: string) => (path.includes('week=0') ? seasonXi : week))
    renderAt(<BestXiCard leagueId="7" />)
    expect(await screen.findByText('Keeper')).toBeInTheDocument()
    expect(screen.getByText(i18n.t('stats.bestXi.week', { week: 5 }))).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('stats.bestXi.next') }))
    expect(await screen.findByText('SeasonKeeper')).toBeInTheDocument()
    expect(screen.getByText(i18n.t('stats.bestXi.season'))).toBeInTheDocument()
  })

  it('reads an empty state before the first matchday ends', async () => {
    get.mockResolvedValue({ season: 3, week: 0, weeks: [], players: [] })
    renderAt(<BestXiCard leagueId="7" />)
    expect(await screen.findByText(i18n.t('stats.bestXi.empty'))).toBeInTheDocument()
  })

  it('lays each line out on its own row, goalkeeper deepest', () => {
    const placed = placeBestXi([xiPlayer(10, 4, 'A'), xiPlayer(1, 1, 'G'), xiPlayer(2, 2, 'D1'), xiPlayer(3, 2, 'D2')])
    const y = (name: string) => placed.find((p) => p.player.lastName === name)!.y
    expect(placed).toHaveLength(4)
    expect(y('G')).toBeGreaterThan(y('D1'))
    expect(y('D1')).toBe(y('D2'))
    expect(y('D1')).toBeGreaterThan(y('A'))
  })
})

describe('ClubRecordsSection', () => {
  it('shows the records the club holds and leaves the others out', async () => {
    const records: ClubRecords = {
      biggestWin: { kind: 'league', matchId: 9, matchDate: '2026-10-01T18:00:00Z', season: 2, homeTeamId: 1, homeTeamName: 'Us', awayTeamId: 2, awayTeamName: 'Them', homeGoals: 6, awayGoals: 0 },
      longestUnbeatenStreak: { length: 12, from: '2026-09-01T18:00:00Z', to: '2026-11-01T18:00:00Z', ongoing: true },
      topScorers: [{ playerId: 4, firstName: 'Goal', lastName: 'Machine', goals: 80, appearances: 120 }],
      mostAppearances: [],
    }
    get.mockResolvedValue(records)
    renderAt(<ClubRecordsSection teamId="1" />)
    expect(await screen.findByText(i18n.t('stats.records.clubTitle'))).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Us 6-0 Them/ })).toHaveAttribute('href', '/match/9')
    expect(screen.getByText(i18n.t('stats.records.ongoing'))).toBeInTheDocument()
    expect(screen.getByText('Goal Machine')).toBeInTheDocument()
    expect(screen.queryByText(i18n.t('stats.records.biggestDefeat'))).not.toBeInTheDocument()
  })

  it('shows nothing for a club without any record', async () => {
    get.mockResolvedValue({ topScorers: [], mostAppearances: [] })
    renderAt(<ClubRecordsSection teamId="1" />)
    await waitFor(() => expect(get).toHaveBeenCalled())
    expect(screen.queryByText(i18n.t('stats.records.clubTitle'))).not.toBeInTheDocument()
  })
})

describe('helpers', () => {
  it('names a competition in the viewer language', () => {
    const t = i18n.t.bind(i18n)
    expect(competitionLabel(t, { competition: 4, competitionName: 'x', competitionRegion: 'europe' })).toBe(i18n.t('cup.names.europe'))
    expect(competitionLabel(t, { competition: 6, competitionName: '' })).toBe(i18n.t('stats.competitions.clubFriendly'))
    expect(competitionLabel(t, { competition: 2, competitionName: 'Copa' })).toBe(i18n.t('cup.names.national'))
  })

  it('sorts squad stats by the chosen figure, unrated last for the rating', () => {
    const p = (id: number, over: Partial<TeamPlayerStats>): TeamPlayerStats => ({ ...line, averageRating: undefined, playerId: id, firstName: '', lastName: '', inSquad: true, ...over })
    const rows = [p(1, { goals: 2, averageRating: 6 }), p(2, { goals: 5 }), p(3, { goals: 2, averageRating: 7, minutes: 900 })]
    expect(sortTeamStats(rows, 'goals').map((r) => r.playerId)).toEqual([2, 3, 1])
    expect(sortTeamStats(rows, 'rating').map((r) => r.playerId)).toEqual([3, 1, 2])
  })
})
