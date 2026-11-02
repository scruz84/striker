import { fireEvent, render as baseRender, screen } from '@testing-library/react'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import type { ReactElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  FORMATION_BALANCED,
  POSITION_DEFENSE,
  POSITION_GOALKEEPER,
  POSITION_LATERAL_DEFENSE_LEFT,
  POSITION_LATERAL_DEFENSE_RIGHT,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_LATERAL_MIDFIELD_RIGHT,
  POSITION_MIDFIELD,
  POSITION_STRIKER,
  type Lineup,
  type Player,
} from '../../api/types'
import i18n from '../../i18n'
import { MatchLineupForm } from './MatchLineupForm'

const testTheme = createTheme({ palette: { rating: ['#ccc', '#bbb', '#aaa', '#999', '#888'] } })
const render = (ui: ReactElement) => baseRender(<ThemeProvider theme={testTheme}>{ui}</ThemeProvider>)

const guard = vi.hoisted(() => ({ dirty: [] as boolean[] }))

vi.mock('../../api/hooks', () => ({
  useMe: () => ({ data: { id: 1 } }),
  useUserTeam: () => ({ data: { team: { id: 7, name: 'Test FC' }, players: [] } }),
  useTeamMatches: () => ({ data: [] }),
  useMatchDetails: () => ({ data: undefined, isFetching: false, isError: false }),
}))
vi.mock('./useUnsavedChangesGuard', () => ({
  useUnsavedChangesGuard: (dirty: boolean) => {
    guard.dirty.push(dirty)
  },
}))

const slots = [
  POSITION_GOALKEEPER,
  POSITION_DEFENSE,
  POSITION_DEFENSE,
  POSITION_LATERAL_DEFENSE_LEFT,
  POSITION_LATERAL_DEFENSE_RIGHT,
  POSITION_MIDFIELD,
  POSITION_MIDFIELD,
  POSITION_LATERAL_MIDFIELD_LEFT,
  POSITION_LATERAL_MIDFIELD_RIGHT,
  POSITION_STRIKER,
  POSITION_STRIKER,
]
const players = slots.map((_, i) => ({ id: i + 1, firstName: 'P', lastName: String(i + 1), shirtNumber: i + 1, goalKeeper: 50, defense: 50, lateral: 50, pass: 50, midfield: 50, dribbling: 50, shot: 50 }) as Player)
const lineup: Lineup = {
  matchId: 0,
  teamId: 7,
  formation: FORMATION_BALANCED,
  positions: slots.map((position, i) => ({ position, playerId: i + 1 })),
  substitutions: [],
  mentality: 0,
}

describe('MatchLineupForm mentality', () => {
  beforeEach(async () => {
    guard.dirty.length = 0
    await i18n.changeLanguage('en')
  })

  it('starts from the saved mentality, marks the lineup dirty when changed and sends it on save', () => {
    const onSave = vi.fn()
    render(
      <MemoryRouter>
        <MatchLineupForm titleKey="lineup.title" teamId={7} players={players} lineup={lineup} onSave={onSave} isSaving={false} isError={false} isSuccess={false} />
      </MemoryRouter>,
    )
    expect(screen.getByRole('button', { name: 'Balanced' })).toHaveAttribute('aria-pressed', 'true')
    expect(guard.dirty.at(-1)).toBe(false)

    fireEvent.click(screen.getByRole('button', { name: 'Attacking' }))
    expect(screen.getByRole('button', { name: 'Attacking' })).toHaveAttribute('aria-pressed', 'true')
    expect(guard.dirty.at(-1)).toBe(true)

    fireEvent.click(screen.getByRole('button', { name: 'Save lineup' }))
    expect(onSave).toHaveBeenCalledWith(FORMATION_BALANCED, lineup.positions, [], 1)
  })

  it('shows a lineup saved as Defensive as Defensive', () => {
    render(
      <MemoryRouter>
        <MatchLineupForm
          titleKey="lineup.title"
          teamId={7}
          players={players}
          lineup={{ ...lineup, mentality: -1 }}
          onSave={vi.fn()}
          isSaving={false}
          isError={false}
          isSuccess={false}
        />
      </MemoryRouter>,
    )
    expect(screen.getByRole('button', { name: 'Defensive' })).toHaveAttribute('aria-pressed', 'true')
  })
})
