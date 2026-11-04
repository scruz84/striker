import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api/client'
import type { DivisionDetails, LeagueSeasons, LeagueSeasonTable, WeekMatch } from '../../api/types'
import i18n from '../../i18n'
import { DivisionDetailsPage } from './DivisionDetailsPage'

vi.mock('../../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/client')>()),
  api: { get: vi.fn() },
}))

const viewport = vi.hoisted(() => ({ desktop: true }))

vi.mock('@mui/material/useMediaQuery', () => ({ default: () => viewport.desktop }))

const TEAMS = 10
const LONG_AGO = '2020-01-01T00:00:00Z'

function weekMatch(week: number, season: number): WeekMatch {
  return {
    matchId: season * 1000 + week,
    divisionId: 7,
    season,
    weekNumber: week,
    matchDate: LONG_AGO,
    played: true,
    inSimulation: false,
    home: 1,
    homeName: `Home ${week} S${season}`,
    homeStadiumName: '',
    away: 2,
    awayName: `Away ${week} S${season}`,
    homeGoals: 4,
    awayGoals: 3,
    events: null,
    matchInstants: null,
    homeLineup: null,
    awayLineup: null,
    homePlayers: null,
    awayPlayers: null,
  }
}

// The current season (3) at matchday 5: Team 1..10 in team id order.
const current: DivisionDetails = {
  id: 7,
  season: 3,
  name: 'First division',
  classification: {
    positions: Array.from({ length: TEAMS }, (_unused, index) => ({
      position: index + 1,
      teamId: index + 1,
      teamName: `Team ${index + 1}`,
      points: 0,
      matches: 5,
      wins: 0,
      draws: 0,
      losses: 0,
      goalsScored: 0,
      goalsReceived: 0,
    })),
  },
  previousWeekMatches: [weekMatch(5, 3)],
  nextWeekMatches: [weekMatch(6, 3)],
  promotionSlots: 0,
  relegationSlots: 2,
} as DivisionDetails

// Season 2's final table: club 10 champion and promoted, clubs 2 and 1 relegated.
const season2: LeagueSeasonTable = {
  id: 7,
  name: 'First division',
  season: 2,
  totalWeeks: (TEAMS - 1) * 2,
  positions: Array.from({ length: TEAMS }, (_unused, index) => ({
    position: index + 1,
    teamId: TEAMS - index,
    name: `Club ${TEAMS - index}`,
    matches: 18,
    points: 40 - index,
    wins: 10,
    draws: 0,
    losses: 8,
    goalsScored: 20,
    goalsReceived: 18,
    promoted: index === 0,
    relegated: index >= TEAMS - 2,
  })),
}

const get = vi.mocked(api.get)

function serve(seasons: LeagueSeasons) {
  get.mockImplementation(async (path: string) => {
    const weeks = /\/weeks\/(\d+)(?:\?season=(\d+))?$/.exec(path)
    if (weeks) return [weekMatch(Number(weeks[1]), Number(weeks[2] ?? 3))]
    if (path.endsWith('/seasons')) return seasons
    if (path.endsWith('/seasons/2')) return season2
    if (path.endsWith('/top-scorers') || path.endsWith('/recent-transfers')) return []
    if (path === '/v1/user/team') return { team: { id: 0 } }
    return current
  })
}

function Location() {
  const location = useLocation()
  return <div data-testid="location">{location.pathname + location.search}</div>
}

