import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { POSITION_MIDFIELD, PLAYER_TRANSFER_LISTING, type TransferComparablesResult } from '../api/types'
import i18n from '../i18n'
import { TransferComparables } from './TransferComparables'

function result(overrides: Partial<TransferComparablesResult> = {}): TransferComparablesResult {
  return {
    position: POSITION_MIDFIELD,
    age: 26,
    averageSkill: 45,
    broadened: false,
    count: 2,
    medianFee: 135000,
    minFee: 100000,
    maxFee: 170000,
    comparables: [
      {
        transferId: 2,
        playerId: 11,
        playerName: 'Ana Gil',
        position: POSITION_MIDFIELD,
        age: 25,
        averageSkill: 44,
        fee: 170000,
        kind: PLAYER_TRANSFER_LISTING,
        createdAt: '2026-10-20T12:00:00Z',
        fromTeamId: 1,
        fromTeamName: 'Alpha FC',
        toTeamId: 2,
        toTeamName: 'Beta FC',
      },
      {
        transferId: 1,
        playerId: 12,
        playerName: 'Luis Mora',
        position: POSITION_MIDFIELD,
        age: 27,
        averageSkill: 46,
        fee: 100000,
        kind: PLAYER_TRANSFER_LISTING,
        createdAt: '2026-10-10T12:00:00Z',
      },
    ],
    ...overrides,
  }
}

function renderIt(props: Parameters<typeof TransferComparables>[0]) {
  return render(
    <MemoryRouter>
      <TransferComparables {...props} />
    </MemoryRouter>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})
afterEach(cleanup)

describe('TransferComparables', () => {
  it('shows skeleton rows while loading', () => {
    renderIt({ isLoading: true, isError: false })
    expect(screen.getByTestId('comparables-loading')).toBeTruthy()
    expect(screen.getByText('Recent comparable transfers')).toBeTruthy()
  })

  it('shows a short note on error', () => {
    renderIt({ isLoading: false, isError: true })
    expect(screen.getByText(/Could not load comparable transfers/)).toBeTruthy()
  })

  it('replaces the error note with a custom errorText', () => {
    renderIt({ isLoading: false, isError: true, errorText: 'Custom failure note' })
    expect(screen.getByText('Custom failure note')).toBeTruthy()
    expect(screen.queryByText(/You can still send your offer/)).toBeNull()
  })

  it('explains an empty result', () => {
    renderIt({ result: result({ count: 0, comparables: [], medianFee: undefined, minFee: undefined, maxFee: undefined }), isLoading: false, isError: false })
    expect(screen.getByText('No similar players have been transferred in the last 60 days')).toBeTruthy()
  })

  it('lists the summary and one row per comparable, linking each player', () => {
    renderIt({ result: result(), isLoading: false, isError: false })
    expect(screen.getByText(/2 similar transfers in the last 60 days · median .*135.*range/)).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Ana Gil' }).getAttribute('href')).toBe('/player/11')
    expect(screen.getByRole('link', { name: 'Luis Mora' }).getAttribute('href')).toBe('/player/12')
    expect(screen.getByText(/Alpha FC → Beta FC/)).toBeTruthy()
    expect(screen.getByText(/Midfield · 25 years · average skill 44/)).toBeTruthy()
    expect(screen.queryByText(/widened/)).toBeNull()
  })

  it('notes a broadened search', () => {
    renderIt({ result: result({ broadened: true }), isLoading: false, isError: false })
    expect(screen.getByText(/search was widened/)).toBeTruthy()
  })
})
