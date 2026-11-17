import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ConfirmContext } from '../../components/confirmDialogContext'
import { NotifyContext } from '../../components/notifyContext'
import i18n from '../../i18n'
import { setPhone } from './testPhone'
import { WorldTab } from './WorldTab'

const setWorldConfig = vi.hoisted(() => vi.fn())
const setCountryStructure = vi.hoisted(() => vi.fn())
const confirmMock = vi.hoisted(() => vi.fn(async () => true))
const setCountryKickoff = vi.hoisted(() => vi.fn())

vi.mock('../../api/hooks', () => {
  const idle = () => ({ mutate: vi.fn(), isPending: false })
  return {
    useAdminWorldConfig: () => ({
      isLoading: false,
      data: {
        tiers: [
          { tier: 1, name: 'Primera', groups: 1, teamsPerLeague: 10 },
          { tier: 2, name: 'Segunda', groups: 2, teamsPerLeague: 10 },
        ],
      },
    }),
    useAdminOverview: () => ({ isLoading: false, data: { pendingRestructure: ['es'] } }),
    useAdminCountries: () => ({
      isLoading: false,
      data: [
        { code: 'es', name: 'Spain', active: true, leagueKickoffMinute: 840 },
        { code: 'fr', name: 'France', active: false },
      ],
    }),
    useAdminLeagues: () => ({
      isLoading: false,
      data: [
        { leagueId: 1, country: 'es', name: 'Primera', tier: 1, schedule: { weekdaysMask: 127, hour: 12, minute: 0 }, hasOverride: true },
        { leagueId: 2, country: 'es', name: 'Segunda A', tier: 1, schedule: { weekdaysMask: 127, hour: 12, minute: 0 }, hasOverride: false },
      ],
    }),
    useSetWorldConfig: () => ({ mutate: setWorldConfig, isPending: false }),
    useAdminSetCountryStructure: () => ({ mutate: setCountryStructure, isPending: false }),
    useAdminClearCountryStructure: idle,
    // The real helper reads an ApiError's 409 body; tests hand the counts over directly.
    dissolveCountsOf: (error: { dissolve?: Record<string, number> } | undefined) => error?.dissolve ?? null,
    useAdminAddCountry: idle,
    useAdminAddDivision: idle,
    useAdminRecreateLeagues: idle,
    useAdminRescheduleCountry: idle,
    useAdminSetCountryKickoff: () => ({ mutate: setCountryKickoff, isPending: false, isError: false }),
    useClearLeagueSchedule: idle,
    useSetLeagueSchedule: idle,
  }
})

function renderTab() {
  render(
    <ConfirmContext.Provider value={confirmMock}>
      <NotifyContext.Provider value={{ success: vi.fn(), error: vi.fn() }}>
        <WorldTab />
      </NotifyContext.Provider>
    </ConfirmContext.Provider>,
  )
}

