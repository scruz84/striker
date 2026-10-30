import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { UNAVAILABLE_INJURY, UNAVAILABLE_SUSPENSION } from '../../api/types'
import i18n from '../../i18n'
import { UnavailabilityBadge } from './UnavailabilityBadge'

describe('UnavailabilityBadge', () => {
  it('labels an injury with the matches it still has to run', () => {
    render(<UnavailabilityBadge unavailability={{ playerId: 1, reason: UNAVAILABLE_INJURY, matchesRemaining: 3 }} />)
    expect(screen.getByRole('img')).toHaveAttribute('aria-label', i18n.t('playerPage.status.injured', { count: 3 }))
  })

  it('labels a suspension', () => {
    render(<UnavailabilityBadge unavailability={{ playerId: 1, reason: UNAVAILABLE_SUSPENSION, matchesRemaining: 1 }} />)
    expect(screen.getByRole('img')).toHaveAttribute('aria-label', i18n.t('playerPage.status.suspended', { count: 1 }))
  })

  it('adds who replaces him to the label when known', () => {
    render(<UnavailabilityBadge unavailability={{ playerId: 1, reason: UNAVAILABLE_INJURY, matchesRemaining: 3 }} replacedBy="Ana Ruiz" />)
    expect(screen.getByRole('img')).toHaveAttribute(
      'aria-label',
      `${i18n.t('playerPage.status.injured', { count: 3 })} · ${i18n.t('lineup.replacedBy', { name: 'Ana Ruiz' })}`,
    )
  })
})
