import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { NotifyProvider } from './NotifyProvider'
import { useNotify } from './notifyContext'

function Buttons() {
  const notify = useNotify()
  return (
    <>
      <button onClick={() => notify.success('Saved')}>ok</button>
      <button onClick={() => notify.error('Failed')}>bad</button>
    </>
  )
}

function renderProvider() {
  render(
    <NotifyProvider>
      <Buttons />
    </NotifyProvider>,
  )
}

describe('NotifyProvider', () => {
  it('shows a success message', () => {
    renderProvider()
    fireEvent.click(screen.getByText('ok'))
    expect(screen.getByText('Saved')).toBeInTheDocument()
  })

  it('shows an error message as an alert', () => {
    renderProvider()
    fireEvent.click(screen.getByText('bad'))
    expect(screen.getByRole('alert')).toHaveTextContent('Failed')
  })

  it('replaces the message on screen with the newest one', () => {
    renderProvider()
    fireEvent.click(screen.getByText('ok'))
    fireEvent.click(screen.getByText('bad'))
    expect(screen.getByText('Failed')).toBeInTheDocument()
    expect(screen.queryByText('Saved')).not.toBeInTheDocument()
  })

  it('throws a clear error when used outside a provider', () => {
    expect(() => render(<Buttons />)).toThrow(/NotifyProvider/)
  })
})
