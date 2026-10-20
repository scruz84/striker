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

export function OverviewTab() {
  const { data: overview, isLoading } = useAdminOverview(true)

  if (isLoading || !overview) {
    return <Skeleton variant="rectangular" height={100} sx={{ mb: 2 }} />
  }

  return (
    <>
      <Overview overview={overview} />
      <OperationsSection overview={overview} />
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

  if (seasons.length === 0) {
    return null
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
function OperationsSection({ overview }: { overview: NonNullable<ReturnType<typeof useAdminOverview>['data']> }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const notify = useNotify()
  const feedback = useAdminFeedback()
  const { transferWindowOpen } = overview
  const simulateNow = useAdminSimulateNow()
  const finishStuckMatches = useAdminFinishStuckMatches()
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

  const handleFinishStuckMatches = async () => {
    const ok = await confirm({
      message: t('admin.actions.confirmFinishStuckMatches', { count: overview.stuckMatches }),
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
          onClick={handleFinishStuckMatches}
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
