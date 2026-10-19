import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import {
  PLAYER_TRANSFER_FREE_AGENT,
  PLAYER_TRANSFER_LISTING,
  PLAYER_TRANSFER_RELEASE,
  type PlayerTransfer,
} from '../../api/types'
import i18n from '../../i18n'
import { PlayerTransferHistory } from './PlayerTransferHistory'

function renderHistory(transfers: PlayerTransfer[]) {
  return render(
    <MemoryRouter>
      <PlayerTransferHistory transfers={transfers} />
    </MemoryRouter>,
  )
}

describe('PlayerTransferHistory', () => {
  it('shows an empty message when the player has never moved', () => {
    renderHistory([])
    expect(screen.getByText(i18n.t('playerPage.history.none'))).toBeInTheDocument()
  })

  it('renders one row per transfer with linked clubs, translated type and price', () => {
    renderHistory([
      {
        id: 2,
        fromTeamId: 7,
        fromTeamName: 'Rovers',
        toTeamId: 9,
        toTeamName: 'United',
        fee: 1500000,
        kind: PLAYER_TRANSFER_LISTING,
        createdAt: '2026-10-26T10:00:00Z',
      },
      { id: 1, toTeamId: 7, toTeamName: 'Rovers', fee: 0, kind: PLAYER_TRANSFER_FREE_AGENT, createdAt: '2026-09-01T10:00:00Z' },
    ])

    const rows = screen.getAllByRole('row').slice(1)
    expect(rows).toHaveLength(2)

    const first = within(rows[0])
    expect(first.getByRole('link', { name: 'Rovers' })).toHaveAttribute('href', '/team/7')
    expect(first.getByRole('link', { name: 'United' })).toHaveAttribute('href', '/team/9')
    expect(first.getAllByText(i18n.t('playerPage.history.kinds.listing')).length).toBeGreaterThan(0)
    expect(first.getByText('$1,500,000')).toBeInTheDocument()
  })

  it('labels a missing origin as free agent and a missing destination as released', () => {
    renderHistory([
      { id: 2, fromTeamId: 7, fromTeamName: 'Rovers', fee: 0, kind: PLAYER_TRANSFER_RELEASE, createdAt: '2026-10-26T10:00:00Z' },
      { id: 1, toTeamId: 7, toTeamName: 'Rovers', fee: 0, kind: PLAYER_TRANSFER_FREE_AGENT, createdAt: '2026-09-01T10:00:00Z' },
    ])

    const rows = screen.getAllByRole('row').slice(1)
    const releasedRoute = within(rows[0]).getAllByRole('cell')[1]
    expect(releasedRoute).toHaveTextContent(`Rovers → ${i18n.t('playerPage.history.released')}`)
    const signedRoute = within(rows[1]).getAllByRole('cell')[1]
    expect(signedRoute).toHaveTextContent(`${i18n.t('playerPage.history.freeAgent')} → Rovers`)
  })

  it('keeps the date from wrapping', () => {
    renderHistory([
      { id: 1, toTeamId: 7, toTeamName: 'Rovers', fee: 0, kind: PLAYER_TRANSFER_FREE_AGENT, createdAt: '2026-10-26T10:00:00Z' },
    ])
    const dateCell = screen.getAllByRole('cell')[0]
    expect(getComputedStyle(dateCell).whiteSpace).toBe('nowrap')
  })
})
