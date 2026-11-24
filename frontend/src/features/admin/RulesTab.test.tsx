import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { NotifyContext } from '../../components/notifyContext'
import i18n from '../../i18n'
import { RulesTab } from './RulesTab'

type MutateOptions = { onSuccess?: () => void; onError?: () => void }

const mutations = vi.hoisted(() => ({
  maxFriendlies: vi.fn(),
  matchSchedule: vi.fn(),
  instantArchive: vi.fn(),
  friendlyInstantRetention: vi.fn(),
}))

vi.mock('../../api/hooks', () => {
  const idle = () => ({ mutate: vi.fn(), isPending: false })
  return {
    useAdminOverview: () => ({
      isLoading: false,
      data: {
        matchScheduleWeekdaysMask: 127,
        matchScheduleHour: 12,
        matchScheduleMinute: 0,
        maxFriendliesPerWeek: 3,
        friendlyScheduleWeekdays: 127,
        freeAgentInjectionsPerWeek: 5,
        offerNegotiationWindowMinutes: 30,
        instantArchiveEnabled: false,
        instantArchiveKeepSeasons: 1,
        friendlyInstantKeepDays: 7,
      },
    }),
    useAdminSetMaxFriendliesPerWeek: () => ({ mutate: mutations.maxFriendlies, isPending: false }),
    useSetMatchSchedule: () => ({ mutate: mutations.matchSchedule, isPending: false }),
    useAdminSetFriendlySchedule: idle,
    useAdminSetFreeAgentInjectionsPerWeek: idle,
    useAdminSetOfferNegotiationWindow: idle,
    useAdminSetInstantArchive: () => ({ mutate: mutations.instantArchive, isPending: false }),
    useAdminSetFriendlyInstantRetention: () => ({ mutate: mutations.friendlyInstantRetention, isPending: false }),
  }
})

const notify = { success: vi.fn(), error: vi.fn() }

function renderTab() {
  render(
    <NotifyContext.Provider value={notify}>
      <RulesTab />
    </NotifyContext.Provider>,
  )
}

// The card whose title is `title`, so a field/button is looked up inside the right form.
function card(title: string) {
  return screen.getByText(title).closest('form') as HTMLElement
}

