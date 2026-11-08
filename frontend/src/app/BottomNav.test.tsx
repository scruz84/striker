import { fireEvent, render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import i18n from '../i18n'
import { lightTheme } from '../theme'
import { BottomNav } from './BottomNav'

vi.mock('../api/hooks', () => ({
  useMe: () => ({ data: { id: '1' } }),
  useUserTeam: () => ({ data: { team: { id: 2 } } }),
}))

const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`

function renderAt(path: string) {
  render(
    <ThemeProvider theme={lightTheme}>
      <MemoryRouter initialEntries={[path]}>
        <BottomNav onMore={vi.fn()} />
      </MemoryRouter>
    </ThemeProvider>,
  )
}

const entry = (name: string) => screen.getByRole('link', { name: i18n.t(`nav.bottom.${name}`) })

// app-navigation "Navigation bar" and "Navigation icons".
describe('BottomNav', () => {
  it('marks the current entry with a 64x32 secondary-container pill and a filled icon (Current entry)', () => {
    renderAt('/transfers')
    const icon = entry('market').querySelector('svg')!
    expect(icon).toHaveAttribute('data-testid', 'SwapHorizIcon')
    const pill = getComputedStyle(icon)
    expect(pill.width).toBe('64px')
    expect(pill.height).toBe('32px')
    expect(pill.backgroundColor).toBe(rgb('#d5e8cf'))
    for (const other of ['dashboard', 'squad', 'lineup']) {
      expect(getComputedStyle(entry(other).querySelector('svg')!).backgroundColor).not.toBe(rgb('#d5e8cf'))
    }
  })

  it('is 80px tall with 12px labels on the surface container', () => {
    renderAt('/dashboard')
    const bar = document.querySelector('.MuiBottomNavigation-root')!
    expect(getComputedStyle(bar).height).toBe('80px')
    expect(getComputedStyle(bar).backgroundColor).toBe(rgb('#eeeee8'))
    const label = entry('dashboard').querySelector('.MuiBottomNavigationAction-label')!
    expect(getComputedStyle(label).fontSize).toBe('12px')
  })

  it('swaps the icons when the player moves to another entry (Icon swaps on selection)', () => {
    renderAt('/dashboard')
    expect(entry('dashboard').querySelector('svg')).toHaveAttribute('data-testid', 'DashboardIcon')
    expect(entry('market').querySelector('svg')).toHaveAttribute('data-testid', 'SwapHorizOutlinedIcon')

    fireEvent.click(entry('market'))
    expect(entry('dashboard').querySelector('svg')).toHaveAttribute('data-testid', 'DashboardOutlinedIcon')
    expect(entry('market').querySelector('svg')).toHaveAttribute('data-testid', 'SwapHorizIcon')
  })
})
