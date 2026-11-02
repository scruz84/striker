import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  FORMATION_FOUR_THREE_THREE,
  MATCH_KIND_LEAGUE,
  POSITION_GOALKEEPER,
  POSITION_STRIKER,
  type OpponentPreview,
  type TeamQualification,
} from '../../api/types'
import '../../i18n'
import { OpponentPreviewCard } from './OpponentPreviewCard'

const hook = vi.hoisted(() => ({ result: { data: undefined, isLoading: false } as { data: unknown; isLoading: boolean }, calls: [] as unknown[][] }))
const zone = vi.hoisted(() => ({ props: [] as Array<Record<string, unknown>> }))

vi.mock('../../api/hooks', () => ({
  useOpponentPreview: (...args: unknown[]) => {
    hook.calls.push(args)
    return hook.result
  },
}))
vi.mock('../match/ZoneComparisonRows', () => ({
  ZoneComparisonRows: (props: Record<string, unknown>) => {
    zone.props.push(props)
    return <div data-testid="zone-duel" />
  },
}))
// Desktop width: the card starts expanded.
vi.mock('@mui/material/useMediaQuery', () => ({ default: () => false }))

const qua = (n: number): TeamQualification => ({
  goalkeeper: n, leftDefense: n, rightDefense: n, defense: n, leftMidfield: n, rightMidfield: n, midfield: n, leftAttack: n, rightAttach: n, attack: n,
})

const fullPreview: OpponentPreview = {
  opponent: {
    id: 9,
    name: 'Rivals FC',
    position: 3,
    form: [
      { kind: 'league', matchId: 31, opponentId: 5, opponentName: 'Beta', isHome: true, matchDate: '2026-10-20T20:00:00Z', played: true, homeGoals: 2, awayGoals: 0 },
      { kind: 'cup', matchId: 12, opponentId: 6, opponentName: 'Gamma', isHome: false, matchDate: '2026-10-13T20:00:00Z', played: true, homeGoals: 1, awayGoals: 1 },
    ],
  },
  likely: {
    kind: 'league',
    matchId: 31,
    formation: FORMATION_FOUR_THREE_THREE,
    players: [
      { playerId: 1, name: 'Ana Keeper', position: POSITION_GOALKEEPER },
      { playerId: 2, name: 'Bea Striker', position: POSITION_STRIKER },
    ],
  },
  inForm: [{ playerId: 2, name: 'Bea Striker', position: POSITION_STRIKER, averageRating: 7.8, appearances: 4 }],
  projection: { own: qua(60), opponent: qua(50), ownGoalkeeper: 70, opponentGoalkeeper: 55 },
}

function renderCard() {
  render(
    <MemoryRouter>
      <OpponentPreviewCard kind={MATCH_KIND_LEAGUE} matchId={77} teamId={4} teamName="Home FC" />
    </MemoryRouter>,
  )
}

describe('OpponentPreviewCard', () => {
  afterEach(() => {
    cleanup()
    zone.props.length = 0
    hook.calls.length = 0
  })

  it('shows the form, the likely XI, the in-form players and the projected zone duel', () => {
    hook.result = { data: fullPreview, isLoading: false }

    renderCard()

    expect(hook.calls[0]).toEqual(['league', '77'])
    expect(screen.getByText('Preview: Rivals FC')).toBeInTheDocument()
    expect(screen.getByText('Currently 3º in the league')).toBeInTheDocument()
    expect(screen.getByLabelText(/Win: Beta 2-0/)).toBeInTheDocument()
    expect(screen.getByLabelText(/Draw: Gamma 1-1/)).toBeInTheDocument()
    expect(screen.getByText(/Likely XI · /)).toBeInTheDocument()
    expect(screen.getByText('Ana Keeper')).toBeInTheDocument()
    expect(screen.getByText(/Bea Striker · 7\.8/)).toBeInTheDocument()
    expect(screen.getByTestId('zone-duel')).toBeInTheDocument()
    expect(zone.props[0]).toMatchObject({ homeId: 4, homeName: 'Home FC', awayId: 9, awayName: 'Rivals FC', homeGoalkeeperRating: 70, awayGoalkeeperRating: 55 })
    expect(zone.props[0].homeQua).toEqual(qua(60))
  })

  it('says there is no recent data, with no XI or duel, for an opponent without history', () => {
    hook.result = {
      data: { opponent: { id: 9, name: 'Rivals FC', position: 0, form: [] }, likely: null, inForm: [], projection: null },
      isLoading: false,
    }

    renderCard()

    expect(screen.getByText(/has no recent league or cup matches yet/)).toBeInTheDocument()
    expect(screen.queryByTestId('zone-duel')).toBeNull()
    expect(screen.queryByText(/Likely XI/)).toBeNull()
  })

  it('renders nothing when the preview is refused (locked or not the owner\'s fixture)', () => {
    hook.result = { data: undefined, isLoading: false }

    renderCard()

    expect(screen.queryByText(/Preview:/)).toBeNull()
  })
})
