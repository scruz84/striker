import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
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
import { useFinances, useMe, useSetTicketPrice } from '../../api/hooks'
import { TRANSACTION_LABELS, type Stadium } from '../../api/types'
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

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="overline" color="text.secondary">
                Budget
              </Typography>
              <Typography variant="h4" color={finances.budget < 0 ? 'error' : 'text.primary'}>
                {formatCurrency(finances.budget)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>{finances.stadium && <StadiumCard stadium={finances.stadium} />}</Grid>
      </Grid>

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

function StadiumCard({ stadium }: { stadium: Stadium }) {
  const setTicketPrice = useSetTicketPrice()
  const [price, setPrice] = useState(String(stadium.ticketPrice))

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const priceNum = Number(price)
    if (!priceNum) {
      return
    }
    setTicketPrice.mutate(priceNum)
  }

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="overline" color="text.secondary">
          Stadium
        </Typography>
        <Typography variant="h4">{stadium.capacity.toLocaleString()}</Typography>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
          seats
        </Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            <TextField
              label="Ticket price"
              size="small"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              sx={{ width: 140 }}
            />
            <Button type="submit" size="small" variant="outlined" disabled={setTicketPrice.isPending}>
              Save
            </Button>
          </Stack>
          {setTicketPrice.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>Ticket price updated.</Alert>}
          {setTicketPrice.isError && <Alert severity="error" sx={{ mt: 1 }}>Failed to update ticket price.</Alert>}
        </Box>
      </CardContent>
    </Card>
  )
}
