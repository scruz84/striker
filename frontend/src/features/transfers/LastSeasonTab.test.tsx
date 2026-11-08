import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { POSITION_STRIKER, type Player, type PreContractMarketEntry } from '../../api/types'
import i18n from '../../i18n'
import { formatCurrency } from '../../utils/formatCurrency'
import { LastSeasonRow } from './LastSeasonTab'

vi.mock('../../components/WatchToggle', () => ({ WatchToggle: () => null }))

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
