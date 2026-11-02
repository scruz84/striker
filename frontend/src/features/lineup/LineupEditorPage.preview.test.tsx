import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { TeamMatchSummary } from '../../api/types'
import '../../i18n'
import { LineupEditorPage } from './LineupEditorPage'

const fixtures = vi.hoisted(() => ({ matches: [] as TeamMatchSummary[] }))

vi.mock('../../api/hooks', () => {
  const lineup = { matchId: 0, teamId: 7, formation: 0, positions: [], substitutions: [] }
  const mutation = () => ({ mutate: vi.fn(), isPending: false, isSuccess: false, isError: false })
  return {
    useMe: () => ({ data: { id: 1 } }),
    useUserTeam: () => ({ data: { team: { id: 7, name: 'Test FC' }, players: [] }, isLoading: false }),
    useLineup: () => ({ data: lineup, isLoading: false }),
    useFriendlyDefaultLineup: () => ({ data: lineup, isLoading: false }),
    useTeamMatches: () => ({ data: fixtures.matches }),
    useSetLineup: mutation,
    useSetFriendlyDefaultLineup: mutation,
  }
})
vi.mock('./MatchLineupForm', () => ({ MatchLineupForm: () => <div>lineup form</div> }))
vi.mock('./OpponentPreviewCard', () => ({
  OpponentPreviewCard: (props: { kind: string; matchId: number; teamId: number }) => (
    <div data-testid="preview">{`${props.kind}:${props.matchId}:${props.teamId}`}</div>
  ),
}))

function fixture(kind: TeamMatchSummary['kind'], matchId: number, inMinutes: number): TeamMatchSummary {
  return {
    kind,
    matchId,
    opponentId: 100 + matchId,
    opponentName: `Rival ${matchId}`,
    isHome: true,
    matchDate: new Date(Date.now() + inMinutes * 60_000).toISOString(),
    played: false,
    homeGoals: 0,
    awayGoals: 0,
  }
}

describe('LineupEditorPage opponent preview', () => {
  afterEach(cleanup)

  it('previews the next fixture, whatever its competition', () => {
    fixtures.matches = [fixture('cup', 5, 24 * 60), fixture('league', 6, 3 * 24 * 60)]

    render(<LineupEditorPage />)

    expect(screen.getByTestId('preview')).toHaveTextContent('cup:5:7')
  })

  it('skips a next fixture whose lineup is already locked', () => {
    fixtures.matches = [fixture('league', 8, 10), fixture('friendly', 9, 24 * 60)]

    render(<LineupEditorPage />)

    expect(screen.getByTestId('preview')).toHaveTextContent('friendly:9:7')
  })

  it('shows no preview when every upcoming fixture is locked or there are none', () => {
    fixtures.matches = [fixture('league', 8, 10)]

    render(<LineupEditorPage />)

    expect(screen.queryByTestId('preview')).toBeNull()
  })
})
