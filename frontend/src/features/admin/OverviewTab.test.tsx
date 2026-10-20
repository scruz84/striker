import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ConfirmContext } from '../../components/confirmDialogContext'
import { NotifyContext } from '../../components/notifyContext'
import i18n from '../../i18n'
import { setPhone } from './testPhone'
import { OverviewTab } from './OverviewTab'

const finishStuckMatches = vi.hoisted(() => vi.fn())
const overview = vi.hoisted(() => ({
  current: {
    seasons: [{ country: 'es', season: 3, matchesTotal: 380, matchesUnplayed: 120 }],
    userCount: 12,
    teamCount: 210,
    leagueCount: 21,
    stuckMatches: 3,
    activeCups: 2,
    transferWindowOpen: true,
    freeAgentInjectionsPerWeek: 5,
  } as Record<string, unknown>,
}))

vi.mock('../../api/hooks', () => {
  const idle = () => ({ mutate: vi.fn(), isPending: false })
  return {
    useAdminOverview: () => ({ isLoading: false, data: overview.current }),
    useAdminFinishStuckMatches: () => ({ mutate: finishStuckMatches, isPending: false }),
    useAdminSimulateNow: idle,
    useAdminAdvanceSeason: idle,
    useAdminPayWagesNow: idle,
    useAdminInjectFreeAgentsNow: idle,
    useSetTransferWindow: idle,
  }
})

const notify = { success: vi.fn(), error: vi.fn() }
const confirm = vi.fn()

function renderTab() {
  render(
    <ConfirmContext.Provider value={confirm}>
      <NotifyContext.Provider value={notify}>
        <OverviewTab />
      </NotifyContext.Provider>
    </ConfirmContext.Provider>,
  )
}

describe('OverviewTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    confirm.mockResolvedValue(true)
    overview.current = { ...overview.current, stuckMatches: 3 }
    setPhone(false)
  })

  it('warns about stuck matches at the top, with the button to finish them right beside it', () => {
    renderTab()
    const alert = screen.getByRole('alert')
    expect(alert).toHaveTextContent(i18n.t('admin.overview.stuckMatchesAlert', { count: 3 }))
    expect(screen.getByRole('button', { name: i18n.t('admin.overview.stuckMatchesAction') })).toBeInTheDocument()
  })

  it('shows no warning when nothing is stuck', () => {
    overview.current = { ...overview.current, stuckMatches: 0 }
    renderTab()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('finishes the stuck matches from the warning, after confirming, and reports the result', async () => {
    finishStuckMatches.mockImplementation((_vars: unknown, options: { onSuccess: (r: { leagueMatches: number; friendlies: number }) => void }) =>
      options.onSuccess({ leagueMatches: 2, friendlies: 1 }),
    )
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.overview.stuckMatchesAction') }))

    await waitFor(() => expect(finishStuckMatches).toHaveBeenCalledTimes(1))
    expect(confirm).toHaveBeenCalledTimes(1)
    expect(notify.success).toHaveBeenCalledWith(i18n.t('admin.actions.finishStuckMatchesSuccess', { leagueMatches: 2, friendlies: 1 }))
  })

  it('does nothing when the confirmation is declined', async () => {
    confirm.mockResolvedValue(false)
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.overview.stuckMatchesAction') }))
    await waitFor(() => expect(confirm).toHaveBeenCalled())
    expect(finishStuckMatches).not.toHaveBeenCalled()
  })

  it('lists the season progress per country in a table on a wide screen', () => {
    renderTab()
    expect(screen.getByRole('table')).toBeInTheDocument()
  })

  it('lists the season progress per country as cards on a phone, with no table', () => {
    setPhone(true)
    renderTab()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByText('ES')).toBeInTheDocument()
    expect(screen.getByText(`${i18n.t('admin.overview.matchesUnplayed')}: 120 / 380`)).toBeInTheDocument()
  })

  it('keeps the same action among the other operations', async () => {
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.actions.finishStuckMatches', { count: 3 }) }))
    await waitFor(() => expect(finishStuckMatches).toHaveBeenCalledTimes(1))
  })
})
