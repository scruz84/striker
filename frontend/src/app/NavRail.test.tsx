import { fireEvent, render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import i18n from '../i18n'
import { lightTheme } from '../theme'
import { NavRail } from './NavRail'

vi.mock('../api/hooks', () => ({
  useMe: () => ({ data: { id: '1' } }),
  useUserTeam: () => ({ data: { team: { id: 2 } } }),
}))

const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`

function renderAt(path: string, onMenu = vi.fn(), menuOpen = false) {
  render(
    <ThemeProvider theme={lightTheme}>
      <MemoryRouter initialEntries={[path]}>
        <NavRail menuOpen={menuOpen} onMenu={onMenu} />
      </MemoryRouter>
    </ThemeProvider>,
  )
}

const entry = (name: string) => screen.getByRole('link', { name: i18n.t(`nav.bottom.${name}`) })

// app-navigation "Navigation rail".
describe('NavRail', () => {
  it('marks the current destination with the pill and a filled icon (Rail entry selected)', () => {
    renderAt('/dashboard/lineup')
    const lineup = entry('lineup')
    expect(lineup).toHaveAttribute('aria-current', 'page')
    expect(lineup.querySelector('svg')).toHaveAttribute('data-testid', 'SportsSoccerIcon')
    const pill = getComputedStyle(lineup.querySelector('.nav-rail-pill')!)
    expect(pill.backgroundColor).toBe(rgb('#d5e8cf'))
    expect(pill.width).toBe('56px')
    expect(pill.height).toBe('32px')
    expect(entry('dashboard')).not.toHaveAttribute('aria-current')
    expect(entry('dashboard').querySelector('svg')).toHaveAttribute('data-testid', 'DashboardOutlinedIcon')
  })

  it('lists the four daily destinations, each at least 48px tall, in an 80px sticky column', () => {
    renderAt('/dashboard')
    for (const name of ['dashboard', 'squad', 'lineup', 'market']) {
      expect(parseInt(getComputedStyle(entry(name)).minHeight)).toBeGreaterThanOrEqual(48)
    }
    const rail = screen.getByRole('navigation')
    expect(getComputedStyle(rail).width).toBe('80px')
    expect(getComputedStyle(rail).position).toBe('sticky')
    expect(getComputedStyle(rail).top).toBe('64px')
  })

  it('opens the full navigation from its menu button', () => {
    const onMenu = vi.fn()
    renderAt('/dashboard', onMenu)
    const menu = screen.getByRole('button', { name: i18n.t('nav.bottom.more') })
    expect(menu).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(menu)
    expect(onMenu).toHaveBeenCalledOnce()
  })
})
