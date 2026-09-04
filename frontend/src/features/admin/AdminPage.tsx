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
  const { data: loggedUser, isLoading: meLoading } = useMe()
  const { data: overview, isLoading: overviewLoading } = useAdminOverview(!!loggedUser)
  const { data: users, isLoading: usersLoading } = useAdminUsers(!!loggedUser)

  if (meLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (!loggedUser?.isAdmin) {
    return <Typography color="text.secondary">You don't have admin access.</Typography>
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        Admin
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
        Users
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
  const stats: { label: string; value: number }[] = [
    { label: 'Season', value: overview.season },
    { label: 'Users', value: overview.userCount },
    { label: 'Teams', value: overview.teamCount },
    { label: 'Leagues', value: overview.leagueCount },
    { label: 'Matches this season', value: overview.matchesTotal },
    { label: 'Matches unplayed', value: overview.matchesUnplayed },
    { label: 'Active cups', value: overview.activeCups },
  ]

  return (
    <Grid container spacing={2} sx={{ mb: 2 }}>
      {stats.map((stat) => (
        <Grid key={stat.label} size={{ xs: 6, sm: 4, md: 3 }}>
          <Card>
            <CardContent>
              <Typography variant="overline" color="text.secondary">
                {stat.label}
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
              Transfer window
            </Typography>
            <Box>
              <Chip
                size="small"
                color={overview.transferWindowOpen ? 'success' : 'default'}
                label={overview.transferWindowOpen ? 'Open' : 'Closed'}
              />
            </Box>
          </CardContent>
        </Card>
      </Grid>
    </Grid>
  )
}

function Actions({ transferWindowOpen }: { transferWindowOpen: boolean }) {
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
            Manual triggers
          </Typography>
          <Stack direction="row" spacing={1}>
            <Button variant="outlined" onClick={() => simulateNow.mutate()} disabled={simulateNow.isPending}>
              Run simulation now
            </Button>
            <Button variant="outlined" onClick={() => advanceSeason.mutate()} disabled={advanceSeason.isPending}>
              Check season advance
            </Button>
            <Button
              variant="outlined"
              color={transferWindowOpen ? 'warning' : 'success'}
              onClick={() => setTransferWindow.mutate(!transferWindowOpen)}
              disabled={setTransferWindow.isPending}
            >
              {transferWindowOpen ? 'Close transfer window' : 'Open transfer window'}
            </Button>
          </Stack>
          {simulateNow.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>Simulation pass triggered.</Alert>}
          {advanceSeason.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>Season advance check triggered.</Alert>}
          {setTransferWindow.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>Transfer window updated.</Alert>}
        </Box>

        <Box component="form" onSubmit={handleAdjustBudget}>
          <Typography variant="subtitle1" gutterBottom>
            Adjust team budget
          </Typography>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
            <TextField
              label="Team id"
              size="small"
              value={teamId}
              onChange={(e) => setTeamId(e.target.value)}
              sx={{ width: 100 }}
            />
            <TextField
              label="Amount"
              size="small"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              helperText="negative to deduct"
              sx={{ width: 140 }}
            />
            <TextField
              label="Description"
              size="small"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              sx={{ width: 260 }}
            />
            <Button type="submit" variant="contained" disabled={adjustBudget.isPending}>
              Apply
            </Button>
          </Stack>
          {adjustBudget.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>Budget adjusted.</Alert>}
          {adjustBudget.isError && <Alert severity="error" sx={{ mt: 1 }}>Failed to adjust budget.</Alert>}
        </Box>
      </Stack>
    </Paper>
  )
}

function UsersTable({ users }: { users: NonNullable<ReturnType<typeof useAdminUsers>['data']> }) {
  return (
    <TableContainer component={Paper}>
      <Table size="small" aria-label="Users">
        <TableHead>
          <TableRow>
            <TableCell>Id</TableCell>
            <TableCell>Name</TableCell>
            <TableCell>Email</TableCell>
            <TableCell>Country</TableCell>
            <TableCell>Team</TableCell>
            <TableCell>Admin</TableCell>
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
              <TableCell>{user.isAdmin && <Chip size="small" color="primary" label="Admin" />}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}
