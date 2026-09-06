import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
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
  useAdminAdjustBudget,
  useAdminAdvanceSeason,
  useAdminOverview,
  useAdminSimulateNow,
  useAdminUsers,
  useMe,
  useSetMatchSchedule,
  useSetTransferWindow,
} from '../../api/hooks'
import { WEEKDAY_OPTIONS } from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { formatCurrency } from '../../utils/formatCurrency'

export function AdminPage() {
  const { t } = useTranslation()
  const { data: loggedUser, isLoading: meLoading } = useMe()
  const { data: overview, isLoading: overviewLoading } = useAdminOverview(!!loggedUser)
  const { data: users, isLoading: usersLoading } = useAdminUsers(!!loggedUser)

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

      <Typography variant="h6" sx={{ mt: 3, mb: 1 }}>
        {t('admin.usersTitle')}
      </Typography>
      {usersLoading || !users ? (
        <Skeleton variant="rectangular" height={200} />
      ) : (
        <UsersTable users={users} />
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

        <MatchScheduleForm overview={overview} />
      </Stack>
    </Paper>
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

function UsersTable({ users }: { users: NonNullable<ReturnType<typeof useAdminUsers>['data']> }) {
  const { t } = useTranslation()
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
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}
