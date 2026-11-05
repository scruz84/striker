import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api/client'
import type { ClassificationPosition, DivisionDetails, LeagueLiveScore, WeekMatch } from '../../api/types'
import i18n from '../../i18n'
import { DivisionDetailsPage } from './DivisionDetailsPage'

vi.mock('../../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/client')>()),
  api: { get: vi.fn() },
}))

const viewport = vi.hoisted(() => ({ desktop: true }))

vi.mock('@mui/material/useMediaQuery', () => ({ default: () => viewport.desktop }))

const TEAMS = 16
const TOTAL_WEEKS = (TEAMS - 1) * 2
const LONG_AGO = '2020-01-01T00:00:00Z'

function weekMatch(week: number, overrides: Partial<WeekMatch> = {}): WeekMatch {
  return {
    matchId: week * 100,
    divisionId: 7,
    season: 1,
    weekNumber: week,
    matchDate: LONG_AGO,
    played: true,
    inSimulation: false,
    home: 1,
    homeName: `Home ${week}`,
    homeStadiumName: '',
    away: 2,
    awayName: `Away ${week}`,
    homeGoals: 1,
    awayGoals: 0,
    events: null,
    matchInstants: null,
    homeLineup: null,
    awayLineup: null,
    homePlayers: null,
    awayPlayers: null,
    ...overrides,
  }
}

function standings(playedWeeks: number): ClassificationPosition[] {
  return Array.from({ length: TEAMS }, (_unused, index) => ({
    position: index + 1,
    teamId: index + 1,
    teamName: `Team ${index + 1}`,
    points: 0,
    matches: playedWeeks,
    wins: 0,
    draws: 0,
    losses: 0,
    goalsScored: 0,
    goalsReceived: 0,
  }))
}

function division(playedWeeks: number, previous: WeekMatch[], next: WeekMatch[]): DivisionDetails {
  return {
    id: 7,
    season: 1,
    name: 'First division',
    classification: { positions: standings(playedWeeks) },
    previousWeekMatches: previous,
    nextWeekMatches: next,
  } as DivisionDetails
}

const get = vi.mocked(api.get)

