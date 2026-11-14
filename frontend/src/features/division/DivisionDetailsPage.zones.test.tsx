import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../api/client'
import type { CupTie, DivisionDetails, PyramidZones } from '../../api/types'
import i18n from '../../i18n'
import { DivisionDetailsPage } from './DivisionDetailsPage'
import { zoneFor, zonesLegend } from './zones'

vi.mock('../../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/client')>()),
  api: { get: vi.fn() },
}))

vi.mock('@mui/material/useMediaQuery', () => ({ default: () => true }))

const TEAMS = 10
const middleTier: PyramidZones = { directUp: 1, playoffUp: 1, playoffDown: 2, directDown: 2, candidate: 0, legacy: false }
const topTier: PyramidZones = { directUp: 0, playoffUp: 0, playoffDown: 2, directDown: 2, candidate: 0, legacy: false }
const bottomTier: PyramidZones = { directUp: 1, playoffUp: 1, playoffDown: 0, directDown: 0, candidate: 0, legacy: false }
const legacyMiddle: PyramidZones = { directUp: 0, playoffUp: 0, playoffDown: 0, directDown: 2, candidate: 1, legacy: true }

const zonesOf = (zones: PyramidZones) => Array.from({ length: TEAMS }, (_unused, i) => zoneFor(i + 1, TEAMS, zones))

describe('zoneFor', () => {
  it('colours a tier-2 group of 10 (Tier-2 group of 10)', () => {
    expect(zonesOf(middleTier)).toEqual([
      'promotion', 'playoff', undefined, undefined, undefined, undefined, 'playoff', 'playoff', 'relegation', 'relegation',
    ])
  })

  it('marks no promotion in the top division (Top division)', () => {
    expect(zonesOf(topTier).slice(0, 6).every((zone) => zone === undefined)).toBe(true)
    expect(zonesLegend(topTier)).toEqual({ promotion: false, playoff: true, relegation: true, candidate: false })
  })

  it('marks no relegation in the bottom tier (Bottom tier)', () => {
    expect(zonesOf(bottomTier)).toEqual(['promotion', 'playoff', ...Array(8).fill(undefined)])
  })

  it('shows a legacy winner as a promotion candidate, never as promoted (Legacy table)', () => {
    expect(zonesOf(legacyMiddle)[0]).toBe('playoff')
    expect(zonesOf(legacyMiddle).filter((zone) => zone === 'promotion')).toHaveLength(0)
    expect(zonesLegend(legacyMiddle).candidate).toBe(true)
  })

  it('has no zones without zone data', () => {
    expect(zonesOf({ directUp: 0, playoffUp: 0, playoffDown: 0, directDown: 0, candidate: 0, legacy: false }).every((z) => z === undefined)).toBe(true)
    expect(zoneFor(1, TEAMS, undefined)).toBeUndefined()
  })
})

function details(zones: PyramidZones, playoff: CupTie[] = []): DivisionDetails {
  return {
    id: 7,
    season: 3,
    name: 'II.1',
    classification: {
      positions: Array.from({ length: TEAMS }, (_unused, index) => ({
        position: index + 1,
        teamId: index + 1,
        teamName: `Team ${index + 1}`,
        points: 50 - index,
        matches: 18,
        wins: 0,
        draws: 0,
        losses: 0,
        goalsScored: 0,
        goalsReceived: 0,
      })),
    },
    previousWeekMatches: [],
    nextWeekMatches: [],
    zones,
    playoff,
  } as DivisionDetails
}

const get = vi.mocked(api.get)

function serve(division: DivisionDetails) {
  get.mockImplementation(async (path: string) => {
    if (path.endsWith('/seasons')) return { current: 3, past: [] }
    if (path.includes('/weeks/')) return []
    if (path.endsWith('/top-scorers') || path.endsWith('/recent-transfers')) return []
    if (path === '/v1/user/team') return { team: { id: 2 } }
    return division
  })
}

function renderPage() {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={['/division/7']}>
        <Routes>
          <Route path="/division/:id" element={<DivisionDetailsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const tie: CupTie = {
  tieId: 55,
  cupId: 9,
  round: 1,
  slotNumber: 1,
  homeTeamId: 99,
  homeTeamName: 'Upper Club',
  awayTeamId: 2,
  awayTeamName: 'Team 2',
  matchDate: '2099-01-01T13:00:00Z',
  played: false,
  homeGoals: 0,
  awayGoals: 0,
  wentToPenalties: false,
  winnerTeamId: null,
}

describe('DivisionDetailsPage zones', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('lists only the zones a middle tier has, and its playoff ties', async () => {
    serve(details(middleTier, [tie]))
    renderPage()

    expect(await screen.findByText('Team 1')).toBeInTheDocument()
    const legend = screen.getByLabelText(i18n.t('division.legend.title'))
    expect(within(legend).getByText(i18n.t('division.legend.promotion'))).toBeInTheDocument()
    expect(within(legend).getByText(i18n.t('division.legend.playoff'))).toBeInTheDocument()
    expect(within(legend).getByText(i18n.t('division.legend.relegation'))).toBeInTheDocument()

    const card = screen.getByRole('region', { name: i18n.t('division.playoff.title') })
    expect(within(card).getByText('Upper Club')).toBeInTheDocument()
    // My team's tie is marked.
    expect(within(card).getByText(i18n.t('common.you'))).toBeInTheDocument()
  })

  it('labels a legacy winner a promotion candidate and shows no playoff card before the ties exist', async () => {
    serve(details(legacyMiddle))
    renderPage()

    expect(await screen.findByText('Team 1')).toBeInTheDocument()
    const legend = screen.getByLabelText(i18n.t('division.legend.title'))
    expect(within(legend).getByText(i18n.t('division.legend.candidate'))).toBeInTheDocument()
    expect(within(legend).queryByText(i18n.t('division.legend.promotion'))).not.toBeInTheDocument()
    expect(screen.queryByRole('region', { name: i18n.t('division.playoff.title') })).not.toBeInTheDocument()
  })
})
