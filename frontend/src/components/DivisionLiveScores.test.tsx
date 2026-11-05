import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import type { LeagueLiveScore } from '../api/types'
import { setPhone } from '../features/admin/testPhone'
import i18n from '../i18n'
import { DivisionLiveScores } from './DivisionLiveScores'

function entry(overrides: Partial<LeagueLiveScore> = {}): LeagueLiveScore {
  return {
    matchId: 1,
    weekNumber: 12,
    matchDate: new Date(2026, 10, 5, 18, 30).toISOString(),
    home: 1,
    homeName: 'Alpha',
    away: 2,
    awayName: 'Beta',
    status: 'live',
    minute: 63,
    homeGoals: 1,
    awayGoals: 2,
    ...overrides,
  }
}

const renderList = (entries: LeagueLiveScore[]) =>
  render(
    <MemoryRouter>
      <DivisionLiveScores entries={entries} />
    </MemoryRouter>,
  )

beforeEach(async () => {
  setPhone(false)
  await i18n.changeLanguage('en')
})
afterEach(cleanup)

describe('DivisionLiveScores', () => {
  it('lists each match with its goals, state chip and a link to the match page', () => {
    renderList([
      entry({ matchId: 1 }),
      entry({ matchId: 2, homeName: 'Gamma', awayName: 'Delta', status: 'halftime', minute: 45, homeGoals: 0, awayGoals: 0 }),
      entry({ matchId: 3, homeName: 'Eta', awayName: 'Theta', status: 'finished', minute: 90, homeGoals: 2, awayGoals: 2 }),
      entry({ matchId: 4, homeName: 'Iota', awayName: 'Kappa', status: 'pending', minute: 0, homeGoals: 0, awayGoals: 0 }),
    ])
    expect(screen.getByText("63'")).toBeTruthy()
    expect(screen.getByText('Half-time')).toBeTruthy()
    expect(screen.getByText('FT')).toBeTruthy()
    expect(screen.getByText('18:30')).toBeTruthy()
    expect(screen.getAllByRole('link').map((link) => link.getAttribute('href'))).toEqual(['/match/1', '/match/2', '/match/3', '/match/4'])
    // The live row shows its goals; the pending row shows none, only the kickoff time.
    expect((screen.getByText('Alpha').closest('a') as HTMLElement).textContent).toBe("ALAlpha1BEBeta263'")
    expect((screen.getByText('Iota').closest('a') as HTMLElement).textContent).toBe('IOIotaKAKappa18:30')
  })

  it('expands by default on desktop and lists every match', () => {
    renderList([entry({ matchId: 1 }), entry({ matchId: 2, homeName: 'Gamma', awayName: 'Delta' })])
    expect(screen.getAllByRole('link')).toHaveLength(2)
    expect(screen.getByText('Rest of the matchday')).toBeTruthy()
    expect(screen.getByRole('button', { name: /2 matches · 6 goals/ })).toBeTruthy()
  })

  it('starts collapsed on a 360 px phone behind the summary toggle, and a tap lists the matches', () => {
    setPhone(true)
    renderList([entry({ matchId: 1 }), entry({ matchId: 2, homeName: 'Gamma', awayName: 'Delta' })])
    expect(screen.queryAllByRole('link')).toHaveLength(0)
    const toggle = screen.getByRole('button', { name: /2 matches · 6 goals/ })
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    fireEvent.click(toggle)
    expect(screen.getAllByRole('link')).toHaveLength(2)
  })

  it('reads the Spanish summary "1 partido · 0 goles"', async () => {
    setPhone(true)
    await i18n.changeLanguage('es')
    renderList([entry({ homeGoals: 0, awayGoals: 0, minute: 10 })])
    expect(screen.getByRole('button', { name: /1 partido · 0 goles/ })).toBeTruthy()
  })
})
