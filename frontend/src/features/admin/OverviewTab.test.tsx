import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ConfirmContext } from '../../components/confirmDialogContext'
import { NotifyContext } from '../../components/notifyContext'
import i18n from '../../i18n'
import { setPhone } from './testPhone'
import { OverviewTab } from './OverviewTab'

const finishStuckMatches = vi.hoisted(() => vi.fn())
const defaultSimulation = vi.hoisted(() => ({
  stuck: 3,
  failingCounts: { leagueMatches: 1, friendlies: 0, cupTies: 0 },
  failing: [
    {
      kind: 'league',
      id: 77,
      homeName: 'Real Norte',
      awayName: 'Atlético Sur',
      matchDate: '2026-03-05T14:30:00Z',
      attempts: 2,
      lastError: 'index out of range [11] with length 11',
    },
  ],
  standingsMismatches: 0,
}))
const overview = vi.hoisted(() => ({
  current: {
    seasons: [{ country: 'es', season: 3, matchesTotal: 380, matchesUnplayed: 120 }],
    userCount: 12,
    teamCount: 210,
    leagueCount: 21,
    simulation: defaultSimulation,
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

const simulation = (overrides: Record<string, unknown>) => ({ ...defaultSimulation, ...overrides })

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
    overview.current = { ...overview.current, simulation: defaultSimulation, aiLevels: undefined }
    setPhone(false)
  })

  it('warns about stuck and failing matches at the top, with the button to finish them right beside it', () => {
    renderTab()
    const alert = screen.getByRole('alert')
    // 3 stuck rows plus 1 failing fixture.
    expect(alert).toHaveTextContent(i18n.t('admin.overview.stuckMatchesAlert', { count: 4 }))
    expect(screen.getByRole('button', { name: i18n.t('admin.overview.stuckMatchesAction') })).toBeInTheDocument()
  })

  it('shows no warning when nothing is stuck or failing', () => {
    overview.current = {
      ...overview.current,
      simulation: simulation({ stuck: 0, failingCounts: { leagueMatches: 0, friendlies: 0, cupTies: 0 }, failing: [] }),
    }
    renderTab()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.queryByText(i18n.t('admin.overview.failingLastError'))).not.toBeInTheDocument()
  })

  it('warns when standings no longer match the matches played', () => {
    overview.current = { ...overview.current, simulation: simulation({ standingsMismatches: 5 }) }
    renderTab()
    expect(screen.getByText(i18n.t('admin.overview.standingsMismatchAlert', { count: 5 }))).toBeInTheDocument()
  })

  it('lists the failing matches with their attempts and last error', () => {
    renderTab()
    expect(screen.getByText(i18n.t('admin.overview.failingMatchesTitle', { count: 1 }))).toBeInTheDocument()
    expect(screen.getByText('Real Norte - Atlético Sur')).toBeInTheDocument()
    expect(screen.getByText(i18n.t('admin.overview.failingKind.league'))).toBeInTheDocument()
    expect(screen.getByText('index out of range [11] with length 11')).toBeInTheDocument()
    // The attempts column of the failing match's row.
    expect(within(screen.getByText('Real Norte - Atlético Sur').closest('tr') as HTMLElement).getByText('2')).toBeInTheDocument()
  })

  it('compares the clubs of each AI level, skipping levels that run no club', () => {
    overview.current = {
      ...overview.current,
      aiLevels: [
        { level: 'human', teams: 4, averageBudget: 500000, insolvent: 0, pointsPerMatch: 1.75 },
        { level: 'caretaker', teams: 0, averageBudget: 0, insolvent: 0, pointsPerMatch: 0 },
        { level: 'manager', teams: 30, averageBudget: 250000, insolvent: 2, pointsPerMatch: 1.31 },
        { level: 'director', teams: 12, averageBudget: 25000, insolvent: 1, pointsPerMatch: 1.4 },
      ],
    }
    renderTab()
    expect(screen.getByText(i18n.t('admin.overview.aiLevels.title'))).toBeInTheDocument()
    expect(screen.queryByText(i18n.t('admin.overview.aiLevels.caretaker'))).not.toBeInTheDocument()
    const director = screen.getByText(i18n.t('admin.overview.aiLevels.director')).closest('tr') as HTMLElement
    expect(within(director).getByText('12')).toBeInTheDocument()
    expect(within(director).getByText('1.40')).toBeInTheDocument()
    expect(within(director).getByText('1')).toBeInTheDocument()
    expect(within(screen.getByText(i18n.t('admin.overview.aiLevels.human')).closest('tr') as HTMLElement).getByText('1.75')).toBeInTheDocument()
  })

  it('lists the failing matches as cards on a phone', () => {
    setPhone(true)
    renderTab()
    expect(screen.getByText('Real Norte - Atlético Sur')).toBeInTheDocument()
    expect(screen.getByText('index out of range [11] with length 11')).toBeInTheDocument()
    expect(screen.queryAllByRole('columnheader')).toHaveLength(0)
  })

  it('finishes the stuck matches from the warning, after confirming, and reports the result', async () => {
    finishStuckMatches.mockImplementation((_vars: unknown, options: { onSuccess: (r: unknown) => void }) =>
      options.onSuccess({
        resimulated: { leagueMatches: 2, friendlies: 1, cupTies: 0 },
        released: { leagueMatches: 3, friendlies: 0, cupTies: 1 },
        defaulted: { leagueMatches: 1, friendlies: 0, cupTies: 1 },
        stillFailing: { leagueMatches: 0, friendlies: 1, cupTies: 0 },
      }),
    )
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.overview.stuckMatchesAction') }))

    await waitFor(() => expect(finishStuckMatches).toHaveBeenCalledTimes(1))
    expect(confirm).toHaveBeenCalledTimes(1)
    expect(notify.success).toHaveBeenCalledWith(
      i18n.t('admin.actions.finishStuckMatchesSuccess', { resimulated: 3, released: 4, defaulted: 2, stillFailing: 1 }),
    )
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
    // One table for the season progress, one for the failing matches.
    expect(screen.getAllByRole('table')).toHaveLength(2)
    expect(screen.getByText(i18n.t('admin.overview.matchesTotal'))).toBeInTheDocument()
  })

  it('lists the season progress per country as cards on a phone, with no table', () => {
    setPhone(true)
    renderTab()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByText(`${i18n.t('countries.es')} (ES)`)).toBeInTheDocument()
    expect(screen.getByText(`${i18n.t('admin.overview.matchesUnplayed')}: 120 / 380`)).toBeInTheDocument()
  })

  it('keeps the same action among the other operations', async () => {
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.actions.finishStuckMatches', { count: 4 }) }))
    await waitFor(() => expect(finishStuckMatches).toHaveBeenCalledTimes(1))
  })
})
