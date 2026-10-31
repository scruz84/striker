import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { COUNTRY_NAME_KEYS } from '../api/types'
import i18n from '../i18n'
import { CountryFlag } from './CountryFlag'
import { TeamNameWithFlag } from './TeamNameWithFlag'
// Every country the game can generate (mirrors backend common.KnownCountries; the backend's
// TestKnownCountriesHaveFlagFiles keeps the two lists and public/flags/ in step).
const KNOWN_COUNTRY_CODES = Object.keys(COUNTRY_NAME_KEYS)

// public/flags/<code>.svg contents by code.
const FLAG_SVGS: Record<string, string> = Object.fromEntries(
  Object.entries(import.meta.glob('../../public/flags/*.svg', { query: '?raw', import: 'default', eager: true })).map(
    ([path, svg]) => [path.replace(/^.*\/(\w+)\.svg$/, '$1'), svg as string],
  ),
)

describe('country flag assets', () => {
  it('has a flag file for every country code', () => {
    expect(KNOWN_COUNTRY_CODES.length).toBeGreaterThanOrEqual(211)
    for (const code of KNOWN_COUNTRY_CODES) {
      expect(FLAG_SVGS[code], `public/flags/${code}.svg`).toBeTruthy()
    }
  })

  it('never shares one flag image between two countries', () => {
    const codeBySvg = new Map<string, string>()
    for (const code of KNOWN_COUNTRY_CODES) {
      expect(codeBySvg.get(FLAG_SVGS[code]), `${code} has the same image as ${codeBySvg.get(FLAG_SVGS[code])}`).toBeUndefined()
      codeBySvg.set(FLAG_SVGS[code], code)
    }
  })
})

describe('CountryFlag', () => {
  it('shows the Scottish flag file with the localized country name', () => {
    render(<CountryFlag code="sco" />)
    const img = screen.getByRole('img', { name: i18n.t('countries.sco') })
    expect(img).toHaveAttribute('src', '/flags/sco.svg')
  })

  it('gives each home nation its own flag, not the UK one', () => {
    const srcs = ['en', 'sco', 'wal', 'nir'].map((code) => {
      const { container, unmount } = render(<CountryFlag code={code} />)
      const src = container.querySelector('img')?.getAttribute('src')
      unmount()
      return src
    })
    expect(new Set(srcs).size).toBe(4)
  })

  it('shows the Northern Ireland and Tahiti flags from their own files', () => {
    for (const code of ['nir', 'pf', 'xk']) {
      const { container, unmount } = render(<CountryFlag code={code} />)
      expect(container.querySelector('img')).toHaveAttribute('src', `/flags/${code}.svg`)
      unmount()
    }
    // Tahiti's FIFA association plays under the French Polynesia flag, Northern Ireland under the
    // Ulster Banner (flag-icons gb-nir) - not the UK flag.
    expect(FLAG_SVGS.nir).not.toBe(FLAG_SVGS.en)
  })

  it('renders nothing for a missing or unknown code', () => {
    const { container, rerender } = render(<CountryFlag />)
    expect(container).toBeEmptyDOMElement()
    rerender(<CountryFlag code="zz" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('is hidden from assistive tech when decorative', () => {
    const { container } = render(<CountryFlag code="es" decorative />)
    expect(container.querySelector('img')).toHaveAttribute('alt', '')
  })
})

describe('TeamNameWithFlag', () => {
  it('shows the flag and localized name of a national team', () => {
    const { container } = render(<TeamNameWithFlag rawName="Scotland" />)
    expect(screen.getByText(i18n.t('countries.sco'))).toBeInTheDocument()
    expect(container.querySelector('img')).toHaveAttribute('src', '/flags/sco.svg')
  })

  it('shows an ordinary club name as plain text with no flag', () => {
    const { container } = render(<TeamNameWithFlag rawName="Real Madrid" />)
    expect(screen.getByText('Real Madrid')).toBeInTheDocument()
    expect(container.querySelector('img')).toBeNull()
  })
})
