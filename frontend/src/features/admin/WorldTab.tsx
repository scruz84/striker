import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogTitle from '@mui/material/DialogTitle'
import FormControl from '@mui/material/FormControl'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import {
  useAdminAddCountry,
  useAdminAddDivision,
  useAdminClearCountryStructure,
  useAdminCountries,
  useAdminLeagues,
  useAdminRecreateLeagues,
  useAdminOverview,
  useAdminRescheduleCountry,
  useAdminSetCountryKickoff,
  useAdminSetCountryStructure,
  useAdminWorldConfig,
  useClearLeagueSchedule,
  useSetLeagueSchedule,
  useSetWorldConfig,
  dissolveCountsOf,
} from '../../api/hooks'
import {
  ALL_WEEKDAYS_MASK,
  WEEKDAY_OPTIONS,
  type AdminLeagueInfo,
  type CountryStatus,
  type ShapeDifference,
  type TierConfig,
} from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { useNotify } from '../../components/notifyContext'
import { AdminSection } from './components/AdminSection'
import { DangerZone } from './components/DangerZone'
import { TierEditor } from './components/TierEditor'
import { WeekdayPicker } from './components/WeekdayPicker'
import { useAdminFeedback } from './useAdminFeedback'
import { useIsPhone } from './useIsPhone'
import { localizedCountryName, useSortedCountries } from '../../utils/countryOrder'
import { localHourMinuteToSchedule, scheduleHourMinuteToLocal } from '../../utils/scheduleTime'
import { pyramidError } from './pyramidRules'

export function WorldTab() {
  const { data: worldConfig, isLoading } = useAdminWorldConfig(true)

  return (
    <>
      {isLoading || !worldConfig ? (
        <Skeleton variant="rectangular" height={150} sx={{ mb: 2 }} />
      ) : (
        <WorldStructureSection tiers={worldConfig.tiers} />
      )}

      <CountriesSection />

      <LeagueSchedulesSection />

      <DangerZone>
        <RecreateLeaguesAction />
      </DangerZone>
    </>
  )
}

function RecreateLeaguesAction() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const feedback = useAdminFeedback()
  const recreateLeagues = useAdminRecreateLeagues()

  const handleRecreate = async () => {
    const ok = await confirm({
      message: t('admin.worldStructure.confirmRecreate'),
      confirmLabel: t('admin.worldStructure.recreateButton'),
      destructive: true,
    })
    if (ok) {
      recreateLeagues.mutate(undefined, feedback('admin.worldStructure.recreateSuccess', 'admin.worldStructure.recreateError'))
    }
  }

  return (
    <>
      <Typography variant="subtitle2" color="error" gutterBottom>
        {t('admin.worldStructure.recreateTitle')}
      </Typography>
      <Button variant="outlined" color="error" onClick={handleRecreate} disabled={recreateLeagues.isPending}>
        {t('admin.worldStructure.recreateButton')}
      </Button>
    </>
  )
}

// describeDifference explains one way a pending country's leagues differ from its structure
// (backend common.ShapeDifference), e.g. "ES: group III.4 is missing".
function describeDifference(t: TFunction, country: string, difference: ShapeDifference): string {
  const params = {
    country: country.toUpperCase(),
    league: difference.league ?? '',
    tier: difference.tier ?? 0,
    group: difference.group ?? 0,
    teams: difference.teams ?? 0,
    expected: difference.expected ?? 0,
    message: difference.message ?? '',
  }
  switch (difference.kind) {
    case 'extra':
      return t('admin.worldStructure.pendingDetail.extra', params)
    case 'missing':
      return t('admin.worldStructure.pendingDetail.missing', params)
    case 'size':
      return t('admin.worldStructure.pendingDetail.size', params)
    case 'ratio':
      return t('admin.worldStructure.pendingDetail.ratio', params)
    case 'invalid_structure':
      return t('admin.worldStructure.pendingDetail.invalidStructure', params)
    default:
      return t('admin.worldStructure.pendingDetail.other', params)
  }
}

