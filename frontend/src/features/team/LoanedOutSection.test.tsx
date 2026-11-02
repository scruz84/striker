import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import type { LoanedOutPlayer, Player } from '../../api/types'
import i18n from '../../i18n'
import { LoanedOutSection } from './LoanedOutSection'

function loan(overrides: Partial<LoanedOutPlayer> & { id: number; lastName: string }): LoanedOutPlayer {
  const { id, lastName, ...rest } = overrides
  return {
    player: { id, firstName: 'Leo', lastName, age: 19, birthCountry: 'es' } as Player,
    borrowingTeamId: 90 + id,
    borrowingTeamName: `Borrower ${id}`,
    borrowingDivisionId: 5,
    borrowingDivisionName: '1B',
    returnSeason: 5,
    wageContributionPercent: 0,
    buyOptionFee: 0,
    goalsForBorrower: 2,
    ...rest,
  }
}

function renderIt(loanedOut: LoanedOutPlayer[]) {
  return render(
    <MemoryRouter>
      <LoanedOutSection loanedOut={loanedOut} isOwnTeam={false} />
    </MemoryRouter>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(cleanup)

describe('LoanedOutSection', () => {
  it('renders nothing when nobody is out', () => {
    const { container } = renderIt([])
    expect(container.textContent).toBe('')
    expect(screen.queryByTestId('loaned-out-section')).toBeNull()
  })

  it('lists a loan without wage share or buy option: club link, division, return season, goals', () => {
    renderIt([loan({ id: 1, lastName: 'Vega' })])
    expect(screen.getByText('Loaned out (1)')).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Leo Vega' }).getAttribute('href')).toBe('/player/1')
    expect(screen.getByRole('link', { name: 'Borrower 1' }).getAttribute('href')).toBe('/team/91')
    expect(screen.getByText('1B')).toBeTruthy()
    expect(screen.getByText('Returns season 5')).toBeTruthy()
    expect(screen.getByText('Goals: 2')).toBeTruthy()
    expect(screen.queryByText(/Wage share/)).toBeNull()
    expect(screen.queryByText(/Buy option/)).toBeNull()
  })

  it('shows the wage share and the buy option when the loan carries them', () => {
    renderIt([loan({ id: 2, lastName: 'Mora', wageContributionPercent: 50, buyOptionFee: 1200000 })])
    expect(screen.getByText('Wage share 50%')).toBeTruthy()
    expect(screen.getByText(/Buy option .*1.*200.*000/)).toBeTruthy()
  })

  it('counts and lists every player out', () => {
    renderIt([loan({ id: 3, lastName: 'Gil' }), loan({ id: 4, lastName: 'Ruiz' })])
    expect(screen.getByText('Loaned out (2)')).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Leo Gil' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Leo Ruiz' })).toBeTruthy()
  })
})
