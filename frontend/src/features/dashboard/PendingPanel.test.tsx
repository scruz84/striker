import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { FinancesResponse, Lineup, Player, PlayerUnavailability, TeamMatchSummary } from '../../api/types'
import '../../i18n'
import { setPhone } from '../admin/testPhone'
import { PendingPanel } from './PendingPanel'

const data = vi.hoisted(() => ({
  academy: { canPromote: false } as { canPromote: boolean },
  offers: [] as unknown[],
  challenges: [] as unknown[],
  trainer: { id: 1 } as unknown,
  lineup: undefined as unknown,
  unavailability: new Map<number, unknown>(),
}))

vi.mock('../../api/hooks', () => ({
  useAcademy: () => ({ data: data.academy }),
  useOffersReceived: () => ({ data: data.offers }),
  useFriendlyChallengesReceived: () => ({ data: data.challenges }),
  useTeamTrainer: () => ({ data: data.trainer, isSuccess: true }),
  useMatchLineup: () => ({ data: data.lineup }),
  useCupTieLineup: () => ({ data: data.lineup }),
  useFriendlyLineup: () => ({ data: data.lineup }),
}))
vi.mock('../lineup/useSquadUnavailability', () => ({ useSquadUnavailability: () => data.unavailability }))

const player = (id: number, firstName: string, contractYearsRemaining = 3): Player => ({ id, firstName, lastName: 'X', contractYearsRemaining }) as Player
const squad = [player(1, 'Keeper'), player(2, 'LeftBack'), player(3, 'Bench')]
const nextMatch: TeamMatchSummary = {
  kind: 'league',
  matchId: 12,
  opponentId: 9,
  opponentName: 'Rivals',
  isHome: true,
  matchDate: '2099-01-01T00:00:00Z',
  played: false,
  homeGoals: 0,
  awayGoals: 0,
}
const finances = (overrides: Partial<FinancesResponse> = {}) =>
  ({ budget: 1000, transactions: [], maxDebt: 100, insolvent: false, recommendedTicketPrice: 11, stadium: { ticketPrice: 11 }, ...overrides }) as FinancesResponse

function renderPanel(props: Partial<Parameters<typeof PendingPanel>[0]> = {}) {
  return render(
    <MemoryRouter>
      <PendingPanel teamId={7} players={squad} finances={finances()} nextMatch={nextMatch} {...props} />
    </MemoryRouter>,
  )
}

describe('PendingPanel', () => {
  beforeEach(() => {
    data.academy = { canPromote: false }
    data.offers = []
    data.challenges = []
    data.trainer = { id: 1 }
    data.lineup = undefined
    data.unavailability = new Map()
    window.localStorage.clear()
    setPhone(false)
  })
  afterEach(cleanup)

  it('shows no panel when nothing is pending', () => {
    const { container } = renderPanel()

    expect(container).toBeEmptyDOMElement()
  })

  it('names the projected replacement of an injured starter and links to that match', () => {
    const lineup: Lineup = {
      matchId: 12,
      teamId: 7,
      formation: 0,
      positions: [
        { playerId: 1, position: 0 },
        { playerId: 2, position: 2 },
      ],
      substitutions: [],
      projectedReplacements: [{ playerOutId: 2, playerInId: 3, position: 2 }],
    }
    data.lineup = lineup
    data.unavailability = new Map([[2, { playerId: 2 } as PlayerUnavailability]])

    renderPanel()

    expect(screen.getByText(/LeftBack X is unavailable for your next match against Rivals; Bench X would take the slot/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Set lineup' })).toHaveAttribute('href', '/match/12')
  })

  it('ignores an injured player who is not in the next fixture\'s XI', () => {
    data.lineup = { matchId: 12, teamId: 7, formation: 0, positions: [{ playerId: 1, position: 0 }], substitutions: [] }
    data.unavailability = new Map([[3, { playerId: 3 } as PlayerUnavailability]])

    const { container } = renderPanel()

    expect(container).toBeEmptyDOMElement()
  })

  it('offers no snooze on insolvency, and snoozes any other item for the day', () => {
    data.offers = [{}]
    renderPanel({ finances: finances({ insolvent: true }) })

    expect(screen.getAllByRole('button', { name: 'Remind me tomorrow' })).toHaveLength(1) // the offers row only
    fireEvent.click(screen.getByRole('button', { name: 'Remind me tomorrow' }))

    expect(screen.queryByText(/transfer offer/)).toBeNull()
    expect(screen.getByText(/deep in debt/)).toBeInTheDocument()
    expect(window.localStorage.getItem('striker.pending.snoozed')).toContain('offers:1')
  })

  it('brings a snoozed item back when its content changes', () => {
    data.offers = [{}]
    const first = renderPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Remind me tomorrow' }))
    first.unmount()

    data.offers = [{}, {}]
    renderPanel()

    expect(screen.getByText(/2 transfer offers/)).toBeInTheDocument()
  })

  describe('on a phone', () => {
    const fiveItems = () => {
      data.offers = [{}]
      data.challenges = [{}]
      data.academy = { canPromote: true }
      data.trainer = null
      return { players: [...squad, player(4, 'Old', 1)] }
    }

    it('shows 3 items and "Show all (5)", which reveals the rest', () => {
      setPhone(true)
      renderPanel(fiveItems())

      expect(screen.getAllByRole('alert')).toHaveLength(3)
      fireEvent.click(screen.getByRole('button', { name: 'Show all (5)' }))
      expect(screen.getAllByRole('alert')).toHaveLength(5)
      expect(screen.getByRole('button', { name: 'Show less' })).toBeInTheDocument()
    })

    it('shows every item and no toggle on a wide screen', () => {
      renderPanel(fiveItems())

      expect(screen.getAllByRole('alert')).toHaveLength(5)
      expect(screen.queryByRole('button', { name: /Show all/ })).toBeNull()
    })
  })
})
