import { cleanup, render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { CurrentWorldCup, NationalMatch } from '../../api/types'
import i18n from '../../i18n'
import { teamTabs } from '../team/teamTabs'
import { NationalMatchRow } from './NationalMatchRow'
import { NationalTeamsPage } from './NationalTeamsPage'

const state = vi.hoisted(() => ({
  round: [] as NationalMatch[],
  worldCup: undefined as CurrentWorldCup | undefined,
}))

vi.mock('../../api/hooks', async (importActual) => ({
  ...(await importActual<typeof import('../../api/hooks')>()),
  useNationalTeamRound: () => ({ data: state.round, isLoading: false, isError: false }),
  useNationalTeamResults: () => ({ data: { pages: [] }, isLoading: false, isError: false, hasNextPage: false }),
  useUserTeam: () => ({ data: undefined }),
  useCurrentWorldCup: () => ({ data: state.worldCup, isLoading: false, isError: false }),
  useCupDetails: () => ({ data: undefined, isLoading: false }),
  useWorldCupHistory: () => ({ data: [] }),
}))

function match(overrides: Partial<NationalMatch> = {}): NationalMatch {
  return {
    kind: 'friendly',
    id: 1,
    homeTeamId: 10,
    homeTeamName: 'Northern Ireland',
    homeCountry: 'nir',
    awayTeamId: 20,
    awayTeamName: 'Sao Tome and Principe',
    awayCountry: 'st',
    kickoff: '2026-11-07T21:00:00Z',
    status: 'scheduled',
    ...overrides,
  }
}

function renderAt(path: string, ui = <NationalTeamsPage />) {
  return render(<MemoryRouter initialEntries={[path]}>{ui}</MemoryRouter>)
}

beforeEach(async () => {
  await i18n.changeLanguage('es')
  state.round = []
  state.worldCup = undefined
})
afterEach(cleanup)

describe('NationalMatchRow', () => {
  it('keeps long country names on one line', () => {
    renderAt('/', <NationalMatchRow match={match()} />)
    const home = screen.getByText(i18n.t('countries.nir'))
    const away = screen.getByText(i18n.t('countries.st'))
    for (const name of [home, away]) {
      expect(name.className).toMatch(/noWrap/)
    }
  })

  it('shows a live match at its broadcast minute and score', () => {
    renderAt('/', <NationalMatchRow match={match({ status: 'live', minute: 40, homeGoals: 1, awayGoals: 0 })} />)
    expect(screen.getByText("EN VIVO 40'")).toBeTruthy()
    expect(screen.getByText('1 - 0')).toBeTruthy()
  })

  it('shows no score before kickoff and badges the viewer\'s called-up players', () => {
    renderAt('/', <NationalMatchRow match={match({ myPlayers: 2 })} />)
    expect(screen.getByText('vs')).toBeTruthy()
    expect(screen.getByText('2 jugadores tuyos')).toBeTruthy()
  })
})

describe('NationalTeamsPage', () => {
  it('opens on the round and lists every match, not just ten', () => {
    state.round = Array.from({ length: 30 }, (_, i) => match({ id: i + 1 }))
    renderAt('/national-teams')
    expect(screen.getByRole('tab', { name: 'Jornada', selected: true })).toBeTruthy()
    expect(screen.getAllByTestId('national-match-row')).toHaveLength(30)
  })

  it('opens the World Cup tab from the URL and shows the next World Cup estimate', () => {
    state.worldCup = { worldCup: null, groups: [], nextWorldCupEstimate: '2027-01-29T03:00:00Z' }
    renderAt('/national-teams?tab=world-cup')
    expect(screen.getByRole('tab', { name: 'Mundial', selected: true })).toBeTruthy()
    expect(screen.getByText(/Próximo Mundial: hacia el/)).toBeTruthy()
  })

  it('shows full group tables', () => {
    state.worldCup = {
      worldCup: { worldCupId: 1, season: 4, name: 'Mundial', stage: 'groups', completed: false, championTeamId: null, championTeamName: '', knockoutCupId: null },
      groups: [
        {
          groupId: 1,
          groupName: 'A',
          standings: [
            { teamId: 10, teamName: 'Spain', played: 1, won: 1, drawn: 0, lost: 0, goalsFor: 2, goalsAgainst: 0 },
            { teamId: 20, teamName: 'Portugal', played: 1, won: 0, drawn: 0, lost: 1, goalsFor: 0, goalsAgainst: 2 },
          ],
          matches: [
            { matchId: 5, round: 1, homeTeamId: 10, homeTeamName: 'Spain', awayTeamId: 20, awayTeamName: 'Portugal', matchDate: '2026-10-31T20:00:00Z', status: 'finished', homeGoals: 2, awayGoals: 0 },
          ],
        },
      ],
    }
    renderAt('/national-teams?tab=world-cup')
    const headers = screen.getAllByRole('columnheader').map((cell) => cell.textContent)
    expect(headers).toEqual(['Equipo', 'PJ', 'G', 'E', 'P', 'GF', 'GC', 'DG', 'Pts'])
    const spain = screen.getAllByTestId('group-standing-row')[0]
    expect(within(spain).getAllByRole('cell').map((cell) => cell.textContent).slice(1)).toEqual(['1', '1', '0', '0', '2', '0', '2', '3'])
    expect(screen.getByText(/Jornada 1/)).toBeTruthy()
  })
})

describe('teamTabs', () => {
  it('gives a national team a matches tab and a club none', () => {
    expect(teamTabs(true, false)).toContain('matches')
    expect(teamTabs(false, false)).not.toContain('matches')
    expect(teamTabs(false, true)).not.toContain('matches')
  })
})
