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
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePlayerDetails, useUserTeam } from '../../api/hooks'
import { TRANSACTION_REASON_KEYS } from '../../api/types'
import { PlayerAttributesPanel } from '../../components/PlayerAttributesPanel'
import { PlayerProfileRail } from '../../components/PlayerProfileRail'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

export function PlayerDetailsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: player, isLoading } = usePlayerDetails(id)
  const { data: userTeam } = useUserTeam(true)

  if (isLoading || !player) {
    return <Skeleton variant="rectangular" height={300} />
  }

  // A rostered player's wage, contract, and financial history are the owning manager's business
  // - the backend already redacts them for anyone else (see common.Player.RedactFinancials), so
  // this only decides whether to render the sections at all. A free agent (teamId 0) has no
  // current team to protect that information for, same reasoning as the transfer market listing.
  const canSeeFinancials = player.teamId === 0 || userTeam?.team.id === player.teamId
  // A rostered player's skill ratings are likewise the owning manager's business, unless this
  // specific player is listed on the transfer market - the backend already zeroes the underlying
  // fields for anyone else (see common.Player.RedactSkills), so this just decides whether the
  // attribute panel/traits are worth rendering at all rather than showing an all-zero radar chart.
  const canSeeSkills = canSeeFinancials || player.transferListingAskingPrice !== undefined

  return (
    <Stack spacing={{ xs: 1.5, sm: 2 }}>
      <Grid container spacing={{ xs: 1.5, sm: 2 }}>
        <Grid size={{ xs: 12, md: 4 }}>
          <PlayerProfileRail player={player} canSeeFinancials={canSeeFinancials} canSeeSkills={canSeeSkills} />
        </Grid>
        {canSeeSkills && (
          <Grid size={{ xs: 12, md: 8 }}>
            <PlayerAttributesPanel player={player} />
          </Grid>
        )}
      </Grid>

      {canSeeFinancials && (
        <Stack spacing={1}>
          <Typography variant="h6">{t('playerPage.history.title')}</Typography>
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
        </Stack>
      )}
    </Stack>
  )
}
