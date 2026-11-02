import { cleanup, render as baseRender, screen } from '@testing-library/react'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import type { ReactElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { LoanedInPlayer, Player } from '../../api/types'
import i18n from '../../i18n'
import { SquadList } from './SquadList'

// The cards colour the rating from the app theme's rating scale (src/theme.ts pulls in a colour
// library that doesn't load under jsdom), so the test supplies a plain scale of its own.
const testTheme = createTheme({ palette: { rating: ['#ccc', '#bbb', '#aaa', '#999', '#888'] } })
const render = (ui: ReactElement) => baseRender(<ThemeProvider theme={testTheme}>{ui}</ThemeProvider>)

vi.mock('../../api/hooks', () => ({
  useReleasePlayer: () => ({ mutate: vi.fn(), isPending: false, isError: false }),
}))

function player(id: number, lastName: string): Player {
  return { id, firstName: 'Leo', lastName, age: 24, wage: 1000, birthCountry: 'es', averageSkill: 50 } as Player
}

const loanedIn: LoanedInPlayer[] = [{ playerId: 2, parentTeamId: 7, parentTeamName: 'Parent FC', returnSeason: 6 }]

function renderIt(loans: LoanedInPlayer[]) {
  return render(
    <MemoryRouter>
      <SquadList
        players={[player(1, 'Own'), player(2, 'Borrowed')]}
        isOwnTeam={false}
        unavailability={new Map()}
        transferListed={new Set()}
        loanedIn={loans}
      />
    </MemoryRouter>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(cleanup)

describe('SquadList loan chip', () => {
  it('marks only the loaned-in player, with the parent club and return season in the tooltip', () => {
    renderIt(loanedIn)
    const chips = screen.getAllByText('Loan')
    expect(chips).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Leo Borrowed' }).textContent).toContain('Loan')
    expect(screen.getByRole('button', { name: 'Leo Own' }).textContent).not.toContain('Loan')
    expect(screen.getByLabelText('On loan from Parent FC until season 6')).toBeTruthy()
  })

  it('shows no chip when nobody is on loan', () => {
    renderIt([])
    expect(screen.queryByText('Loan')).toBeNull()
  })
})
