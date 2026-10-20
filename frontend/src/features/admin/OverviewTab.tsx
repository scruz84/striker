import Box from '@mui/material/Box'
import Alert from '@mui/material/Alert'
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
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useAdminAdvanceSeason,
  useAdminFinishStuckMatches,
  useAdminInjectFreeAgentsNow,
  useAdminOverview,
  useAdminPayWagesNow,
  useAdminSimulateNow,
  useSetTransferWindow,
} from '../../api/hooks'

import { useConfirm } from '../../components/confirmDialogContext'
import { useNotify } from '../../components/notifyContext'
import { AdminSection } from './components/AdminSection'
import { useAdminFeedback } from './useAdminFeedback'
import { useIsPhone } from './useIsPhone'

export function OverviewTab() {
  const { data: overview, isLoading } = useAdminOverview(true)

  if (isLoading || !overview) {
    return <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
  }

  return <OverviewContent overview={overview} />
}

type Overview = NonNullable<ReturnType<typeof useAdminOverview>['data']>

interface FinishStuckMatches {
  run: () => void
  isPending: boolean
}

// The "finish stuck matches" action is offered twice - as a warning at the top of the tab, where an
// admin who just arrived sees it, and among the other operations - so it is built once here and
// shared, keeping a single request in flight and a single confirmation.
function useFinishStuckMatches(stuckMatches: number): FinishStuckMatches {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const notify = useNotify()
  const finishStuckMatches = useAdminFinishStuckMatches()

  const run = async () => {
    const ok = await confirm({
      message: t('admin.actions.confirmFinishStuckMatches', { count: stuckMatches }),
      destructive: true,
    })
    if (ok) {
      finishStuckMatches.mutate(undefined, {
        onSuccess: (result) =>
          notify.success(
            t('admin.actions.finishStuckMatchesSuccess', {
              leagueMatches: result.leagueMatches,
              friendlies: result.friendlies,
            }),
          ),
        onError: () => notify.error(t('admin.genericError')),
      })
    }
  }

  return { run, isPending: finishStuckMatches.isPending }
}

function OverviewContent({ overview }: { overview: Overview }) {
  const { t } = useTranslation()
  const finishStuckMatches = useFinishStuckMatches(overview.stuckMatches)

  return (
    <>
      {overview.stuckMatches > 0 && (
        <Alert
          severity="warning"
          sx={{ mb: 2 }}
          action={
            <Button color="inherit" size="small" onClick={finishStuckMatches.run} disabled={finishStuckMatches.isPending}>
              {t('admin.overview.stuckMatchesAction')}
            </Button>
          }
        >
          {t('admin.overview.stuckMatchesAlert', { count: overview.stuckMatches })}
        </Alert>
      )}
      <Overview overview={overview} />
      <OperationsSection overview={overview} finishStuckMatches={finishStuckMatches} />
    </>
  )
}

