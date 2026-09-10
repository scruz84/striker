import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Chip from '@mui/material/Chip'
import Dialog from '@mui/material/Dialog'
import DialogActions from '@mui/material/DialogActions'
import DialogContent from '@mui/material/DialogContent'
import DialogTitle from '@mui/material/DialogTitle'
import FormControl from '@mui/material/FormControl'
import Grid from '@mui/material/Grid'
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
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useAdminAddCountry,
  useAdminAdjustBudget,
  useAdminAdvanceSeason,
  useAdminCountries,
  useAdminLeagues,
  useAdminOverview,
  useAdminRecreateLeagues,
  useAdminSetAiLevel,
  useAdminSimulateNow,
  useAdminUsers,
  useAdminWorldConfig,
  useClearLeagueSchedule,
  useMe,
  useSetLeagueSchedule,
  useSetMatchSchedule,
  useSetTransferWindow,
  useSetUserAdmin,
  useSetWorldConfig,
} from '../../api/hooks'
import {
  AI_LEVEL_CARETAKER,
  AI_LEVEL_DIRECTOR,
  AI_LEVEL_MANAGER,
  ALL_WEEKDAYS_MASK,
  WEEKDAY_OPTIONS,
  type AdminLeagueInfo,
  type CountryStatus,
  type TierConfig,
} from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { formatCurrency } from '../../utils/formatCurrency'

export function AdminPage() {
  const { t } = useTranslation()
  const { data: loggedUser, isLoading: meLoading } = useMe()
  const { data: overview, isLoading: overviewLoading } = useAdminOverview(!!loggedUser)
  const { data: users, isLoading: usersLoading } = useAdminUsers(!!loggedUser)
  const { data: worldConfig, isLoading: worldConfigLoading } = useAdminWorldConfig(!!loggedUser)

  if (meLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (!loggedUser?.isAdmin) {
    return <Typography color="text.secondary">{t('admin.noAccess')}</Typography>
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('admin.title')}
      </Typography>

      {overviewLoading || !overview ? (
        <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
      ) : (
        <>
          <Overview overview={overview} />
          <Actions overview={overview} />
        </>
      )}

      {worldConfigLoading || !worldConfig ? (
        <Skeleton variant="rectangular" height={150} sx={{ mb: 2 }} />
      ) : (
        <WorldStructureSection tiers={worldConfig.tiers} />
      )}

      <CountriesSection />

      <LeagueSchedulesSection />

      <Typography variant="h6" sx={{ mt: 3, mb: 1 }}>
        {t('admin.usersTitle')}
      </Typography>
      {usersLoading || !users ? (
        <Skeleton variant="rectangular" height={200} />
      ) : (
        <UsersTable users={users} loggedUserId={Number(loggedUser.id)} />
      )}
    </Box>
  )
}

