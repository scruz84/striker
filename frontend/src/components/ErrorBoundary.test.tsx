import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { ErrorBoundary } from './ErrorBoundary'

function Bomb(): never {
  throw new Error('boom')
}

describe('ErrorBoundary', () => {
  it('renders children normally when nothing throws', () => {
    render(
      <MemoryRouter>
        <ErrorBoundary>
          <div>All good</div>
        </ErrorBoundary>
      </MemoryRouter>,
    )
    expect(screen.getByText('All good')).toBeInTheDocument()
  })

  it('catches a render error and shows a recoverable fallback instead of unmounting the app', () => {
    // React logs the error to the console by default even though it's caught - silence that
    // expected noise so the test output doesn't look like a real failure.
    vi.spyOn(console, 'error').mockImplementation(() => {})

    render(
      <MemoryRouter>
        <ErrorBoundary>
          <Bomb />
        </ErrorBoundary>
      </MemoryRouter>,
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Back to home' })).toBeInTheDocument()
  })

  it('recovers when "Try again" is clicked and the child no longer throws', () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})

    let shouldThrow = true
    function MaybeBomb() {
      if (shouldThrow) {
        throw new Error('boom')
      }
      return <div>Recovered</div>
    }

    const { rerender } = render(
      <MemoryRouter>
        <ErrorBoundary>
          <MaybeBomb />
        </ErrorBoundary>
      </MemoryRouter>,
    )
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()

    shouldThrow = false
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))
    rerender(
      <MemoryRouter>
        <ErrorBoundary>
          <MaybeBomb />
        </ErrorBoundary>
      </MemoryRouter>,
    )

    expect(screen.getByText('Recovered')).toBeInTheDocument()
  })
})
