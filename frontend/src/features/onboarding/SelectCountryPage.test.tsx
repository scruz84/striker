import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { SelectCountryPage } from './SelectCountryPage'

vi.mock('../../api/hooks', () => ({
  // The backend sends its own English `name` in KnownCountries order; the page must not show it.
  useOnboardingCountries: () => ({
    isLoading: false,
    data: [
      { code: 'es', name: 'Spain' },
      { code: 'en', name: 'England' },
      { code: 'fr', name: 'France' },
      { code: 'de', name: 'Germany' },
    ],
  }),
  useSelectCountry: () => ({ mutate: vi.fn(), isPending: false, isError: false }),
}))

describe('SelectCountryPage', () => {
  afterEach(async () => {
    cleanup()
    await i18n.changeLanguage('en')
  })

  it('offers localized names in alphabetical order', async () => {
    await i18n.changeLanguage('es')
    render(
      <MemoryRouter>
        <SelectCountryPage />
      </MemoryRouter>,
    )
    fireEvent.mouseDown(screen.getByRole('combobox'))
    const options = within(screen.getByRole('listbox')).getAllByRole('option')
    expect(options.map((option) => option.textContent)).toEqual(['Alemania', 'España', 'Francia', 'Inglaterra'])
  })
})
