import { useState } from 'react'
import Alert from '@mui/material/Alert'
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
import { useTranslation } from 'react-i18next'
import {
  useAdminAddCountry,
  useAdminAddDivision,
  useAdminCountries,
  useAdminLeagues,
  useAdminRecreateLeagues,
  useAdminRescheduleCountry,
  useAdminWorldConfig,
  useClearLeagueSchedule,
  useSetLeagueSchedule,
  useSetWorldConfig,
} from '../../api/hooks'
import {
  ALL_WEEKDAYS_MASK,
  WEEKDAY_OPTIONS,
  type AdminLeagueInfo,
  type CountryStatus,
  type TierConfig,
} from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { AdminSection } from './components/AdminSection'
import { DangerZone } from './components/DangerZone'
import { WeekdayPicker } from './components/WeekdayPicker'
import { useAdminFeedback } from './useAdminFeedback'
import { useIsPhone } from './useIsPhone'
import { localHourMinuteToSchedule, scheduleHourMinuteToLocal } from '../../utils/scheduleTime'

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

function WorldStructureSection({ tiers: initialTiers }: { tiers: TierConfig[] }) {
  const { t } = useTranslation()
  const [tiers, setTiers] = useState<TierConfig[]>(initialTiers)
  const feedback = useAdminFeedback()
  const isPhone = useIsPhone()
  const setWorldConfig = useSetWorldConfig()

  const updateTier = (index: number, patch: Partial<TierConfig>) =>
    setTiers((prev) => prev.map((tier, i) => (i === index ? { ...tier, ...patch } : tier)))

  const addTier = () => {
    const nextTierNumber = tiers.length > 0 ? Math.max(...tiers.map((tier) => tier.tier)) + 1 : 1
    setTiers((prev) => [...prev, { tier: nextTierNumber, name: String(nextTierNumber), groups: 1, teamsPerLeague: 10 }])
  }

  const removeTier = (index: number) => setTiers((prev) => prev.filter((_, i) => i !== index))

  const handleSave = () => setWorldConfig.mutate(tiers, feedback('admin.worldStructure.saveSuccess', 'admin.worldStructure.saveError'))

  return (
    <AdminSection title={t('admin.worldStructure.title')} description={t('admin.worldStructure.description')}>
      {isPhone ? (
        <Stack spacing={1.5}>
          {tiers.map((tier, index) => (
            <Paper key={index} variant="outlined" sx={{ p: 1.5 }}>
              <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
                <Typography variant="subtitle2">
                  {t('admin.worldStructure.tier')} {tier.tier}
                </Typography>
                <Button size="small" color="secondary" onClick={() => removeTier(index)}>
                  {t('admin.worldStructure.removeTier')}
                </Button>
              </Stack>
              <Stack spacing={1.5}>
                <TextField
                  label={t('admin.worldStructure.tierName')}
                  size="small"
                  fullWidth
                  value={tier.name}
                  onChange={(e) => updateTier(index, { name: e.target.value })}
                />
                <TextField
                  label={t('admin.worldStructure.groups')}
                  size="small"
                  type="number"
                  fullWidth
                  value={tier.groups}
                  onChange={(e) => updateTier(index, { groups: Number(e.target.value) })}
                />
                <TextField
                  label={t('admin.worldStructure.teamsPerLeague')}
                  size="small"
                  type="number"
                  fullWidth
                  value={tier.teamsPerLeague}
                  onChange={(e) => updateTier(index, { teamsPerLeague: Number(e.target.value) })}
                />
              </Stack>
            </Paper>
          ))}
        </Stack>
      ) : (
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>{t('admin.worldStructure.tier')}</TableCell>
                <TableCell>{t('admin.worldStructure.tierName')}</TableCell>
                <TableCell align="right">{t('admin.worldStructure.groups')}</TableCell>
                <TableCell align="right">{t('admin.worldStructure.teamsPerLeague')}</TableCell>
                <TableCell align="right" />
              </TableRow>
            </TableHead>
            <TableBody>
              {tiers.map((tier, index) => (
                <TableRow key={index}>
                  <TableCell>{tier.tier}</TableCell>
                  <TableCell>
                    <TextField
                      size="small"
                      value={tier.name}
                      onChange={(e) => updateTier(index, { name: e.target.value })}
                      sx={{ width: 80 }}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <TextField
                      size="small"
                      type="number"
                      value={tier.groups}
                      onChange={(e) => updateTier(index, { groups: Number(e.target.value) })}
                      sx={{ width: 90 }}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <TextField
                      size="small"
                      type="number"
                      value={tier.teamsPerLeague}
                      onChange={(e) => updateTier(index, { teamsPerLeague: Number(e.target.value) })}
                      sx={{ width: 90 }}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <Button size="small" color="secondary" onClick={() => removeTier(index)}>
                      {t('admin.worldStructure.removeTier')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mt: 2 }}>
        <Button size="small" onClick={addTier}>
          {t('admin.worldStructure.addTier')}
        </Button>
        <Button size="small" variant="contained" onClick={handleSave} disabled={setWorldConfig.isPending}>
          {t('admin.worldStructure.save')}
        </Button>
      </Stack>
    </AdminSection>
  )
}

function CountriesSection() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: countries, isLoading } = useAdminCountries(true)
  const { data: worldConfig } = useAdminWorldConfig(true)
  const { data: leagues } = useAdminLeagues(true)
  const addCountry = useAdminAddCountry()
  const addDivision = useAdminAddDivision()
  const rescheduleCountry = useAdminRescheduleCountry()
  const feedback = useAdminFeedback()
  const [addingCountry, setAddingCountry] = useState<CountryStatus | null>(null)

  const handleReschedule = async (country: CountryStatus) => {
    const ok = await confirm({
      message: t('admin.countries.confirmReschedule', { country: country.name }),
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
        { country: country.name, division: tier.name },
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
          const missingTiers = (worldConfig?.tiers ?? []).filter(
            (tier) => !existingTiersByCountry.get(country.code)?.has(tier.tier),
          )
          return (
            <Stack key={country.code} spacing={0.5}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5 }}>
                <Typography sx={{ flex: '1 1 10rem', minWidth: 0 }}>
                  {country.name} ({country.code})
                </Typography>
                {country.active ? (
                  <>
                    <Chip size="small" color="success" label={t('admin.countries.active')} />
                    <Button size="small" onClick={() => handleReschedule(country)} disabled={rescheduleCountry.isPending}>
                      {t('admin.countries.reschedule')}
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
    </AdminSection>
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
      <DialogTitle>{t('admin.countries.addDialogTitle', { name: country.name })}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <Typography variant="body2" color="text.secondary">
            {t('admin.countries.addDialogDescription', { name: country.name })}
          </Typography>
          <TextField
            label={t('admin.countries.kickoffLabel')}
            type="time"
            size="small"
            value={time}
            onChange={(e) => setTime(e.target.value)}
            helperText={t('admin.countries.kickoffHelp', { name: country.name })}
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

function describeSchedule(t: ReturnType<typeof useTranslation>['t'], schedule: { weekdaysMask: number; hour: number; minute: number }) {
  const dayLabel =
    schedule.weekdaysMask === ALL_WEEKDAYS_MASK
      ? t('admin.leagueSchedules.everyDay')
      : WEEKDAY_OPTIONS.filter((d) => (schedule.weekdaysMask & d.bit) !== 0)
          .map((d) => t(`common.weekdaysShort.${d.key}`))
          .join(', ')
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
                  {describeSchedule(t, league.schedule)}
                </Typography>
                {league.hasOverride && <Chip size="small" label={t('admin.leagueSchedules.override')} />}
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
                    {describeSchedule(t, league.schedule)}
                    {league.hasOverride && <Chip size="small" label={t('admin.leagueSchedules.override')} sx={{ ml: 1 }} />}
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
