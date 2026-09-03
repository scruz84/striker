import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SkillBar } from './SkillBar'

describe('SkillBar', () => {
  it('renders the label and the value', () => {
    render(<SkillBar label="Shot" value={72} />)
    expect(screen.getByText('Shot')).toBeInTheDocument()
    expect(screen.getByText('72')).toBeInTheDocument()
  })

  it('caps the progress bar at 100 even for out-of-range values', () => {
    render(<SkillBar label="Pass" value={150} />)
    const progressBar = screen.getByRole('progressbar')
    expect(progressBar).toHaveAttribute('aria-valuenow', '100')
  })
})
