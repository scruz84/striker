import { render as baseRender, screen } from '@testing-library/react'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import type { ReactElement } from 'react'
import { describe, expect, it } from 'vitest'
import { UNAVAILABLE_INJURY, type Player, type PlayerUnavailability } from '../../api/types'
import i18n from '../../i18n'
import { PlayerChip } from './PlayerChip'

// The chip colours its rating from the app theme's rating scale (src/theme.ts, which pulls in a
// colour library that doesn't load under jsdom), so the test supplies a plain scale of its own.
const testTheme = createTheme({ palette: { rating: ['#ccc', '#bbb', '#aaa', '#999', '#888'] } })
const render = (ui: ReactElement) => baseRender(<ThemeProvider theme={testTheme}>{ui}</ThemeProvider>)

function player(id: number, lastName: string): Player {
  return {
    id,
    teamId: 1,
    firstName: 'P',
    lastName,
    birthDay: '2000-01-01',
    goalKeeper: 50,
    defense: 50,
    lateral: 50,
    pass: 50,
    midfield: 50,
    dribbling: 50,
    shot: 50,
    header: false,
    quick: false,
    creative: false,
    technical: false,
    stamina: 70,
    form: 50,
    wage: 0,
    age: 25,
    contractYearsRemaining: 1,
  }
}

const injured: PlayerUnavailability = { playerId: 1, reason: UNAVAILABLE_INJURY, matchesRemaining: 3 }

describe('PlayerChip on the pitch', () => {
  it('says who replaces an unavailable starter', () => {
    render(<PlayerChip player={player(1, 'Injured')} variant="field" unavailability={injured} replacedBy={player(2, 'Garcia')} />)

    expect(screen.getByText(i18n.t('lineup.replacedBy', { name: 'Garcia' }))).toBeInTheDocument()
    expect(screen.getByRole('img')).toHaveAttribute(
      'aria-label',
      `${i18n.t('playerPage.status.injured', { count: 3 })} · ${i18n.t('lineup.replacedBy', { name: 'P Garcia' })}`,
    )
  })

  it('shows no replacement for an available starter', () => {
    render(<PlayerChip player={player(1, 'Fit')} variant="field" replacedBy={player(2, 'Garcia')} />)

    expect(screen.queryByText(i18n.t('lineup.replacedBy', { name: 'Garcia' }))).not.toBeInTheDocument()
  })

  it('shows no replacement line when none is known', () => {
    render(<PlayerChip player={player(1, 'Injured')} variant="field" unavailability={injured} />)

    expect(screen.queryByText(/Replaced by|Le sustituye/)).not.toBeInTheDocument()
  })
})
