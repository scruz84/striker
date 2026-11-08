import { act, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { LoggedUser } from '../api/types'
import i18n from '../i18n'
import { ThemeModeProvider } from './ThemeModeProvider'
import { Header } from './Header'

const me = vi.hoisted(() => ({ data: undefined as LoggedUser | null | undefined }))

vi.mock('../api/hooks', () => ({
  useMe: () => me,
  useLogout: () => ({ mutate: vi.fn() }),
  useSetUserLanguage: () => ({ mutate: vi.fn() }),
}))
vi.mock('./NotificationBell', () => ({ NotificationBell: () => null }))

function renderHeader(data: LoggedUser | null | undefined) {
  me.data = data
  render(
    <MemoryRouter>
      <Header />
    </MemoryRouter>,
  )
}

const signIn = () => screen.queryByRole('link', { name: i18n.t('header.login') })

describe('Header (phone)', () => {
  it('offers sign-in once the server says there is no session', () => {
    renderHeader(null)
    expect(signIn()).toBeInTheDocument()
    expect(screen.getByRole('button', { name: i18n.t('header.more') })).toBeInTheDocument()
  })

  it('shows neither sign-in nor the guest menu while the session is unknown (loading or server unreachable)', () => {
    renderHeader(undefined)
    expect(signIn()).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: i18n.t('header.more') })).not.toBeInTheDocument()
  })

  it('shows the account menu for a signed-in user', () => {
    renderHeader({ id: '1', pictureUrl: '' } as LoggedUser)
    expect(signIn()).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: i18n.t('header.accountMenu') })).toBeInTheDocument()
  })
})

// ui-theme "Top app bar": rendered with the app's light theme; phone unless `desktop` stubs
// matchMedia.
describe('Header top app bar', () => {
  const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`

  function renderThemed(desktop = false) {
    if (desktop) {
      vi.stubGlobal('matchMedia', (query: string) => ({
        matches: query.includes('min-width'),
        media: query,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      }))
    }
    me.data = { id: '1', pictureUrl: '' } as LoggedUser
    const { container } = render(
      <ThemeModeProvider>
        <MemoryRouter>
          <Header />
        </MemoryRouter>
      </ThemeModeProvider>,
    )
    return container.querySelector<HTMLElement>('.MuiAppBar-root')!
  }

  function scrollTo(y: number) {
    act(() => {
      Object.defineProperty(window, 'pageYOffset', { value: y, configurable: true })
      window.dispatchEvent(new Event('scroll'))
    })
  }

  afterEach(() => {
    scrollTo(0)
    vi.unstubAllGlobals()
  })

  it('sits on the page surface with the logo in primary and onSurface icons (Page at the top)', () => {
    const bar = renderThemed()
    expect(getComputedStyle(bar).backgroundColor).toBe(rgb('#f9faf4'))
    expect(getComputedStyle(bar).boxShadow === '' || getComputedStyle(bar).boxShadow === 'none').toBe(true)
    expect(getComputedStyle(screen.getByRole('link', { name: /striker/i })).color).toBe(rgb('#006e1c'))
    expect(getComputedStyle(screen.getByRole('link', { name: i18n.t('header.searchIcon') })).color).toBe(rgb('#1a1c19'))
  })

  it('takes surfaceContainer and stays visible while content scrolls under it on a desktop (Content under the header)', () => {
    const bar = renderThemed(true)
    scrollTo(200)
    expect(getComputedStyle(bar).backgroundColor).toBe(rgb('#eeeee8'))
    expect(bar.style.transform === '' || bar.style.transform === 'none').toBe(true)
  })

  it('still slides away on a phone after scrolling 64px down', () => {
    const bar = renderThemed()
    scrollTo(200)
    expect(bar.style.transform).toContain('translateY')
  })
})
