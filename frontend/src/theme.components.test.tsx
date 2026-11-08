import { render as baseRender, screen } from '@testing-library/react'
import CheckIcon from '@mui/icons-material/Check'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import Chip from '@mui/material/Chip'
import Dialog from '@mui/material/Dialog'
import Switch from '@mui/material/Switch'
import { ThemeProvider, type Theme } from '@mui/material/styles'
import type { ReactElement } from 'react'
import { describe, expect, it } from 'vitest'
import { CORE_POSITIONS, PositionFilter } from './features/transfers/marketUi'
import { darkTheme, lightTheme } from './theme'
import { TOUCH } from './touchMedia'

function render(ui: ReactElement, theme: Theme = lightTheme) {
  return baseRender(<ThemeProvider theme={theme}>{ui}</ThemeProvider>)
}

// jsdom does not evaluate media queries, so touch-screen sizes are read from the CSS that MUI
// emitted for the element: the value of `property` in rules inside `@media <TOUCH>` that match it.
function touchValue(element: Element, property: string): string | undefined {
  const media = TOUCH.replace(/^@media /, '')
  let value: string | undefined
  for (const sheet of Array.from(document.styleSheets)) {
    for (const rule of Array.from(sheet.cssRules)) {
      if (!(rule instanceof CSSMediaRule) || rule.conditionText !== media) continue
      for (const inner of Array.from(rule.cssRules)) {
        if (inner instanceof CSSStyleRule && element.matches(inner.selectorText) && inner.style.getPropertyValue(property)) {
          value = inner.style.getPropertyValue(property)
        }
      }
    }
  }
  return value
}