function Overview({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const stats: { key: string; value: number }[] = [
    { key: 'season', value: overview.season },
    { key: 'users', value: overview.userCount },
    { key: 'teams', value: overview.teamCount },
    { key: 'leagues', value: overview.leagueCount },
    { key: 'matchesTotal', value: overview.matchesTotal },
    { key: 'matchesUnplayed', value: overview.matchesUnplayed },
    { key: 'activeCups', value: overview.activeCups },
  ]

  return (
    <Grid container spacing={2} sx={{ mb: 2 }}>
      {stats.map((stat) => (
        <Grid key={stat.key} size={{ xs: 6, sm: 4, md: 3 }}>
          <Card>
            <CardContent>
              <Typography variant="overline" color="text.secondary">
                {t(`admin.overview.${stat.key}`)}
              </Typography>
              <Typography variant="h5">{stat.value}</Typography>
            </CardContent>
          </Card>
        </Grid>
      ))}
      <Grid size={{ xs: 6, sm: 4, md: 3 }}>
        <Card>
          <CardContent>
            <Typography variant="overline" color="text.secondary">
              {t('admin.overview.transferWindow')}
            </Typography>
            <Box>
              <Chip
                size="small"
                color={overview.transferWindowOpen ? 'success' : 'default'}
                label={overview.transferWindowOpen ? t('admin.overview.open') : t('admin.overview.closed')}
              />
            </Box>
          </CardContent>
        </Card>
      </Grid>
    </Grid>
  )
}

function Actions({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { transferWindowOpen } = overview
  const simulateNow = useAdminSimulateNow()
  const advanceSeason = useAdminAdvanceSeason()
  const adjustBudget = useAdminAdjustBudget()
  const setTransferWindow = useSetTransferWindow()
  const [teamId, setTeamId] = useState('')
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')

  const handleSimulateNow = async () => {
    const ok = await confirm({ message: t('admin.actions.confirmRunSimulation'), destructive: true })
    if (ok) {
      simulateNow.mutate()
    }
  }

  const handleAdvanceSeason = async () => {
    const ok = await confirm({ message: t('admin.actions.confirmCheckSeasonAdvance'), destructive: true })
    if (ok) {
      advanceSeason.mutate()
    }
  }

  const handleToggleTransferWindow = async () => {
    const ok = await confirm({
      message: transferWindowOpen
        ? t('admin.actions.confirmCloseWindow')
        : t('admin.actions.confirmOpenWindow'),
      destructive: transferWindowOpen,
    })
    if (ok) {
      setTransferWindow.mutate(!transferWindowOpen)
    }
  }

  const handleAdjustBudget = async (event: FormEvent) => {
    event.preventDefault()
    const teamIdNum = Number(teamId)
    const amountNum = Number(amount)
    if (!teamIdNum || !amountNum || !description) {
      return
    }
    const ok = await confirm({
      message: t('admin.adjustBudget.confirm', { teamId: teamIdNum, amount: formatCurrency(amountNum) }),
      confirmLabel: t('admin.adjustBudget.apply'),
      destructive: true,
    })
    if (!ok) {
      return
    }
    adjustBudget.mutate(
      { teamId: teamIdNum, amount: amountNum, description },
      { onSuccess: () => setDescription('') },
    )
  }

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Stack spacing={2}>
        <Box>
          <Typography variant="subtitle1" gutterBottom>
            {t('admin.actions.title')}
          </Typography>
          <Stack direction="row" spacing={1}>
            <Button variant="outlined" onClick={handleSimulateNow} disabled={simulateNow.isPending}>
              {t('admin.actions.runSimulation')}
            </Button>
            <Button variant="outlined" onClick={handleAdvanceSeason} disabled={advanceSeason.isPending}>
              {t('admin.actions.checkSeasonAdvance')}
            </Button>
            <Button
              variant="outlined"
              color={transferWindowOpen ? 'warning' : 'success'}
              onClick={handleToggleTransferWindow}
              disabled={setTransferWindow.isPending}
            >
              {transferWindowOpen ? t('admin.actions.closeWindow') : t('admin.actions.openWindow')}
            </Button>
          </Stack>
          {simulateNow.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.actions.simSuccess')}</Alert>}
          {advanceSeason.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.actions.seasonSuccess')}</Alert>}
          {setTransferWindow.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.actions.windowSuccess')}</Alert>}
        </Box>

        <Box component="form" onSubmit={handleAdjustBudget}>
          <Typography variant="subtitle1" gutterBottom>
            {t('admin.adjustBudget.title')}
          </Typography>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
            <TextField
              label={t('admin.adjustBudget.teamId')}
              size="small"
              value={teamId}
              onChange={(e) => setTeamId(e.target.value)}
              sx={{ width: 100 }}
            />
            <TextField
              label={t('admin.adjustBudget.amount')}
              size="small"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              helperText={t('admin.adjustBudget.amountHelper')}
              sx={{ width: 140 }}
            />
            <TextField
              label={t('admin.adjustBudget.description')}
              size="small"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              sx={{ width: 260 }}
            />
            <Button type="submit" variant="contained" disabled={adjustBudget.isPending}>
              {t('admin.adjustBudget.apply')}
            </Button>
          </Stack>
          {adjustBudget.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.adjustBudget.success')}</Alert>}
          {adjustBudget.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.adjustBudget.error')}</Alert>}
        </Box>

        <SetAiLevelForm />

        <MatchScheduleForm overview={overview} />
      </Stack>
    </Paper>
  )
}

const AI_LEVEL_OPTIONS = [
  { value: AI_LEVEL_CARETAKER, labelKey: 'admin.setAiLevel.caretaker' },
  { value: AI_LEVEL_MANAGER, labelKey: 'admin.setAiLevel.manager' },
  { value: AI_LEVEL_DIRECTOR, labelKey: 'admin.setAiLevel.director' },
]

function SetAiLevelForm() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const setAiLevel = useAdminSetAiLevel()
  const [teamId, setTeamId] = useState('')
  const [aiLevel, setAiLevelValue] = useState<number>(AI_LEVEL_CARETAKER)

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    const teamIdNum = Number(teamId)
    if (!teamIdNum) {
      return
    }
    const ok = await confirm({
      message: t('admin.setAiLevel.confirm', { teamId: teamIdNum, level: aiLevel }),
      confirmLabel: t('admin.setAiLevel.apply'),
    })
    if (!ok) {
      return
    }
    setAiLevel.mutate({ teamId: teamIdNum, aiLevel })
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.setAiLevel.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.setAiLevel.description')}
      </Typography>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <TextField
          label={t('admin.setAiLevel.teamId')}
          size="small"
          value={teamId}
          onChange={(e) => setTeamId(e.target.value)}
          sx={{ width: 100 }}
        />
        <FormControl size="small" sx={{ width: 160 }}>
          <InputLabel id="ai-level-select-label">{t('admin.setAiLevel.level')}</InputLabel>
          <Select
            labelId="ai-level-select-label"
            label={t('admin.setAiLevel.level')}
            value={aiLevel}
            onChange={(e) => setAiLevelValue(Number(e.target.value))}
          >
            {AI_LEVEL_OPTIONS.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {t(option.labelKey)}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button type="submit" variant="contained" disabled={setAiLevel.isPending}>
          {t('admin.setAiLevel.apply')}
        </Button>
      </Stack>
      {setAiLevel.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.setAiLevel.success')}</Alert>}
      {setAiLevel.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.setAiLevel.error')}</Alert>}
    </Box>
  )
}

function MatchScheduleForm({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const setMatchSchedule = useSetMatchSchedule()
  const [weekdaysMask, setWeekdaysMask] = useState(overview.matchScheduleWeekdaysMask)
  const [time, setTime] = useState(
    `${String(overview.matchScheduleHour).padStart(2, '0')}:${String(overview.matchScheduleMinute).padStart(2, '0')}`,
  )

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (weekdaysMask === 0 || !time) {
      return
    }
    const [hourStr, minuteStr] = time.split(':')
    setMatchSchedule.mutate({ weekdaysMask, hour: Number(hourStr), minute: Number(minuteStr) })
  }

  return (
    <Box component="form" onSubmit={handleSubmit}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.matchSchedule.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {t('admin.matchSchedule.description')}
      </Typography>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <ToggleButtonGroup
          size="small"
          value={WEEKDAY_OPTIONS.filter((d) => (weekdaysMask & d.bit) !== 0).map((d) => d.bit)}
          onChange={(_event, newBits: number[]) => setWeekdaysMask(newBits.reduce((mask, bit) => mask | bit, 0))}
        >
          {WEEKDAY_OPTIONS.map((d) => (
            <ToggleButton key={d.bit} value={d.bit}>
              {t(`common.weekdaysShort.${d.key}`)}
            </ToggleButton>
          ))}
        </ToggleButtonGroup>
        <TextField
          label={t('admin.matchSchedule.time')}
          type="time"
          size="small"
          value={time}
          onChange={(e) => setTime(e.target.value)}
          sx={{ width: 140 }}
          slotProps={{ inputLabel: { shrink: true } }}
        />
        <Button type="submit" variant="contained" disabled={weekdaysMask === 0 || setMatchSchedule.isPending}>
          {t('admin.matchSchedule.save')}
        </Button>
      </Stack>
      {weekdaysMask === 0 && (
        <Typography variant="caption" color="error">
          {t('admin.matchSchedule.noWeekdaysError')}
        </Typography>
      )}
      {setMatchSchedule.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.matchSchedule.success')}</Alert>}
      {setMatchSchedule.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.matchSchedule.error')}</Alert>}
    </Box>
  )
}

function WorldStructureSection({ tiers: initialTiers }: { tiers: TierConfig[] }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const [tiers, setTiers] = useState<TierConfig[]>(initialTiers)
  const setWorldConfig = useSetWorldConfig()
  const recreateLeagues = useAdminRecreateLeagues()

  const updateTier = (index: number, patch: Partial<TierConfig>) =>
    setTiers((prev) => prev.map((tier, i) => (i === index ? { ...tier, ...patch } : tier)))

  const addTier = () => {
    const nextTierNumber = tiers.length > 0 ? Math.max(...tiers.map((tier) => tier.tier)) + 1 : 1
    setTiers((prev) => [...prev, { tier: nextTierNumber, name: String(nextTierNumber), groups: 1, teamsPerLeague: 10 }])
  }

  const removeTier = (index: number) => setTiers((prev) => prev.filter((_, i) => i !== index))

  const handleSave = () => setWorldConfig.mutate(tiers)

  const handleRecreate = async () => {
    const ok = await confirm({
      message: t('admin.worldStructure.confirmRecreate'),
      confirmLabel: t('admin.worldStructure.recreateButton'),
      destructive: true,
    })
    if (ok) {
      recreateLeagues.mutate()
    }
  }

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.worldStructure.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('admin.worldStructure.description')}
      </Typography>
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
      <Stack direction="row" spacing={1} sx={{ mt: 2 }}>
        <Button size="small" onClick={addTier}>
          {t('admin.worldStructure.addTier')}
        </Button>
        <Button size="small" variant="contained" onClick={handleSave} disabled={setWorldConfig.isPending}>
          {t('admin.worldStructure.save')}
        </Button>
      </Stack>
      {setWorldConfig.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.worldStructure.saveSuccess')}</Alert>}
      {setWorldConfig.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.worldStructure.saveError')}</Alert>}

      <Box sx={{ mt: 3, pt: 2, borderTop: 1, borderColor: 'divider' }}>
        <Typography variant="subtitle2" color="error" gutterBottom>
          {t('admin.worldStructure.recreateTitle')}
        </Typography>
        <Button variant="outlined" color="error" onClick={handleRecreate} disabled={recreateLeagues.isPending}>
          {t('admin.worldStructure.recreateButton')}
        </Button>
        {recreateLeagues.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('admin.worldStructure.recreateSuccess')}</Alert>}
        {recreateLeagues.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.worldStructure.recreateError')}</Alert>}
      </Box>
    </Paper>
  )
}

function CountriesSection() {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: countries, isLoading } = useAdminCountries(true)
  const addCountry = useAdminAddCountry()

  const handleAdd = async (country: CountryStatus) => {
    const ok = await confirm({
      message: t('admin.countries.confirmAdd', { name: country.name }),
      confirmLabel: t('admin.countries.add'),
    })
    if (ok) {
      addCountry.mutate(country.code)
    }
  }

  if (isLoading || !countries) {
    return <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
  }

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.countries.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('admin.countries.description')}
      </Typography>
      <Stack spacing={1}>
        {countries.map((country) => (
          <Stack key={country.code} direction="row" spacing={2} sx={{ alignItems: 'center' }}>
            <Typography sx={{ width: 160 }}>
              {country.name} ({country.code})
            </Typography>
            {country.active ? (
              <Chip size="small" color="success" label={t('admin.countries.active')} />
            ) : (
              <Button size="small" variant="outlined" onClick={() => handleAdd(country)} disabled={addCountry.isPending}>
                {t('admin.countries.add')}
              </Button>
            )}
          </Stack>
        ))}
      </Stack>
      {addCountry.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('admin.countries.addError')}</Alert>}
    </Paper>
  )
}

function describeSchedule(t: ReturnType<typeof useTranslation>['t'], schedule: { weekdaysMask: number; hour: number; minute: number }) {
  const dayLabel =
    schedule.weekdaysMask === ALL_WEEKDAYS_MASK
      ? t('admin.leagueSchedules.everyDay')
      : WEEKDAY_OPTIONS.filter((d) => (schedule.weekdaysMask & d.bit) !== 0)
          .map((d) => t(`common.weekdaysShort.${d.key}`))
          .join(', ')
  const time = `${String(schedule.hour).padStart(2, '0')}:${String(schedule.minute).padStart(2, '0')}`
  return `${dayLabel} ${time}`
}

function LeagueSchedulesSection() {
  const { t } = useTranslation()
  const { data: leagues, isLoading } = useAdminLeagues(true)
  const [countryFilter, setCountryFilter] = useState('')
  const [editingLeague, setEditingLeague] = useState<AdminLeagueInfo | null>(null)

  if (isLoading || !leagues) {
    return <Skeleton variant="rectangular" height={200} sx={{ mb: 2 }} />
  }

  const countries = Array.from(new Set(leagues.map((l) => l.country))).sort()
  const filteredLeagues = countryFilter ? leagues.filter((l) => l.country === countryFilter) : leagues

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('admin.leagueSchedules.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('admin.leagueSchedules.description')}
      </Typography>
      <FormControl size="small" sx={{ mb: 2, minWidth: 180 }}>
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
                  {league.hasOverride && (
                    <Chip size="small" label={t('admin.leagueSchedules.override')} sx={{ ml: 1 }} />
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
      {editingLeague && <LeagueScheduleDialog league={editingLeague} onClose={() => setEditingLeague(null)} />}
    </Paper>
  )
}

function LeagueScheduleDialog({ league, onClose }: { league: AdminLeagueInfo; onClose: () => void }) {
  const { t } = useTranslation()
  const setLeagueSchedule = useSetLeagueSchedule()
  const clearLeagueSchedule = useClearLeagueSchedule()
  const [weekdaysMask, setWeekdaysMask] = useState(league.schedule.weekdaysMask)
  const [time, setTime] = useState(
    `${String(league.schedule.hour).padStart(2, '0')}:${String(league.schedule.minute).padStart(2, '0')}`,
  )

  const handleSave = () => {
    if (weekdaysMask === 0 || !time) {
      return
    }
    const [hourStr, minuteStr] = time.split(':')
    setLeagueSchedule.mutate(
      { leagueId: league.leagueId, schedule: { weekdaysMask, hour: Number(hourStr), minute: Number(minuteStr) } },
      { onSuccess: onClose },
    )
  }

  const handleReset = () => {
    clearLeagueSchedule.mutate(league.leagueId, { onSuccess: onClose })
  }

  return (
    <Dialog open onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>
        {t('admin.leagueSchedules.dialogTitle', { league: `${league.country.toUpperCase()} ${league.name}` })}
      </DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <ToggleButtonGroup
            size="small"
            value={WEEKDAY_OPTIONS.filter((d) => (weekdaysMask & d.bit) !== 0).map((d) => d.bit)}
            onChange={(_event, newBits: number[]) => setWeekdaysMask(newBits.reduce((mask, bit) => mask | bit, 0))}
          >
            {WEEKDAY_OPTIONS.map((d) => (
              <ToggleButton key={d.bit} value={d.bit}>
                {t(`common.weekdaysShort.${d.key}`)}
              </ToggleButton>
            ))}
          </ToggleButtonGroup>
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

function UsersTable({
  users,
  loggedUserId,
}: {
  users: NonNullable<ReturnType<typeof useAdminUsers>['data']>
  loggedUserId: number
}) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const setUserAdmin = useSetUserAdmin()

  const handleToggleAdmin = async (user: (typeof users)[number]) => {
    const makeAdmin = !user.isAdmin
    const ok = await confirm({
      message: t(makeAdmin ? 'admin.usersTable.confirmGrantAdmin' : 'admin.usersTable.confirmRevokeAdmin', { name: user.name }),
      confirmLabel: t(makeAdmin ? 'admin.usersTable.grantAdmin' : 'admin.usersTable.revokeAdmin'),
      destructive: !makeAdmin,
    })
    if (ok) {
      setUserAdmin.mutate({ userId: user.userId, isAdmin: makeAdmin })
    }
  }

  return (
    <TableContainer component={Paper}>
      <Table size="small" aria-label="Users">
        <TableHead>
          <TableRow>
            <TableCell>{t('admin.usersTable.id')}</TableCell>
            <TableCell>{t('admin.usersTable.name')}</TableCell>
            <TableCell>{t('admin.usersTable.email')}</TableCell>
            <TableCell>{t('admin.usersTable.country')}</TableCell>
            <TableCell>{t('admin.usersTable.team')}</TableCell>
            <TableCell>{t('admin.usersTable.admin')}</TableCell>
            <TableCell align="right" />
          </TableRow>
        </TableHead>
        <TableBody>
          {users.map((user) => (
            <TableRow key={user.userId}>
              <TableCell>{user.userId}</TableCell>
              <TableCell>{user.name}</TableCell>
              <TableCell>{user.email}</TableCell>
              <TableCell>{user.country}</TableCell>
              <TableCell>{user.teamName || '-'}</TableCell>
              <TableCell>{user.isAdmin && <Chip size="small" color="primary" label={t('admin.usersTable.admin')} />}</TableCell>
              <TableCell align="right">
                <Button
                  size="small"
                  color={user.isAdmin ? 'secondary' : 'primary'}
                  onClick={() => handleToggleAdmin(user)}
                  disabled={setUserAdmin.isPending || user.userId === loggedUserId}
                >
                  {user.isAdmin ? t('admin.usersTable.revokeAdmin') : t('admin.usersTable.grantAdmin')}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}
