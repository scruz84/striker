import { cleanup, fireEvent, render as baseRender, screen } from '@testing-library/react'
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

describe('SquadList contract sort', () => {
  // Real skills so each player lands in a position group: 1 goalkeeper and 2 strikers.
  function rostered(id: number, lastName: string, contractYearsRemaining: number, keeper = false): Player {
    return {
      ...player(id, lastName),
      contractYearsRemaining,
      goalKeeper: keeper ? 80 : 10,
      defense: 10,
      midfield: 10,
      pass: 10,
      dribbling: 10,
      shot: keeper ? 10 : 80,
    } as Player
  }
  const squad = [rostered(1, 'Keeper', 2, true), rostered(2, 'Short', 1), rostered(3, 'Long', 3), rostered(4, 'Alpha', 2)]

  function renderSquad(isOwnTeam: boolean, loans: LoanedInPlayer[] = []) {
    return render(
      <MemoryRouter>
        <SquadList players={squad} isOwnTeam={isOwnTeam} unavailability={new Map()} transferListed={new Set()} loanedIn={loans} />
      </MemoryRouter>,
    )
  }
  const cardOrder = () => screen.getAllByRole('button', { name: /^Leo / }).map((b) => b.getAttribute('aria-label'))
  const pickContractSort = () => {
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Sort by' }))
    fireEvent.click(screen.getByRole('option', { name: 'Contract' }))
  }

  it('lists the whole squad from the longest contract down, ties by name, then flips with the arrow', () => {
    renderSquad(true)
    pickContractSort()
    expect(cardOrder()).toEqual(['Leo Long', 'Leo Alpha', 'Leo Keeper', 'Leo Short'])
    // One flat list - no position headings - with the seasons left on every card.
    expect(screen.queryByText(/Goalkeepers/)).toBeNull()
    expect(screen.getByRole('button', { name: 'Leo Long' }).textContent).toContain('3 seasons left')
    expect(screen.getByRole('button', { name: 'Leo Short' }).textContent).toContain('1 season left')

    fireEvent.click(screen.getByRole('button', { name: 'Descending order' }))
    expect(cardOrder()).toEqual(['Leo Short', 'Leo Alpha', 'Leo Keeper', 'Leo Long'])
  })

  it('flags only the own players in their last season, whatever the sort', () => {
    renderSquad(true)
    expect(screen.getAllByText('Last season')).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Leo Short' }).textContent).toContain('Last season')
    // The seasons left only show in the contract sort.
    expect(screen.queryByText(/seasons? left/)).toBeNull()
  })

  it('never flags a loaned-in player, whose contract is the parent club\'s', () => {
    renderSquad(true, [{ playerId: 2, parentTeamId: 7, parentTeamName: 'Parent FC', returnSeason: 6 }])
    expect(screen.queryByText('Last season')).toBeNull()
  })

  it('offers no contract sort and no warning on a rival squad', () => {
    renderSquad(false)
    expect(screen.queryByText('Last season')).toBeNull()
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Sort by' }))
    expect(screen.queryByRole('option', { name: 'Contract' })).toBeNull()
  })
})