// Answers each endpoint the page reads. `/weeks/<n>` returns one match titled after its week, so a
// test can tell which matchdays were fetched.
function serve(details: DivisionDetails, liveScores: LeagueLiveScore[] | Error = []) {
  get.mockImplementation(async (path: string) => {
    if (path.endsWith('/live')) {
      if (liveScores instanceof Error) throw liveScores
      return liveScores
    }
    const weeks = /\/weeks\/(\d+)$/.exec(path)
    if (weeks) return [weekMatch(Number(weeks[1]))]
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

const liveCalls = () => get.mock.calls.map(([path]) => path).filter((path) => path.endsWith('/live'))
const minutesAgo = (minutes: number) => new Date(Date.now() - minutes * 60 * 1000).toISOString()

function liveScore(match: WeekMatch, overrides: Partial<LeagueLiveScore> = {}): LeagueLiveScore {
  return {
    matchId: match.matchId,
    weekNumber: match.weekNumber,
    matchDate: match.matchDate,
    home: match.home,
    homeName: match.homeName,
    away: match.away,
    awayName: match.awayName,
    status: 'live',
    minute: 63,
    homeGoals: 1,
    awayGoals: 0,
    ...overrides,
  }
}

const matchdayCalls = () => get.mock.calls.map(([path]) => path).filter((path) => path.includes('/weeks/'))
const heading = (week: number) => screen.findByRole('heading', { name: i18n.t('division.matchday', { week }) })

describe('DivisionDetailsPage matchday navigation', () => {
  beforeEach(() => {
    get.mockReset()
    viewport.desktop = true
  })

  it('opens mid-season on the current pair without requesting any matchday', async () => {
    serve(division(12, [weekMatch(12)], [weekMatch(13)]))
    renderPage()

    expect(await heading(12)).toBeInTheDocument()
    expect(await heading(13)).toBeInTheDocument()
    expect(matchdayCalls()).toEqual([])
  })

  it('requests only the earlier matchday while the latest one is still broadcasting', async () => {
    // Matchday 12 was simulated but kicked off 40 minutes ago: it reads as the next matchday, and
    // matchday 11 (the reverted current week) is the only one the division response doesn't carry.
    const live = weekMatch(12, { matchDate: new Date(Date.now() - 40 * 60 * 1000).toISOString(), home: 1, away: 2 })
    serve(division(12, [live], [weekMatch(13)]))
    renderPage()

    expect(await heading(11)).toBeInTheDocument()
    expect(await heading(12)).toBeInTheDocument()
    await waitFor(() => expect(matchdayCalls()).toEqual(['/v1/teams/division/7/weeks/11']))
    // The live scores endpoint answers with nothing for this match, so the LIVE chip stays.
    expect(await screen.findAllByText(i18n.t('match.live'))).toHaveLength(1)
    expect(liveCalls()).toEqual(['/v1/teams/division/7/live'])
  })

  it('shows the broadcast score and minute of a match in play, never the stored result (Match in play)', async () => {
    const live = weekMatch(12, { matchDate: minutesAgo(40), homeGoals: 3, awayGoals: 0 })
    serve(division(12, [live], [weekMatch(13)]), [liveScore(live, { minute: 63, homeGoals: 1, awayGoals: 0 })])
    renderPage()

    const link = (await screen.findByText('Home 12')).closest('a')!
    expect(await within(link).findByText("63'")).toBeInTheDocument()
    expect(within(link).getByText('1')).toBeInTheDocument()
    expect(within(link).getByText('0')).toBeInTheDocument()
    expect(within(link).queryByText('3')).not.toBeInTheDocument()
    expect(within(link).queryByText(i18n.t('match.live'))).not.toBeInTheDocument()
  })

  it('shows FT when the broadcast is over but the window is still open (Broadcast over, window still open)', async () => {
    const live = weekMatch(12, { matchDate: minutesAgo(110) })
    serve(division(12, [live], [weekMatch(13)]), [liveScore(live, { status: 'finished', minute: 90, homeGoals: 2, awayGoals: 2 })])
    renderPage()

    const link = (await screen.findByText('Home 12')).closest('a')!
    expect(await within(link).findByText(i18n.t('liveScores.fullTime'))).toBeInTheDocument()
    expect(within(link).getAllByText('2')).toHaveLength(2)
  })

  it('keeps the LIVE chip and no score for a match simulated ahead of kickoff (Simulated ahead of kickoff)', async () => {
    const early = weekMatch(12, { matchDate: new Date(Date.now() + 5 * 60 * 1000).toISOString(), homeGoals: 4, awayGoals: 1 })
    serve(division(12, [early], [weekMatch(13)]), [liveScore(early, { status: 'pending', minute: 0, homeGoals: 0, awayGoals: 0 })])
    renderPage()

    const link = (await screen.findByText('Home 12')).closest('a')!
    await waitFor(() => expect(liveCalls()).toHaveLength(1))
    expect(within(link).getByText(i18n.t('match.live'))).toBeInTheDocument()
    expect(within(link).queryByText('4')).not.toBeInTheDocument()
    expect(within(link).queryByText('0')).not.toBeInTheDocument()
  })

  it('falls back to the LIVE chip when the live scores request fails (Endpoint down)', async () => {
    const live = weekMatch(12, { matchDate: minutesAgo(40), homeGoals: 3 })
    serve(division(12, [live], [weekMatch(13)]), new Error('boom'))
    renderPage()

    const link = (await screen.findByText('Home 12')).closest('a')!
    await waitFor(() => expect(liveCalls()).toHaveLength(1))
    expect(within(link).getByText(i18n.t('match.live'))).toBeInTheDocument()
    expect(within(link).queryByText('3')).not.toBeInTheDocument()
  })

  it('makes no live scores request when nothing is in play (Quiet day)', async () => {
    serve(division(12, [weekMatch(12)], [weekMatch(13)]))
    renderPage()

    expect(await heading(12)).toBeInTheDocument()
    expect(await screen.findByText('Home 12')).toBeInTheDocument()
    expect(liveCalls()).toEqual([])
  })

  it('steps through matchdays and returns to the current pair', async () => {
    serve(division(12, [weekMatch(12)], [weekMatch(13)]))
    renderPage()
    await heading(12)

    fireEvent.click(screen.getByRole('button', { name: i18n.t('division.nextMatchday') }))
    expect(await heading(13)).toBeInTheDocument()
    expect(await heading(14)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: i18n.t('division.backToCurrent') }))
    expect(await heading(12)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: i18n.t('division.backToCurrent') })).not.toBeInTheDocument()
  })

  it('disables ‹ on matchday 1 and › on the last pair', async () => {
    serve(division(0, [], [weekMatch(1)]))
    renderPage('/division/7?week=1')
    await heading(1)
    expect(screen.getByRole('button', { name: i18n.t('division.previousMatchday') })).toBeDisabled()
  })

  it('clamps an out-of-range ?week= to the last pair', async () => {
    serve(division(12, [weekMatch(12)], [weekMatch(13)]))
    renderPage('/division/7?week=99')

    expect(await heading(TOTAL_WEEKS - 1)).toBeInTheDocument()
    expect(await heading(TOTAL_WEEKS)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: i18n.t('division.nextMatchday') })).toBeDisabled()
  })

  it('ignores a non-numeric ?week=', async () => {
    serve(division(12, [weekMatch(12)], [weekMatch(13)]))
    renderPage('/division/7?week=abc')

    expect(await heading(12)).toBeInTheDocument()
    expect(matchdayCalls()).toEqual([])
  })

  it('hides the score of a still-broadcasting match on a browsed matchday', async () => {
    serve(division(12, [weekMatch(12)], [weekMatch(13)]))
    const live = weekMatch(5, { matchDate: new Date(Date.now() - 40 * 60 * 1000).toISOString(), homeGoals: 3, awayGoals: 2 })
    get.mockImplementation(async (path: string) => {
      if (path.endsWith('/weeks/5')) return [live]
      if (path.includes('/weeks/')) return []
      if (path.endsWith('/top-scorers') || path.endsWith('/recent-transfers')) return []
      if (path === '/v1/user/team') return { team: { id: 0 } }
      return division(12, [weekMatch(12)], [weekMatch(13)])
    })
    renderPage('/division/7?week=5')

    await heading(5)
    const link = (await screen.findByText('Home 5')).closest('a')!
    expect(within(link).getByText(i18n.t('match.live'))).toBeInTheDocument()
    expect(within(link).queryByText('3')).not.toBeInTheDocument()
    expect(within(link).queryByText('2')).not.toBeInTheDocument()
  })

  it('keeps polling while browsing an old matchday mid-broadcast, whose cards show their stored scores (Browsing an old matchday mid-broadcast)', async () => {
    const live = weekMatch(12, { matchDate: minutesAgo(40), homeGoals: 3 })
    const old = weekMatch(3, { homeGoals: 4, awayGoals: 1 })
    get.mockImplementation(async (path: string) => {
      if (path.endsWith('/live')) return [liveScore(live)]
      if (path.endsWith('/weeks/3')) return [old]
      if (path.endsWith('/weeks/4')) return [weekMatch(4)]
      if (path.endsWith('/top-scorers') || path.endsWith('/recent-transfers')) return []
      if (path === '/v1/user/team') return { team: { id: 0 } }
      return division(12, [live], [weekMatch(13)])
    })
    renderPage('/division/7?week=3')

    await heading(3)
    const link = (await screen.findByText('Home 3')).closest('a')!
    expect(within(link).getByText('4')).toBeInTheDocument()
    expect(within(link).getByText('1')).toBeInTheDocument()
    await waitFor(() => expect(liveCalls()).toEqual(['/v1/teams/division/7/live']))
  })

  it('opens on the Fixtures tab content only after tapping it on a phone', async () => {
    viewport.desktop = false
    serve(division(12, [weekMatch(12)], [weekMatch(13)]))
    renderPage()

    fireEvent.click(await screen.findByRole('tab', { name: i18n.t('division.tabs.fixtures') }))
    expect(await heading(12)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: i18n.t('division.nextMatchday') })).toBeInTheDocument()
  })

  it('splits top scorers and recent transfers into their own phone tabs', async () => {
    viewport.desktop = false
    serve(division(12, [weekMatch(12)], [weekMatch(13)]))
    renderPage()

    const transfersHeading = () => screen.queryByRole('heading', { name: i18n.t('recentTransfers.title') })
    expect(await screen.findAllByRole('tab')).toHaveLength(4)

    fireEvent.click(await screen.findByRole('tab', { name: i18n.t('division.tabs.scorers') }))
    expect(await screen.findByRole('heading', { name: i18n.t('division.topScorers.title') })).toBeInTheDocument()
    expect(transfersHeading()).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('tab', { name: i18n.t('division.tabs.transfers') }))
    expect(await screen.findByRole('heading', { name: i18n.t('recentTransfers.title') })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: i18n.t('division.topScorers.title') })).not.toBeInTheDocument()
  })
})
