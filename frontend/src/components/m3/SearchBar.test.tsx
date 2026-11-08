import { fireEvent, render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'
import i18n from '../../i18n'
import { lightTheme } from '../../theme'
import { SearchBar } from './SearchBar'

const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`

function Harness({ compact = false, initial = '' }: { compact?: boolean; initial?: string }) {
  const [value, setValue] = useState(initial)
  return <SearchBar compact={compact} value={value} onChange={setValue} placeholder="Players, teams, coaches" />
}

function renderBar(props: { compact?: boolean; initial?: string } = {}) {
  render(
    <ThemeProvider theme={lightTheme}>
      <Harness {...props} />
    </ThemeProvider>,
  )
  const field = screen.getByRole('searchbox', { name: 'Players, teams, coaches' })
  return { field, pill: field.closest('.MuiInputBase-root')!.parentElement! }
}

const clearButton = () => screen.queryByRole('button', { name: i18n.t('common.clearSearch') })

// ui-theme "Search bar".
describe('SearchBar', () => {
  it('is a 56px pill on the high surface container with no clear button when empty (Search page)', () => {
    const { pill } = renderBar()
    const style = getComputedStyle(pill)
    expect(style.height).toBe('56px')
    expect(style.borderRadius).toBe('9999px')
    expect(style.backgroundColor).toBe(rgb('#e8e9e2'))
    expect(pill.querySelector('[data-testid="SearchIcon"]')).not.toBeNull()
    expect(clearButton()).not.toBeInTheDocument()
  })

  it('makes the whole pill height the input tap target', () => {
    const { field } = renderBar()
    expect(getComputedStyle(field).height).toBe('100%')
  })

  it('empties the field, keeps the focus and hides the button (Clearing a search)', () => {
    const { field } = renderBar()
    fireEvent.change(field, { target: { value: 'ribera' } })
    expect(field).toHaveValue('ribera')
    fireEvent.click(clearButton()!)
    expect(field).toHaveValue('')
    expect(field).toHaveFocus()
    expect(clearButton()).not.toBeInTheDocument()
  })

  it('is 48px tall when compact (Compact filter)', () => {
    const { pill } = renderBar({ compact: true })
    expect(getComputedStyle(pill).height).toBe('48px')
  })
})