function WorldStructureSection({ tiers: initialTiers }: { tiers: TierConfig[] }) {
  const { t } = useTranslation()
  const [tiers, setTiers] = useState<TierConfig[]>(initialTiers)
  const setWorldConfig = useSetWorldConfig()
  const saveStructure = useStructureSave()
  const { data: overview } = useAdminOverview(true)
  const pending = overview?.pendingRestructure ?? []
  const pendingDetails = overview?.pendingRestructureDetails ?? []
  const ownStructures = overview?.countryStructures ?? []

  const handleSave = () =>
    saveStructure(
      (confirmDissolve, callbacks) => setWorldConfig.mutate({ tiers, confirmDissolve }, callbacks),
      'admin.worldStructure.saveSuccess',
      'admin.worldStructure.saveError',
    )

  return (
    <AdminSection title={t('admin.worldStructure.title')} description={t('admin.worldStructure.description')}>
      {pending.length > 0 && (
        <Alert severity="info" sx={{ mb: 2 }}>
          {t('admin.worldStructure.pending', { countries: pending.map((code) => code.toUpperCase()).join(', ') })}
          {pendingDetails.length > 0 && (
            <Box component="ul" sx={{ m: 0, mt: 0.5, pl: 2.5 }}>
              {pendingDetails.flatMap(({ country, differences }) =>
                differences.map((difference, i) => (
                  <li key={`${country}-${i}`}>
                    <Typography variant="body2">{describeDifference(t, country, difference)}</Typography>
                  </li>
                )),
              )}
            </Box>
          )}
        </Alert>
      )}
      {ownStructures.length > 0 && (
        <Alert severity="info" sx={{ mb: 2 }}>
          {t('admin.countryStructure.overridden', { countries: ownStructures.map((code) => code.toUpperCase()).join(', ') })}
        </Alert>
      )}
      <TierEditor
        tiers={tiers}
        onChange={setTiers}
        actions={
          <Button size="small" variant="contained" onClick={handleSave} disabled={setWorldConfig.isPending || pyramidError(tiers) !== null}>
            {t('admin.worldStructure.save')}
          </Button>
        }
      />
    </AdminSection>
  )
}

type MutateCallbacks = { onSuccess: () => void; onError: (error: unknown) => void }

// useStructureSave saves a structure (global or a country's own) and, when the backend refuses it
// because it would dissolve CPU teams, asks the admin to confirm with the teams each country loses
// before sending it again with confirmDissolve (openspec/specs/league-pyramid "Confirming a
// structure that dissolves teams").
function useStructureSave() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const notify = useNotify()

  return (
    save: (confirmDissolve: boolean, callbacks: MutateCallbacks) => void,
    successKey: string,
    errorKey: string,
    onSaved?: () => void,
  ) => {
    const done = {
      onSuccess: () => {
        notify.success(t(successKey))
        onSaved?.()
      },
      onError: () => notify.error(t(errorKey)),
    }
    save(false, {
      onSuccess: done.onSuccess,
      onError: async (error) => {
        const dissolve = dissolveCountsOf(error)
        if (!dissolve) {
          done.onError()
          return
        }
        const list = Object.entries(dissolve)
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([country, count]) => t('admin.countryStructure.dissolveCount', { country: country.toUpperCase(), count }))
          .join(', ')
        const ok = await confirm({
          message: t('admin.countryStructure.confirmDissolve', { list }),
          confirmLabel: t('admin.countryStructure.confirmDissolveButton'),
          destructive: true,
        })
        if (ok) {
          save(true, done)
        }
      },
    })
  }
}

