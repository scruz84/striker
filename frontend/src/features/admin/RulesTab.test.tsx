import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { NotifyContext } from '../../components/notifyContext'
import i18n from '../../i18n'
import { RulesTab } from './RulesTab'

type MutateOptions = { onSuccess?: () => void; onError?: () => void }

const mutations = vi.hoisted(() => ({
  maxFriendlies: vi.fn(),
  matchSchedule: vi.fn(),
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
        promotionSlotsPerGroup: 2,
        relegationSlotsPerGroup: 2,
        maxFriendliesPerWeek: 3,
        friendlyScheduleWeekdays: 127,
        freeAgentInjectionsPerWeek: 5,
        offerNegotiationWindowMinutes: 30,
      },
    }),
    useAdminSetMaxFriendliesPerWeek: () => ({ mutate: mutations.maxFriendlies, isPending: false }),
    useSetMatchSchedule: () => ({ mutate: mutations.matchSchedule, isPending: false }),
    useAdminSetPromotionRelegation: idle,
    useAdminSetFriendlySchedule: idle,
    useAdminSetFreeAgentInjectionsPerWeek: idle,
    useAdminSetOfferNegotiationWindow: idle,
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
    for (const key of ['matchSchedule', 'promotionRelegation', 'maxFriendlies', 'friendlySchedule', 'freeAgentInjections', 'offerNegotiationWindow']) {
      expect(screen.getByText(i18n.t(`admin.${key}.title`))).toBeInTheDocument()
    }
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
})
