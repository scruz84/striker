import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { PLAYER_TRANSFER_KIND_KEYS, type PlayerTransfer } from '../../api/types'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDate } from '../../utils/formatDate'

// A player's transfer history: one row per move (origin -> destination, type, price). Public
// market information, unlike the owning team's accounting ledger - see backend
// common.GetPlayerTransfers.
export function PlayerTransferHistory({ transfers }: { transfers: PlayerTransfer[] }) {
  const { t } = useTranslation()

  return (
    <Stack spacing={1}>
      <Typography variant="h6">{t('playerPage.history.title')}</Typography>
      <TableContainer component={Paper}>
        <Table size="small" aria-label="Transfer history">
          <TableHead>
            <TableRow>
              <TableCell>{t('playerPage.history.date')}</TableCell>
              <TableCell>{t('playerPage.history.route')}</TableCell>
              <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>{t('playerPage.history.kind')}</TableCell>
              <TableCell align="right">{t('playerPage.history.price')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {transfers.length === 0 && (
              <TableRow>
                <TableCell colSpan={4}>
                  <Typography variant="body2" color="text.secondary">
                    {t('playerPage.history.none')}
                  </Typography>
                </TableCell>
              </TableRow>
            )}
            {transfers.map((transfer) => (
              <TransferRow key={transfer.id} transfer={transfer} />
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
  )
}

function TransferRow({ transfer }: { transfer: PlayerTransfer }) {
  const { t } = useTranslation()
  const kind = t(PLAYER_TRANSFER_KIND_KEYS[transfer.kind] ?? 'finances.reasons.other')

  return (
    <TableRow>
      <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatDate(transfer.createdAt)}</TableCell>
      <TableCell>
        <TransferClub teamId={transfer.fromTeamId} name={transfer.fromTeamName} none={t('playerPage.history.freeAgent')} />
        {' → '}
        <TransferClub teamId={transfer.toTeamId} name={transfer.toTeamName} none={t('playerPage.history.released')} />
        <Typography variant="caption" color="text.secondary" component="div" sx={{ display: { xs: 'block', sm: 'none' } }}>
          {kind}
        </Typography>
      </TableCell>
      <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>{kind}</TableCell>
      <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
        {transfer.fee > 0 ? formatCurrency(transfer.fee) : '-'}
      </TableCell>
    </TableRow>
  )
}

// A club links to its page; a missing one (no origin = free agent, no destination = released)
// shows the given fallback label instead.
function TransferClub({ teamId, name, none }: { teamId?: number; name?: string; none: string }) {
  if (teamId === undefined) {
    return <>{none}</>
  }
  return <TeamNameLink teamId={teamId}>{name}</TeamNameLink>
}
