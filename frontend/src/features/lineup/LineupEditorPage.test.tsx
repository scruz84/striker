import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { LineupSubstitution } from '../../api/types'
import '../../i18n'
import { LineupEditorPage } from './LineupEditorPage'

const setFriendlyLineup = vi.hoisted(() => vi.fn())
const setLineup = vi.hoisted(() => vi.fn())
const planned: LineupSubstitution[] = [{ playerOutId: 1, playerInId: 2, minute: 60 }]

vi.mock('../../api/hooks', () => {
  const lineup = { matchId: 0, teamId: 7, formation: 0, positions: [], substitutions: [] }
  const mutation = (mutate: unknown) => () => ({ mutate, isPending: false, isSuccess: false, isError: false })
  return {
    useMe: () => ({ data: { id: 1 } }),
    useUserTeam: () => ({ data: { team: { id: 7, name: 'Test FC' }, players: [] }, isLoading: false }),
    useLineup: () => ({ data: lineup, isLoading: false }),
    useFriendlyDefaultLineup: () => ({ data: lineup, isLoading: false }),
    useTeamMatches: () => ({ data: [] }),
    useSetLineup: mutation(setLineup),
    useSetFriendlyDefaultLineup: mutation(setFriendlyLineup),
  }
})

// The real form needs a whole squad; what matters here is how each tab drives it.
vi.mock('./MatchLineupForm', () => ({
  MatchLineupForm: (props: {
    substitutionsEnabled?: boolean
    isFriendly?: boolean
    onSave: (formation: number, positions: never[], substitutions: LineupSubstitution[]) => void
  }) => (
    <div>
      <span data-testid="flags">{`subs=${!!props.substitutionsEnabled} friendly=${!!props.isFriendly}`}</span>
      <button onClick={() => props.onSave(2, [], planned)}>save</button>
    </div>
  ),
}))

describe('LineupEditorPage standing substitutions', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows the substitutions planner on the League/Cup tab and saves its own set', () => {
    render(<LineupEditorPage />)

    expect(screen.getByTestId('flags').textContent).toBe('subs=true friendly=false')
    fireEvent.click(screen.getByText('save'))
    expect(setLineup).toHaveBeenCalledWith({ formation: 2, positions: [], substitutions: planned })
    expect(setFriendlyLineup).not.toHaveBeenCalled()
  })

  it('shows the substitutions planner on the Friendly tab with friendly rules and saves them', () => {
    render(<LineupEditorPage />)
    fireEvent.click(screen.getAllByRole('tab')[1])

    expect(screen.getByTestId('flags').textContent).toBe('subs=true friendly=true')
    expect(screen.queryByText(/shared with the league|se comparten con/i)).toBeNull()
    fireEvent.click(screen.getByText('save'))
    expect(setFriendlyLineup).toHaveBeenCalledWith({ formation: 2, positions: [], substitutions: planned })
    expect(setLineup).not.toHaveBeenCalled()
  })
})
