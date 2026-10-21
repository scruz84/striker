import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import type { LoggedUser } from '../api/types'
import i18n from '../i18n'
import { Header } from './Header'

const me = vi.hoisted(() => ({ data: undefined as LoggedUser | null | undefined }))

vi.mock('../api/hooks', () => ({
  useMe: () => me,
  useLogout: () => ({ mutate: vi.fn() }),
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
