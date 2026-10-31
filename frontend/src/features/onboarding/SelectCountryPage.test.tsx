import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { SelectCountryPage } from './SelectCountryPage'

const user = vi.hoisted(() => ({
  data: { id: '1', termsAccepted: true, termsVersion: 1 } as Record<string, unknown> | undefined,
}))
const selectCountry = vi.hoisted(() => vi.fn())
const acceptTerms = vi.hoisted(() => vi.fn())

vi.mock('../../api/hooks', () => ({
  useMe: () => user,
  useAcceptTerms: () => ({ mutateAsync: acceptTerms, isPending: false, isError: false }),
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
  useSelectCountry: () => ({ mutate: selectCountry, isPending: false, isError: false }),
}))

function renderPage() {
  render(
    <MemoryRouter>
      <SelectCountryPage />
    </MemoryRouter>,
  )
}

async function pickCountry(name: string) {
  fireEvent.mouseDown(screen.getByRole('combobox'))
  fireEvent.click(within(screen.getByRole('listbox')).getByRole('option', { name }))
}

describe('SelectCountryPage', () => {
  afterEach(async () => {
    cleanup()
    vi.clearAllMocks()
    user.data = { id: '1', termsAccepted: true, termsVersion: 1 }
    await i18n.changeLanguage('en')
  })

  it('tells a user whose club was released for inactivity, naming the club', () => {
    user.data = { id: '1', termsAccepted: true, termsVersion: 1, releasedTeamName: 'Atlético Riverside' }
    renderPage()
    expect(screen.getByText(/Atlético Riverside/)).toBeInTheDocument()
    expect(screen.getByText(/60 days/)).toBeInTheDocument()
  })

  it('assigns no club until the age and terms box is ticked, then records the acceptance first', async () => {
    user.data = { id: '1', termsAccepted: false, termsVersion: 1 }
    acceptTerms.mockResolvedValue({})
    renderPage()
    await pickCountry('Spain')
    const confirm = screen.getByRole('button', { name: i18n.t('onboarding.confirm') })
    expect(confirm).toBeDisabled()
    expect(screen.getByText(i18n.t('onboarding.ageAndTermsRequired'))).toBeInTheDocument()

    fireEvent.click(screen.getByRole('checkbox'))
    expect(confirm).toBeEnabled()
    fireEvent.click(confirm)
    await waitFor(() => expect(selectCountry).toHaveBeenCalledWith('es', expect.any(Object)))
    expect(acceptTerms).toHaveBeenCalledWith({ version: 1, ageConfirmed: true })
  })

  it('shows no checkbox to a user who already accepted', async () => {
    renderPage()
    await pickCountry('Spain')
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('onboarding.confirm') }))
    expect(selectCountry).toHaveBeenCalledWith('es', expect.any(Object))
    expect(acceptTerms).not.toHaveBeenCalled()
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
