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
import { lightTheme } from '../../theme'
import { TOUCH } from '../../touchMedia'
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
    expect(onSave).toHaveBeenCalledWith(FORMATION_BALANCED, lineup.positions, [], 1, [], {})
  })

  // ui-theme "Compact mentality picker": the app theme draws it as M3 segmented buttons while the
  // picker keeps its own compact height (32px with a pointer, 36px on touch screens).
  it('keeps its compact height with the app theme and takes the segmented-button colours', () => {
    baseRender(
      <ThemeProvider theme={lightTheme}>
        <MemoryRouter>
          <MatchLineupForm titleKey="lineup.title" teamId={7} players={players} lineup={lineup} onSave={vi.fn()} isSaving={false} isError={false} isSuccess={false} />
        </MemoryRouter>
      </ThemeProvider>,
    )
    const balanced = screen.getByRole('button', { name: 'Balanced' })
    const style = getComputedStyle(balanced)
    expect(style.minHeight).toBe('32px')
    expect(style.backgroundColor).toBe('rgb(213, 232, 207)')
    expect(getComputedStyle(screen.getByRole('button', { name: 'Defensive' })).borderTopLeftRadius).toBe('9999px')
    const touchMinHeights = Array.from(document.styleSheets)
      .flatMap((sheet) => Array.from(sheet.cssRules))
      .filter((rule): rule is CSSMediaRule => rule instanceof CSSMediaRule && `@media ${rule.conditionText}` === TOUCH)
      .flatMap((rule) => Array.from(rule.cssRules))
      .filter((rule): rule is CSSStyleRule => rule instanceof CSSStyleRule && balanced.matches(rule.selectorText))
      .filter((rule) => rule.style.getPropertyValue('min-height'))
      // The winning declaration: highest specificity (class count, enough for emotion's
      // class-only selectors), then the later rule.
      .map((rule, index) => ({ value: rule.style.getPropertyValue('min-height'), classes: rule.selectorText.split('.').length, index }))
      .sort((a, b) => a.classes - b.classes || a.index - b.index)
    expect(touchMinHeights.at(-1)?.value).toBe('36px')
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

describe('MatchLineupForm conditional orders', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  function renderForm(onSave = vi.fn(), ordersEnabled = true) {
    render(
      <MemoryRouter>
        <MatchLineupForm
          titleKey="lineup.title"
          teamId={7}
          players={players}
          lineup={lineup}
          onSave={onSave}
          isSaving={false}
          isError={false}
          isSuccess={false}
          substitutionsEnabled
          ordersEnabled={ordersEnabled}
        />
      </MemoryRouter>,
    )
    return onSave
  }

  it('adds an order, rejects a from-minute outside 1-89 and sends the order on save', () => {
    const onSave = renderForm()
    fireEvent.click(screen.getByRole('button', { name: 'Add order' }))
    const minute = screen.getByLabelText('From minute')

    fireEvent.change(minute, { target: { value: '0' } })
    expect(screen.getByText('Between 1 and 89')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save lineup' })).toBeDisabled()
    // Above the range the field stops at 89.
    fireEvent.change(minute, { target: { value: '95' } })
    expect(minute).toHaveValue('89')

    fireEvent.change(minute, { target: { value: '70' } })
    expect(screen.queryByText('Between 1 and 89')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save lineup' }))
    expect(onSave).toHaveBeenCalledWith(FORMATION_BALANCED, lineup.positions, [], 0, [{ fromMinute: 70, condition: 0, action: 0, mentality: 1 }], {})
  })

  it('allows at most 3 orders', () => {
    renderForm()
    const add = screen.getByRole('button', { name: 'Add order' })
    fireEvent.click(add)
    fireEvent.click(add)
    fireEvent.click(add)
    expect(screen.getAllByText(/^Order \d$/)).toHaveLength(3)
    expect(add).toBeDisabled()
  })

  it('has no orders section where orders are not enabled', () => {
    renderForm(vi.fn(), false)
    expect(screen.queryByRole('button', { name: 'Add order' })).not.toBeInTheDocument()
  })
})

describe('MatchLineupForm lineup roles', () => {
  beforeEach(async () => {
    guard.dirty.length = 0
    await i18n.changeLanguage('en')
  })

  function renderForm(onSave = vi.fn(), saved: Lineup = lineup) {
    render(
      <MemoryRouter>
        <MatchLineupForm titleKey="lineup.title" teamId={7} players={players} lineup={saved} onSave={onSave} isSaving={false} isError={false} isSuccess={false} />
      </MemoryRouter>,
    )
    return onSave
  }
  // Picks a player up on the pitch, which opens the selected-player panel with the role toggles.
  const pick = (lastName: string) => fireEvent.click(screen.getByRole('button', { name: `P ${lastName} (undefined)` }))
  const toggle = (name: string) => fireEvent.click(screen.getByRole('button', { name }))
  const summary = () => screen.getByTestId('lineup-roles-summary').textContent

  it('moves a role from one player to another, marks the lineup dirty and sends the roles on save', () => {
    const onSave = renderForm()
    expect(summary()).toBe('Captain: automatic · Penalties: automatic · Free kicks: automatic')

    pick('2')
    toggle('Captain')
    toggle('Penalties')
    expect(screen.getByRole('button', { name: 'Captain' })).toHaveAttribute('aria-pressed', 'true')
    expect(summary()).toBe('Captain: 2 · Penalties: 2 · Free kicks: automatic')
    expect(guard.dirty.at(-1)).toBe(true)

    // The armband moves: one holder per role. (Tapping another pitch player while one is picked up
    // would swap them, so drop the first one before picking the next.)
    fireEvent.click(screen.getByRole('button', { name: 'Drop player' }))
    pick('6')
    toggle('Captain')
    expect(summary()).toBe('Captain: 6 · Penalties: 2 · Free kicks: automatic')
    expect(screen.getAllByTestId('role-badge').map((b) => b.textContent)).toEqual(expect.arrayContaining(['P', 'C']))

    fireEvent.click(screen.getByRole('button', { name: 'Save lineup' }))
    expect(onSave).toHaveBeenCalledWith(FORMATION_BALANCED, lineup.positions, [], 0, [], { captainId: 6, penaltyTakerId: 2 })
  })

  it('never lets the goalkeeper take set pieces, but he may be captain', () => {
    renderForm()
    pick('1')
    expect(screen.getByRole('button', { name: 'Penalties' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Free kicks' })).toBeDisabled()
    toggle('Captain')
    expect(summary()).toBe('Captain: 1 · Penalties: automatic · Free kicks: automatic')
  })

  it('starts from the saved roles and drops one whose player left the squad', () => {
    renderForm(vi.fn(), { ...lineup, captainId: 3, freeKickTakerId: 99 })
    expect(summary()).toBe('Captain: 3 · Penalties: automatic · Free kicks: automatic')
    expect(guard.dirty.at(-1)).toBe(false)
  })
})
