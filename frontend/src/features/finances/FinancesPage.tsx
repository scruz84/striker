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
import { useTranslation } from 'react-i18next'
import { useFinances, useMe, useSetTicketPrice, useUpgradeStadium } from '../../api/hooks'
import { TRANSACTION_REASON_KEYS, type Stadium } from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

export function FinancesPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: finances, isLoading } = useFinances(!!loggedUser)

  if (isLoading || !finances) {
    return <Skeleton variant="rectangular" height={300} />
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('finances.title')}
      </Typography>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="overline" color="text.secondary">
                {t('finances.budget')}
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
              <TableCell>{t('finances.table.date')}</TableCell>
              <TableCell>{t('finances.table.reason')}</TableCell>
              <TableCell>{t('finances.table.description')}</TableCell>
              <TableCell align="right">{t('finances.table.amount')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {finances.transactions.length === 0 && (
              <TableRow>
                <TableCell colSpan={4}>
                  <Typography variant="body2" color="text.secondary">
                    {t('finances.noTransactions')}
                  </Typography>
                </TableCell>
              </TableRow>
            )}
            {finances.transactions.map((tx) => (
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
  )
}

function StadiumCard({ stadium }: { stadium: Stadium }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const setTicketPrice = useSetTicketPrice()
  const upgradeStadium = useUpgradeStadium()
  const [price, setPrice] = useState(String(stadium.ticketPrice))

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const priceNum = Number(price)
    if (!priceNum) {
      return
    }
    setTicketPrice.mutate(priceNum)
  }

  const handleUpgrade = async () => {
    const ok = await confirm({ message: t('finances.confirmExpand'), confirmLabel: t('finances.expand') })
    if (ok) {
      upgradeStadium.mutate()
    }
  }

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="overline" color="text.secondary">
          {t('finances.stadium')}
        </Typography>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'baseline' }}>
          <Typography variant="h4">{stadium.capacity.toLocaleString()}</Typography>
          <Typography variant="caption" color="text.secondary">
            {t('finances.seats')}
          </Typography>
        </Stack>
        <Button size="small" variant="text" sx={{ mb: 1, px: 0 }} onClick={handleUpgrade} disabled={upgradeStadium.isPending}>
          {t('finances.expand')}
        </Button>
        {upgradeStadium.isSuccess && <Alert severity="success" sx={{ mb: 1 }}>{t('finances.expandSuccess')}</Alert>}
        {upgradeStadium.isError && <Alert severity="error" sx={{ mb: 1 }}>{t('finances.expandError')}</Alert>}

        <Box component="form" onSubmit={handleSubmit}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            <TextField
              label={t('finances.ticketPrice')}
              size="small"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              sx={{ width: 140 }}
            />
            <Button type="submit" size="small" variant="outlined" disabled={setTicketPrice.isPending}>
              {t('common.save')}
            </Button>
          </Stack>
          {setTicketPrice.isSuccess && <Alert severity="success" sx={{ mt: 1 }}>{t('finances.ticketPriceSuccess')}</Alert>}
          {setTicketPrice.isError && <Alert severity="error" sx={{ mt: 1 }}>{t('finances.ticketPriceError')}</Alert>}
        </Box>
      </CardContent>
    </Card>
  )
}
