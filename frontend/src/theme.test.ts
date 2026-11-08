import { describe, expect, it } from 'vitest'
import type { Palette } from '@mui/material/styles'
import { darkTheme, lightTheme } from './theme'

// WCAG 2.x relative luminance and contrast ratio of two #rrggbb colours.
function luminance(hex: string): number {
  const [r, g, b] = [1, 3, 5].map((i) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

function contrastRatio(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

const modes = [
  ['light', lightTheme.palette],
  ['dark', darkTheme.palette],
] as const

function surfaceLevels(p: Palette): string[] {
  return [p.surfaceContainerLowest, p.surfaceContainerLow, p.surfaceContainer, p.surfaceContainerHigh, p.surfaceContainerHighest]
}

describe('ui-theme colour roles', () => {
  it('keeps the brand colours (Brand colour unchanged)', () => {
    expect(lightTheme.palette.primary.main).toBe('#006e1c')
    expect(lightTheme.palette.primaryContainer).toBe('#94f990')
    expect(darkTheme.palette.primary.main).toBe('#78dc77')
    expect(darkTheme.palette.primaryContainer).toBe('#005313')
  })

  it('exposes the five surface-container levels in light mode', () => {
    expect(surfaceLevels(lightTheme.palette)).toEqual(['#ffffff', '#f3f4ee', '#eeeee8', '#e8e9e2', '#e2e3dd'])
    expect(lightTheme.palette.background.default).toBe('#f9faf4')
  })

  it('exposes the five surface-container levels in dark mode', () => {
    expect(surfaceLevels(darkTheme.palette)).toEqual(['#0c0f0c', '#1a1c19', '#1e201d', '#282b27', '#333531'])
    expect(darkTheme.palette.background.default).toBe('#121411')
  })

  it('puts outlined panels on the page colour with an outline-variant border (Outlined panel)', () => {
    expect(lightTheme.palette.background.paper).toBe(lightTheme.palette.background.default)
    expect(lightTheme.palette.outlineVariant).toBe('#c2c9bd')
  })

  it('leaves the rating scale unchanged', () => {
    expect(lightTheme.palette.rating).toEqual(['#61675e', '#49624c', '#315d3a', '#185827', '#005315'])
    expect(darkTheme.palette.rating).toEqual(['#9da39a', '#97b295', '#92c290', '#8cd18a', '#86e085'])
  })

  it('keeps text on a container readable (Text on a container)', () => {
    expect(lightTheme.palette.onErrorContainer).toBe('#93000a')
    expect(contrastRatio(lightTheme.palette.onErrorContainer, lightTheme.palette.errorContainer)).toBeGreaterThanOrEqual(4.5)
  })

  it.each(modes)('reaches 4.5:1 for every on-role pair in %s mode', (_mode, p) => {
    const pairs: [string, string, string][] = [
      ['primary', p.primary.contrastText, p.primary.main],
      ['secondary', p.secondary.contrastText, p.secondary.main],
      ['tertiary', p.tertiary.contrastText, p.tertiary.main],
      ['error', p.error.contrastText, p.error.main],
      ['primaryContainer', p.onPrimaryContainer, p.primaryContainer],
      ['secondaryContainer', p.onSecondaryContainer, p.secondaryContainer],
      ['tertiaryContainer', p.onTertiaryContainer, p.tertiaryContainer],
      ['errorContainer', p.onErrorContainer, p.errorContainer],
      ['inverseSurface', p.inverseOnSurface, p.inverseSurface],
    ]
    for (const surface of [p.background.default, ...surfaceLevels(p)]) {
      pairs.push([`onSurface on ${surface}`, p.onSurface, surface])
      pairs.push([`onSurfaceVariant on ${surface}`, p.onSurfaceVariant, surface])
    }
    for (const [name, fg, bg] of pairs) {
      expect(contrastRatio(fg, bg), name).toBeGreaterThanOrEqual(4.5)
    }
  })

  it.each(modes)('reaches 3:1 for the outline on the page in %s mode (Outline on the page)', (_mode, p) => {
    expect(contrastRatio(p.outline, p.background.default)).toBeGreaterThanOrEqual(3)
  })
})

describe('ui-theme surface hierarchy', () => {
  // The MuiPaper elevation1 override is a function of the theme; resolve it the way MUI does.
  function raisedPanelBackground(theme: typeof lightTheme): unknown {
    const override = theme.components?.MuiPaper?.styleOverrides?.elevation1 as (props: { theme: typeof lightTheme }) => { backgroundColor: string }
    return override({ theme }).backgroundColor
  }

  it('raises panels one surface level without tint (Raised panel in light mode)', () => {
    expect(raisedPanelBackground(lightTheme)).toBe('#f3f4ee')
  })

  it('raises panels one surface level in dark mode (Raised panel in dark mode)', () => {
    expect(raisedPanelBackground(darkTheme)).toBe('#1a1c19')
  })
})
