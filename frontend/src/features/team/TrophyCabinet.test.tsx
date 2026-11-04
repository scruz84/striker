import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import type { SeasonHistoryEntry } from '../../api/types'
import i18n from '../../i18n'
import { TrophyCabinet } from './TeamDetailsPage'

function entry(season: number, leagueId: number | null, leagueName: string | null): SeasonHistoryEntry {
  return { teamId: 1, season, leagueId, leagueName, finalPosition: 3, promoted: false, relegated: false, cupResult: null }
}

function Location() {
  const location = useLocation()
  return <div data-testid="location">{location.pathname + location.search}</div>
}

function renderCabinet(history: SeasonHistoryEntry[]) {
  render(
    <MemoryRouter initialEntries={['/team/1']}>
      <Routes>
        <Route path="*" element={<Location />} />
      </Routes>
      <TrophyCabinet history={history} />
    </MemoryRouter>,
  )
}

describe('TrophyCabinet', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('opens the past table of the league the team played in that season', () => {
    renderCabinet([entry(3, 7, 'First'), entry(2, 12, 'Second')])

    fireEvent.click(screen.getAllByRole('link').find((link) => link.getAttribute('href') === '/division/12?season=2')!)
    expect(screen.getByTestId('location')).toHaveTextContent('/division/12?season=2')
  })

  it('does not link a season with no recorded league', () => {
    renderCabinet([entry(1, null, null)])

    expect(screen.queryAllByRole('link')).toHaveLength(0)
  })
})