function renderPage(url: string) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[url]}>
        <Routes>
          <Route
            path="/division/:id"
            element={
              <>
                <DivisionDetailsPage />
                <Location />
              </>
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const heading = (week: number) => screen.findByRole('heading', { name: i18n.t('division.matchday', { week }) })
const picker = () => screen.queryByRole('combobox', { name: i18n.t('division.seasonPicker') })
const tableRows = () => within(screen.getByRole('table')).getAllByRole('row').slice(1)
const weekCalls = () => get.mock.calls.map(([path]) => path).filter((path) => path.includes('/weeks/'))

describe('DivisionDetailsPage past seasons', () => {
  beforeEach(() => {
    get.mockReset()
    viewport.desktop = true
  })

  it('shows no season picker in a league’s first season', async () => {
    serve({ current: 1, past: [] })
    renderPage('/division/7')

    expect(await screen.findByText('Team 1')).toBeInTheDocument()
    expect(picker()).not.toBeInTheDocument()
  })

  it('picks season 2 into the URL and back to the current season', async () => {
    serve({ current: 3, past: [2, 1] })
    renderPage('/division/7')

    fireEvent.mouseDown(await screen.findByRole('combobox', { name: i18n.t('division.seasonPicker') }))
    const options = screen.getAllByRole('option').map((option) => option.textContent)
    expect(options).toEqual([
      i18n.t('division.seasonCurrent', { season: 3 }),
      i18n.t('division.seasonOption', { season: 2 }),
      i18n.t('division.seasonOption', { season: 1 }),
    ])
    fireEvent.click(screen.getByRole('option', { name: i18n.t('division.seasonOption', { season: 2 }) }))

    expect(await screen.findByText(i18n.t('division.seasonFinal', { season: 2 }))).toBeInTheDocument()
    expect(screen.getByTestId('location')).toHaveTextContent('/division/7?season=2')

    fireEvent.mouseDown(screen.getByRole('combobox', { name: i18n.t('division.seasonPicker') }))
    fireEvent.click(screen.getByRole('option', { name: i18n.t('division.seasonCurrent', { season: 3 }) }))
    expect(await screen.findByText('Team 1')).toBeInTheDocument()
    expect(screen.getByTestId('location').textContent).toBe('/division/7')
  })

  it('browses season 2: final table, last matchdays with scores, no scorers or transfers', async () => {
    serve({ current: 3, past: [2, 1] })
    renderPage('/division/7?season=2')

    expect(await screen.findByText(i18n.t('division.seasonFinal', { season: 2 }))).toBeInTheDocument()
    const rows = tableRows()
    expect(rows).toHaveLength(TEAMS)
    expect(within(rows[0]).getByText('Club 10')).toBeInTheDocument()
    expect(within(rows[9]).getByText('Club 1')).toBeInTheDocument()

    expect(await heading(17)).toBeInTheDocument()
    expect(await heading(18)).toBeInTheDocument()
    expect(weekCalls().sort()).toEqual(['/v1/teams/division/7/weeks/17?season=2', '/v1/teams/division/7/weeks/18?season=2'])
    const card = (await screen.findByText('Home 18 S2')).closest('a')!
    expect(within(card).getByText('4')).toBeInTheDocument()
    expect(screen.queryByText(i18n.t('division.lastWeek'))).not.toBeInTheDocument()
    expect(screen.queryByText(i18n.t('division.nextWeek'))).not.toBeInTheDocument()

    expect(screen.queryByRole('heading', { name: i18n.t('division.topScorers.title') })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: i18n.t('recentTransfers.title') })).not.toBeInTheDocument()
    expect(get.mock.calls.some(([path]) => path.endsWith('/top-scorers') || path.endsWith('/recent-transfers'))).toBe(false)
  })

  it('marks the promoted champion with the trophy and the promotion zone', async () => {
    serve({ current: 3, past: [2, 1] })
    renderPage('/division/7?season=2')

    await screen.findByText(i18n.t('division.seasonFinal', { season: 2 }))
    const rows = tableRows()
    expect(within(rows[0]).getByTitle(i18n.t('division.champion'))).toBeInTheDocument()
    expect(within(rows[1]).queryByTitle(i18n.t('division.champion'))).not.toBeInTheDocument()
    // Both zones come from what happened that season, though the league has no promotion slot now.
    expect(screen.getByText(i18n.t('division.legend.promotion'))).toBeInTheDocument()
    expect(screen.getByText(i18n.t('division.legend.relegation'))).toBeInTheDocument()
  })

  it('shows the current season for a season that is not a past one', async () => {
    serve({ current: 3, past: [2, 1] })
    renderPage('/division/7?season=5')

    expect(await screen.findByText('Team 1')).toBeInTheDocument()
    expect(screen.queryByText(i18n.t('division.seasonFinal', { season: 5 }))).not.toBeInTheDocument()
  })

  it('shows only the Table and Fixtures tabs for a past season on a phone', async () => {
    viewport.desktop = false
    serve({ current: 3, past: [2, 1] })
    renderPage('/division/7?season=2')

    const tabs = await screen.findAllByRole('tab')
    expect(tabs.map((tab) => tab.textContent)).toEqual([i18n.t('division.tabs.table'), i18n.t('division.tabs.fixtures')])
    fireEvent.click(screen.getByRole('tab', { name: i18n.t('division.tabs.fixtures') }))
    expect(await heading(17)).toBeInTheDocument()
  })
})
