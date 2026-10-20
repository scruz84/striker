import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { WEEKDAY_OPTIONS } from '../../../api/types'
import i18n from '../../../i18n'
import { WeekdayPicker } from './WeekdayPicker'

describe('WeekdayPicker', () => {
  const label = (index: number) => i18n.t(`common.weekdaysShort.${WEEKDAY_OPTIONS[index].key}`)

  it('shows the days in the mask as pressed', () => {
    render(<WeekdayPicker value={WEEKDAY_OPTIONS[0].bit | WEEKDAY_OPTIONS[2].bit} onChange={() => {}} />)
    expect(screen.getByRole('button', { name: label(0) })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: label(1) })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: label(2) })).toHaveAttribute('aria-pressed', 'true')
  })

  it('adds a day to the mask when it is pressed', () => {
    const onChange = vi.fn()
    render(<WeekdayPicker value={WEEKDAY_OPTIONS[0].bit} onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: label(1) }))
    expect(onChange).toHaveBeenCalledWith(WEEKDAY_OPTIONS[0].bit | WEEKDAY_OPTIONS[1].bit)
  })

  it('removes a day from the mask when it is pressed again', () => {
    const onChange = vi.fn()
    render(<WeekdayPicker value={WEEKDAY_OPTIONS[0].bit | WEEKDAY_OPTIONS[1].bit} onChange={onChange} />)
    fireEvent.click(screen.getByRole('button', { name: label(1) }))
    expect(onChange).toHaveBeenCalledWith(WEEKDAY_OPTIONS[0].bit)
  })
})