function CountriesSection() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data, isLoading } = useAdminCountries(true)
  // Alphabetical by the name in the UI language, and shown in it (the API's English name is only
  // the fallback for an unknown code).
  const { sorted: countries, countryName } = useSortedCountries(data, (country) => country.code)
  const { data: worldConfig } = useAdminWorldConfig(true)
  const { data: leagues } = useAdminLeagues(true)
  const addCountry = useAdminAddCountry()
  const addDivision = useAdminAddDivision()
  const rescheduleCountry = useAdminRescheduleCountry()
  const feedback = useAdminFeedback()
  const [addingCountry, setAddingCountry] = useState<CountryStatus | null>(null)
  const [editingKickoff, setEditingKickoff] = useState<CountryStatus | null>(null)
  const [editingStructure, setEditingStructure] = useState<CountryStatus | null>(null)

  const handleReschedule = async (country: CountryStatus) => {
    const ok = await confirm({
      message: t('admin.countries.confirmReschedule', { country: countryName(country.code) }),
      confirmLabel: t('admin.countries.reschedule'),
    })
    if (ok) {
      rescheduleCountry.mutate(country.code, feedback('admin.countries.rescheduleSuccess', 'admin.countries.rescheduleError'))
    }
  }

  const handleAddDivision = async (country: CountryStatus, tier: TierConfig, withMatches: boolean) => {
    const ok = await confirm({
      message: t(
        withMatches ? 'admin.countries.confirmAddDivisionWithMatches' : 'admin.countries.confirmAddDivisionTeamsOnly',
        { country: countryName(country.code), division: tier.name },
      ),
      confirmLabel: t('admin.countries.addDivision'),
    })
    if (ok) {
      addDivision.mutate(
        { country: country.code, tier: tier.tier, withMatches },
        feedback('admin.countries.addDivisionSuccess', 'admin.countries.addDivisionError'),
      )
    }
  }

  if (isLoading || !countries) {
    return <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
  }

  // Every tier the country's leagues table doesn't have yet, from the current world structure -
  // see handlers.AdminAddDivisionHandler (issue #68).
  const existingTiersByCountry = new Map<string, Set<number>>()
  for (const league of leagues ?? []) {
    const tiers = existingTiersByCountry.get(league.country) ?? new Set<number>()
    tiers.add(league.tier)
    existingTiersByCountry.set(league.country, tiers)
  }

  return (
    <AdminSection title={t('admin.countries.title')} description={t('admin.countries.description')}>
      <Stack spacing={2}>
        {countries.map((country) => {
          const missingTiers = (country.structure ?? worldConfig?.tiers ?? []).filter(
            (tier) => !existingTiersByCountry.get(country.code)?.has(tier.tier),
          )
          return (
            <Stack key={country.code} spacing={0.5}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5 }}>
                <Typography sx={{ flex: '1 1 10rem', minWidth: 0 }}>
                  {countryName(country.code)} ({country.code})
                </Typography>
                {country.active ? (
                  <>
                    <Chip size="small" color="success" label={t('admin.countries.active')} />
                    {country.leagueKickoffMinute !== undefined && (
                      <Button size="small" onClick={() => setEditingKickoff(country)}>
                        {t('admin.countries.kickoffButton', { time: minuteOfDayToTime(country.leagueKickoffMinute) })}
                      </Button>
                    )}
                    <Button size="small" onClick={() => handleReschedule(country)} disabled={rescheduleCountry.isPending}>
                      {t('admin.countries.reschedule')}
                    </Button>
                    {country.structure && <Chip size="small" color="info" label={t('admin.countryStructure.ownChip')} />}
                    <Button size="small" onClick={() => setEditingStructure(country)}>
                      {t('admin.countryStructure.edit')}
                    </Button>
                  </>
                ) : (
                  <Button size="small" variant="outlined" onClick={() => setAddingCountry(country)} disabled={addCountry.isPending}>
                    {t('admin.countries.add')}
                  </Button>
                )}
              </Stack>
              {country.active &&
                missingTiers.map((tier) => (
                  <Stack
                    key={tier.tier}
                    direction="row"
                    spacing={1}
                    sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5, ml: { xs: 1, sm: 2 }, pl: 1.5, borderLeft: 2, borderColor: 'divider' }}
                  >
                    <Typography variant="body2" color="text.secondary" sx={{ flex: '1 1 10rem', minWidth: 0 }}>
                      {t('admin.countries.missingDivision', { name: tier.name })}
                    </Typography>
                    <Button size="small" onClick={() => handleAddDivision(country, tier, true)} disabled={addDivision.isPending}>
                      {t('admin.countries.addDivisionWithMatches')}
                    </Button>
                    <Button size="small" onClick={() => handleAddDivision(country, tier, false)} disabled={addDivision.isPending}>
                      {t('admin.countries.addDivisionTeamsOnly')}
                    </Button>
                  </Stack>
                ))}
            </Stack>
          )
        })}
      </Stack>
      {addingCountry && <AddCountryDialog country={addingCountry} onClose={() => setAddingCountry(null)} />}
      {editingKickoff && <CountryKickoffDialog country={editingKickoff} onClose={() => setEditingKickoff(null)} />}
      {editingStructure && worldConfig && (
        <CountryStructureDialog
          country={editingStructure}
          countryName={countryName(editingStructure.code)}
          globalTiers={worldConfig.tiers}
          onClose={() => setEditingStructure(null)}
        />
      )}
    </AdminSection>
  )
}

