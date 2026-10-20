import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ConfirmContext } from '../../components/confirmDialogContext'
import { NotifyContext } from '../../components/notifyContext'
import i18n from '../../i18n'
import { TeamsTab } from './TeamsTab'

const adjustBudget = vi.hoisted(() => vi.fn())
const setAiLevel = vi.hoisted(() => vi.fn())
const confirm = vi.hoisted(() => vi.fn())

vi.mock('../../api/hooks', () => ({
  useAdminOverview: () => ({ isLoading: false, data: { teamCount: 210 } }),
  useAdminAdjustBudget: () => ({ mutate: adjustBudget, isPending: false }),
  useAdminInjectAllTeamsBudget: () => ({ mutate: vi.fn(), isPending: false }),
  useAdminSetAiLevel: () => ({ mutate: setAiLevel, isPending: false }),
  useSearch: () => ({
    data: { teams: [{ id: 12, name: 'Real Betis', country: 'es', divisionName: 'Primera' }], players: [], coaches: [], stadiums: [] },
    isFetching: false,
  }),
}))

function renderTab() {
  render(
    <ConfirmContext.Provider value={confirm}>
      <NotifyContext.Provider value={{ success: vi.fn(), error: vi.fn() }}>
        <TeamsTab />
      </NotifyContext.Provider>
    </ConfirmContext.Provider>,
  )
}

// The card whose title is `title`, so fields and buttons are looked up inside the right form.
const card = (title: string) => screen.getByText(title).closest('form') as HTMLElement

async function pickTeam(form: HTMLElement) {
  const input = form.querySelector('input[role="combobox"]') as HTMLInputElement
  fireEvent.focus(input)
  fireEvent.change(input, { target: { value: 'bet' } })
  fireEvent.click(await screen.findByText('Real Betis (#12)'))
}

describe('TeamsTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    confirm.mockResolvedValue(true)
  })

  it('cannot adjust a budget until a team is picked', () => {
    renderTab()
    const form = card(i18n.t('admin.adjustBudget.title'))
    fireEvent.change(form.querySelector('input[inputmode], input:not([role])') as HTMLInputElement, { target: { value: '500' } })
    expect(form.querySelector('button[type="submit"]')).toBeDisabled()
  })

  it('adjusts the budget of the picked team, naming it in the confirmation', async () => {
    renderTab()
    const form = card(i18n.t('admin.adjustBudget.title'))
    await pickTeam(form)
    fireEvent.change(within(form).getByLabelText(i18n.t('admin.adjustBudget.amount')), { target: { value: '5000' } })
    fireEvent.change(within(form).getByLabelText(i18n.t('admin.adjustBudget.description')), { target: { value: 'bonus' } })
    fireEvent.click(form.querySelector('button[type="submit"]') as HTMLButtonElement)

    await waitFor(() => expect(adjustBudget).toHaveBeenCalled())
    expect(adjustBudget.mock.calls[0][0]).toEqual({ teamId: 12, amount: 5000, description: 'bonus' })
    expect(JSON.stringify(confirm.mock.calls[0][0].message)).toContain('Real Betis (#12)')
  })

  it('sets the AI level of the picked team', async () => {
    renderTab()
    const form = card(i18n.t('admin.setAiLevel.title'))
    await pickTeam(form)
    fireEvent.click(form.querySelector('button[type="submit"]') as HTMLButtonElement)

    await waitFor(() => expect(setAiLevel).toHaveBeenCalled())
    expect(setAiLevel.mock.calls[0][0]).toMatchObject({ teamId: 12 })
  })
})
