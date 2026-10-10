import { useState, type ChangeEvent, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import EditIcon from '@mui/icons-material/Edit'
import FormControlLabel from '@mui/material/FormControlLabel'
import Grid from '@mui/material/Grid'
import IconButton from '@mui/material/IconButton'
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
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import { useFinances, useMe, useRenameStadium, useSetSponsorDeal, useSetTicketPrice, useUpgradeStadium } from '../../api/hooks'
import {
  calculateStadiumMaintenanceCost,
  calculateStadiumUpgradeCost,
  DEFAULT_STADIUM_UPGRADE_SEATS,
  MAX_STADIUM_UPGRADE_SEATS,
  MIN_STADIUM_UPGRADE_SEATS,
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
import { ratingTierColor } from '../../utils/playerRating'
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
        <Grid size={{ xs: 12, sm: 4 }}>
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
        <Grid size={{ xs: 12, sm: 4 }}>{finances.stadium && <StadiumCard stadium={finances.stadium} />}</Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
          <FanbaseCard fanbase={finances.fanbase} />
        </Grid>
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
  const confirm = useConfirm()
  const setSponsorDeal = useSetSponsorDeal()
  // Shows the just-clicked option as selected right away, while the confirm dialog below is
  // open - RadioGroup is controlled by sponsorDeal (the server-confirmed value), so without this
  // the radio would just sit on the old selection until (if) the choice is confirmed. Reset back
  // to null either way once the dialog settles, so the group falls back to reflecting sponsorDeal.
  const [pendingDeal, setPendingDeal] = useState<number | null>(null)
  const amountsByDeal = new Map(options.map((option) => [option.deal, option]))

  const handleChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const deal = Number(event.target.value)
    setPendingDeal(deal)
    const ok = await confirm({
      message: t('finances.sponsorDeal.confirmChange'),
      confirmLabel: t('finances.sponsorDeal.confirmLabel'),
    })
    if (ok) {
      setSponsorDeal.mutate(deal, { onSettled: () => setPendingDeal(null) })
    } else {
      setPendingDeal(null)
    }
  }

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
        <RadioGroup value={pendingDeal ?? sponsorDeal} onChange={handleChange}>
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

// FanbaseCard shows this team's current 0-99 supporter-base index (see backend common.GetFanbase)
// - read-only, unlike SponsorDealCard/StadiumCard, since Fanbase drifts on its own from match
// results rather than being something a manager directly sets.
function FanbaseCard({ fanbase }: { fanbase: number }) {
  const { t } = useTranslation()
  const theme = useTheme()

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="overline" color="text.secondary">
          {t('finances.fanbase.title')}
        </Typography>
        <Typography variant="h4" sx={{ color: ratingTierColor(fanbase, theme.palette) }}>
          {fanbase}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {t('finances.fanbase.description')}
        </Typography>
      </CardContent>
    </Card>
  )
}

function StadiumCard({ stadium }: { stadium: Stadium }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const setTicketPrice = useSetTicketPrice()
  const upgradeStadium = useUpgradeStadium()
  const renameStadium = useRenameStadium()
  const [price, setPrice] = useState(String(stadium.ticketPrice))
  const [seats, setSeats] = useState(String(DEFAULT_STADIUM_UPGRADE_SEATS))
  const [editingName, setEditingName] = useState(false)
  const [name, setName] = useState(stadium.name)

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const priceNum = Number(price)
    if (!priceNum) {
      return
    }
    setTicketPrice.mutate(priceNum)
  }

  const seatsNum = Number(seats)
  const seatsValid = Number.isInteger(seatsNum) && seatsNum >= MIN_STADIUM_UPGRADE_SEATS && seatsNum <= MAX_STADIUM_UPGRADE_SEATS
  const expandCost = seatsValid ? calculateStadiumUpgradeCost(stadium.capacity, seatsNum) : null

  const handleUpgrade = async () => {
    if (!seatsValid || expandCost == null) {
      return
    }
    const ok = await confirm({
      message: t('finances.confirmExpand', { seats: seatsNum, cost: formatCurrency(expandCost) }),
      confirmLabel: t('finances.expand'),
    })
    if (ok) {
      upgradeStadium.mutate(seatsNum)
    }
  }

  const handleRenameSubmit = (event: FormEvent) => {
    event.preventDefault()
    const trimmed = name.trim()
    if (!trimmed || trimmed === stadium.name) {
      setEditingName(false)
      return
    }
    renameStadium.mutate(trimmed, { onSuccess: () => setEditingName(false) })
  }

  const upgradePending = stadium.pendingCapacity != null && stadium.upgradeCompletesAt != null

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="overline" color="text.secondary">
          {t('finances.stadium')}
        </Typography>
        {editingName ? (
          <Box component="form" onSubmit={handleRenameSubmit} sx={{ mb: 0.5 }}>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
              <TextField
                label={t('finances.stadiumNameLabel')}
                size="small"
                autoFocus
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
              <Button type="submit" size="small" variant="outlined" disabled={renameStadium.isPending}>
                {t('common.save')}
              </Button>
            </Stack>
          </Box>
        ) : (
          <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
            <Typography variant="subtitle1">{stadium.name}</Typography>
            <IconButton
              size="small"
              aria-label={t('finances.renameStadium')}
              onClick={() => {
                setName(stadium.name)
                setEditingName(true)
              }}
            >
              <EditIcon fontSize="inherit" />
            </IconButton>
          </Stack>
        )}
        {renameStadium.isError && (
          <Alert severity="error" sx={{ mb: 1 }}>
            {t('finances.renameStadiumError')}
          </Alert>
        )}
        <Stack direction="row" spacing={1} sx={{ alignItems: 'baseline' }}>
          <Typography variant="h4">{stadium.capacity.toLocaleString()}</Typography>
          <Typography variant="caption" color="text.secondary">
            {t('finances.seats')}
          </Typography>
        </Stack>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
          {t('finances.maintenanceCost', { cost: formatCurrency(calculateStadiumMaintenanceCost(stadium.capacity)) })}
        </Typography>
        {upgradePending ? (
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
            {t('finances.expandInProgress', {
              capacity: stadium.pendingCapacity?.toLocaleString(),
              date: formatDateTime(stadium.upgradeCompletesAt ?? ''),
            })}
          </Typography>
        ) : (
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1 }}>
            <TextField
              label={t('finances.newSeats')}
              size="small"
              type="number"
              value={seats}
              onChange={(e) => setSeats(e.target.value)}
              error={!seatsValid}
              helperText={
                seatsValid
                  ? t('finances.expandCost', { cost: formatCurrency(expandCost ?? 0) })
                  : t('finances.expandSeatsRange', { min: MIN_STADIUM_UPGRADE_SEATS, max: MAX_STADIUM_UPGRADE_SEATS })
              }
              slotProps={{ htmlInput: { min: MIN_STADIUM_UPGRADE_SEATS, max: MAX_STADIUM_UPGRADE_SEATS } }}
              sx={{ width: 140 }}
            />
            <Button size="small" variant="text" onClick={handleUpgrade} disabled={upgradeStadium.isPending || !seatsValid}>
              {t('finances.expand')}
            </Button>
          </Stack>
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
