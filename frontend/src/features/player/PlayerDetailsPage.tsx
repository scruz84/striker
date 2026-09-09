import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Link from '@mui/material/Link'
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
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePlayerDetails } from '../../api/hooks'
import { POSITION_LABEL_KEYS, TRANSACTION_REASON_KEYS, UNAVAILABLE_INJURY, type PlayerDetails } from '../../api/types'
import { PlayerCard } from '../../components/PlayerCard'
import { TeamBadge } from '../../components/TeamBadge'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

export function PlayerDetailsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: player, isLoading } = usePlayerDetails(id)

  if (isLoading || !player) {
    return <Skeleton variant="rectangular" height={300} />
  }

  const isFreeAgent = player.teamId === 0

  return (
    <Stack spacing={2}>
      <Box>
        <Typography variant="h4" gutterBottom>
          {player.firstName} {player.lastName}
        </Typography>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', mb: isFreeAgent ? 0 : 1 }}>
          <Chip size="small" variant="outlined" label={t(POSITION_LABEL_KEYS[player.bestPosition])} />
          {isFreeAgent && <Chip size="small" label={t('playerPage.freeAgent')} />}
        </Stack>
        {!isFreeAgent && (
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
            <TeamBadge teamId={player.teamId} teamName={player.teamName} size={24} />
            <Link component={RouterLink} to={`/team/${player.teamId}`} underline="hover">
              {player.teamName}
            </Link>
            <Typography color="text.secondary">&middot;</Typography>
            <Link component={RouterLink} to={`/division/${player.divisionId}`} underline="hover" color="text.secondary">
              {player.divisionName}
            </Link>
          </Stack>
        )}
      </Box>

      <PlayerStatusBanner player={player} />

      <PlayerCard player={player} />

      <Grid container spacing={2}>
        <Grid size={{ xs: 6, sm: 4 }}>
          <StatCard label={t('playerPage.stats.goals')} value={player.goals} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4 }}>
          <StatCard label={t('playerPage.stats.yellowCards')} value={player.yellowCards} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4 }}>
          <StatCard label={t('playerPage.stats.redCards')} value={player.redCards} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4 }}>
          <StatCard
            label={t('playerPage.stats.contract')}
            value={t('team.contractSeasons', { count: player.contractYearsRemaining })}
          />
        </Grid>
        <Grid size={{ xs: 6, sm: 4 }}>
          <StatCard label={t('playerPage.stats.wage')} value={formatCurrency(player.wage)} />
        </Grid>
      </Grid>

      <Box>
        <Typography variant="h6" sx={{ mb: 1 }}>
          {t('playerPage.history.title')}
        </Typography>
        <TableContainer component={Paper}>
          <Table size="small" aria-label="Transfer history">
            <TableHead>
              <TableRow>
                <TableCell>{t('finances.table.date')}</TableCell>
                <TableCell>{t('finances.table.reason')}</TableCell>
                <TableCell>{t('finances.table.description')}</TableCell>
                <TableCell align="right">{t('finances.table.amount')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {player.history.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4}>
                    <Typography variant="body2" color="text.secondary">
                      {t('playerPage.history.none')}
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
              {player.history.map((tx) => (
                <TableRow key={tx.id}>
                  <TableCell>{formatDateTime(tx.createdAt)}</TableCell>
                  <TableCell>{t(TRANSACTION_REASON_KEYS[tx.reason] ?? 'finances.reasons.other')}</TableCell>
                  <TableCell>{tx.description}</TableCell>
                  <TableCell align="right" sx={{ color: tx.amount < 0 ? 'error.main' : 'success.main' }}>
                    {formatCurrency(tx.amount)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Box>
    </Stack>
  )
}

function StatCard({ label, value }: { label: string; value: number | string }) {
  return (
    <Card variant="outlined">
      <CardContent>
        <Typography variant="overline" color="text.secondary">
          {label}
        </Typography>
        <Typography variant="h5">{value}</Typography>
      </CardContent>
    </Card>
  )
}

// One Alert per currently-true status - a player can be e.g. injured AND listed for transfer at
// the same time, so these are independent rather than mutually exclusive.
function PlayerStatusBanner({ player }: { player: PlayerDetails }) {
  const { t } = useTranslation()
  const alerts: { key: string; severity: 'warning' | 'info'; text: string }[] = []

  if (player.unavailableReason !== undefined) {
    const key = player.unavailableReason === UNAVAILABLE_INJURY ? 'playerPage.status.injured' : 'playerPage.status.suspended'
    alerts.push({ key: 'unavailable', severity: 'warning', text: t(key, { count: player.unavailableMatchesRemaining }) })
  }
  if (player.onLoanFromTeamName) {
    alerts.push({
      key: 'loan',
      severity: 'info',
      text: t('playerPage.status.onLoan', { team: player.onLoanFromTeamName, season: player.onLoanReturnSeason }),
    })
  }
  if (player.transferListingAskingPrice !== undefined) {
    alerts.push({
      key: 'listedTransfer',
      severity: 'info',
      text: t('playerPage.status.listedForTransfer', { price: formatCurrency(player.transferListingAskingPrice) }),
    })
  }
  if (player.loanListingDurationSeasons !== undefined) {
    alerts.push({
      key: 'listedLoan',
      severity: 'info',
      text: t('playerPage.status.listedForLoan', { count: player.loanListingDurationSeasons }),
    })
  }

  if (alerts.length === 0) {
    return null
  }

  return (
    <Stack spacing={1}>
      {alerts.map((a) => (
        <Alert key={a.key} severity={a.severity}>
          {a.text}
        </Alert>
      ))}
    </Stack>
  )
}
