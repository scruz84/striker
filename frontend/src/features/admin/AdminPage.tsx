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
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useAdminAdjustBudget,
  useAdminAdvanceSeason,
  useAdminOverview,
  useAdminSimulateNow,
  useAdminUsers,
  useMe,
  useSetTransferWindow,
} from '../../api/hooks'

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
          <Actions transferWindowOpen={overview.transferWindowOpen} />
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

function Actions({ transferWindowOpen }: { transferWindowOpen: boolean }) {
  const { t } = useTranslation()
  const simulateNow = useAdminSimulateNow()
  const advanceSeason = useAdminAdvanceSeason()
  const adjustBudget = useAdminAdjustBudget()
  const setTransferWindow = useSetTransferWindow()
  const [teamId, setTeamId] = useState('')
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')

  const handleAdjustBudget = (event: FormEvent) => {
    event.preventDefault()
    const teamIdNum = Number(teamId)
    const amountNum = Number(amount)
    if (!teamIdNum || !amountNum || !description) {
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
            <Button variant="outlined" onClick={() => simulateNow.mutate()} disabled={simulateNow.isPending}>
              {t('admin.actions.runSimulation')}
            </Button>
            <Button variant="outlined" onClick={() => advanceSeason.mutate()} disabled={advanceSeason.isPending}>
              {t('admin.actions.checkSeasonAdvance')}
            </Button>
            <Button
              variant="outlined"
              color={transferWindowOpen ? 'warning' : 'success'}
              onClick={() => setTransferWindow.mutate(!transferWindowOpen)}
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
      </Stack>
    </Paper>
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
