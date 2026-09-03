import Box from '@mui/material/Box'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { useFinances, useMe } from '../../api/hooks'
import { TRANSACTION_LABELS } from '../../api/types'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

export function FinancesPage() {
  const { data: loggedUser } = useMe()
  const { data: finances, isLoading } = useFinances(!!loggedUser)

  if (isLoading || !finances) {
    return <Skeleton variant="rectangular" height={300} />
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        Finances
      </Typography>

      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Typography variant="overline" color="text.secondary">
            Budget
          </Typography>
          <Typography variant="h4" color={finances.budget < 0 ? 'error' : 'text.primary'}>
            {formatCurrency(finances.budget)}
          </Typography>
        </CardContent>
      </Card>

      <TableContainer component={Paper}>
        <Table size="small" aria-label="Transactions">
          <TableHead>
            <TableRow>
              <TableCell>Date</TableCell>
              <TableCell>Reason</TableCell>
              <TableCell>Description</TableCell>
              <TableCell align="right">Amount</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {finances.transactions.length === 0 && (
              <TableRow>
                <TableCell colSpan={4}>
                  <Typography variant="body2" color="text.secondary">
                    No transactions yet.
                  </Typography>
                </TableCell>
              </TableRow>
            )}
            {finances.transactions.map((tx) => (
              <TableRow key={tx.id}>
                <TableCell>{formatDateTime(tx.createdAt)}</TableCell>
                <TableCell>{TRANSACTION_LABELS[tx.reason] ?? 'Other'}</TableCell>
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
  )
}