describe('WorldTab', () => {
  beforeEach(() => vi.clearAllMocks())

  it('uses tables on a wide screen', () => {
    setPhone(false)
    renderTab()
    expect(screen.getAllByRole('table').length).toBeGreaterThanOrEqual(2)
  })

  it('has no table at all on a phone - everything stacks as cards', () => {
    setPhone(true)
    renderTab()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('edits the world structure from a phone card and saves the changed tiers', () => {
    setPhone(true)
    renderTab()
    expect(screen.getByText(i18n.t('admin.worldStructure.places', { count: 30 }))).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.worldStructure.addTier') }))
    const names = screen.getAllByLabelText(i18n.t('admin.worldStructure.tierName'))
    fireEvent.change(names[2], { target: { value: 'Tercera' } })
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.worldStructure.save') }))

    // A new tier follows the pyramid rule: twice the groups of the deepest one.
    expect(setWorldConfig).toHaveBeenCalledWith(
      {
        tiers: [
          { tier: 1, name: 'Primera', groups: 1, teamsPerLeague: 10 },
          { tier: 2, name: 'Segunda', groups: 2, teamsPerLeague: 10 },
          { tier: 3, name: 'Tercera', groups: 4, teamsPerLeague: 10 },
        ],
        confirmDissolve: false,
      },
      expect.any(Object),
    )
  })

  it('explains a structure that breaks the pyramid rule and does not save it', () => {
    setPhone(true)
    renderTab()
    const groups = screen.getAllByLabelText(i18n.t('admin.worldStructure.groups'))
    fireEvent.change(groups[1], { target: { value: '4' } })
    expect(screen.getByText(i18n.t('admin.worldStructure.errors.doubling', { tier: 2, groups: 2, upper: 1 }))).toBeInTheDocument()
    expect(screen.getByRole('button', { name: i18n.t('admin.worldStructure.save') })).toBeDisabled()
  })

  it('asks before saving a structure that dissolves CPU teams, then saves it confirmed', async () => {
    setPhone(true)
    type Callbacks = { onSuccess: () => void; onError: (error: unknown) => void }
    setWorldConfig.mockImplementationOnce((_args: unknown, callbacks: Callbacks) => callbacks.onError({ dissolve: { es: 160, it: 5 } }))
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.worldStructure.save') }))

    const list = ['ES', 'IT'].map((country, i) => i18n.t('admin.countryStructure.dissolveCount', { country, count: [160, 5][i] })).join(', ')
    await vi.waitFor(() => expect(confirmMock).toHaveBeenCalledWith(expect.objectContaining({ message: i18n.t('admin.countryStructure.confirmDissolve', { list }) })))
    await vi.waitFor(() => expect(setWorldConfig).toHaveBeenCalledTimes(2))
    expect(setWorldConfig.mock.calls[1][0]).toEqual(expect.objectContaining({ confirmDissolve: true }))
  })

  it('does not resend a structure the admin did not confirm', async () => {
    setPhone(true)
    type Callbacks = { onSuccess: () => void; onError: (error: unknown) => void }
    setWorldConfig.mockImplementationOnce((_args: unknown, callbacks: Callbacks) => callbacks.onError({ dissolve: { es: 160 } }))
    confirmMock.mockResolvedValueOnce(false)
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.worldStructure.save') }))
    await vi.waitFor(() => expect(confirmMock).toHaveBeenCalled())
    expect(setWorldConfig).toHaveBeenCalledTimes(1)
  })

  it('gives an active country its own structure from its row', () => {
    setPhone(false)
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.countryStructure.edit') }))
    const dialog = screen.getByRole('dialog')
    // A country without its own structure starts from the global one, and has nothing to remove.
    expect(within(dialog).queryByRole('button', { name: i18n.t('admin.countryStructure.useGlobal') })).not.toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: i18n.t('admin.countryStructure.save') }))
    expect(setCountryStructure).toHaveBeenCalledWith(
      {
        code: 'es',
        tiers: [
          { tier: 1, name: 'Primera', groups: 1, teamsPerLeague: 10 },
          { tier: 2, name: 'Segunda', groups: 2, teamsPerLeague: 10 },
        ],
        confirmDissolve: false,
      },
      expect.any(Object),
    )
  })

  it('lists the countries waiting for a restructure', () => {
    setPhone(false)
    renderTab()
    expect(screen.getByText(i18n.t('admin.worldStructure.pending', { countries: 'ES' }))).toBeInTheDocument()
  })

  it('changes a country\'s league kickoff on the quarter-hour', () => {
    setPhone(false)
    renderTab()
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.countries.kickoffButton', { time: '14:00' }) }))
    const dialog = screen.getByRole('dialog')
    const field = within(dialog).getByLabelText(i18n.t('admin.countries.kickoffLabel'))
    fireEvent.change(field, { target: { value: '13:10' } })
    expect(within(dialog).getByRole('button', { name: i18n.t('admin.countries.kickoffSave') })).toBeDisabled()
    fireEvent.change(field, { target: { value: '13:15' } })
    fireEvent.click(within(dialog).getByRole('button', { name: i18n.t('admin.countries.kickoffSave') }))
    expect(setCountryKickoff).toHaveBeenCalledWith({ country: 'es', hour: 13, minute: 15 }, expect.any(Object))
  })

  it('marks a league that follows its country\'s kickoff apart from an override', () => {
    setPhone(false)
    renderTab()
    expect(screen.getByText(i18n.t('admin.leagueSchedules.override'))).toBeInTheDocument()
    expect(screen.getByText(i18n.t('admin.leagueSchedules.derived', { minutes: 0 }))).toBeInTheDocument()
  })

  it('lists each league as a card on a phone and opens its editor full screen', () => {
    setPhone(true)
    renderTab()
    expect(screen.getByText('ES Segunda A')).toBeInTheDocument()
    fireEvent.click(screen.getAllByRole('button', { name: i18n.t('admin.leagueSchedules.edit') })[0])
    const dialog = screen.getByRole('dialog')
    expect(dialog.className).toContain('MuiDialog-paperFullScreen')
    expect(within(dialog).getByRole('button', { name: i18n.t('admin.leagueSchedules.save') })).toBeInTheDocument()
  })

  it('opens the league editor as a normal dialog on a wide screen', () => {
    setPhone(false)
    renderTab()
    fireEvent.click(screen.getAllByRole('button', { name: i18n.t('admin.leagueSchedules.edit') })[0])
    expect(screen.getByRole('dialog').className).not.toContain('MuiDialog-paperFullScreen')
  })

  it('offers to add the missing divisions of an active country, and keeps the danger zone apart', () => {
    setPhone(true)
    renderTab()
    expect(screen.getByText(i18n.t('admin.countries.missingDivision', { name: 'Segunda' }))).toBeInTheDocument()
    expect(screen.getByText(i18n.t('admin.dangerZone.title'))).toBeInTheDocument()
  })

  it('lists the countries alphabetically by their name in the UI language, and re-sorts on a language switch', async () => {
    const countryRows = () =>
      screen
        .getAllByText(/\([a-z]{2,3}\)$/)
        .map((node) => node.textContent)
    const previous = i18n.language
    try {
      await i18n.changeLanguage('en')
      renderTab()
      expect(countryRows()).toEqual(['France (fr)', 'Spain (es)'])
      cleanup()
      await i18n.changeLanguage('es')
      renderTab()
      expect(countryRows()).toEqual(['España (es)', 'Francia (fr)'])
    } finally {
      await i18n.changeLanguage(previous)
    }
  })
})
