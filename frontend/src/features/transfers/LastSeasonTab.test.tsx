import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { POSITION_DEFENSE, POSITION_STRIKER, type Player, type PreContractMarketEntry } from '../../api/types'
import i18n from '../../i18n'
import { formatCurrency } from '../../utils/formatCurrency'
import { LastSeasonRow, LastSeasonTab } from './LastSeasonTab'

vi.mock('../../components/WatchToggle', () => ({ WatchToggle: () => null }))

const market = vi.hoisted(() => ({ current: [] as PreContractMarketEntry[] }))
vi.mock('../../api/hooks', () => ({
  useMe: () => ({ data: { id: 1 } }),
  useUserTeam: () => ({ data: { team: { id: 1 }, players: [] } }),
  usePreContractMarket: () => ({ isLoading: false, data: market.current }),
}))

const entry: PreContractMarketEntry = {
  player: { id: 7, teamId: 2, firstName: 'Leo', lastName: 'Vega', age: 27, wage: 2000, averageSkill: 71, contractYearsRemaining: 0, contractExpiring: true, releaseClause: 900000 } as Player,
  teamName: 'Rival FC',
  bestPosition: POSITION_STRIKER,
  open: true,
  offered: false,
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(cleanup)

// pre-contracts "Pre-contract market" and release-clauses "Clause visibility and notifications".
describe('LastSeasonRow', () => {
  it('shows the player, his clause and an open pre-contract action', () => {
    render(
      <MemoryRouter>
        <LastSeasonRow entry={entry} />
      </MemoryRouter>,
    )
    expect(screen.getByText('Rival FC')).toBeTruthy()
    expect(screen.getByText(`Release clause ${formatCurrency(900000)}`)).toBeTruthy()
    expect(screen.getByText('Last season')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Pre-contract' }) as HTMLButtonElement).disabled).toBe(false)
  })

  it('keeps the pre-contract action closed before half the season and after an offer', () => {
    render(
      <MemoryRouter>
        <LastSeasonRow entry={{ ...entry, open: false }} />
        <LastSeasonRow entry={{ ...entry, player: { ...entry.player, id: 8 }, offered: true }} />
      </MemoryRouter>,
    )
    expect(screen.getByText('Pre-contracts open once his league is halfway through.')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Pre-contract' }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: 'Offer sent' }) as HTMLButtonElement).disabled).toBe(true)
  })
})

const marketEntry = (id: number, lastName: string, age: number, overrides: Partial<PreContractMarketEntry> = {}, player: Partial<Player> = {}): PreContractMarketEntry => ({
  ...entry,
  ...overrides,
  player: { ...entry.player, id, lastName, age, ...player },
})

const renderTab = (query = '') =>
  render(
    <MemoryRouter initialEntries={[`/transfers${query}`]}>
      <LastSeasonTab />
    </MemoryRouter>,
  )

const rowNames = () => screen.getAllByTestId('last-season-row').map((row) => row.querySelector('a')?.textContent)

// The tab's search and filters, as on the other market tabs.
describe('LastSeasonTab', () => {
  beforeEach(() => {
    market.current = [
      marketEntry(1, 'Vega', 27, {}, { averageSkill: 71, wage: 2000, releaseClause: 900000 }),
      marketEntry(2, 'Rojo', 31, { bestPosition: POSITION_DEFENSE }, { averageSkill: 65, wage: 5000, releaseClause: 300000 }),
      marketEntry(3, 'Sol', 22, {}, { averageSkill: 60, wage: 1000, releaseClause: undefined }),
    ]
  })

  it('lists every entry by rating when no filter is set', () => {
    renderTab()
    expect(rowNames()).toEqual(['Leo Vega', 'Leo Rojo', 'Leo Sol'])
  })

  it('searches by name', () => {
    renderTab('?lsSearch=rojo')
    expect(rowNames()).toEqual(['Leo Rojo'])
  })

  it('filters by position, age, salary and clause, and sorts', () => {
    renderTab(`?lsPosition=${POSITION_STRIKER}`)
    expect(rowNames()).toEqual(['Leo Vega', 'Leo Sol'])
    cleanup()
    renderTab('?lsMaxAge=30&lsSort=age&lsDir=asc')
    expect(rowNames()).toEqual(['Leo Sol', 'Leo Vega'])
    cleanup()
    renderTab('?lsMinWage=1500&lsMaxClause=500000')
    expect(rowNames()).toEqual(['Leo Rojo'])
  })

  it('says so when no player matches', () => {
    renderTab('?lsMinSkill=90')
    expect(screen.getByText('No players match your filters.')).toBeTruthy()
  })
})
