import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { DevUser } from '../api/types'
import i18n from '../i18n'
import { DevLoginWarning } from './DevLoginWarning'

const roster = vi.hoisted(() => ({ data: undefined as DevUser[] | undefined }))

vi.mock('../api/hooks', () => ({
  useDevLoginUsers: () => roster,
}))

describe('DevLoginWarning', () => {
  it('shows the warning strip when the backend lists dev-login users', () => {
    roster.data = [{ email: 'admin@dev.local', name: 'Dev Admin', isAdmin: true }]
    render(<DevLoginWarning />)
    expect(screen.getByRole('alert')).toHaveTextContent(i18n.t('devLogin.warningStrip'))
  })

  it.each([[undefined], [[]]])('shows nothing without a roster (%j)', (data) => {
    roster.data = data as DevUser[] | undefined
    render(<DevLoginWarning />)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
