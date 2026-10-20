import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { SettingCard } from './SettingCard'

function renderCard(props: Partial<React.ComponentProps<typeof SettingCard>> = {}) {
  const onSubmit = vi.fn()
  render(
    <SettingCard title="Max friendlies" description="How many per week" submitLabel="Save" onSubmit={onSubmit} {...props}>
      <input aria-label="value" />
    </SettingCard>,
  )
  return { onSubmit, button: screen.getByRole('button', { name: 'Save' }) }
}

describe('SettingCard', () => {
  it('shows its title, description and fields', () => {
    renderCard()
    expect(screen.getByText('Max friendlies')).toBeInTheDocument()
    expect(screen.getByText('How many per week')).toBeInTheDocument()
    expect(screen.getByLabelText('value')).toBeInTheDocument()
  })

  it('submits when the button is pressed', () => {
    const { onSubmit, button } = renderCard()
    fireEvent.click(button)
    expect(onSubmit).toHaveBeenCalledTimes(1)
  })

  it('does not submit while the input is invalid, and says why', () => {
    const { onSubmit, button } = renderCard({ invalid: true, errorText: 'Pick at least one day' })
    expect(button).toBeDisabled()
    fireEvent.click(button)
    fireEvent.submit(button.closest('form')!)
    expect(onSubmit).not.toHaveBeenCalled()
    expect(screen.getByText('Pick at least one day')).toBeInTheDocument()
  })

  it('does not submit twice while a request is pending', () => {
    const { onSubmit, button } = renderCard({ pending: true })
    expect(button).toBeDisabled()
    fireEvent.submit(button.closest('form')!)
    expect(onSubmit).not.toHaveBeenCalled()
  })
})