function Overview({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const stats: { key: string; value: number }[] = [
    { key: 'users', value: overview.userCount },
    { key: 'teams', value: overview.teamCount },
    { key: 'leagues', value: overview.leagueCount },
    { key: 'stuckMatches', value: overview.stuckMatches },
    { key: 'activeCups', value: overview.activeCups },
  ]

  return (
    <>
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
      <SeasonsByCountryTable seasons={overview.seasons} />
    </>
  )
}

// SeasonsByCountryTable shows each country's own current season and match progress - current_
// season now advances per country rather than globally (see api/types.ts's CountrySeason), so
// there's no single season number left to show as one of the stat tiles above.
function SeasonsByCountryTable({ seasons }: { seasons: NonNullable<ReturnType<typeof useAdminOverview>['data']>['seasons'] }) {
  const { t } = useTranslation()

  const isPhone = useIsPhone()

  if (seasons.length === 0) {
    return null
  }

  // On a phone each country is a small card - four columns with long headers don't fit 360px.
  if (isPhone) {
    return (
      <Stack spacing={1} sx={{ mb: 2 }}>
        {seasons.map((countrySeason) => (
          <Paper key={countrySeason.country} variant="outlined" sx={{ p: 1.5 }}>
            <Stack direction="row" sx={{ alignItems: 'baseline', justifyContent: 'space-between' }}>
              <Typography sx={{ fontWeight: 600 }}>{countrySeason.country.toUpperCase()}</Typography>
              <Typography variant="body2" color="text.secondary">
                {t('admin.overview.season')} {countrySeason.season}
              </Typography>
            </Stack>
            <Typography variant="body2">
              {t('admin.overview.matchesUnplayed')}: {countrySeason.matchesUnplayed} / {countrySeason.matchesTotal}
            </Typography>
          </Paper>
        ))}
      </Stack>
    )
  }

  return (
    <TableContainer component={Paper} sx={{ mb: 2 }}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t('admin.overview.country')}</TableCell>
            <TableCell>{t('admin.overview.season')}</TableCell>
            <TableCell>{t('admin.overview.matchesTotal')}</TableCell>
            <TableCell>{t('admin.overview.matchesUnplayed')}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {seasons.map((countrySeason) => (
            <TableRow key={countrySeason.country}>
              <TableCell>{countrySeason.country.toUpperCase()}</TableCell>
              <TableCell>{countrySeason.season}</TableCell>
              <TableCell>{countrySeason.matchesTotal}</TableCell>
              <TableCell>{countrySeason.matchesUnplayed}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

function OperationsSection({ overview, finishStuckMatches }: { overview: Overview; finishStuckMatches: FinishStuckMatches }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const feedback = useAdminFeedback()
  const { transferWindowOpen } = overview
  const simulateNow = useAdminSimulateNow()
  const advanceSeason = useAdminAdvanceSeason()
  const payWagesNow = useAdminPayWagesNow()
  const injectFreeAgentsNow = useAdminInjectFreeAgentsNow()
  const setTransferWindow = useSetTransferWindow()

  const handleSimulateNow = async () => {
    const ok = await confirm({ message: t('admin.actions.confirmRunSimulation'), destructive: true })
    if (ok) {
      simulateNow.mutate(undefined, feedback('admin.actions.simSuccess'))
    }
  }

  const handleAdvanceSeason = async () => {
    const ok = await confirm({ message: t('admin.actions.confirmCheckSeasonAdvance'), destructive: true })
    if (ok) {
      advanceSeason.mutate(undefined, feedback('admin.actions.seasonSuccess'))
    }
  }

  const handlePayWagesNow = async () => {
    const ok = await confirm({ message: t('admin.actions.confirmPayWages'), destructive: true })
    if (ok) {
      payWagesNow.mutate(undefined, feedback('admin.actions.payWagesSuccess'))
    }
  }

  const handleInjectFreeAgentsNow = async () => {
    const ok = await confirm({
      message: t('admin.actions.confirmInjectFreeAgents', { count: overview.freeAgentInjectionsPerWeek }),
    })
    if (ok) {
      injectFreeAgentsNow.mutate(undefined, feedback('admin.actions.injectFreeAgentsSuccess'))
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
      setTransferWindow.mutate(!transferWindowOpen, feedback('admin.actions.windowSuccess'))
    }
  }

  return (
    <AdminSection title={t('admin.actions.title')}>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ flexWrap: 'wrap', rowGap: 1 }}>
        <Button variant="outlined" onClick={handleSimulateNow} disabled={simulateNow.isPending}>
          {t('admin.actions.runSimulation')}
        </Button>
        <Button
          variant="outlined"
          color={overview.stuckMatches > 0 ? 'warning' : 'inherit'}
          onClick={finishStuckMatches.run}
          disabled={finishStuckMatches.isPending || overview.stuckMatches === 0}
        >
          {t('admin.actions.finishStuckMatches', { count: overview.stuckMatches })}
        </Button>
        <Button variant="outlined" onClick={handleAdvanceSeason} disabled={advanceSeason.isPending}>
          {t('admin.actions.checkSeasonAdvance')}
        </Button>
        <Button variant="outlined" onClick={handlePayWagesNow} disabled={payWagesNow.isPending}>
          {t('admin.actions.payWagesNow')}
        </Button>
        <Button variant="outlined" onClick={handleInjectFreeAgentsNow} disabled={injectFreeAgentsNow.isPending}>
          {t('admin.actions.injectFreeAgentsNow')}
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
    </AdminSection>
  )
}
