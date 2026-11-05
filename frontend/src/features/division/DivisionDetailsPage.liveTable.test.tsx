import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api/client'
import type { ClassificationPosition, DivisionDetails, LeagueLiveScore, LeagueSeasons, WeekMatch } from '../../api/types'
import i18n from '../../i18n'
import { DivisionDetailsPage } from './DivisionDetailsPage'

vi.mock('../../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/client')>()),
  api: { get: vi.fn() },
}))

const viewport = vi.hoisted(() => ({ desktop: true }))

vi.mock('@mui/material/useMediaQuery', () => ({ default: () => viewport.desktop }))

const minutesAgo = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString()

// Six teams before matchday 12, all with 11 matches: T1 30 pts, T2 25, T3 20, T4 15, T5 14, T6 5.
const PRE_POINTS = [30, 25, 20, 15, 14, 5]

function preRow(teamId: number): ClassificationPosition {
  return {
    position: teamId,
    teamId,
    teamName: `Team ${teamId}`,
    points: PRE_POINTS[teamId - 1],
    matches: 11,
    wins: 0,
    draws: 0,
    losses: 0,
    goalsScored: 0,
    goalsReceived: 0,
  }
}

// What the backend table says once the stored final result of a team's match is counted.
function withStoredResult(row: ClassificationPosition, goalsFor: number, goalsAgainst: number): ClassificationPosition {
  const won = goalsFor > goalsAgainst
  const drawn = goalsFor === goalsAgainst
  return {
    ...row,
    points: row.points + (won ? 3 : drawn ? 1 : 0),
    matches: row.matches + 1,
    wins: row.wins + (won ? 1 : 0),
    draws: row.draws + (drawn ? 1 : 0),
    losses: row.losses + (!won && !drawn ? 1 : 0),
    goalsScored: row.goalsScored + goalsFor,
    goalsReceived: row.goalsReceived + goalsAgainst,
  }
}

function weekMatch(matchId: number, home: number, away: number, homeGoals: number, awayGoals: number, matchDate = minutesAgo(40)): WeekMatch {
  return {
    matchId,
    divisionId: 7,
    season: 3,
    weekNumber: 12,
    matchDate,
    played: true,
    inSimulation: false,
    home,
    homeName: `Team ${home}`,
    homeStadiumName: '',
    away,
    awayName: `Team ${away}`,
    homeGoals,
    awayGoals,
    events: null,
    matchInstants: null,
    homeLineup: null,
    awayLineup: null,
    homePlayers: null,
    awayPlayers: null,
  }
}

// T4 beat T6 2-0 and T3 beat T5 2-0 as stored results.
const MATCH_A = weekMatch(1201, 4, 6, 2, 0)
const MATCH_B = weekMatch(1202, 3, 5, 2, 0)

const details = {
  id: 7,
  season: 3,
  name: 'First division',
  classification: {
    positions: [
      preRow(1),
      preRow(2),
      withStoredResult(preRow(3), 2, 0),
      withStoredResult(preRow(4), 2, 0),
      withStoredResult(preRow(5), 0, 2),
      withStoredResult(preRow(6), 0, 2),
    ].sort((a, b) => b.points - a.points || a.teamId - b.teamId).map((row, index) => ({ ...row, position: index + 1 })),
  },
  previousWeekMatches: [MATCH_A, MATCH_B],
  nextWeekMatches: [],
  promotionSlots: 0,
  relegationSlots: 2,
} as DivisionDetails

function entry(match: WeekMatch, overrides: Partial<LeagueLiveScore>): LeagueLiveScore {
  return {
    matchId: match.matchId,
    weekNumber: match.weekNumber,
    matchDate: match.matchDate,
    home: match.home,
    homeName: match.homeName,
    away: match.away,
    awayName: match.awayName,
    status: 'live',
    minute: 30,
    homeGoals: 0,
    awayGoals: 0,
    ...overrides,
  }
}

// T4 trails T6 0-1 and T5 leads T3 1-0 at minute 30: T5 climbs to 4th, T4 drops into the zone.
const LIVE: LeagueLiveScore[] = [entry(MATCH_A, { homeGoals: 0, awayGoals: 1 }), entry(MATCH_B, { homeGoals: 0, awayGoals: 1 })]

const get = vi.mocked(api.get)

function serve(live: LeagueLiveScore[] | Error, seasons: LeagueSeasons = { current: 3, past: [] }) {
  get.mockImplementation(async (path: string) => {
    if (path.endsWith('/live')) {
      if (live instanceof Error) throw live
      return live
    }
    if (path.endsWith('/seasons')) return seasons
    if (path.endsWith('/seasons/2')) throw new Error('not needed')
    if (/\/weeks\/\d+/.test(path)) return []
    if (path.endsWith('/top-scorers') || path.endsWith('/recent-transfers')) return []
    if (path === '/v1/user/team') return { team: { id: 0 } }
    return details
  })
}

