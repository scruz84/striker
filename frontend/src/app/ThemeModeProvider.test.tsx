import { render, screen } from '@testing-library/react'
import { useTheme } from '@mui/material/styles'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ThemeModeProvider } from './ThemeModeProvider'

function stubMedia(matching: string[]) {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: matching.some((m) => query.includes(m)),
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  }))
}

function Probe() {
  return <span data-testid="primary">{useTheme().palette.primary.main}</span>
}

function primaryColour() {
  render(
    <ThemeModeProvider>
      <Probe />
    </ThemeModeProvider>,
  )
  return screen.getByTestId('primary').textContent
}

// ui-theme "High contrast".
describe('ThemeModeProvider contrast', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('uses the high-contrast scheme when the system asks for more contrast (Increase contrast in light mode)', () => {
    stubMedia(['prefers-contrast: more'])
    expect(primaryColour()).toBe('#003509')
  })

  it('uses the high-contrast dark scheme in dark mode', () => {
    stubMedia(['prefers-contrast: more', 'prefers-color-scheme: dark'])
    expect(primaryColour()).toBe('#c5ffbc')
  })

  it('keeps the standard scheme without a preference (No preference)', () => {
    stubMedia([])
    expect(primaryColour()).toBe('#006e1c')
  })
})
