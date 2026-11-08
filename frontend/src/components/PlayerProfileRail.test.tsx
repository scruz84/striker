import { cleanup, render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { PlayerDetails } from '../api/types'
import i18n from '../i18n'
import { PlayerProfileRail } from './PlayerProfileRail'
import { lightTheme } from '../theme'

vi.mock('../api/hooks', () => ({
  useExerciseLoanBuyOption: () => ({ mutate: vi.fn(), isPending: false }),
  useTransferWindowStatus: () => ({ data: { open: true } }),
}))
vi.mock('./WatchToggle', () => ({ WatchToggle: () => null }))
vi.mock('./TransferOfferDialog', () => ({ TransferOfferDialog: () => null }))
vi.mock('./ListPlayerForTransferDialog', () => ({ ListPlayerForTransferDialog: () => null }))
vi.mock('./confirmDialogContext', () => ({ useConfirm: () => vi.fn() }))

const player = {
  id: 7,
  teamId: 2,
  teamName: 'Rival FC',
  firstName: 'Leo',
  lastName: 'Vega',
  age: 27,
  wage: 2000,
  averageSkill: 70,
  birthDay: '2000-01-01T00:00:00Z',
  divisionId: 1,
  divisionName: 'Liga',
  caps: 0,
  goals: 0,
  goalsForTeam: 0,
  yellowCards: 0,
  redCards: 0,
  seasonAwards: { goldenBoots: 0, playerOfSeasons: 0 },
  contractYearsRemaining: 0,
} as unknown as PlayerDetails

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(cleanup)

// player-negotiation "Negotiation insight": "After a refused offer" and "Another club".
describe('PlayerProfileRail negotiator row', () => {
  it('shows the tier once the club has negotiated with him', () => {
    render(
      <ThemeProvider theme={lightTheme}>
        <MemoryRouter>
        <PlayerProfileRail player={{ ...player, negotiationStyle: 4 }} canSeeFinancials={false} canSeeSkills={false} />
      </MemoryRouter>
      </ThemeProvider>,
    )
    expect(screen.getByText('Negotiator')).toBeTruthy()
    expect(screen.getByText('Shrewd')).toBeTruthy()
  })

  it('shows nothing before any negotiation', () => {
    render(
      <ThemeProvider theme={lightTheme}>
        <MemoryRouter>
        <PlayerProfileRail player={player} canSeeFinancials={false} canSeeSkills={false} />
      </MemoryRouter>
      </ThemeProvider>,
    )
    expect(screen.queryByText('Negotiator')).toBeNull()
  })
})