function renderPage(url = '/division/7') {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[url]}>
        <Routes>
          <Route path="/division/:id" element={<DivisionDetailsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const liveSwitch = () => screen.queryByRole('switch', { name: i18n.t('division.liveTable') })
const rows = () => within(screen.getByRole('table')).getAllByRole('row').slice(1)
const order = () => rows().map((row) => within(row).getByRole('link').textContent)
const liveCalls = () => get.mock.calls.map(([path]) => path).filter((path) => path.endsWith('/live'))

describe('DivisionDetailsPage live table', () => {
  beforeEach(async () => {
    get.mockReset()
    viewport.desktop = true
    await i18n.changeLanguage('en')
  })

  it('shows the Live switch on, with the live table, while the matchday is under way (Matchday under way)', async () => {
    serve(LIVE)
    renderPage()

    await waitFor(() => expect(liveSwitch()).toBeChecked())
    // Pre-matchday: T1 30, T2 25, T3 20, T4 15, T5 14, T6 5. Live: T6 and T5 win -> T5 17 4th,
    // T4 15 5th, T3 20 stays 3rd, T6 8.
    expect(order()).toEqual(['Team 1', 'Team 2', 'Team 3', 'Team 5', 'Team 4', 'Team 6'])
  })

  it('shows the table with every still-live result reverted when the switch is off (Switch off)', async () => {
    serve(LIVE)
    renderPage()

    fireEvent.click(await screen.findByRole('switch', { name: i18n.t('division.liveTable') }))
    expect(liveSwitch()).not.toBeChecked()
    expect(order()).toEqual(['Team 1', 'Team 2', 'Team 3', 'Team 4', 'Team 5', 'Team 6'])
    expect(screen.queryByLabelText(i18n.t('division.positionUp', { count: 1 }))).not.toBeInTheDocument()
  })

  it('shows no switch while nothing is on air, or when the request fails (Nothing live)', async () => {
    serve([entry(MATCH_A, { status: 'pending', minute: 0 }), entry(MATCH_B, { status: 'pending', minute: 0 })])
    renderPage()
    await waitFor(() => expect(liveCalls()).toHaveLength(1))
    expect(await screen.findByText('Team 6')).toBeInTheDocument()
    expect(liveSwitch()).not.toBeInTheDocument()
    expect(order()).toEqual(['Team 1', 'Team 2', 'Team 3', 'Team 4', 'Team 5', 'Team 6'])
  })

  it('shows no switch when the live scores request fails (Endpoint down)', async () => {
    serve(new Error('boom'))
    renderPage()
    await waitFor(() => expect(liveCalls()).toHaveLength(1))
    expect(await screen.findByText('Team 6')).toBeInTheDocument()
    expect(liveSwitch()).not.toBeInTheDocument()
    expect(order()).toEqual(['Team 1', 'Team 2', 'Team 3', 'Team 4', 'Team 5', 'Team 6'])
  })

  it('shows no switch and requests no live scores on a past season (Past season)', async () => {
    serve(LIVE, { current: 3, past: [2] })
    renderPage('/division/7?season=2')
    // The past-season table request is not stubbed, so the view never renders; only the absence of
    // a live scores request matters here.
    await waitFor(() => expect(get.mock.calls.some(([path]) => path.endsWith('/seasons/2'))).toBe(true))
    expect(liveCalls()).toEqual([])
    expect(liveSwitch()).not.toBeInTheDocument()
  })

  it('marks the places gained and lost and the teams on air (Position markers)', async () => {
    serve(LIVE)
    renderPage()

    await waitFor(() => expect(liveSwitch()).toBeChecked())
    const team5 = rows()[3]
    const team4 = rows()[4]
    expect(within(team5).getByText('▲1')).toBeInTheDocument()
    expect(within(team5).getByLabelText(i18n.t('division.positionUp', { count: 1 }))).toBeInTheDocument()
    expect(within(team4).getByText('▼1')).toBeInTheDocument()
    expect(within(team4).getByLabelText(i18n.t('division.positionDown', { count: 1 }))).toBeInTheDocument()
    // Every team of both matches is on air; Teams 1 and 2 are not.
    expect(within(team5).getByRole('img', { name: i18n.t('match.live') })).toBeInTheDocument()
    expect(within(team4).getByRole('img', { name: i18n.t('match.live') })).toBeInTheDocument()
    expect(within(rows()[0]).queryByRole('img', { name: i18n.t('match.live') })).not.toBeInTheDocument()
    // A team that did not move shows no marker.
    expect(within(rows()[0]).queryByText(/[▲▼]/)).not.toBeInTheDocument()
  })

  it('draws a team that drops into the relegation zone in the zone (Into the relegation zone)', async () => {
    serve(LIVE)
    renderPage()

    await waitFor(() => expect(liveSwitch()).toBeChecked())
    // 6 teams, 2 relegation slots: positions 5 and 6. Team 4 drops 4th -> 5th, Team 5 climbs out.
    const zoneBar = (row: HTMLElement) => window.getComputedStyle(row.querySelector('th, td') as HTMLElement).boxShadow
    expect(zoneBar(rows()[4])).not.toBe('')
    expect(zoneBar(rows()[4])).not.toBe('none')
    expect(within(rows()[4]).getByText('▼1')).toBeInTheDocument()
    expect(zoneBar(rows()[3]) || 'none').toBe('none')
  })

  it('keeps the switch and the markers on a phone (Live table on a phone)', async () => {
    viewport.desktop = false
    serve(LIVE)
    renderPage()

    await waitFor(() => expect(liveSwitch()).toBeChecked())
    expect(within(rows()[3]).getByText('▲1')).toBeInTheDocument()
    expect(within(rows()[4]).getByText('▼1')).toBeInTheDocument()
  })
})
