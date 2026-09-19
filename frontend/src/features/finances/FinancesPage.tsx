import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import FormControlLabel from '@mui/material/FormControlLabel'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Radio from '@mui/material/Radio'
import RadioGroup from '@mui/material/RadioGroup'
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
import { useFinances, useMe, useSetSponsorDeal, useSetTicketPrice, useUpgradeStadium } from '../../api/hooks'
import {
  SPONSOR_DEAL_BALANCED,
  SPONSOR_DEAL_DESCRIPTION_KEYS,
  SPONSOR_DEAL_LABEL_KEYS,
  SPONSOR_DEAL_PERFORMANCE,
  SPONSOR_DEAL_STEADY,
  TRANSACTION_REASON_KEYS,
  type SponsorDealOption,
  type Stadium,
} from '../../api/types'
import { useConfirm } from '../../components/confirmDialogContext'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { BudgetOverTimeChart } from './BudgetOverTimeChart'
import { CashFlowForecastCard } from './CashFlowForecastCard'
import { IncomeBreakdownChart } from './IncomeBreakdownChart'

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

      {finances.insolvent && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {t('finances.insolvent')}
        </Alert>
      )}

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

      <SponsorDealCard sponsorDeal={finances.sponsorDeal} options={finances.sponsorDealOptions} locked={finances.sponsorDealLocked} />

      <CashFlowForecastCard transactions={finances.transactions} budget={finances.budget} maxDebt={finances.maxDebt} />

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <BudgetOverTimeChart transactions={finances.transactions} budget={finances.budget} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <IncomeBreakdownChart transactions={finances.transactions} />
        </Grid>
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

// Lets the owner pick between a higher guaranteed per-match sponsorship payment (Steady) and a
// lower guaranteed payment made up for by a bigger win/draw bonus (Performance), with Balanced in
// between - see backend engine.sponsorshipAmount for the actual multipliers. Takes effect from
// either side's next match, same as ticket price.
function SponsorDealCard({
  sponsorDeal,
  options,
  locked,
}: {
  sponsorDeal: number
  options: SponsorDealOption[]
  locked: boolean
}) {
  const { t } = useTranslation()
  const setSponsorDeal = useSetSponsorDeal()
  const amountsByDeal = new Map(options.map((option) => [option.deal, option]))

  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        <Typography variant="overline" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
          {t('finances.sponsorDeal.title')}
        </Typography>
        {locked && (
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
            {t('finances.sponsorDeal.locked')}
          </Typography>
        )}
        <RadioGroup
          value={sponsorDeal}
          onChange={(event) => setSponsorDeal.mutate(Number(event.target.value))}
        >
          {[SPONSOR_DEAL_STEADY, SPONSOR_DEAL_BALANCED, SPONSOR_DEAL_PERFORMANCE].map((deal) => {
            const amounts = amountsByDeal.get(deal)
            return (
              <FormControlLabel
                key={deal}
                value={deal}
                disabled={setSponsorDeal.isPending || locked}
                sx={{ alignItems: 'flex-start', mb: 0.5 }}
                control={<Radio size="small" sx={{ pt: 0 }} />}
                label={
                  <Box>
                    <Typography variant="body2">{t(SPONSOR_DEAL_LABEL_KEYS[deal])}</Typography>
                    <Typography variant="caption" color="text.secondary">
                      {t(SPONSOR_DEAL_DESCRIPTION_KEYS[deal])}
                    </Typography>
                    {amounts && (
                      <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                        {t('finances.sponsorDeal.amounts', {
                          win: formatCurrency(amounts.win),
                          draw: formatCurrency(amounts.draw),
                          loss: formatCurrency(amounts.loss),
                        })}
                      </Typography>
                    )}
                  </Box>
                }
              />
            )
          })}
        </RadioGroup>
        {setSponsorDeal.isError && (
          <Alert severity="error" sx={{ mt: 1 }}>
            {t('finances.sponsorDeal.error')}
          </Alert>
        )}
      </CardContent>
    </Card>
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

  const upgradePending = stadium.pendingCapacity != null && stadium.upgradeCompletesAt != null

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="overline" color="text.secondary">
          {t('finances.stadium')}
        </Typography>
        <Typography variant="subtitle1">{stadium.name}</Typography>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'baseline' }}>
          <Typography variant="h4">{stadium.capacity.toLocaleString()}</Typography>
          <Typography variant="caption" color="text.secondary">
            {t('finances.seats')}
          </Typography>
        </Stack>
        {upgradePending ? (
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
            {t('finances.expandInProgress', {
              capacity: stadium.pendingCapacity?.toLocaleString(),
              date: formatDateTime(stadium.upgradeCompletesAt ?? ''),
            })}
          </Typography>
        ) : (
          <Button size="small" variant="text" sx={{ mb: 1, px: 0 }} onClick={handleUpgrade} disabled={upgradeStadium.isPending}>
            {t('finances.expand')}
          </Button>
        )}
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