describe('RulesTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows one card per rule', () => {
    renderTab()
    for (const key of ['matchSchedule', 'maxFriendlies', 'friendlySchedule', 'freeAgentInjections', 'offerNegotiationWindow', 'instantArchive', 'friendlyInstantRetention']) {
      expect(screen.getByText(i18n.t(`admin.${key}.title`))).toBeInTheDocument()
    }
    // Promotion and relegation follow the league pyramid; there is no setting for them any more.
    expect(screen.queryByText(/promotion|ascens/i)).not.toBeInTheDocument()
  })

  it('saves only the match days, keeping the stored kickoff time', () => {
    renderTab()
    const form = card(i18n.t('admin.matchSchedule.title'))
    expect(within(form).queryByLabelText(i18n.t('admin.matchSchedule.time'))).not.toBeInTheDocument()
    fireEvent.click(within(form).getByRole('button', { name: i18n.t('admin.matchSchedule.save') }))
    expect(mutations.matchSchedule).toHaveBeenCalledWith({ weekdaysMask: 127, hour: 12, minute: 0 }, expect.any(Object))
  })

  it('saves a setting with its value and reports success as a notification', () => {
    mutations.maxFriendlies.mockImplementation((_value: number, options: MutateOptions) => options.onSuccess?.())
    renderTab()
    const form = card(i18n.t('admin.maxFriendlies.title'))
    fireEvent.change(within(form).getByRole('spinbutton'), { target: { value: '7' } })
    fireEvent.click(within(form).getByRole('button', { name: i18n.t('admin.maxFriendlies.save') }))

    expect(mutations.maxFriendlies).toHaveBeenCalledWith(7, expect.any(Object))
    expect(notify.success).toHaveBeenCalledWith(i18n.t('admin.maxFriendlies.success'))
  })

  it('reports a failed save as an error notification', () => {
    mutations.maxFriendlies.mockImplementation((_value: number, options: MutateOptions) => options.onError?.())
    renderTab()
    fireEvent.click(within(card(i18n.t('admin.maxFriendlies.title'))).getByRole('button', { name: i18n.t('admin.maxFriendlies.save') }))
    expect(notify.error).toHaveBeenCalledWith(i18n.t('admin.maxFriendlies.error'))
  })

  it('blocks saving a negative number', () => {
    renderTab()
    const form = card(i18n.t('admin.maxFriendlies.title'))
    fireEvent.change(within(form).getByRole('spinbutton'), { target: { value: '-1' } })
    expect(within(form).getByRole('button', { name: i18n.t('admin.maxFriendlies.save') })).toBeDisabled()
  })

  it('blocks saving the match schedule with no weekday selected, and says so', () => {
    renderTab()
    const form = card(i18n.t('admin.matchSchedule.title'))
    for (const pressed of within(form).getAllByRole('button', { pressed: true })) {
      fireEvent.click(pressed)
    }
    expect(within(form).getByRole('button', { name: i18n.t('admin.matchSchedule.save') })).toBeDisabled()
    expect(within(form).getByText(i18n.t('admin.matchSchedule.noWeekdaysError'))).toBeInTheDocument()
    expect(mutations.matchSchedule).not.toHaveBeenCalled()
  })

  it('shows the season archive off, keeping 1 season, and saves it turned on', () => {
    renderTab()
    const form = card(i18n.t('admin.instantArchive.title'))
    const toggle = within(form).getByRole('switch', { name: i18n.t('admin.instantArchive.enabled') })
    expect(toggle).not.toBeChecked()
    expect(within(form).getByRole('spinbutton')).toHaveValue(1)

    fireEvent.click(toggle)
    fireEvent.change(within(form).getByRole('spinbutton'), { target: { value: '2' } })
    fireEvent.click(within(form).getByRole('button', { name: i18n.t('admin.instantArchive.save') }))
    expect(mutations.instantArchive).toHaveBeenCalledWith({ enabled: true, keepSeasons: 2 }, expect.any(Object))
  })

  it('refuses to save a season archive count outside 0-10', () => {
    renderTab()
    const form = card(i18n.t('admin.instantArchive.title'))
    fireEvent.change(within(form).getByRole('spinbutton'), { target: { value: '11' } })
    expect(within(form).getByText(i18n.t('admin.instantArchive.rangeError'))).toBeInTheDocument()
    expect(within(form).getByRole('button', { name: i18n.t('admin.instantArchive.save') })).toBeDisabled()
    expect(mutations.instantArchive).not.toHaveBeenCalled()
  })

  it('shows friendlies keeping 7 days of detail and saves a new count', () => {
    renderTab()
    const form = card(i18n.t('admin.friendlyInstantRetention.title'))
    expect(within(form).getByRole('spinbutton')).toHaveValue(7)

    fireEvent.change(within(form).getByRole('spinbutton'), { target: { value: '14' } })
    fireEvent.click(within(form).getByRole('button', { name: i18n.t('admin.friendlyInstantRetention.save') }))
    expect(mutations.friendlyInstantRetention).toHaveBeenCalledWith(14, expect.any(Object))
  })

  it('refuses to save a friendly detail count outside 0-365', () => {
    renderTab()
    const form = card(i18n.t('admin.friendlyInstantRetention.title'))
    fireEvent.change(within(form).getByRole('spinbutton'), { target: { value: '366' } })
    expect(within(form).getByText(i18n.t('admin.friendlyInstantRetention.rangeError'))).toBeInTheDocument()
    expect(within(form).getByRole('button', { name: i18n.t('admin.friendlyInstantRetention.save') })).toBeDisabled()
    expect(mutations.friendlyInstantRetention).not.toHaveBeenCalled()
  })
})