// CountryStructureDialog edits one active country's own structure (openspec/specs/league-pyramid
// "Per-country structure"), starting from it or, for a country without one, from the global
// structure. "Use the global structure" removes it. Either takes effect at the country's next
// rollover, and one that dissolves CPU teams asks first (see useStructureSave).
function CountryStructureDialog({
  country,
  countryName,
  globalTiers,
  onClose,
}: {
  country: CountryStatus
  countryName: string
  globalTiers: TierConfig[]
  onClose: () => void
}) {
  const { t } = useTranslation()
  const [tiers, setTiers] = useState<TierConfig[]>(country.structure ?? globalTiers)
  const setCountryStructure = useAdminSetCountryStructure()
  const clearCountryStructure = useAdminClearCountryStructure()
  const saveStructure = useStructureSave()
  const busy = setCountryStructure.isPending || clearCountryStructure.isPending

  const handleSave = () =>
    saveStructure(
      (confirmDissolve, callbacks) => setCountryStructure.mutate({ code: country.code, tiers, confirmDissolve }, callbacks),
      'admin.countryStructure.saveSuccess',
      'admin.countryStructure.saveError',
      onClose,
    )
  const handleClear = () =>
    saveStructure(
      (confirmDissolve, callbacks) => clearCountryStructure.mutate({ code: country.code, confirmDissolve }, callbacks),
      'admin.countryStructure.clearSuccess',
      'admin.countryStructure.saveError',
      onClose,
    )

  return (
    <Dialog open onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>{t('admin.countryStructure.title', { country: countryName })}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t('admin.countryStructure.description')}
        </Typography>
        <TierEditor tiers={tiers} onChange={setTiers} />
      </DialogContent>
      <DialogActions sx={{ flexWrap: 'wrap', rowGap: 1 }}>
        {country.structure && (
          <Button onClick={handleClear} disabled={busy}>
            {t('admin.countryStructure.useGlobal')}
          </Button>
        )}
        <Button onClick={onClose}>{t('common.cancel')}</Button>
        <Button variant="contained" onClick={handleSave} disabled={busy || pyramidError(tiers) !== null}>
          {t('admin.countryStructure.save')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

// AddCountryDialog lets the commissioner pick what local time of day country's matches kick off
// at (see common.DefaultKickoffHour/LocalTimeToServerSchedule on the backend) before activating
// it - defaulting to 14:00 in that country's own time zone, not the viewer's, since that's the
// whole point of asking (unlike LeagueScheduleDialog's time field, which does convert to/from the
// viewer's local time because that field's wire format is server-time, not country-time).
function AddCountryDialog({ country, onClose }: { country: CountryStatus; onClose: () => void }) {
  const { t } = useTranslation()
  const addCountry = useAdminAddCountry()
  const isPhone = useIsPhone()
  const [time, setTime] = useState('14:00')
  const name = localizedCountryName(t, country.code)

  const handleAdd = () => {
    if (!time) {
      return
    }
    const [hourStr, minuteStr] = time.split(':')
    addCountry.mutate(
      { code: country.code, kickoffHour: Number(hourStr), kickoffMinute: Number(minuteStr) },
      { onSuccess: onClose },
    )
  }

  return (
    <Dialog open onClose={onClose} maxWidth="xs" fullWidth fullScreen={isPhone}>
      <DialogTitle>{t('admin.countries.addDialogTitle', { name })}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <Typography variant="body2" color="text.secondary">
            {t('admin.countries.addDialogDescription', { name })}
          </Typography>
          <TextField
            label={t('admin.countries.kickoffLabel')}
            type="time"
            size="small"
            value={time}
            onChange={(e) => setTime(e.target.value)}
            helperText={t('admin.countries.kickoffHelp', { name })}
            slotProps={{ inputLabel: { shrink: true } }}
          />
          {addCountry.isError && <Alert severity="error">{t('admin.countries.addError')}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t('common.cancel')}</Button>
        <Button onClick={handleAdd} variant="contained" disabled={!time || addCountry.isPending}>
          {t('admin.countries.add')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function minuteOfDayToTime(minute: number) {
  return `${String(Math.floor(minute / 60)).padStart(2, '0')}:${String(minute % 60).padStart(2, '0')}`
}

// CountryKickoffDialog changes an active country's league kickoff: tier 1's local time, every lower
// tier 15 minutes later. Saving re-dates the country's unplayed matches of leagues without an
// override (backend handlers.AdminSetCountryKickoffHandler).
function CountryKickoffDialog({ country, onClose }: { country: CountryStatus; onClose: () => void }) {
  const { t } = useTranslation()
  const setKickoff = useAdminSetCountryKickoff()
  const isPhone = useIsPhone()
  const [time, setTime] = useState(minuteOfDayToTime(country.leagueKickoffMinute ?? 14 * 60))
  const name = localizedCountryName(t, country.code)
  const [hourStr, minuteStr] = time.split(':')
  const minute = Number(minuteStr)
  const invalid = !time || Number.isNaN(minute) || minute % 15 !== 0

  return (
    <Dialog open onClose={onClose} maxWidth="xs" fullWidth fullScreen={isPhone}>
      <DialogTitle>{t('admin.countries.kickoffDialogTitle', { name })}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <Typography variant="body2" color="text.secondary">
            {t('admin.countries.kickoffDialogDescription', { name })}
          </Typography>
          <TextField
            label={t('admin.countries.kickoffLabel')}
            type="time"
            size="small"
            value={time}
            onChange={(e) => setTime(e.target.value)}
            error={invalid}
            helperText={invalid ? t('admin.countries.kickoffQuarterHour') : t('admin.countries.kickoffHelp', { name })}
            slotProps={{ inputLabel: { shrink: true }, htmlInput: { step: 900 } }}
          />
          {setKickoff.isError && <Alert severity="error">{t('admin.countries.kickoffError')}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t('common.cancel')}</Button>
        <Button
          variant="contained"
          disabled={invalid || setKickoff.isPending}
          onClick={() => setKickoff.mutate({ country: country.code, hour: Number(hourStr), minute }, { onSuccess: onClose })}
        >
          {t('admin.countries.kickoffSave')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function describeSchedule(
  t: ReturnType<typeof useTranslation>['t'],
  schedule: { weekdaysMask: number; hour: number; minute: number },
  localTime = false,
) {
  const dayLabel =
    schedule.weekdaysMask === ALL_WEEKDAYS_MASK
      ? t('admin.leagueSchedules.everyDay')
      : WEEKDAY_OPTIONS.filter((d) => (schedule.weekdaysMask & d.bit) !== 0)
          .map((d) => t(`common.weekdaysShort.${d.key}`))
          .join(', ')
  // A league without an override is on its country's own clock; an override on the server's,
  // shown in the viewer's time.
  if (localTime) {
    return `${dayLabel} ${minuteOfDayToTime(schedule.hour * 60 + schedule.minute)} ${t('admin.leagueSchedules.countryTime')}`
  }
  const local = scheduleHourMinuteToLocal(schedule.hour, schedule.minute)
  const time = `${String(local.hour).padStart(2, '0')}:${String(local.minute).padStart(2, '0')}`
  return `${dayLabel} ${time}`
}

function LeagueSchedulesSection() {
  const { t } = useTranslation()
  const { data: leagues, isLoading } = useAdminLeagues(true)
  const isPhone = useIsPhone()
  const [countryFilter, setCountryFilter] = useState('')
  const [editingLeague, setEditingLeague] = useState<AdminLeagueInfo | null>(null)

  if (isLoading || !leagues) {
    return <Skeleton variant="rectangular" height={200} sx={{ mb: 2 }} />
  }

  const countries = Array.from(new Set(leagues.map((l) => l.country))).sort()
  const filteredLeagues = countryFilter ? leagues.filter((l) => l.country === countryFilter) : leagues

  return (
    <AdminSection title={t('admin.leagueSchedules.title')} description={t('admin.leagueSchedules.description')}>
      <FormControl size="small" sx={{ mb: 2, minWidth: 180, width: { xs: '100%', sm: 'auto' } }}>
        <InputLabel id="league-schedule-country-label">{t('admin.leagueSchedules.filterCountry')}</InputLabel>
        <Select
          labelId="league-schedule-country-label"
          label={t('admin.leagueSchedules.filterCountry')}
          value={countryFilter}
          onChange={(e) => setCountryFilter(e.target.value)}
        >
          <MenuItem value="">{t('admin.leagueSchedules.allCountries')}</MenuItem>
          {countries.map((c) => (
            <MenuItem key={c} value={c}>
              {c.toUpperCase()}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      {isPhone ? (
        <Stack spacing={1.5}>
          {filteredLeagues.map((league) => (
            <Paper key={league.leagueId} variant="outlined" sx={{ p: 1.5 }}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'baseline', justifyContent: 'space-between' }}>
                <Typography sx={{ fontWeight: 600, minWidth: 0, overflowWrap: 'anywhere' }}>
                  {league.country.toUpperCase()} {league.name}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ flexShrink: 0 }}>
                  {t('admin.leagueSchedules.tier')} {league.tier}
                </Typography>
              </Stack>
              <Stack direction="row" spacing={1} sx={{ mt: 0.5, alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5 }}>
                <Typography variant="body2" sx={{ flex: '1 1 auto' }}>
                  {describeSchedule(t, league.schedule, league.localTime)}
                </Typography>
                {league.hasOverride ? (
                  <Chip size="small" label={t('admin.leagueSchedules.override')} />
                ) : (
                  <Chip size="small" variant="outlined" label={t('admin.leagueSchedules.derived', { minutes: (league.tier - 1) * 15 })} />
                )}
                <Button size="small" onClick={() => setEditingLeague(league)}>
                  {t('admin.leagueSchedules.edit')}
                </Button>
              </Stack>
            </Paper>
          ))}
        </Stack>
      ) : (
        <TableContainer sx={{ maxHeight: 400 }}>
          <Table size="small" stickyHeader>
            <TableHead>
              <TableRow>
                <TableCell>{t('admin.leagueSchedules.league')}</TableCell>
                <TableCell align="right">{t('admin.leagueSchedules.tier')}</TableCell>
                <TableCell>{t('admin.leagueSchedules.schedule')}</TableCell>
                <TableCell align="right" />
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredLeagues.map((league) => (
                <TableRow key={league.leagueId}>
                  <TableCell>
                    {league.country.toUpperCase()} {league.name}
                  </TableCell>
                  <TableCell align="right">{league.tier}</TableCell>
                  <TableCell>
                    {describeSchedule(t, league.schedule, league.localTime)}
                    {league.hasOverride ? (
                      <Chip size="small" label={t('admin.leagueSchedules.override')} sx={{ ml: 1 }} />
                    ) : (
                      <Chip size="small" variant="outlined" label={t('admin.leagueSchedules.derived', { minutes: (league.tier - 1) * 15 })} sx={{ ml: 1 }} />
                    )}
                  </TableCell>
                  <TableCell align="right">
                    <Button size="small" onClick={() => setEditingLeague(league)}>
                      {t('admin.leagueSchedules.edit')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      {editingLeague && <LeagueScheduleDialog league={editingLeague} onClose={() => setEditingLeague(null)} />}
    </AdminSection>
  )
}

function LeagueScheduleDialog({ league, onClose }: { league: AdminLeagueInfo; onClose: () => void }) {
  const { t } = useTranslation()
  const setLeagueSchedule = useSetLeagueSchedule()
  const clearLeagueSchedule = useClearLeagueSchedule()
  const isPhone = useIsPhone()
  const [weekdaysMask, setWeekdaysMask] = useState(league.schedule.weekdaysMask)
  const initialLocalTime = scheduleHourMinuteToLocal(league.schedule.hour, league.schedule.minute)
  const [time, setTime] = useState(
    `${String(initialLocalTime.hour).padStart(2, '0')}:${String(initialLocalTime.minute).padStart(2, '0')}`,
  )

  const handleSave = () => {
    if (weekdaysMask === 0 || !time) {
      return
    }
    const [hourStr, minuteStr] = time.split(':')
    const { hour, minute } = localHourMinuteToSchedule(Number(hourStr), Number(minuteStr))
    setLeagueSchedule.mutate(
      { leagueId: league.leagueId, schedule: { weekdaysMask, hour, minute } },
      { onSuccess: onClose },
    )
  }

  const handleReset = () => {
    clearLeagueSchedule.mutate(league.leagueId, { onSuccess: onClose })
  }

  return (
    <Dialog open onClose={onClose} maxWidth="xs" fullWidth fullScreen={isPhone}>
      <DialogTitle>
        {t('admin.leagueSchedules.dialogTitle', { league: `${league.country.toUpperCase()} ${league.name}` })}
      </DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <WeekdayPicker value={weekdaysMask} onChange={setWeekdaysMask} />
          <TextField
            label={t('admin.matchSchedule.time')}
            type="time"
            size="small"
            value={time}
            onChange={(e) => setTime(e.target.value)}
            slotProps={{ inputLabel: { shrink: true } }}
          />
          {weekdaysMask === 0 && (
            <Typography variant="caption" color="error">
              {t('admin.matchSchedule.noWeekdaysError')}
            </Typography>
          )}
          {setLeagueSchedule.isError && <Alert severity="error">{t('admin.leagueSchedules.saveError')}</Alert>}
          {clearLeagueSchedule.isError && <Alert severity="error">{t('admin.leagueSchedules.resetError')}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        {league.hasOverride && (
          <Button onClick={handleReset} color="secondary" disabled={clearLeagueSchedule.isPending}>
            {t('admin.leagueSchedules.reset')}
          </Button>
        )}
        <Button onClick={onClose}>{t('common.cancel')}</Button>
        <Button onClick={handleSave} variant="contained" disabled={weekdaysMask === 0 || setLeagueSchedule.isPending}>
          {t('admin.leagueSchedules.save')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
