import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { LegalConfig } from '../../config'
import i18n from '../../i18n'
import { LegalLinks } from './LegalLinks'
import { LegalNoticePage } from './LegalNoticePage'
import { PrivacyPage } from './PrivacyPage'
import { TermsPage } from './TermsPage'

const legal = vi.hoisted(() => ({
  config: {} as LegalConfig,
}))
vi.mock('../../config', () => ({ getLegalConfig: () => legal.config }))

const complete: LegalConfig = {
  controllerName: 'Example Games SL',
  contactEmail: 'privacy@example.org',
  taxId: 'B12345678',
  postalAddress: 'Calle Mayor 1, 28001 Madrid',
  hostingProvider: 'Acme Cloud',
  hostingRegion: 'Frankfurt',
  noticeEnabled: true,
}
const empty: LegalConfig = { ...complete, controllerName: '', contactEmail: '', taxId: '', postalAddress: '', hostingProvider: '', hostingRegion: '', noticeEnabled: false }

function renderPage(page: React.ReactElement) {
  render(<MemoryRouter>{page}</MemoryRouter>)
}

describe('legal pages', () => {
  afterEach(async () => {
    cleanup()
    await i18n.changeLanguage('en')
  })

  it('names the configured controller and contact in the privacy policy', () => {
    legal.config = complete
    renderPage(<PrivacyPage />)
    expect(screen.getAllByText(/Example Games SL/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/privacy@example\.org/).length).toBeGreaterThan(0)
    expect(screen.getByText(/Calle Mayor 1, 28001 Madrid/)).toBeInTheDocument()
    expect(screen.getByText(/Acme Cloud, with servers in Frankfurt/)).toBeInTheDocument()
    expect(screen.queryByText(i18n.t('legal.notConfigured'))).not.toBeInTheDocument()
  })

  it('states the retention periods and the rights in the privacy policy', () => {
    legal.config = complete
    renderPage(<PrivacyPage />)
    expect(screen.getByText(/60 days/)).toBeInTheDocument()
    expect(screen.getByText(/365 days/)).toBeInTheDocument()
    expect(screen.getByText(/at most 30 days/)).toBeInTheDocument()
    expect(screen.getByText(/AEPD/)).toBeInTheDocument()
  })

  it('shows a visible warning instead of the missing required values', () => {
    legal.config = empty
    renderPage(<TermsPage />)
    expect(screen.getByText(i18n.t('legal.notConfigured'))).toBeInTheDocument()
    expect(screen.getByText(/LEGAL_CONTROLLER_NAME, LEGAL_CONTACT_EMAIL/)).toBeInTheDocument()
    expect(screen.getAllByText(new RegExp(i18n.t('legal.notConfiguredValue').replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))).length).toBeGreaterThan(0)
  })

  it('omits the address and hosting sentences when those properties are not set', () => {
    legal.config = { ...complete, postalAddress: '', hostingProvider: '' }
    renderPage(<PrivacyPage />)
    expect(screen.queryByText(/Postal address/)).not.toBeInTheDocument()
    expect(screen.queryByText(/hosting provider is/)).not.toBeInTheDocument()
  })

  it('renders in Spanish', async () => {
    legal.config = complete
    await i18n.changeLanguage('es')
    renderPage(<PrivacyPage />)
    expect(screen.getByRole('heading', { name: 'Política de privacidad' })).toBeInTheDocument()
    expect(screen.getByText(/Agencia Española de Protección de Datos/)).toBeInTheDocument()
  })

  it('shows the legal notice with the configured identity and drops unset optional lines', () => {
    legal.config = { ...complete, postalAddress: '' }
    renderPage(<LegalNoticePage />)
    expect(screen.getByText('Example Games SL')).toBeInTheDocument()
    expect(screen.getByText('B12345678')).toBeInTheDocument()
    expect(screen.queryByText(i18n.t('legal.notice.address'))).not.toBeInTheDocument()
  })

  it('links the legal notice only when the deployment enabled it', () => {
    legal.config = { ...complete, noticeEnabled: false }
    renderPage(<LegalLinks />)
    expect(screen.getByRole('link', { name: i18n.t('legal.privacyLink') })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: i18n.t('legal.termsLink') })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: i18n.t('legal.noticeLink') })).not.toBeInTheDocument()

    cleanup()
    legal.config = complete
    renderPage(<LegalLinks />)
    expect(screen.getByRole('link', { name: i18n.t('legal.noticeLink') })).toHaveAttribute('href', '/legal-notice')
  })
})
