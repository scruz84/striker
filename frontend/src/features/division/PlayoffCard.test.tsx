import { fireEvent, render, screen, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import type { CupTie } from '../../api/types'
import i18n from '../../i18n'
import { PlayoffCard } from './PlayoffCard'

const HOUR = 60 * 60 * 1000

function tie(overrides: Partial<CupTie>): CupTie {
  return {
    tieId: 1,
    cupId: 9,
    round: 1,
    slotNumber: 1,
    homeTeamId: 10,
    homeTeamName: 'Upper Club',
    awayTeamId: 20,
    awayTeamName: 'Lower Club',
    matchDate: new Date(Date.now() + 20 * HOUR).toISOString(),
    played: false,
    homeGoals: 0,
    awayGoals: 0,
    wentToPenalties: false,
    winnerTeamId: null,
    ...overrides,
  }
}

function Location() {
  return <div data-testid="location">{useLocation().pathname}</div>
}

function renderCard(ties: CupTie[], myTeamId?: number) {
  render(
    <MemoryRouter initialEntries={['/division/7']}>
      <Routes>
        <Route
          path="*"
          element={
            <>
              <PlayoffCard ties={ties} myTeamId={myTeamId} />
              <Location />
            </>
          }
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('PlayoffCard', () => {
  it('shows an upcoming tie with its kickoff and no score', () => {
    renderCard([tie({})])
    const card = screen.getByRole('region', { name: i18n.t('division.playoff.title') })
    expect(within(card).getByText('Upper Club')).toBeInTheDocument()
    expect(within(card).getByText('Lower Club')).toBeInTheDocument()
    expect(within(card).queryByText('0')).not.toBeInTheDocument()
  })

  it('shows the score and the penalties once the broadcast is over', () => {
    renderCard([
      tie({ played: true, matchDate: new Date(Date.now() - 3 * HOUR).toISOString(), homeGoals: 1, awayGoals: 1, wentToPenalties: true, winnerTeamId: 20 }),
    ])
    expect(screen.getAllByText('1')).toHaveLength(2)
    expect(screen.getByText(i18n.t('cup.pensWinner', { team: 'Lower Club' }))).toBeInTheDocument()
  })

  it('keeps a tie still broadcasting live, with no result', () => {
    renderCard([tie({ played: true, matchDate: new Date(Date.now() - 30 * 60 * 1000).toISOString(), homeGoals: 2, awayGoals: 0, winnerTeamId: 10 })])
    expect(screen.getByText(i18n.t('match.live'))).toBeInTheDocument()
    expect(screen.queryByText('2')).not.toBeInTheDocument()
  })

  it('opens the tie page and marks my team', () => {
    renderCard([tie({ tieId: 42 })], 20)
    expect(screen.getByText(i18n.t('common.you'))).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('division.playoff.openTie', { home: 'Upper Club', away: 'Lower Club' }) }))
    expect(screen.getByTestId('location')).toHaveTextContent('/cup/ties/42')
  })

  it('fits a 360 px phone: team names wrap instead of widening the card', () => {
    renderCard([tie({ homeTeamName: 'A very long club name that would not fit on one line of a phone' })])
    const name = screen.getByText('A very long club name that would not fit on one line of a phone')
    const cell = name.closest('p') as HTMLElement
    const style = window.getComputedStyle(cell)
    expect(style.overflowWrap).toBe('anywhere')
    expect(style.minWidth).toBe('0px')
  })
})
