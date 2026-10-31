import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { LeaguesPage } from './LeaguesPage'

vi.mock('../../api/hooks', () => ({
  useLeagueDirectory: () => ({
    isLoading: false,
    data: ['fr', 'de', 'es', 'en'].map((country, i) => ({
      country,
      leagues: [{ leagueId: i + 1, name: `League ${country}`, tier: 1, groupNumber: 1 }],
    })),
  }),
}))

function renderPage() {
  return render(
    <MemoryRouter>
      <LeaguesPage />
    </MemoryRouter>,
  )
}

const tabNames = () => screen.getAllByRole('tab').map((tab) => tab.textContent)

describe('LeaguesPage country tabs', () => {
  afterEach(async () => {
    cleanup()
    await i18n.changeLanguage('en')
  })

  it('lists the tabs alphabetically in Spanish', async () => {
    await i18n.changeLanguage('es')
    renderPage()
    expect(tabNames()).toEqual(['Alemania', 'España', 'Francia', 'Inglaterra'])
  })

  it('lists the tabs alphabetically in English', async () => {
    await i18n.changeLanguage('en')
    renderPage()
    expect(tabNames()).toEqual(['England', 'France', 'Germany', 'Spain'])
  })

  it('keeps the selected country across a language switch', async () => {
    await i18n.changeLanguage('en')
    renderPage()
    fireEvent.click(screen.getByRole('tab', { name: 'Spain' }))
    expect(screen.getByText('League es')).toBeInTheDocument()

    await i18n.changeLanguage('es')
    // The tabs re-sorted ("España" is now second), and Spain is still the selected one.
    expect(await screen.findByRole('tab', { name: 'España', selected: true })).toBeInTheDocument()
    expect(tabNames()).toEqual(['Alemania', 'España', 'Francia', 'Inglaterra'])
    expect(screen.getByText('League es')).toBeInTheDocument()
  })
})
