import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { AcceptTermsPage } from './AcceptTermsPage'

const mutate = vi.hoisted(() => vi.fn())
vi.mock('../../api/hooks', () => ({
  useMe: () => ({ data: { id: '1', termsAccepted: false, termsVersion: 3, teamName: 'CD Norte' } }),
  useAcceptTerms: () => ({ mutate, isPending: false, isError: false }),
  useDeleteAccount: () => ({ mutate: vi.fn(), isPending: false, isError: false }),
}))
vi.mock('../../config', () => ({ getLegalConfig: () => ({ noticeEnabled: false }) }))

describe('AcceptTermsPage', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('needs the age and terms box ticked, then accepts the version the server asked for', () => {
    render(
      <MemoryRouter>
        <AcceptTermsPage />
      </MemoryRouter>,
    )
    const accept = screen.getByRole('button', { name: i18n.t('acceptTerms.accept') })
    expect(accept).toBeDisabled()
    fireEvent.click(screen.getByRole('checkbox'))
    fireEvent.click(accept)
    expect(mutate).toHaveBeenCalledWith({ version: 3, ageConfirmed: true }, expect.any(Object))
  })

  it('offers deleting the account instead of accepting', () => {
    render(
      <MemoryRouter>
        <AcceptTermsPage />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByRole('button', { name: i18n.t('acceptTerms.deleteInstead') }))
    expect(screen.getByText(i18n.t('deleteAccount.title'))).toBeInTheDocument()
  })
})
