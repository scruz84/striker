import { render, screen } from '@testing-library/react'
import Button from '@mui/material/Button'
import { ThemeProvider } from '@mui/material/styles'
import { describe, expect, it, vi } from 'vitest'
import i18n from '../../i18n'
import { lightTheme } from '../../theme'
import { FilterSheet } from '../../features/transfers/marketUi'
import { BottomSheet } from './BottomSheet'

const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`

// ui-theme "Bottom sheet".
describe('BottomSheet', () => {
  it('slides up on the low surface container with 28px corners and a drag handle', () => {
    render(
      <ThemeProvider theme={lightTheme}>
        <BottomSheet open onClose={vi.fn()} title="Filters" footer={<Button>Apply</Button>}>
          <span>body</span>
        </BottomSheet>
      </ThemeProvider>,
    )
    const paper = getComputedStyle(document.querySelector('.MuiDrawer-paper')!)
    expect(paper.backgroundColor).toBe(rgb('#f3f4ee'))
    expect(paper.borderTopLeftRadius).toBe('28px')
    expect(paper.borderTopRightRadius).toBe('28px')
    expect(paper.maxWidth).toBe('640px')
    const handle = getComputedStyle(document.querySelector('.bottom-sheet-handle')!)
    expect(handle.width).toBe('32px')
    expect(handle.height).toBe('4px')
    expect(screen.getByRole('heading', { name: 'Filters' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: i18n.t('common.close') })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Apply' })).toBeInTheDocument()
  })

  it('holds the market filters with Clear and Show N results (Market filters)', () => {
    render(
      <ThemeProvider theme={lightTheme}>
        <FilterSheet open onClose={vi.fn()} resultCount={12} onClear={vi.fn()} hasActiveFilters>
          <span>position filter</span>
        </FilterSheet>
      </ThemeProvider>,
    )
    expect(document.querySelector('.bottom-sheet-handle')).not.toBeNull()
    expect(screen.getByRole('heading', { name: i18n.t('transfers.filters.title') })).toBeInTheDocument()
    expect(screen.getByText('position filter')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: i18n.t('transfers.filters.clear') })).toBeEnabled()
    expect(screen.getByRole('button', { name: i18n.t('transfers.filters.showResults', { count: 12 }) })).toBeInTheDocument()
  })
})
