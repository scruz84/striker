import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { InternationalCupPage } from './InternationalCupPage'

// Only the europe and americas regions have an edition; asia and africa have none yet.
const cupIdsByRegion: Record<string, number> = { europe: 11, americas: 22 }
const cupsById: Record<number, { name: string; season: number }> = {
  11: { name: 'European Cup', season: 3 },
  22: { name: 'Copa America', season: 1 },
}
let userCountry = 'ar'

vi.mock('../../api/hooks', () => ({
  useMe: () => ({ data: { country: userCountry } }),
  useLatestInternationalCupId: (region: string | undefined) => ({
    isLoading: false,
    data: region && cupIdsByRegion[region] ? { cupId: cupIdsByRegion[region] } : undefined,
  }),
  useCupDetails: (cupId: number | undefined) => ({
    isLoading: false,
    data:
      cupId !== undefined && cupsById[cupId]
        ? { cup: { cupId, completed: false, championTeamName: '', ...cupsById[cupId] }, ties: [] }
        : undefined,
  }),
}))

function renderPage() {
  return render(
    <MemoryRouter>
      <InternationalCupPage />
    </MemoryRouter>,
  )
}

describe('InternationalCupPage region tabs', () => {
  beforeEach(async () => {
    userCountry = 'ar'
    await i18n.changeLanguage('es')
  })

  afterEach(async () => {
    cleanup()
    await i18n.changeLanguage('en')
  })

  it('offers one tab per region', () => {
    renderPage()
    expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual([
      'Copa de Europa',
      'Copa América',
      'Copa de Asia',
      'Copa de África',
    ])
  })

  it("opens on the user's own region", () => {
    renderPage()
    expect(screen.getByRole('tab', { name: 'Copa América', selected: true })).toBeInTheDocument()
    expect(screen.getByText('Copa América - Edición 1')).toBeInTheDocument()
  })

  it("shows another region's cup to a user from a different region", () => {
    renderPage()
    fireEvent.click(screen.getByRole('tab', { name: 'Copa de Europa' }))
    expect(screen.getByText('Copa de Europa - Edición 3')).toBeInTheDocument()
  })

  it('explains when a region has no edition yet', () => {
    renderPage()
    fireEvent.click(screen.getByRole('tab', { name: 'Copa de Asia' }))
    expect(screen.getByText('Aún no hay edición')).toBeInTheDocument()
  })
})
