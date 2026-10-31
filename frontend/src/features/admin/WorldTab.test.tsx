import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ConfirmContext } from '../../components/confirmDialogContext'
import { NotifyContext } from '../../components/notifyContext'
import i18n from '../../i18n'
import { setPhone } from './testPhone'
import { WorldTab } from './WorldTab'

const setWorldConfig = vi.hoisted(() => vi.fn())

vi.mock('../../api/hooks', () => {
  const idle = () => ({ mutate: vi.fn(), isPending: false })
  return {
    useAdminWorldConfig: () => ({
      isLoading: false,
      data: {
        tiers: [
          { tier: 1, name: 'Primera', groups: 1, teamsPerLeague: 10 },
          { tier: 2, name: 'Segunda', groups: 4, teamsPerLeague: 10 },
        ],
      },
    }),
    useAdminCountries: () => ({
      isLoading: false,
      data: [
        { code: 'es', name: 'Spain', active: true },
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
    useAdminAddCountry: idle,
    useAdminAddDivision: idle,
    useAdminRecreateLeagues: idle,
    useAdminRescheduleCountry: idle,
    useClearLeagueSchedule: idle,
    useSetLeagueSchedule: idle,
  }
})

function renderTab() {
  render(
    <ConfirmContext.Provider value={async () => true}>
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
    const groups = screen.getAllByLabelText(i18n.t('admin.worldStructure.groups'))
    expect(groups).toHaveLength(2)
    fireEvent.change(groups[1], { target: { value: '8' } })
    fireEvent.click(screen.getByRole('button', { name: i18n.t('admin.worldStructure.save') }))

    expect(setWorldConfig).toHaveBeenCalledWith(
      [
        { tier: 1, name: 'Primera', groups: 1, teamsPerLeague: 10 },
        { tier: 2, name: 'Segunda', groups: 8, teamsPerLeague: 10 },
      ],
      expect.any(Object),
    )
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
