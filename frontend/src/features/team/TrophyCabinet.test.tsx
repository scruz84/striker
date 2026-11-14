import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import type { SeasonHistoryEntry } from '../../api/types'
import i18n from '../../i18n'
import { HonoursSummary, TrophyCabinet } from './TeamDetailsPage'

function entry(
  season: number,
  leagueId: number | null,
  leagueName: string | null,
  extra: Partial<SeasonHistoryEntry> = {},
): SeasonHistoryEntry {
  return { teamId: 1, season, leagueId, leagueName, finalPosition: 3, promoted: false, relegated: false, cupResult: null, ...extra }
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

  it('shows a domestic and an international cup win in the same season', () => {
    renderCabinet([
      entry(5, 7, 'First', { cupResult: 'Champion', internationalCup: { cupId: 41, region: 'europe', edition: 3 } }),
    ])

    expect(screen.getAllByText('Cup winner').length).toBeGreaterThan(0)
    const title = screen.getAllByRole('link').find((link) => link.textContent === 'European Cup winner (Ed. 3)')
    expect(title).toBeDefined()
    fireEvent.click(title!)
    expect(screen.getByTestId('location')).toHaveTextContent('/international-cup?region=europe&cup=41')
  })

  it('shows an international title in a season still without a league line', () => {
    renderCabinet([entry(6, null, null, { finalPosition: null, internationalCup: { cupId: 9, region: 'americas', edition: 1 } })])

    expect(screen.getAllByText('Copa America winner (Ed. 1)').length).toBeGreaterThan(0)
  })
})

describe('HonoursSummary', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('counts tier-1 league titles, cups and each international cup', () => {
    render(
      <HonoursSummary
        history={[
          entry(6, 7, 'First', { finalPosition: 1, leagueTier: 1, internationalCup: { cupId: 4, region: 'europe', edition: 2 } }),
          entry(5, 7, 'First', { finalPosition: 1, leagueTier: 1, cupResult: 'Champion' }),
          entry(4, 12, 'Second', { finalPosition: 1, leagueTier: 2 }),
        ]}
      />,
    )

    expect(screen.getByTestId('honours-summary')).toHaveTextContent('League titles 2 · Cups 1 · European Cup 1')
  })

  it('shows nothing for a club without titles', () => {
    const { container } = render(<HonoursSummary history={[entry(2, 7, 'First', { leagueTier: 1 })]} />)

    expect(container).toBeEmptyDOMElement()
  })
})