const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`

describe('ui-theme buttons', () => {
  it('renders the tonal button in secondary-container colours without border or shadow (Tonal button)', () => {
    render(<Button variant="tonal">Ver previa</Button>)
    const style = getComputedStyle(screen.getByRole('button', { name: 'Ver previa' }))
    expect(style.backgroundColor).toBe(rgb('#d5e8cf'))
    expect(style.color).toBe(rgb('#3b4b38'))
    expect(style.borderStyle === '' || style.borderStyle === 'none' || style.borderWidth === '0px').toBe(true)
    expect(style.boxShadow === '' || style.boxShadow === 'none').toBe(true)
  })

  it('is 40px tall with a pointer and 44px on touch screens (Touch height)', () => {
    render(<Button variant="contained">Edit lineup</Button>)
    const button = screen.getByRole('button', { name: 'Edit lineup' })
    expect(getComputedStyle(button).minHeight).toBe('40px')
    expect(getComputedStyle(button).borderRadius).toBe('9999px')
    expect(touchValue(button, 'min-height')).toBe('44px')
  })
})

describe('ui-theme chips', () => {
  it('draws a compact status chip 24px tall with 8px corners in its own colour (Compact status chip)', () => {
    const { container } = render(<Chip size="small" color="success" label="W" />)
    const style = getComputedStyle(container.querySelector('.MuiChip-root')!)
    expect(style.height).toBe('24px')
    expect(style.borderRadius).toBe('8px')
    expect(style.backgroundColor).toBe(rgb(lightTheme.palette.success.main))
  })

  it('shows a selected filter chip in secondary-container colours with a leading check (Selected filter chip)', () => {
    const { container } = render(<Chip color="primary" icon={<CheckIcon data-testid="check" />} label="Available" />)
    const chip = container.querySelector('.MuiChip-root')!
    const style = getComputedStyle(chip)
    expect(style.height).toBe('32px')
    expect(style.backgroundColor).toBe(rgb('#d5e8cf'))
    expect(style.color).toBe(rgb('#3b4b38'))
    expect(chip.firstElementChild).toBe(screen.getByTestId('check'))
  })
})

describe('ui-theme segmented buttons', () => {
  it('renders the market position filter as segmented buttons (Position filter in the transfer market)', () => {
    render(<PositionFilter value={CORE_POSITIONS[2]} onChange={() => {}} />)
    const buttons = screen.getAllByRole('button')
    const selected = buttons.filter((b) => b.getAttribute('aria-pressed') === 'true')
    expect(selected).toHaveLength(1)
    expect(getComputedStyle(selected[0]).backgroundColor).toBe(rgb('#d5e8cf'))
    for (const unselected of buttons.filter((b) => b !== selected[0])) {
      expect(getComputedStyle(unselected).backgroundColor).not.toBe(rgb('#d5e8cf'))
    }
    const first = getComputedStyle(buttons[0])
    expect(first.borderTopLeftRadius).toBe('9999px')
    expect(first.borderColor).toBe(rgb('#72796f'))
    expect(getComputedStyle(buttons[buttons.length - 1]).borderBottomRightRadius).toBe('9999px')
    expect(touchValue(buttons[0], 'min-height')).toBe('44px')
  })
})

describe('ui-theme cards', () => {
  it('fills a card with the highest surface container and no shadow in dark mode (Filled card in dark mode)', () => {
    const { container } = render(<Card variant="filled">x</Card>, darkTheme)
    const style = getComputedStyle(container.querySelector('.MuiCard-root')!)
    expect(style.backgroundColor).toBe(rgb('#333531'))
    expect(style.boxShadow === '' || style.boxShadow === 'none').toBe(true)
  })

  it('raises an elevated card on the low surface container with a shadow', () => {
    const { container } = render(<Card variant="elevated">x</Card>)
    const style = getComputedStyle(container.querySelector('.MuiCard-root')!)
    expect(style.backgroundColor).toBe(rgb('#f3f4ee'))
    expect(style.boxShadow).toContain('rgba(0, 0, 0, 0.3)')
  })
})

describe('ui-theme switches', () => {
  it('fills the track with primary and grows the thumb when on (Switch on)', () => {
    const { container } = render(<Switch checked readOnly />)
    expect(getComputedStyle(container.querySelector('.MuiSwitch-track')!).backgroundColor).toBe(rgb('#006e1c'))
    const thumb = getComputedStyle(container.querySelector('.MuiSwitch-thumb')!)
    expect(thumb.width).toBe('24px')
    expect(thumb.height).toBe('24px')
    expect(getComputedStyle(container.querySelector('.MuiSwitch-switchBase')!).color).toBe(rgb('#ffffff'))
  })

  it('draws the off state as an outlined track with a 16px outline thumb', () => {
    const { container } = render(<Switch checked={false} readOnly />)
    const track = getComputedStyle(container.querySelector('.MuiSwitch-track')!)
    expect(track.height).toBe('32px')
    expect(track.backgroundColor).toBe(rgb('#e2e3dd'))
    expect(track.border).toContain('2px')
    expect(getComputedStyle(container.querySelector('.MuiSwitch-thumb')!).width).toBe('16px')
    expect(getComputedStyle(container.querySelector('.MuiSwitch-switchBase')!).color).toBe(rgb('#72796f'))
  })

  it('keeps a tap area at least 44px tall (Switch tap area)', () => {
    const { container } = render(<Switch />)
    expect(parseInt(getComputedStyle(container.querySelector('.MuiSwitch-root')!).height)).toBeGreaterThanOrEqual(44)
  })
})

describe('ui-theme dialogs', () => {
  it('uses the high surface container with 28px corners (Dialog in light mode)', () => {
    render(
      <Dialog open>
        <span>Confirm?</span>
      </Dialog>,
    )
    const paper = getComputedStyle(document.querySelector('.MuiDialog-paper')!)
    expect(paper.backgroundColor).toBe(rgb('#e8e9e2'))
    expect(paper.borderRadius).toBe('28px')
  })

  it('squares full-screen dialogs on the page surface (Full-screen dialog on a phone)', () => {
    render(
      <Dialog open fullScreen>
        <span>Sheet</span>
      </Dialog>,
    )
    const paper = getComputedStyle(document.querySelector('.MuiDialog-paperFullScreen')!)
    expect(paper.borderRadius).toBe('0px')
    expect(paper.backgroundColor).toBe(rgb('#f9faf4'))
  })
})
