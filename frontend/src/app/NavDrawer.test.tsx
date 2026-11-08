import { fireEvent, render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import type { LoggedUser } from '../api/types'
import i18n from '../i18n'
import { lightTheme } from '../theme'
import { NavDrawer } from './NavDrawer'

const me = vi.hoisted(() => ({ data: undefined as Partial<LoggedUser> | undefined }))

vi.mock('../api/hooks', () => ({
  useMe: () => ({ data: me.data }),
  useUserTeam: () => ({ data: { team: { id: 2 } } }),
}))

const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`

function renderDrawer(path: string, user: Partial<LoggedUser>, variant: 'modal' | 'standard' = 'modal', onClose = vi.fn()) {
  me.data = user
  render(
    <ThemeProvider theme={lightTheme}>
      <MemoryRouter initialEntries={[path]}>
        <NavDrawer variant={variant} open onClose={onClose} />
      </MemoryRouter>
    </ThemeProvider>,
  )
}

const player = { id: '1', pendingCountrySelection: false, isAdmin: false }
const item = (key: string) => screen.getByRole('link', { name: i18n.t(`header.nav.${key}`) })

// app-navigation "Modal navigation drawer" and "Standard navigation drawer".
describe('NavDrawer', () => {
  it('highlights the section of a sub-route as a secondary-container pill (Current item)', () => {
    renderDrawer('/cup/ties/7', player)
    const cup = item('cup')
    expect(cup).toHaveClass('Mui-selected')
    expect(cup.querySelector('svg')).toHaveAttribute('data-testid', 'EmojiEventsIcon')
    const style = getComputedStyle(cup)
    expect(style.backgroundColor).toBe(rgb('#d5e8cf'))
    expect(style.minHeight).toBe('56px')
    expect(item('leagues').querySelector('svg')).toHaveAttribute('data-testid', 'PublicOutlinedIcon')
  })

  it('opens on the low surface container with rounded open-side corners', () => {
    renderDrawer('/dashboard', player)
    const paper = getComputedStyle(document.querySelector('.MuiDrawer-paper')!)
    expect(paper.backgroundColor).toBe(rgb('#f3f4ee'))
    // Square on the screen edge, 16px on the open side.
    expect(paper.borderRadius).toBe('0 16px 16px 0')
  })

  it('ends the Other group with Admin for an admin only (Admin entry)', () => {
    renderDrawer('/dashboard', { ...player, isAdmin: true })
    expect(item('admin')).toBeInTheDocument()
  })

  it('has no Admin entry for a player who is not an admin', () => {
    renderDrawer('/dashboard', player)
    expect(screen.queryByRole('link', { name: i18n.t('header.nav.admin') })).not.toBeInTheDocument()
  })

  it('closes once an item is chosen', () => {
    const onClose = vi.fn()
    renderDrawer('/dashboard', player, 'modal', onClose)
    fireEvent.click(item('finances'))
    expect(onClose).toHaveBeenCalled()
  })

  it('renders the standard drawer as a column that stays under the header', () => {
    renderDrawer('/dashboard', player, 'standard')
    expect(document.querySelector('.MuiDrawer-root')).toBeNull()
    const column = getComputedStyle(screen.getByRole('navigation'))
    expect(column.position).toBe('sticky')
    expect(column.top).toBe('64px')
  })

  it('shows nothing to a user who is still choosing a country', () => {
    renderDrawer('/onboarding', { ...player, pendingCountrySelection: true })
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
